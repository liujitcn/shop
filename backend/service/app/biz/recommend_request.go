package biz

import (
	"context"
	"time"

	"shop/api/gen/go/app"
	"shop/api/gen/go/common"
	"shop/api/gen/go/conf"
	"shop/pkg/biz"
	"shop/pkg/configs"
	"shop/pkg/gen/data"
	recommendCache "shop/pkg/recommend/cache"
	recommendCandidate "shop/pkg/recommend/candidate"
	recommendCore "shop/pkg/recommend/core"
	recommendDomain "shop/pkg/recommend/domain"
	recommendEvent "shop/pkg/recommend/event"
	recommendOnlineFeature "shop/pkg/recommend/online/feature"
	recommendOnlinePlanner "shop/pkg/recommend/online/planner"
	recommendOnlineRank "shop/pkg/recommend/online/rank"
	recommendOnlineRecord "shop/pkg/recommend/online/record"
	appDto "shop/service/app/dto"

	"github.com/liujitcn/gorm-kit/repo"
)

// RecommendRequestCase 推荐请求子业务处理对象。
type RecommendRequestCase struct {
	*biz.BaseCase
	*data.RecommendRequestRepo
	recommendRequestItemCase         *RecommendRequestItemCase
	goodsInfoCase                    *GoodsInfoCase
	orderGoodsCase                   *OrderGoodsCase
	userCartCase                     *UserCartCase
	recommendExposureCase            *RecommendExposureCase
	recommendUserGoodsPreferenceCase *RecommendUserGoodsPreferenceCase
	recommendUserPreferenceCase      *RecommendUserPreferenceCase
	recommendGoodsRelationCase       *RecommendGoodsRelationCase
	recommendGoodsStatDayCase        *RecommendGoodsStatDayCase
	goodsStatDayCase                 *GoodsStatDayCase
	recommendModelVersionRepo        *data.RecommendModelVersionRepo
	recommendCacheStore              recommendCache.Store
	goodsRecommendConfig             *conf.GoodsRecommendConfig
}

// NewRecommendRequestCase 创建推荐请求子业务处理对象。
func NewRecommendRequestCase(
	baseCase *biz.BaseCase,
	shopConfig *conf.ShopConfig,
	recommendRequestRepo *data.RecommendRequestRepo,
	recommendRequestItemCase *RecommendRequestItemCase,
	goodsInfoCase *GoodsInfoCase,
	orderGoodsCase *OrderGoodsCase,
	userCartCase *UserCartCase,
	recommendExposureCase *RecommendExposureCase,
	recommendUserGoodsPreferenceCase *RecommendUserGoodsPreferenceCase,
	recommendUserPreferenceCase *RecommendUserPreferenceCase,
	recommendGoodsRelationCase *RecommendGoodsRelationCase,
	recommendGoodsStatDayCase *RecommendGoodsStatDayCase,
	goodsStatDayCase *GoodsStatDayCase,
	recommendModelVersionRepo *data.RecommendModelVersionRepo,
	recommendCacheStore recommendCache.Store,
) *RecommendRequestCase {
	goodsRecommendConfig := configs.ParseGoodsRecommendConfig(shopConfig)
	return &RecommendRequestCase{
		BaseCase:                         baseCase,
		RecommendRequestRepo:             recommendRequestRepo,
		recommendRequestItemCase:         recommendRequestItemCase,
		goodsInfoCase:                    goodsInfoCase,
		orderGoodsCase:                   orderGoodsCase,
		userCartCase:                     userCartCase,
		recommendExposureCase:            recommendExposureCase,
		recommendUserGoodsPreferenceCase: recommendUserGoodsPreferenceCase,
		recommendUserPreferenceCase:      recommendUserPreferenceCase,
		recommendGoodsRelationCase:       recommendGoodsRelationCase,
		recommendGoodsStatDayCase:        recommendGoodsStatDayCase,
		goodsStatDayCase:                 goodsStatDayCase,
		recommendModelVersionRepo:        recommendModelVersionRepo,
		recommendCacheStore:              recommendCacheStore,
		goodsRecommendConfig:             goodsRecommendConfig,
	}
}

// bindRecommendRequestActor 将匿名请求主体绑定为登录主体。
func (c *RecommendRequestCase) bindRecommendRequestActor(ctx context.Context, anonymousId, userId int64) error {
	query := c.RecommendRequestRepo.Data.Query(ctx).RecommendRequest
	_, err := query.WithContext(ctx).
		Where(
			query.ActorType.Eq(recommendEvent.ActorTypeAnonymous),
			query.ActorID.Eq(anonymousId),
		).
		Updates(map[string]interface{}{
			"actor_type": recommendEvent.ActorTypeUser,
			"actor_id":   userId,
		})
	return err
}

// listAnonymousRecommendGoods 查询匿名推荐商品列表并执行统一排序。
func (c *RecommendRequestCase) listAnonymousRecommendGoods(ctx context.Context, actor *appDto.RecommendActor, req *app.RecommendGoodsRequest) ([]*app.GoodsInfo, int64, []string, map[string]any, error) {
	requestPlan := recommendOnlinePlanner.NewAnonymousRequestPlan(buildRecommendGoodsRequest(req), map[string]any{})
	candidateLimit := requestPlan.CandidateLimit
	// 匿名态只使用近一段时间内的热度数据。
	startDate := time.Now().AddDate(0, 0, -recommendCandidate.AnonymousRecallDays)
	probeContext, err := c.buildRecommendRecallProbeContext(ctx, int32(req.GetScene()), 0, req.GetGoodsId(), candidateLimit, []int64{req.GetGoodsId()})
	if err != nil {
		return nil, 0, nil, nil, err
	}
	requestPlan = recommendOnlinePlanner.NewAnonymousRequestPlan(&requestPlan.Request, probeContext)

	sceneGoodsIds := make([]int64, 0)
	sceneInput := recommendOnlinePlanner.SceneInput{}
	sceneHotCacheResult, err := c.listCachedSceneHotGoodsIds(ctx, int32(req.GetScene()), candidateLimit, nil)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	requestPlan.MergeCacheReadContext(mergeRecommendCacheReadResult(nil, sceneHotCacheResult))
	sceneGoodsIds = sceneHotCacheResult.Ids
	// 场景热度缓存未命中时，回退到统计表查询。
	if len(sceneGoodsIds) == 0 {
		sceneGoodsIds, err = c.recommendGoodsStatDayCase.listSceneHotGoodsIds(ctx, req.GetScene(), startDate, candidateLimit)
		if err != nil {
			return nil, 0, nil, nil, err
		}
	} else {
		requestPlan.AddCacheHitSource(recommendCacheHitSceneHot)
	}

	// 商品详情场景优先使用当前商品做匿名关联召回。
	switch req.GetScene() {
	case common.RecommendScene_GOODS_DETAIL:
		// 没有商品编号时，无法恢复商品详情上下文。
		if req.GetGoodsId() > 0 {
			sourceGoodsIds := []int64{req.GetGoodsId()}
			similarItemCacheResult, cacheErr := c.listCachedSimilarItemGoodsIds(ctx, req.GetGoodsId(), candidateLimit, sourceGoodsIds)
			if cacheErr != nil {
				return nil, 0, nil, nil, cacheErr
			}
			requestPlan.MergeCacheReadContext(mergeRecommendCacheReadResult(nil, similarItemCacheResult))
			goodsDetailPriorityGoodsIds := similarItemCacheResult.Ids
			// 相似商品缓存未命中时，回退到数据库关系召回。
			if len(goodsDetailPriorityGoodsIds) == 0 {
				goodsDetailPriorityGoodsIds, err = c.recommendGoodsRelationCase.listRelatedGoodsIds(ctx, sourceGoodsIds, candidateLimit)
				if err != nil {
					return nil, 0, nil, nil, err
				}
			}
			goodsDetailCategoryIds, categoryErr := c.goodsInfoCase.listCategoryIdsByGoodsIds(ctx, sourceGoodsIds)
			if categoryErr != nil {
				return nil, 0, nil, nil, categoryErr
			}
			cacheHitSources := make([]string, 0, 1)
			// 相似商品缓存命中时，记录当前详情场景使用了缓存桥接结果。
			if len(similarItemCacheResult.Ids) > 0 {
				cacheHitSources = append(cacheHitSources, recommendCacheHitGoodsDetail)
			}
			sceneInput = recommendOnlinePlanner.BuildGoodsDetailSceneInput(sourceGoodsIds, goodsDetailPriorityGoodsIds, goodsDetailCategoryIds, cacheHitSources)
		}
	}
	requestPlan.ApplySceneInput(sceneInput)
	// 场景热度命中时，补充场景召回来源。
	if len(sceneGoodsIds) > 0 {
		requestPlan.AddRecallSources("scene_hot")
	}
	requestPlan.NormalizeState()

	categoryCandidateIds, err := c.pageCategoryCandidateGoodsIds(ctx, requestPlan.CategoryIds, requestPlan.ExcludeGoodsIds(), candidateLimit)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	requestPlan.SetCategoryCandidateGoodsIds(categoryCandidateIds)

	// 场景热度与类目候选合并后，再交给全站热度兜底补足。
	candidateGoodsIds := recommendCore.DedupeInt64s(requestPlan.BuildAnonymousMergedSceneGoodsIds(sceneGoodsIds))
	// 当前候选池仍未达到上限时，再补全站热度商品。
	if candidateLimit > 0 && int64(len(candidateGoodsIds)) < candidateLimit {
		globalHotGoodsIds, listErr := c.goodsStatDayCase.listGlobalHotGoodsIds(ctx, startDate, candidateLimit)
		if listErr != nil {
			return nil, 0, nil, nil, listErr
		}
		candidateGoodsIds = recommendCore.DedupeInt64s(append(candidateGoodsIds, globalHotGoodsIds...))
	}
	// 合并后的匿名候选超过上限时，直接按候选上限截断。
	if candidateLimit > 0 && int64(len(candidateGoodsIds)) > candidateLimit {
		candidateGoodsIds = candidateGoodsIds[:candidateLimit]
	}
	// 场景热度和类目补足都没有数据，且强召回也为空时，才退回最新商品分页。
	if shouldFallbackToAnonymousLatest(requestPlan, candidateGoodsIds) {
		latestExcludeGoodsIds := make([]int64, 0, 1)
		// 商品详情场景回退到 latest 时，同样排除当前详情商品。
		if requestPlan.IsGoodsDetail() && requestPlan.Request.GoodsId > 0 {
			latestExcludeGoodsIds = append(latestExcludeGoodsIds, requestPlan.Request.GoodsId)
		}
		latestCacheResult, cacheErr := c.listCachedLatestGoodsIds(ctx, int32(req.GetScene()), candidateLimit, latestExcludeGoodsIds)
		if cacheErr != nil {
			return nil, 0, nil, nil, cacheErr
		}
		requestPlan.MergeCacheReadContext(mergeRecommendCacheReadResult(nil, latestCacheResult))
		latestGoodsIds := latestCacheResult.Ids
		// 最新榜缓存命中时，直接按缓存顺序返回商品列表。
		if len(latestGoodsIds) > 0 {
			requestPlan.AddCacheHitSource(recommendCacheHitLatest)
			latestGoodsList, listErr := c.goodsInfoCase.listByGoodsIds(ctx, latestGoodsIds)
			if listErr != nil {
				return nil, 0, nil, nil, listErr
			}
			sourceContext := requestPlan.BuildAnonymousLatestResultSourceContext(sceneInput, sceneGoodsIds, probeContext)
			return latestGoodsList, int64(len(latestGoodsList)), []string{"latest"}, sourceContext, nil
		}
		latestGoodsList, latestTotal, listErr := c.pageLatestFallbackGoods(ctx, latestExcludeGoodsIds, candidateLimit)
		if listErr != nil {
			return nil, 0, nil, nil, listErr
		}
		sourceContext := requestPlan.BuildAnonymousLatestResultSourceContext(sceneInput, sceneGoodsIds, probeContext)
		return latestGoodsList, latestTotal, []string{"latest"}, sourceContext, nil
	}
	// 强召回商品优先排在匿名候选池前面，再做统一去重。
	candidateGoodsIds = requestPlan.BuildAnonymousCandidateGoodsIds(candidateGoodsIds)

	goodsList, err := c.goodsInfoCase.listByGoodsIds(ctx, candidateGoodsIds)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	signalSnapshot := recommendOnlineFeature.BuildAnonymousSignalSnapshot(&requestPlan.Request, goodsList)
	filteredGoodsList := signalSnapshot.GoodsList
	// 后续排序信号加载统一复用 feature 给出的参数计划。
	signalLoadPlan := recommendOnlineFeature.BuildAnonymousSignalLoadPlan(&requestPlan.Request, signalSnapshot)
	// 后续的热度分、曝光惩罚都会按商品 ID 回填。
	candidateGoodsIds = signalLoadPlan.CandidateGoodsIds

	relationScores := make(map[int64]float64)
	// 商品详情场景存在源商品时，补充匿名关联分数。
	if len(signalLoadPlan.RelationSourceGoodsIds) > 0 {
		relationScores, err = c.recommendGoodsRelationCase.loadRelationScores(ctx, signalLoadPlan.RelationSourceGoodsIds)
		if err != nil {
			return nil, 0, nil, nil, err
		}
	}
	scenePopularityScores, sceneExposurePenalties, err := c.recommendGoodsStatDayCase.loadScenePopularitySignals(ctx, signalLoadPlan.Scene, candidateGoodsIds)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	actorExposurePenalties, err := c.recommendExposureCase.loadActorExposurePenalties(ctx, actor, signalLoadPlan.Scene, candidateGoodsIds)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	globalPopularityScores, err := c.goodsStatDayCase.loadGlobalPopularityScores(ctx, candidateGoodsIds)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	anonymousSignals := recommendOnlineFeature.BuildAnonymousSignals(
		relationScores,
		scenePopularityScores,
		globalPopularityScores,
		sceneExposurePenalties,
		actorExposurePenalties,
	)

	// 匿名态不看用户偏好，只使用公共排序信号。
	candidates := recommendCandidate.BuildAnonymous(filteredGoodsList, anonymousSignals, c.goodsRecommendConfig.GetAnonymousRank())
	// 商品详情页的内容相似灰度召回需要显式补到 explain 来源里。
	requestPlan.AppendAnonymousExplainRecallSources(candidates)
	// 这里不仅排序，还会顺带做类目打散。
	rankedGoods := recommendCandidate.RankGoods(candidates)
	pageSnapshot := recommendOnlineRank.BuildRankedPageSnapshot(&requestPlan.Request, rankedGoods)
	// 分页偏移超出候选集范围时，直接返回空页。
	if pageSnapshot.IsEmptyPage {
		sourceContext := requestPlan.BuildAnonymousEmptyOnlineResultContext(sceneInput, sceneGoodsIds, candidateGoodsIds, signalLoadPlan.CandidateGoodsIds, probeContext)
		return []*app.GoodsInfo{}, pageSnapshot.Total, requestPlan.RecallSources, sourceContext, nil
	}
	pageGoods := pageSnapshot.PageGoods
	// explain 只收集当前页，避免响应上下文过大。
	explainSnapshot := recommendOnlineRank.BuildPageExplainSnapshot(pageGoods, candidates)
	sourceContext := requestPlan.BuildAnonymousPageOnlineResultContext(sceneInput, sceneGoodsIds, candidateGoodsIds, signalLoadPlan.CandidateGoodsIds, explainSnapshot, probeContext)
	return pageGoods, pageSnapshot.Total, explainSnapshot.RecallSources, sourceContext, nil
}

// listRecommendGoods 查询推荐商品列表并执行统一排序。
func (c *RecommendRequestCase) listRecommendGoods(
	ctx context.Context,
	actor *appDto.RecommendActor,
	req *app.RecommendGoodsRequest,
	userId int64,
) ([]*app.GoodsInfo, int64, []string, map[string]any, error) {
	pageSize := req.GetPageSize()
	// 每页数量非法时，直接返回空结果避免继续构造候选集。
	if pageSize <= 0 {
		return []*app.GoodsInfo{}, 0, []string{}, map[string]any{}, nil
	}

	requestPlan := recommendOnlinePlanner.NewPersonalizedRequestPlan(buildRecommendGoodsRequest(req), map[string]any{})
	candidateLimit := requestPlan.CandidateLimit
	probeContext, err := c.buildRecommendRecallProbeContext(ctx, int32(req.GetScene()), userId, req.GetGoodsId(), candidateLimit, []int64{req.GetGoodsId()})
	if err != nil {
		return nil, 0, nil, nil, err
	}
	requestPlan = recommendOnlinePlanner.NewPersonalizedRequestPlan(&requestPlan.Request, probeContext)
	// 相似用户当前仍只做观测，因此单独拉取一份偏好商品用于效果比对。
	if len(requestPlan.SimilarUserIds) > 0 {
		similarUserObservedGoodsIds, listErr := c.recommendUserGoodsPreferenceCase.listObservedGoodsIdsByUserIds(ctx, requestPlan.SimilarUserIds, candidateLimit, []int64{req.GetGoodsId()})
		if listErr != nil {
			return nil, 0, nil, nil, listErr
		}
		requestPlan.SetSimilarUserObservedGoodsIds(similarUserObservedGoodsIds)
	}
	profileCategoryIds, err := c.recommendUserPreferenceCase.listPreferredCategoryIds(ctx, userId, 3)
	if err != nil {
		return nil, 0, nil, nil, err
	}

	// 登录态优先消费当前页面最强的业务上下文。
	sceneInput := recommendOnlinePlanner.SceneInput{}
	// 按当前推荐场景决定优先使用哪类业务上下文做召回。
	switch req.GetScene() {
	case common.RecommendScene_CART:
		cartGoodsIds, listErr := c.userCartCase.listGoodsIdsByUserId(ctx, userId)
		if listErr != nil {
			return nil, 0, nil, nil, listErr
		}
		sceneInput = recommendOnlinePlanner.BuildCartSceneInput(cartGoodsIds, nil, nil)
		// 购物车存在商品时，继续做购物车关联召回。
		if len(cartGoodsIds) > 0 {
			cartPriorityGoodsIds, relatedErr := c.recommendGoodsRelationCase.listRelatedGoodsIds(ctx, cartGoodsIds, pageSize)
			if relatedErr != nil {
				return nil, 0, nil, nil, relatedErr
			}
			cartCategoryIds, categoryErr := c.goodsInfoCase.listCategoryIdsByGoodsIds(ctx, cartGoodsIds)
			if categoryErr != nil {
				return nil, 0, nil, nil, categoryErr
			}
			sceneInput = recommendOnlinePlanner.BuildCartSceneInput(cartGoodsIds, cartPriorityGoodsIds, cartCategoryIds)
		}
	case common.RecommendScene_ORDER_DETAIL, common.RecommendScene_ORDER_PAID:
		// 存在订单编号时，继续做订单关联召回。
		if req.GetOrderId() > 0 {
			orderGoodsIds, listErr := c.orderGoodsCase.listGoodsIdsByOrderId(ctx, req.GetOrderId())
			if listErr != nil {
				return nil, 0, nil, nil, listErr
			}
			sceneInput = recommendOnlinePlanner.BuildOrderSceneInput(orderGoodsIds, nil, nil)
			orderPriorityGoodsIds, relatedErr := c.recommendGoodsRelationCase.listRelatedGoodsIds(ctx, orderGoodsIds, pageSize)
			if relatedErr != nil {
				return nil, 0, nil, nil, relatedErr
			}
			orderCategoryIds, categoryErr := c.goodsInfoCase.listCategoryIdsByGoodsIds(ctx, orderGoodsIds)
			if categoryErr != nil {
				return nil, 0, nil, nil, categoryErr
			}
			sceneInput = recommendOnlinePlanner.BuildOrderSceneInput(orderGoodsIds, orderPriorityGoodsIds, orderCategoryIds)
		}
	case common.RecommendScene_GOODS_DETAIL:
		// 存在商品编号时，继续做商品关联召回。
		if req.GetGoodsId() > 0 {
			sourceGoodsIds := []int64{req.GetGoodsId()}
			similarItemCacheResult, cacheErr := c.listCachedSimilarItemGoodsIds(ctx, req.GetGoodsId(), candidateLimit, sourceGoodsIds)
			if cacheErr != nil {
				return nil, 0, nil, nil, cacheErr
			}
			requestPlan.MergeCacheReadContext(mergeRecommendCacheReadResult(nil, similarItemCacheResult))
			goodsDetailPriorityGoodsIds := similarItemCacheResult.Ids
			// 相似商品缓存未命中时，回退到数据库关系召回。
			if len(goodsDetailPriorityGoodsIds) == 0 {
				goodsDetailPriorityGoodsIds, err = c.recommendGoodsRelationCase.listRelatedGoodsIds(ctx, sourceGoodsIds, candidateLimit)
				if err != nil {
					return nil, 0, nil, nil, err
				}
			}
			goodsDetailCategoryIds, categoryErr := c.goodsInfoCase.listCategoryIdsByGoodsIds(ctx, sourceGoodsIds)
			if categoryErr != nil {
				return nil, 0, nil, nil, categoryErr
			}
			cacheHitSources := make([]string, 0, 1)
			// 相似商品缓存命中时，记录当前详情场景使用了缓存桥接结果。
			if len(similarItemCacheResult.Ids) > 0 {
				cacheHitSources = append(cacheHitSources, recommendCacheHitGoodsDetail)
			}
			sceneInput = recommendOnlinePlanner.BuildGoodsDetailSceneInput(sourceGoodsIds, goodsDetailPriorityGoodsIds, goodsDetailCategoryIds, cacheHitSources)
		}
	}
	requestPlan.ApplySceneInput(sceneInput)

	// 用户画像只负责补足，不覆盖强场景召回。
	if len(profileCategoryIds) > 0 {
		requestPlan.ApplyProfileScene(profileCategoryIds)
	}
	// 没有命中任何召回入口时，统一回退到 latest。
	requestPlan.EnsureFallbackLatest()
	// 这里统一去重，避免同一商品或类目重复参与候选计算。
	requestPlan.NormalizeState()

	categoryCandidateIds, err := c.pageCategoryCandidateGoodsIds(ctx, requestPlan.CategoryIds, requestPlan.ExcludeGoodsIds(), candidateLimit)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	requestPlan.SetCategoryCandidateGoodsIds(categoryCandidateIds)

	latestCandidateIds := make([]int64, 0)
	// 候选池仍可扩充时，继续用 latest 召回做兜底补足。
	if candidateLimit > 0 {
		latestExcludeGoodsIds := requestPlan.BuildLatestExcludeGoodsIds()
		latestCacheResult, cacheErr := c.listCachedLatestGoodsIds(ctx, int32(req.GetScene()), candidateLimit, latestExcludeGoodsIds)
		if cacheErr != nil {
			return nil, 0, nil, nil, cacheErr
		}
		requestPlan.MergeCacheReadContext(mergeRecommendCacheReadResult(nil, latestCacheResult))
		latestCandidateIds = latestCacheResult.Ids
		// 最新榜缓存未命中时，回退到数据库最新商品分页。
		if len(latestCandidateIds) == 0 {
			latestCandidateIds, err = c.pageLatestCandidateGoodsIds(ctx, latestExcludeGoodsIds, candidateLimit)
			if err != nil {
				return nil, 0, nil, nil, err
			}
		} else {
			requestPlan.AddCacheHitSource(recommendCacheHitLatest)
		}
	}
	requestPlan.SetLatestCandidateGoodsIds(latestCandidateIds)

	// 最终候选池按“强召回 + 类目补足 + latest 兜底”合并。
	allCandidateIds := requestPlan.BuildPersonalizedCandidateGoodsIds()
	// 候选商品池为空时，直接返回空结果。
	if len(allCandidateIds) == 0 {
		sourceContext := requestPlan.BuildPersonalizedOnlineResultContext(sceneInput, recommendOnlinePlanner.ResultSnapshot{}, []int64{}, []int64{}, probeContext)
		return []*app.GoodsInfo{}, 0, []string{}, sourceContext, nil
	}

	goodsList, err := c.goodsInfoCase.listByGoodsIds(ctx, allCandidateIds)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	signalSnapshot := recommendOnlineFeature.BuildPersonalizedSignalSnapshot(goodsList)
	signalLoadPlan := recommendOnlineFeature.BuildPersonalizedSignalLoadPlan(&requestPlan.Request, requestPlan.PriorityGoodsIds, signalSnapshot)
	// 这份商品 ID 用来对齐各种商品级排序信号。
	candidateGoodsIds := signalLoadPlan.CandidateGoodsIds
	// 这份类目 ID 用来对齐画像类偏好分。
	candidateCategoryIds := signalLoadPlan.CandidateCategoryIds

	relationScores, err := c.recommendGoodsRelationCase.loadRelationScores(ctx, signalLoadPlan.RelationSourceGoodsIds)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	userGoodsScores, recentPaidGoodsMap, err := c.recommendUserGoodsPreferenceCase.loadUserGoodsSignals(ctx, userId, candidateGoodsIds)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	actorExposurePenalties, err := c.recommendExposureCase.loadActorExposurePenalties(ctx, actor, signalLoadPlan.Scene, candidateGoodsIds)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	profileScores, err := c.recommendUserPreferenceCase.loadProfileScores(ctx, userId, candidateCategoryIds)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	scenePopularityScores, sceneExposurePenalties, err := c.recommendGoodsStatDayCase.loadScenePopularitySignals(ctx, signalLoadPlan.Scene, candidateGoodsIds)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	globalPopularityScores, err := c.goodsStatDayCase.loadGlobalPopularityScores(ctx, candidateGoodsIds)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	personalizedSignals := recommendOnlineFeature.BuildPersonalizedSignals(
		relationScores,
		userGoodsScores,
		profileScores,
		scenePopularityScores,
		globalPopularityScores,
		sceneExposurePenalties,
		actorExposurePenalties,
		recentPaidGoodsMap,
	)

	// 登录态会融合关系分、偏好分、热度分和惩罚分。
	candidates := recommendCandidate.BuildPersonalized(signalSnapshot.GoodsList, personalizedSignals, c.goodsRecommendConfig.GetPersonalizedRank())
	// 商品详情页的灰度召回需要显式补到 explain 来源里。
	requestPlan.AppendPersonalizedExplainRecallSources(candidates)
	// 这里同时完成最终排序和类目去扎堆。
	rankedGoods := recommendCandidate.RankGoods(candidates)
	pageSnapshot := recommendOnlineRank.BuildRankedPageSnapshot(&requestPlan.Request, rankedGoods)
	// 分页偏移超出候选集范围时，直接返回空页但保留总数。
	if pageSnapshot.IsEmptyPage {
		sourceContext := requestPlan.BuildPersonalizedEmptyOnlineResultContext(sceneInput, candidateGoodsIds, probeContext)
		return []*app.GoodsInfo{}, pageSnapshot.Total, []string{}, sourceContext, nil
	}
	pageGoods := pageSnapshot.PageGoods
	// 当前页才需要 explain，整池 explain 没必要返回。
	explainSnapshot := recommendOnlineRank.BuildPageExplainSnapshot(pageGoods, candidates)
	sourceContext := requestPlan.BuildPersonalizedPageOnlineResultContext(sceneInput, candidateGoodsIds, explainSnapshot, probeContext)
	return pageGoods, pageSnapshot.Total, explainSnapshot.RecallSources, sourceContext, nil
}

// executeRecommendGoods 执行统一推荐主流程。
func (c *RecommendRequestCase) executeRecommendGoods(ctx context.Context, actor *appDto.RecommendActor, req *app.RecommendGoodsRequest) (recommendDomain.PageResult, error) {
	domainActor := toRecommendDomainActor(actor)
	// 匿名主体统一走匿名推荐主链路，减少各场景内容分裂。
	if domainActor.IsAnonymous() {
		list, total, recallSources, sourceContext, err := c.listAnonymousRecommendGoods(ctx, actor, req)
		if err != nil {
			return recommendDomain.PageResult{}, err
		}
		return recommendDomain.PageResult{
			List:          list,
			Total:         total,
			RecallSources: recallSources,
			SourceContext: sourceContext,
		}, nil
	}

	userId := int64(0)
	// 当前主体存在时，继续提取登录用户编号给召回探针使用。
	if actor != nil {
		userId = actor.UserId()
	}
	list, total, recallSources, sourceContext, err := c.listRecommendGoods(ctx, actor, req, userId)
	if err != nil {
		return recommendDomain.PageResult{}, err
	}
	return recommendDomain.PageResult{
		List:          list,
		Total:         total,
		RecallSources: recallSources,
		SourceContext: sourceContext,
	}, nil
}

// pageCategoryCandidateGoodsIds 按类目补足查询候选商品编号。
func (c *RecommendRequestCase) pageCategoryCandidateGoodsIds(ctx context.Context, categoryIds []int64, excludeGoodsIds []int64, limit int64) ([]int64, error) {
	// 查询条件未启用时，直接返回空集合。
	if limit <= 0 || len(categoryIds) == 0 {
		return []int64{}, nil
	}
	query := c.goodsInfoCase.GoodsInfoRepo.Query(ctx).GoodsInfo
	opts := make([]repo.QueryOption, 0, 1)
	opts = append(opts, repo.Where(query.CategoryID.In(categoryIds...)))
	pageResp, err := c.pageRecommendGoods(ctx, limit, excludeGoodsIds, opts...)
	if err != nil {
		return nil, err
	}
	return recommendOnlineRank.ListGoodsIds(pageResp.List), nil
}

// pageLatestCandidateGoodsIds 按 latest 兜底查询候选商品编号。
func (c *RecommendRequestCase) pageLatestCandidateGoodsIds(ctx context.Context, excludeGoodsIds []int64, limit int64) ([]int64, error) {
	// 查询条件未启用时，直接返回空集合。
	if limit <= 0 {
		return []int64{}, nil
	}
	pageResp, err := c.pageRecommendGoods(ctx, limit, excludeGoodsIds)
	if err != nil {
		return nil, err
	}
	return recommendOnlineRank.ListGoodsIds(pageResp.List), nil
}

// pageLatestFallbackGoods 按 latest 回退查询商品列表。
func (c *RecommendRequestCase) pageLatestFallbackGoods(ctx context.Context, excludeGoodsIds []int64, limit int64) ([]*app.GoodsInfo, int64, error) {
	// 查询条件未启用时，直接返回空结果。
	if limit <= 0 {
		return []*app.GoodsInfo{}, 0, nil
	}
	pageResp, err := c.pageRecommendGoods(ctx, limit, excludeGoodsIds)
	if err != nil {
		return nil, 0, err
	}
	return pageResp.List, int64(pageResp.Total), nil
}

// pageRecommendGoods 执行推荐候选分页查询。
func (c *RecommendRequestCase) pageRecommendGoods(ctx context.Context, limit int64, excludeGoodsIds []int64, extraOpts ...repo.QueryOption) (*app.PageGoodsInfoResponse, error) {
	// 查询上限非法时，直接返回空分页结果。
	if limit <= 0 {
		return &app.PageGoodsInfoResponse{
			List:  []*app.GoodsInfo{},
			Total: 0,
		}, nil
	}
	query := c.goodsInfoCase.GoodsInfoRepo.Query(ctx).GoodsInfo
	opts := make([]repo.QueryOption, 0, len(extraOpts)+1)
	opts = append(opts, extraOpts...)
	// 当前存在排除商品时，不再让这些商品进入当前候选分页结果。
	if len(excludeGoodsIds) > 0 {
		opts = append(opts, repo.Where(query.ID.NotIn(excludeGoodsIds...)))
	}
	return c.goodsInfoCase.PageGoodsInfo(ctx, &app.PageGoodsInfoRequest{
		PageNum:  1,
		PageSize: limit,
	}, opts...)
}

// buildRecommendGoodsRequest 将接口请求桥接为推荐领域请求对象。
func buildRecommendGoodsRequest(req *app.RecommendGoodsRequest) *recommendDomain.GoodsRequest {
	if req == nil {
		return &recommendDomain.GoodsRequest{}
	}
	return &recommendDomain.GoodsRequest{
		Scene:    req.GetScene(),
		OrderId:  req.GetOrderId(),
		GoodsId:  req.GetGoodsId(),
		PageNum:  req.GetPageNum(),
		PageSize: req.GetPageSize(),
	}
}

// saveRecommendRequest 保存推荐请求记录。
func (c *RecommendRequestCase) saveRecommendRequest(ctx context.Context, requestId string, actor *appDto.RecommendActor, req *app.RecommendGoodsRequest, sourceContext map[string]any, list []*app.GoodsInfo, recallSources []string) error {
	createdAt := time.Now()
	entity, err := recommendOnlineRecord.BuildRecommendRequestEntity(
		requestId,
		toRecommendDomainActor(actor),
		buildRecommendGoodsRequest(req),
		sourceContext,
		createdAt,
	)
	if err != nil {
		return err
	}
	return c.RecommendRequestRepo.Data.Transaction(ctx, func(ctx context.Context) error {
		err = c.RecommendRequestRepo.Create(ctx, entity)
		if err != nil {
			return err
		}
		return c.recommendRequestItemCase.batchCreateByRecommendRequest(ctx, entity.ID, req, sourceContext, list, recallSources)
	})
}

// toRecommendDomainActor 将应用层推荐主体桥接为领域主体对象。
func toRecommendDomainActor(actor *appDto.RecommendActor) *recommendDomain.Actor {
	// 主体为空时，不再继续构建领域主体对象。
	if actor == nil {
		return nil
	}
	return &recommendDomain.Actor{
		ActorType: actor.ActorType,
		ActorId:   actor.ActorId,
	}
}

// shouldFallbackToAnonymousLatest 判断匿名态是否需要回退到 latest。
func shouldFallbackToAnonymousLatest(plan *recommendOnlinePlanner.RequestPlan, candidateGoodsIds []int64) bool {
	// 计划对象为空时，只要当前候选为空就允许回退。
	if plan == nil {
		return len(candidateGoodsIds) == 0
	}
	return len(candidateGoodsIds) == 0 && len(plan.PriorityGoodsIds) == 0
}

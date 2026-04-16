package biz

import (
	"context"
	"encoding/json"
	"shop/api/gen/go/app"
	"shop/api/gen/go/common"
	"shop/api/gen/go/conf"
	"shop/pkg/biz"
	"shop/pkg/configs"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	recommendCache "shop/pkg/recommend/cache"
	recommendCandidate "shop/pkg/recommend/candidate"
	recommendDomain "shop/pkg/recommend/domain"
	recommendEvent "shop/pkg/recommend/event"
	recommendOnlinePlanner "shop/pkg/recommend/online/planner"
	recommendOnlineRecord "shop/pkg/recommend/online/record"
	appDto "shop/service/app/dto"
	"time"

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
	requestPlan := recommendOnlinePlanner.NewAnonymousRequestPlan(&recommendDomain.GoodsRequest{
		Scene:    req.GetScene(),
		OrderId:  req.GetOrderId(),
		GoodsId:  req.GetGoodsId(),
		PageNum:  req.GetPageNum(),
		PageSize: req.GetPageSize(),
	}, map[string]any{})
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
	sceneHotCacheResult := &recommendCacheReadResult{}
	sceneHotCacheResult, err = c.listCachedSceneHotGoodsIds(ctx, int32(req.GetScene()), candidateLimit, nil)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	requestPlan.MergeCacheReadContext(mergeRecommendCacheReadResult(nil, sceneHotCacheResult))
	sceneGoodsIds = sceneHotCacheResult.ids
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
			sourceGoodsIdList := []int64{req.GetGoodsId()}
			similarItemCacheResult := &recommendCacheReadResult{}
			similarItemCacheResult, err = c.listCachedSimilarItemGoodsIds(ctx, req.GetGoodsId(), candidateLimit, []int64{req.GetGoodsId()})
			if err != nil {
				return nil, 0, nil, nil, err
			}
			requestPlan.MergeCacheReadContext(mergeRecommendCacheReadResult(nil, similarItemCacheResult))
			goodsDetailPriorityGoodsIds := similarItemCacheResult.ids
			// 相似商品缓存未命中时，回退到数据库关系召回。
			if len(goodsDetailPriorityGoodsIds) == 0 {
				goodsDetailPriorityGoodsIds, err = c.recommendGoodsRelationCase.listRelatedGoodsIds(ctx, sourceGoodsIdList, candidateLimit)
				if err != nil {
					return nil, 0, nil, nil, err
				}
			}
			goodsDetailCategoryIds := make([]int64, 0)
			goodsDetailCategoryIds, err = c.goodsInfoCase.listCategoryIdsByGoodsIds(ctx, sourceGoodsIdList)
			if err != nil {
				return nil, 0, nil, nil, err
			}
			cacheHitSources := make([]string, 0, 1)
			// 相似商品缓存命中时，记录当前详情场景使用了缓存桥接结果。
			if len(similarItemCacheResult.ids) > 0 {
				cacheHitSources = append(cacheHitSources, recommendCacheHitGoodsDetail)
			}
			sceneInput = recommendOnlinePlanner.BuildGoodsDetailSceneInput(sourceGoodsIdList, goodsDetailPriorityGoodsIds, goodsDetailCategoryIds, cacheHitSources)
		}
	}
	requestPlan.ApplySceneInput(sceneInput)
	// 场景热度命中时，补充场景召回来源。
	if len(sceneGoodsIds) > 0 {
		requestPlan.AddRecallSources("scene_hot")
	}
	requestPlan.NormalizeState()
	categoryCandidateIdList := make([]int64, 0)
	categoryQuery := requestPlan.BuildCategoryCandidateQuery()
	// 存在类目候选时，按类目继续补足匿名候选池。
	if categoryQuery.Enabled {
		categoryCandidateIdList, err = c.pageCategoryCandidateGoodsIds(ctx, categoryQuery)
		if err != nil {
			return nil, 0, nil, nil, err
		}
	}
	requestPlan.SetCategoryCandidateGoodsIds(categoryCandidateIdList)
	// 场景热度与类目候选合并后，再交给全站热度兜底补足。
	mergedSceneGoodsIds := requestPlan.BuildAnonymousMergedSceneGoodsIds(sceneGoodsIds)
	candidateGoodsIds := make([]int64, 0)
	candidateGoodsIds, err = c.goodsStatDayCase.mergeAnonymousGoodsIds(ctx, mergedSceneGoodsIds, startDate, candidateLimit)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	// 场景热度和类目补足都没有数据，且强召回也为空时，才退回最新商品分页。
	if requestPlan.ShouldFallbackToAnonymousLatest(candidateGoodsIds) {
		latestQuery := requestPlan.BuildAnonymousLatestFallbackQuery()
		latestCacheResult, cacheErr := c.listCachedLatestGoodsIds(ctx, int32(req.GetScene()), latestQuery.Limit, latestQuery.ExcludeGoodsIds)
		if cacheErr != nil {
			return nil, 0, nil, nil, cacheErr
		}
		requestPlan.MergeCacheReadContext(mergeRecommendCacheReadResult(nil, latestCacheResult))
		latestGoodsIds := latestCacheResult.ids
		// 最新榜缓存命中时，直接按缓存顺序返回商品列表。
		if len(latestGoodsIds) > 0 {
			requestPlan.AddCacheHitSource(recommendCacheHitLatest)
			latestGoodsList := make([]*app.GoodsInfo, 0)
			latestGoodsList, err = c.goodsInfoCase.listByGoodsIds(ctx, latestGoodsIds)
			if err != nil {
				return nil, 0, nil, nil, err
			}
			sourceContext := requestPlan.BuildAnonymousLatestResultSourceContext(sceneInput, sceneGoodsIds, probeContext)
			return latestGoodsList, int64(len(latestGoodsList)), []string{"latest"}, sourceContext, nil
		}
		latestGoodsList := make([]*app.GoodsInfo, 0)
		latestTotal := int64(0)
		latestGoodsList, latestTotal, err = c.pageLatestFallbackGoods(ctx, latestQuery)
		if err != nil {
			return nil, 0, nil, nil, err
		}
		sourceContext := requestPlan.BuildAnonymousLatestResultSourceContext(sceneInput, sceneGoodsIds, probeContext)
		return latestGoodsList, latestTotal, []string{"latest"}, sourceContext, nil
	}
	// 强召回商品优先排在匿名候选池前面，再做统一去重。
	candidateGoodsIds = requestPlan.BuildAnonymousCandidateGoodsIds(candidateGoodsIds)

	goodsList := make([]*app.GoodsInfo, 0)
	goodsList, err = c.goodsInfoCase.listByGoodsIds(ctx, candidateGoodsIds)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	signalSnapshot := requestPlan.BuildAnonymousSignalSnapshot(goodsList)
	filteredGoodsList := signalSnapshot.GoodsList
	// 后续排序信号加载统一复用 planner 给出的参数计划。
	signalLoadPlan := requestPlan.BuildAnonymousSignalLoadPlan(signalSnapshot)
	// 后续的热度分、曝光惩罚都会按商品 ID 回填。
	candidateGoodsIdList := signalLoadPlan.CandidateGoodsIds

	relationScores := make(map[int64]float64)
	// 商品详情场景存在源商品时，补充匿名关联分数。
	if len(signalLoadPlan.RelationSourceGoodsIds) > 0 {
		relationScores, err = c.recommendGoodsRelationCase.loadRelationScores(ctx, signalLoadPlan.RelationSourceGoodsIds)
		if err != nil {
			return nil, 0, nil, nil, err
		}
	}
	scenePopularityScores := make(map[int64]float64)
	sceneExposurePenalties := make(map[int64]float64)
	scenePopularityScores, sceneExposurePenalties, err = c.recommendGoodsStatDayCase.loadScenePopularitySignals(ctx, signalLoadPlan.Scene, candidateGoodsIdList)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	actorExposurePenalties := make(map[int64]float64)
	actorExposurePenalties, err = c.recommendExposureCase.loadActorExposurePenalties(ctx, actor, signalLoadPlan.Scene, candidateGoodsIdList)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	globalPopularityScores := make(map[int64]float64)
	globalPopularityScores, err = c.goodsStatDayCase.loadGlobalPopularityScores(ctx, candidateGoodsIdList)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	anonymousSignals := requestPlan.BuildAnonymousSignals(
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
	pageSnapshot := requestPlan.BuildRankedPageSnapshot(rankedGoods)
	// 分页偏移超出候选集范围时，直接返回空页。
	if pageSnapshot.IsEmptyPage {
		sourceContext := requestPlan.BuildAnonymousEmptyOnlineResultContext(sceneInput, sceneGoodsIds, candidateGoodsIds, candidateGoodsIdList, probeContext)
		return []*app.GoodsInfo{}, pageSnapshot.Total, requestPlan.RecallSources, sourceContext, nil
	}
	pageGoods := pageSnapshot.PageGoods
	// explain 只收集当前页，避免响应上下文过大。
	explainSnapshot := recommendOnlinePlanner.BuildPageExplainSnapshot(pageGoods, candidates)
	sourceContext := requestPlan.BuildAnonymousPageOnlineResultContext(sceneInput, sceneGoodsIds, candidateGoodsIds, candidateGoodsIdList, explainSnapshot, probeContext)
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

	requestPlan := recommendOnlinePlanner.NewPersonalizedRequestPlan(&recommendDomain.GoodsRequest{
		Scene:    req.GetScene(),
		OrderId:  req.GetOrderId(),
		GoodsId:  req.GetGoodsId(),
		PageNum:  req.GetPageNum(),
		PageSize: req.GetPageSize(),
	}, map[string]any{})
	// 分页越深，候选池越大，避免深页直接无货可排。
	candidateLimit := requestPlan.CandidateLimit
	probeContext, err := c.buildRecommendRecallProbeContext(ctx, int32(req.GetScene()), userId, req.GetGoodsId(), candidateLimit, []int64{req.GetGoodsId()})
	if err != nil {
		return nil, 0, nil, nil, err
	}
	requestPlan = recommendOnlinePlanner.NewPersonalizedRequestPlan(&requestPlan.Request, probeContext)
	// 相似用户当前仍只做观测，因此单独拉取一份偏好商品用于效果比对。
	if len(requestPlan.SimilarUserIds) > 0 {
		requestPlan.SimilarUserObservedGoodsIds, err = c.recommendUserGoodsPreferenceCase.listObservedGoodsIdsByUserIds(ctx, requestPlan.SimilarUserIds, candidateLimit, []int64{req.GetGoodsId()})
		if err != nil {
			return nil, 0, nil, nil, err
		}
		requestPlan.SetSimilarUserObservedGoodsIds(requestPlan.SimilarUserObservedGoodsIds)
	}
	profileCategoryIdList := make([]int64, 0)
	profileCategoryIdList, err = c.recommendUserPreferenceCase.listPreferredCategoryIds(ctx, userId, 3)
	if err != nil {
		return nil, 0, nil, nil, err
	}

	// 登录态优先消费当前页面最强的业务上下文。
	// 按当前推荐场景决定优先使用哪类业务上下文做召回。
	sceneInput := recommendOnlinePlanner.SceneInput{}
	switch req.GetScene() {
	case common.RecommendScene_CART:
		var cartGoodsIdList []int64
		cartGoodsIdList, err = c.userCartCase.listGoodsIdsByUserId(ctx, userId)
		if err != nil {
			return nil, 0, nil, nil, err
		}
		sceneInput = recommendOnlinePlanner.BuildCartSceneInput(cartGoodsIdList, nil, nil)
		// 购物车存在商品时，继续做购物车关联召回。
		if len(cartGoodsIdList) > 0 {
			cartPriorityGoodsIds := make([]int64, 0)
			cartPriorityGoodsIds, err = c.recommendGoodsRelationCase.listRelatedGoodsIds(ctx, cartGoodsIdList, pageSize)
			if err != nil {
				return nil, 0, nil, nil, err
			}
			cartCategoryIds := make([]int64, 0)
			cartCategoryIds, err = c.goodsInfoCase.listCategoryIdsByGoodsIds(ctx, cartGoodsIdList)
			if err != nil {
				return nil, 0, nil, nil, err
			}
			sceneInput = recommendOnlinePlanner.BuildCartSceneInput(cartGoodsIdList, cartPriorityGoodsIds, cartCategoryIds)
		}
	case common.RecommendScene_ORDER_DETAIL, common.RecommendScene_ORDER_PAID:
		// 存在订单编号时，继续做订单关联召回。
		if req.GetOrderId() > 0 {
			var orderGoodsIdList []int64
			orderGoodsIdList, err = c.orderGoodsCase.listGoodsIdsByOrderId(ctx, req.GetOrderId())
			if err != nil {
				return nil, 0, nil, nil, err
			}
			sceneInput = recommendOnlinePlanner.BuildOrderSceneInput(orderGoodsIdList, nil, nil)
			orderPriorityGoodsIds := make([]int64, 0)
			orderPriorityGoodsIds, err = c.recommendGoodsRelationCase.listRelatedGoodsIds(ctx, orderGoodsIdList, pageSize)
			if err != nil {
				return nil, 0, nil, nil, err
			}
			orderCategoryIds := make([]int64, 0)
			orderCategoryIds, err = c.goodsInfoCase.listCategoryIdsByGoodsIds(ctx, orderGoodsIdList)
			if err != nil {
				return nil, 0, nil, nil, err
			}
			sceneInput = recommendOnlinePlanner.BuildOrderSceneInput(orderGoodsIdList, orderPriorityGoodsIds, orderCategoryIds)
		}
	case common.RecommendScene_GOODS_DETAIL:
		// 存在商品编号时，继续做商品关联召回。
		if req.GetGoodsId() > 0 {
			sourceGoodsIdList := []int64{req.GetGoodsId()}
			similarItemCacheResult := &recommendCacheReadResult{}
			similarItemCacheResult, err = c.listCachedSimilarItemGoodsIds(ctx, req.GetGoodsId(), candidateLimit, []int64{req.GetGoodsId()})
			if err != nil {
				return nil, 0, nil, nil, err
			}
			requestPlan.MergeCacheReadContext(mergeRecommendCacheReadResult(nil, similarItemCacheResult))
			goodsDetailPriorityGoodsIds := similarItemCacheResult.ids
			// 相似商品缓存未命中时，回退到数据库关系召回。
			if len(goodsDetailPriorityGoodsIds) == 0 {
				goodsDetailPriorityGoodsIds, err = c.recommendGoodsRelationCase.listRelatedGoodsIds(ctx, sourceGoodsIdList, candidateLimit)
				if err != nil {
					return nil, 0, nil, nil, err
				}
			}
			goodsDetailCategoryIds := make([]int64, 0)
			goodsDetailCategoryIds, err = c.goodsInfoCase.listCategoryIdsByGoodsIds(ctx, sourceGoodsIdList)
			if err != nil {
				return nil, 0, nil, nil, err
			}
			cacheHitSources := make([]string, 0, 1)
			// 相似商品缓存命中时，记录当前详情场景使用了缓存桥接结果。
			if len(similarItemCacheResult.ids) > 0 {
				cacheHitSources = append(cacheHitSources, recommendCacheHitGoodsDetail)
			}
			sceneInput = recommendOnlinePlanner.BuildGoodsDetailSceneInput(sourceGoodsIdList, goodsDetailPriorityGoodsIds, goodsDetailCategoryIds, cacheHitSources)
		}
	}
	requestPlan.ApplySceneInput(sceneInput)

	// 用户画像只负责补足，不覆盖强场景召回。
	// 用户画像命中类目时，合并到类目补足候选集中。
	if len(profileCategoryIdList) > 0 {
		requestPlan.ApplyProfileScene(profileCategoryIdList)
	}
	// 没有命中任何召回入口时，统一回退到 latest。
	requestPlan.EnsureFallbackLatest()

	// 这里统一去重，避免同一商品或类目重复参与候选计算。
	requestPlan.NormalizeState()
	categoryCandidateIdList := make([]int64, 0)
	categoryQuery := requestPlan.BuildCategoryCandidateQuery()
	// 存在类目候选时，按类目继续补足候选商品池。
	if categoryQuery.Enabled {
		categoryCandidateIdList, err = c.pageCategoryCandidateGoodsIds(ctx, categoryQuery)
		if err != nil {
			return nil, 0, nil, nil, err
		}
	}
	requestPlan.SetCategoryCandidateGoodsIds(categoryCandidateIdList)
	latestCandidateIdList := make([]int64, 0)
	latestQuery := requestPlan.BuildLatestCandidateQuery()
	// 候选池仍可扩充时，继续用 latest 召回做兜底补足。
	if latestQuery.Enabled {
		latestCacheResult := &recommendCacheReadResult{}
		latestCacheResult, err = c.listCachedLatestGoodsIds(ctx, int32(req.GetScene()), latestQuery.Limit, latestQuery.ExcludeGoodsIds)
		if err != nil {
			return nil, 0, nil, nil, err
		}
		requestPlan.MergeCacheReadContext(mergeRecommendCacheReadResult(nil, latestCacheResult))
		latestCandidateIdList = latestCacheResult.ids
		// 最新榜缓存未命中时，回退到数据库最新商品分页。
		if len(latestCandidateIdList) == 0 {
			latestCandidateIdList, err = c.pageLatestCandidateGoodsIds(ctx, latestQuery)
			if err != nil {
				return nil, 0, nil, nil, err
			}
		} else {
			requestPlan.AddCacheHitSource(recommendCacheHitLatest)
		}
	}
	requestPlan.SetLatestCandidateGoodsIds(latestCandidateIdList)

	// 最终候选池按 强召回 + 类目补足 + latest 兜底 合并。
	allCandidateIdList := requestPlan.BuildPersonalizedCandidateGoodsIds()
	// 候选商品池为空时，直接返回空结果。
	if len(allCandidateIdList) == 0 {
		sourceContext := requestPlan.BuildPersonalizedOnlineResultContext(sceneInput, recommendOnlinePlanner.ResultSnapshot{}, []int64{}, []int64{}, probeContext)
		return []*app.GoodsInfo{}, 0, []string{}, sourceContext, nil
	}

	goodsList := make([]*app.GoodsInfo, 0)
	goodsList, err = c.goodsInfoCase.listByGoodsIds(ctx, allCandidateIdList)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	signalSnapshot := requestPlan.BuildPersonalizedSignalSnapshot(goodsList)
	signalLoadPlan := requestPlan.BuildPersonalizedSignalLoadPlan(signalSnapshot)
	// 这份商品 ID 用来对齐各种商品级排序信号。
	candidateGoodsIdList := signalLoadPlan.CandidateGoodsIds
	// 这份类目 ID 用来对齐画像类偏好分。
	candidateCategoryIdList := signalLoadPlan.CandidateCategoryIds

	relationScores := make(map[int64]float64)
	relationScores, err = c.recommendGoodsRelationCase.loadRelationScores(ctx, signalLoadPlan.RelationSourceGoodsIds)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	userGoodsScores := make(map[int64]float64)
	recentPaidGoodsMap := make(map[int64]struct{})
	userGoodsScores, recentPaidGoodsMap, err = c.recommendUserGoodsPreferenceCase.loadUserGoodsSignals(ctx, userId, candidateGoodsIdList)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	actorExposurePenalties := make(map[int64]float64)
	actorExposurePenalties, err = c.recommendExposureCase.loadActorExposurePenalties(ctx, actor, signalLoadPlan.Scene, candidateGoodsIdList)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	profileScores := make(map[int64]float64)
	profileScores, err = c.recommendUserPreferenceCase.loadProfileScores(ctx, userId, candidateCategoryIdList)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	scenePopularityScores := make(map[int64]float64)
	sceneExposurePenalties := make(map[int64]float64)
	scenePopularityScores, sceneExposurePenalties, err = c.recommendGoodsStatDayCase.loadScenePopularitySignals(ctx, signalLoadPlan.Scene, candidateGoodsIdList)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	globalPopularityScores := make(map[int64]float64)
	globalPopularityScores, err = c.goodsStatDayCase.loadGlobalPopularityScores(ctx, candidateGoodsIdList)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	personalizedSignals := requestPlan.BuildPersonalizedSignals(
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
	pageSnapshot := requestPlan.BuildRankedPageSnapshot(rankedGoods)
	// 分页偏移超出候选集范围时，直接返回空页但保留总数。
	if pageSnapshot.IsEmptyPage {
		sourceContext := requestPlan.BuildPersonalizedEmptyOnlineResultContext(sceneInput, candidateGoodsIdList, probeContext)
		return []*app.GoodsInfo{}, pageSnapshot.Total, []string{}, sourceContext, nil
	}
	pageGoods := pageSnapshot.PageGoods
	// 当前页才需要 explain，整池 explain 没必要返回。
	explainSnapshot := recommendOnlinePlanner.BuildPageExplainSnapshot(pageGoods, candidates)
	sourceContext := requestPlan.BuildPersonalizedPageOnlineResultContext(sceneInput, candidateGoodsIdList, explainSnapshot, probeContext)
	return pageGoods, pageSnapshot.Total, explainSnapshot.RecallSources, sourceContext, nil
}

// pageCategoryCandidateGoodsIds 按类目补足查询候选商品编号。
func (c *RecommendRequestCase) pageCategoryCandidateGoodsIds(ctx context.Context, queryPlan recommendOnlinePlanner.GoodsPoolQuery) ([]int64, error) {
	// 查询计划未启用或缺少必要参数时，直接返回空集合。
	if !queryPlan.SupportsCategoryQuery() {
		return []int64{}, nil
	}
	query := c.goodsInfoCase.GoodsInfoRepo.Query(ctx).GoodsInfo
	opts := make([]repo.QueryOption, 0, 2)
	opts = append(opts, repo.Where(query.CategoryID.In(queryPlan.CategoryIds...)))
	pageResp, err := c.pageGoodsByQueryPlan(ctx, queryPlan, opts...)
	if err != nil {
		return nil, err
	}
	// 类目补足阶段只需要商品编号进入后续候选池。
	return recommendOnlinePlanner.BuildGoodsPoolPageSnapshot(pageResp).GoodsIds, nil
}

// pageLatestCandidateGoodsIds 按 latest 兜底查询候选商品编号。
func (c *RecommendRequestCase) pageLatestCandidateGoodsIds(ctx context.Context, queryPlan recommendOnlinePlanner.GoodsPoolQuery) ([]int64, error) {
	// 查询计划未启用或缺少必要参数时，直接返回空集合。
	if !queryPlan.IsEnabled() {
		return []int64{}, nil
	}
	pageResp, err := c.pageGoodsByQueryPlan(ctx, queryPlan)
	if err != nil {
		return nil, err
	}
	// latest 兜底阶段也只需要商品编号进入候选池。
	return recommendOnlinePlanner.BuildGoodsPoolPageSnapshot(pageResp).GoodsIds, nil
}

// pageLatestFallbackGoods 按 latest 回退查询商品列表。
func (c *RecommendRequestCase) pageLatestFallbackGoods(ctx context.Context, queryPlan recommendOnlinePlanner.GoodsPoolQuery) ([]*app.GoodsInfo, int64, error) {
	// 查询计划未启用或缺少必要参数时，直接返回空结果。
	if !queryPlan.IsEnabled() {
		return []*app.GoodsInfo{}, 0, nil
	}
	pageResp, err := c.pageGoodsByQueryPlan(ctx, queryPlan)
	if err != nil {
		return nil, 0, err
	}
	pageSnapshot := recommendOnlinePlanner.BuildGoodsPoolPageSnapshot(pageResp)
	return pageSnapshot.GoodsList, pageSnapshot.Total, nil
}

// pageGoodsByQueryPlan 按候选池查询计划执行统一分页桥接查询。
func (c *RecommendRequestCase) pageGoodsByQueryPlan(ctx context.Context, queryPlan recommendOnlinePlanner.GoodsPoolQuery, extraOpts ...repo.QueryOption) (*app.PageGoodsInfoResponse, error) {
	// 查询计划未启用或缺少必要参数时，直接返回空分页结果。
	if !queryPlan.IsEnabled() {
		return recommendOnlinePlanner.BuildEmptyGoodsPoolPageResponse(), nil
	}
	query := c.goodsInfoCase.GoodsInfoRepo.Query(ctx).GoodsInfo
	opts := make([]repo.QueryOption, 0, len(extraOpts)+1)
	opts = append(opts, extraOpts...)
	// 当前存在排除商品时，不再让这些商品进入当前候选分页结果。
	if len(queryPlan.ExcludeGoodsIds) > 0 {
		opts = append(opts, repo.Where(query.ID.NotIn(queryPlan.ExcludeGoodsIds...)))
	}
	return c.goodsInfoCase.PageGoodsInfo(ctx, &app.PageGoodsInfoRequest{
		PageNum:  1,
		PageSize: queryPlan.Limit,
	}, opts...)
}

// saveRecommendRequest 保存推荐请求记录。
func (c *RecommendRequestCase) saveRecommendRequest(ctx context.Context, requestId string, actor *appDto.RecommendActor, req *app.RecommendGoodsRequest, sourceContext map[string]any, list []*app.GoodsInfo, recallSources []string) error {
	persistedSourceContext := recommendOnlineRecord.BuildPersistedSourceContext(sourceContext)
	sourceContextJson, err := json.Marshal(persistedSourceContext)
	if err != nil {
		return err
	}

	createdAt := time.Now()
	// 这条记录会被曝光、点击、下单链路按 requestId 回查。
	entity := &models.RecommendRequest{
		RequestID: requestId,
		ActorType: actor.ActorType,
		ActorID:   actor.ActorId,
		Scene:     int32(req.GetScene()),
		// 精简后的上下文仍然保留场景调试信息。
		SourceContext: string(sourceContextJson),
		PageNum:       int32(req.GetPageNum()),
		PageSize:      int32(req.GetPageSize()),
		CreatedAt:     createdAt,
	}
	return c.RecommendRequestRepo.Data.Transaction(ctx, func(ctx context.Context) error {
		err = c.RecommendRequestRepo.Create(ctx, entity)
		if err != nil {
			return err
		}
		return c.recommendRequestItemCase.batchCreateByRecommendRequest(ctx, entity.ID, req, sourceContext, list, recallSources)
	})
}

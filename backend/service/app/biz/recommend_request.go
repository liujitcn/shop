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
	recommendLlmRerank "shop/pkg/llm/rerank"
	recommendCache "shop/pkg/recommend/cache"
	recommendCandidate "shop/pkg/recommend/candidate"
	recommendCore "shop/pkg/recommend/core"
	recommendDomain "shop/pkg/recommend/domain"
	recommendEvent "shop/pkg/recommend/event"
	recommendOnlineFeature "shop/pkg/recommend/online/feature"
	recommendOnlinePlanner "shop/pkg/recommend/online/planner"
	recommendOnlineRank "shop/pkg/recommend/online/rank"
	recommendOnlineRecall "shop/pkg/recommend/online/recall"
	recommendOnlineRecord "shop/pkg/recommend/online/record"
	appDto "shop/service/app/dto"

	"github.com/go-kratos/kratos/v2/log"
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
	recommendModelVersionCase        *RecommendModelVersionCase
	recommendCacheStore              recommendCache.Store
	recommendLlmRerankService        *recommendLlmRerank.Service
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
	recommendModelVersionCase *RecommendModelVersionCase,
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
		recommendModelVersionCase:        recommendModelVersionCase,
		recommendCacheStore:              recommendCacheStore,
		recommendLlmRerankService:        recommendLlmRerank.NewServiceFromEnv(),
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
	requestPlan := recommendOnlinePlanner.NewAnonymousRequestPlan(appDto.BuildRecommendGoodsRequest(req), map[string]any{})
	candidateLimit := requestPlan.CandidateLimit
	// 匿名态只使用近一段时间内的热度数据。
	startDate := time.Now().AddDate(0, 0, -recommendCandidate.AnonymousRecallDays)
	sceneStrategyContext, err := c.recommendModelVersionCase.loadSceneStrategyContext(ctx, int32(req.GetScene()))
	if err != nil {
		return nil, 0, nil, nil, err
	}
	probeContext, err := c.buildRecommendRecallProbeContext(ctx, sceneStrategyContext, 0, req.GetGoodsId(), candidateLimit, []int64{req.GetGoodsId()})
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
	requestPlan.MergeCacheReadContext(recommendCache.MergeReadContext(nil, sceneHotCacheResult))
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
			requestPlan.MergeCacheReadContext(recommendCache.MergeReadContext(nil, similarItemCacheResult))
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
	// 匿名态允许入池的灰度召回统一在这里并入优先候选集合。
	requestPlan.ApplyJoinRecall()
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
	if requestPlan.ShouldFallbackToAnonymousLatest(candidateGoodsIds) {
		latestExcludeGoodsIds := make([]int64, 0, 1)
		// 商品详情场景回退到 latest 时，同样排除当前详情商品。
		if requestPlan.IsGoodsDetail() && requestPlan.Request.GoodsId > 0 {
			latestExcludeGoodsIds = append(latestExcludeGoodsIds, requestPlan.Request.GoodsId)
		}
		latestCacheResult, cacheErr := c.listCachedLatestGoodsIds(ctx, int32(req.GetScene()), candidateLimit, latestExcludeGoodsIds)
		if cacheErr != nil {
			return nil, 0, nil, nil, cacheErr
		}
		requestPlan.MergeCacheReadContext(recommendCache.MergeReadContext(nil, latestCacheResult))
		latestGoodsIds := latestCacheResult.Ids
		// 最新榜缓存命中时，直接按缓存顺序返回商品列表。
		if len(latestGoodsIds) > 0 {
			requestPlan.AddCacheHitSource(recommendCacheHitLatest)
			latestGoodsList, listErr := c.goodsInfoCase.listByGoodsIds(ctx, latestGoodsIds)
			if listErr != nil {
				return nil, 0, nil, nil, listErr
			}
			payload := requestPlan.BuildAnonymousLatestFallbackPayload(sceneInput, sceneGoodsIds, probeContext)
			payload.SourceContext = recommendOnlineRecord.AppendStrategyContext(payload.SourceContext, sceneStrategyContext, nil)
			return latestGoodsList, int64(len(latestGoodsList)), payload.RecallSources, payload.SourceContext, nil
		}
		latestGoodsList, latestTotal, listErr := c.pageLatestFallbackGoods(ctx, latestExcludeGoodsIds, candidateLimit)
		if listErr != nil {
			return nil, 0, nil, nil, listErr
		}
		payload := requestPlan.BuildAnonymousLatestFallbackPayload(sceneInput, sceneGoodsIds, probeContext)
		payload.SourceContext = recommendOnlineRecord.AppendStrategyContext(payload.SourceContext, sceneStrategyContext, nil)
		return latestGoodsList, latestTotal, payload.RecallSources, payload.SourceContext, nil
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
	stageScores, stageCacheReadContext, err := c.loadRecommendStageScores(
		ctx,
		sceneStrategyContext,
		actor.ToDomainActor(),
		&requestPlan.Request,
		candidateGoodsIds,
		filteredGoodsList,
	)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	requestPlan.MergeCacheReadContext(stageCacheReadContext)

	rankingResult := recommendOnlineRank.ExecuteAnonymousRanking(
		&requestPlan.Request,
		filteredGoodsList,
		anonymousSignals,
		c.goodsRecommendConfig.GetAnonymousRank(),
		requestPlan.AppendAnonymousExplainRecallSources,
		sceneStrategyContext.Config,
		stageScores,
	)
	pageSnapshot := rankingResult.PageSnapshot
	// 分页偏移超出候选集范围时，直接返回空页。
	if pageSnapshot.IsEmptyPage {
		payload := requestPlan.BuildAnonymousEmptyOnlinePayload(sceneInput, sceneGoodsIds, candidateGoodsIds, signalLoadPlan.CandidateGoodsIds, probeContext)
		payload.SourceContext = recommendOnlineRecord.AppendStrategyContext(payload.SourceContext, sceneStrategyContext, rankingResult.StageContext)
		return []*app.GoodsInfo{}, pageSnapshot.Total, payload.RecallSources, payload.SourceContext, nil
	}
	pageGoods := pageSnapshot.PageGoods
	// explain 只收集当前页，避免响应上下文过大。
	explainSnapshot := rankingResult.ExplainSnapshot
	payload := requestPlan.BuildAnonymousPageOnlinePayload(sceneInput, sceneGoodsIds, candidateGoodsIds, signalLoadPlan.CandidateGoodsIds, explainSnapshot, probeContext)
	payload.SourceContext = recommendOnlineRecord.AppendStrategyContext(payload.SourceContext, sceneStrategyContext, rankingResult.StageContext)
	return pageGoods, pageSnapshot.Total, payload.RecallSources, payload.SourceContext, nil
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

	requestPlan := recommendOnlinePlanner.NewPersonalizedRequestPlan(appDto.BuildRecommendGoodsRequest(req), map[string]any{})
	candidateLimit := requestPlan.CandidateLimit
	sceneStrategyContext, err := c.recommendModelVersionCase.loadSceneStrategyContext(ctx, int32(req.GetScene()))
	if err != nil {
		return nil, 0, nil, nil, err
	}
	probeContext, err := c.buildRecommendRecallProbeContext(ctx, sceneStrategyContext, userId, req.GetGoodsId(), candidateLimit, []int64{req.GetGoodsId()})
	if err != nil {
		return nil, 0, nil, nil, err
	}
	requestPlan = recommendOnlinePlanner.NewPersonalizedRequestPlan(&requestPlan.Request, probeContext)
	similarUserScores := make(map[int64]float64)
	// 相似用户观测结果会回写调试上下文，并在版本配置允许时并入候选池。
	if len(requestPlan.SimilarUserIds) > 0 {
		similarUserObservedGoodsIds, observedScoreMap, listErr := c.recommendUserGoodsPreferenceCase.listObservedGoodsIdsByUserIds(ctx, requestPlan.SimilarUserIds, candidateLimit, []int64{req.GetGoodsId()})
		if listErr != nil {
			return nil, 0, nil, nil, listErr
		}
		similarUserScores = observedScoreMap
		requestPlan.ApplySimilarUserObservation(
			similarUserObservedGoodsIds,
			recommendOnlineRecall.ShouldJoinProbeCandidate(probeContext, "similarUser"),
		)
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
			requestPlan.MergeCacheReadContext(recommendCache.MergeReadContext(nil, similarItemCacheResult))
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
	// 灰度召回允许入池时，这里统一并入优先候选集合，避免只在单一场景里生效。
	requestPlan.ApplyJoinRecall()

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
		requestPlan.MergeCacheReadContext(recommendCache.MergeReadContext(nil, latestCacheResult))
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
		payload := requestPlan.BuildPersonalizedEmptyOnlinePayload(sceneInput, []int64{}, probeContext)
		payload.SourceContext = recommendOnlineRecord.AppendStrategyContext(payload.SourceContext, sceneStrategyContext, nil)
		return []*app.GoodsInfo{}, 0, payload.RecallSources, payload.SourceContext, nil
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
		similarUserScores,
		profileScores,
		scenePopularityScores,
		globalPopularityScores,
		sceneExposurePenalties,
		actorExposurePenalties,
		recentPaidGoodsMap,
	)
	stageScores, stageCacheReadContext, err := c.loadRecommendStageScores(
		ctx,
		sceneStrategyContext,
		actor.ToDomainActor(),
		&requestPlan.Request,
		candidateGoodsIds,
		signalSnapshot.GoodsList,
	)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	requestPlan.MergeCacheReadContext(stageCacheReadContext)

	rankingResult := recommendOnlineRank.ExecutePersonalizedRanking(
		&requestPlan.Request,
		signalSnapshot.GoodsList,
		personalizedSignals,
		c.goodsRecommendConfig.GetPersonalizedRank(),
		requestPlan.AppendPersonalizedExplainRecallSources,
		sceneStrategyContext.Config,
		stageScores,
	)
	pageSnapshot := rankingResult.PageSnapshot
	// 分页偏移超出候选集范围时，直接返回空页但保留总数。
	if pageSnapshot.IsEmptyPage {
		payload := requestPlan.BuildPersonalizedEmptyOnlinePayload(sceneInput, candidateGoodsIds, probeContext)
		payload.SourceContext = recommendOnlineRecord.AppendStrategyContext(payload.SourceContext, sceneStrategyContext, rankingResult.StageContext)
		return []*app.GoodsInfo{}, pageSnapshot.Total, payload.RecallSources, payload.SourceContext, nil
	}
	pageGoods := pageSnapshot.PageGoods
	// 当前页才需要 explain，整池 explain 没必要返回。
	explainSnapshot := rankingResult.ExplainSnapshot
	payload := requestPlan.BuildPersonalizedPageOnlinePayload(sceneInput, candidateGoodsIds, explainSnapshot, probeContext)
	payload.SourceContext = recommendOnlineRecord.AppendStrategyContext(payload.SourceContext, sceneStrategyContext, rankingResult.StageContext)
	return pageGoods, pageSnapshot.Total, payload.RecallSources, payload.SourceContext, nil
}

// executeRecommendGoods 执行统一推荐主流程。
func (c *RecommendRequestCase) executeRecommendGoods(ctx context.Context, actor *appDto.RecommendActor, req *app.RecommendGoodsRequest) (recommendDomain.PageResult, error) {
	domainActor := actor.ToDomainActor()
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

// saveRecommendRequest 保存推荐请求记录。
func (c *RecommendRequestCase) saveRecommendRequest(ctx context.Context, requestId string, actor *appDto.RecommendActor, req *app.RecommendGoodsRequest, sourceContext map[string]any, list []*app.GoodsInfo, recallSources []string) error {
	createdAt := time.Now()
	entity, err := recommendOnlineRecord.BuildRecommendRequestEntity(
		requestId,
		actor.ToDomainActor(),
		appDto.BuildRecommendGoodsRequest(req),
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

const (
	// recommendCacheHitGoodsDetail 表示商品详情相似商品缓存命中。
	recommendCacheHitGoodsDetail = "goods_detail_cache"
	// recommendCacheHitSceneHot 表示场景热门榜缓存命中。
	recommendCacheHitSceneHot = "scene_hot_cache"
	// recommendCacheHitLatest 表示最新榜缓存命中。
	recommendCacheHitLatest = "latest_cache"
	// recommendCacheHitRanker 表示模型精排缓存命中。
	recommendCacheHitRanker = "ranker_cache"
	// recommendCacheHitLlmRerank 表示 LLM 二次重排缓存命中。
	recommendCacheHitLlmRerank = "llm_rerank_cache"
)

const (
	// recommendRecallProbeSimilarUser 表示相似用户召回探针。
	recommendRecallProbeSimilarUser = "similar_user_probe"
	// recommendRecallProbeCollaborativeFiltering 表示协同过滤召回探针。
	recommendRecallProbeCollaborativeFiltering = "collaborative_filtering_probe"
	// recommendRecallProbeContentBased 表示内容相似召回探针。
	recommendRecallProbeContentBased = "content_based_probe"
)

// listCachedSceneHotGoodsIds 读取场景热门榜缓存商品。
func (c *RecommendRequestCase) listCachedSceneHotGoodsIds(ctx context.Context, scene int32, limit int64, excludeGoodsIds []int64) (*recommendDomain.CacheReadResult, error) {
	version, versionPublishedAt, err := c.recommendModelVersionCase.loadSceneCacheVersion(ctx, scene)
	if err != nil {
		return nil, err
	}
	return c.listCachedGoodsIds(
		ctx,
		recommendCache.NonPersonalized,
		recommendCache.SceneHotSubset(scene, version),
		recommendCacheHitSceneHot,
		version,
		versionPublishedAt,
		limit,
		excludeGoodsIds,
	)
}

// listCachedLatestGoodsIds 读取场景最新榜缓存商品。
func (c *RecommendRequestCase) listCachedLatestGoodsIds(ctx context.Context, scene int32, limit int64, excludeGoodsIds []int64) (*recommendDomain.CacheReadResult, error) {
	version, versionPublishedAt, err := c.recommendModelVersionCase.loadSceneCacheVersion(ctx, scene)
	if err != nil {
		return nil, err
	}
	return c.listCachedGoodsIds(
		ctx,
		recommendCache.NonPersonalized,
		recommendCache.SceneLatestSubset(scene, version),
		recommendCacheHitLatest,
		version,
		versionPublishedAt,
		limit,
		excludeGoodsIds,
	)
}

// listCachedSimilarItemGoodsIds 读取相似商品缓存。
func (c *RecommendRequestCase) listCachedSimilarItemGoodsIds(ctx context.Context, goodsId int64, limit int64, excludeGoodsIds []int64) (*recommendDomain.CacheReadResult, error) {
	// 商品编号非法时，不需要继续读取相似商品缓存。
	if goodsId <= 0 {
		return recommendCache.NewReadResult(
			recommendCacheHitGoodsDetail,
			recommendCache.ItemToItem,
			"",
			recommendCache.DefaultVersion,
			time.Time{},
			limit,
			len(excludeGoodsIds),
		), nil
	}

	version, versionPublishedAt, err := c.recommendModelVersionCase.loadSceneCacheVersion(ctx, int32(common.RecommendScene_GOODS_DETAIL))
	if err != nil {
		return nil, err
	}
	return c.listCachedGoodsIds(
		ctx,
		recommendCache.ItemToItem,
		recommendCache.SimilarItemSubset(goodsId, version),
		recommendCacheHitGoodsDetail,
		version,
		versionPublishedAt,
		limit,
		excludeGoodsIds,
	)
}

// listCachedSimilarUserIds 读取相似用户召回探针缓存。
func (c *RecommendRequestCase) listCachedSimilarUserIds(ctx context.Context, userId int64, version string, versionPublishedAt time.Time, limit int64) (*recommendDomain.CacheReadResult, error) {
	// 登录用户编号非法时，不需要继续读取相似用户缓存。
	if userId <= 0 {
		return recommendCache.NewReadResult(
			recommendRecallProbeSimilarUser,
			recommendCache.UserToUser,
			"",
			version,
			versionPublishedAt,
			limit,
			0,
		), nil
	}
	return c.listCachedInt64Ids(
		ctx,
		recommendCache.UserToUser,
		recommendCache.SimilarUserSubset(userId, version),
		recommendRecallProbeSimilarUser,
		version,
		versionPublishedAt,
		limit,
		nil,
	)
}

// listCachedCollaborativeFilteringGoodsIds 读取协同过滤召回探针缓存。
func (c *RecommendRequestCase) listCachedCollaborativeFilteringGoodsIds(ctx context.Context, userId int64, version string, versionPublishedAt time.Time, limit int64, excludeGoodsIds []int64) (*recommendDomain.CacheReadResult, error) {
	// 登录用户编号非法时，不需要继续读取协同过滤缓存。
	if userId <= 0 {
		return recommendCache.NewReadResult(
			recommendRecallProbeCollaborativeFiltering,
			recommendCache.CollaborativeFiltering,
			"",
			version,
			versionPublishedAt,
			limit,
			len(excludeGoodsIds),
		), nil
	}
	return c.listCachedInt64Ids(
		ctx,
		recommendCache.CollaborativeFiltering,
		recommendCache.CollaborativeFilteringSubset(userId, version),
		recommendRecallProbeCollaborativeFiltering,
		version,
		versionPublishedAt,
		limit,
		excludeGoodsIds,
	)
}

// listCachedContentBasedGoodsIds 读取内容相似召回探针缓存。
func (c *RecommendRequestCase) listCachedContentBasedGoodsIds(ctx context.Context, goodsId int64, version string, versionPublishedAt time.Time, limit int64, excludeGoodsIds []int64) (*recommendDomain.CacheReadResult, error) {
	// 商品编号非法时，不需要继续读取内容相似缓存。
	if goodsId <= 0 {
		return recommendCache.NewReadResult(
			recommendRecallProbeContentBased,
			recommendCache.ContentBased,
			"",
			version,
			versionPublishedAt,
			limit,
			len(excludeGoodsIds),
		), nil
	}
	return c.listCachedInt64Ids(
		ctx,
		recommendCache.ContentBased,
		recommendCache.ContentBasedSubset(goodsId, version),
		recommendRecallProbeContentBased,
		version,
		versionPublishedAt,
		limit,
		excludeGoodsIds,
	)
}

// buildRecommendRecallProbeContext 构建当前请求的召回探针上下文。
func (c *RecommendRequestCase) buildRecommendRecallProbeContext(
	ctx context.Context,
	sceneStrategyContext *recommendDomain.SceneStrategyContext,
	userId int64,
	goodsId int64,
	defaultLimit int64,
	excludeGoodsIds []int64,
) (map[string]any, error) {
	if sceneStrategyContext == nil {
		sceneStrategyContext = &recommendDomain.SceneStrategyContext{
			Version:            recommendCache.DefaultVersion,
			EffectiveVersion:   recommendCache.DefaultVersion,
			VersionPublishedAt: time.Time{},
			Config:             &recommendDomain.StrategyVersionConfig{},
		}
	}
	version := sceneStrategyContext.EffectiveVersion
	versionPublishedAt := sceneStrategyContext.VersionPublishedAt
	probeConfig := &recommendDomain.RecallProbeStrategy{}
	if sceneStrategyContext.Config != nil && sceneStrategyContext.Config.RecallProbe != nil {
		probeConfig = sceneStrategyContext.Config.RecallProbe
	}
	// 当前版本没有启用探针时，不需要额外记录上下文。
	if !probeConfig.HasEnabledProbe() {
		return map[string]any{}, nil
	}

	probeContext := map[string]any{
		"sceneVersion": sceneStrategyContext.Version,
	}
	// 实际读取版本与当前启用版本不一致时，再额外记录生效版本。
	if sceneStrategyContext.EffectiveVersion != "" && sceneStrategyContext.EffectiveVersion != sceneStrategyContext.Version {
		probeContext["effectiveVersion"] = sceneStrategyContext.EffectiveVersion
	}
	// 当前场景存在启用版本时，再补充版本发布时间。
	if !versionPublishedAt.IsZero() {
		probeContext["sceneVersionPublishedAt"] = versionPublishedAt.Format(time.RFC3339Nano)
	}
	observedSources := make([]string, 0, 3)
	if probeConfig.IsSimilarUserEnabled() && userId > 0 {
		limit := probeConfig.SimilarUser.ResolveLimit(defaultLimit)
		similarUserResult, listErr := c.listCachedSimilarUserIds(ctx, userId, version, versionPublishedAt, limit)
		if listErr != nil {
			return nil, listErr
		}
		similarUserIds := similarUserResult.Ids
		probeContext["similarUser"] = map[string]any{
			"enabled":          true,
			"joinCandidate":    probeConfig.SimilarUser.ShouldJoinCandidate(),
			"limit":            limit,
			"userIds":          similarUserIds,
			"cacheReadContext": similarUserResult.ReadContext,
		}
		// 读取到了有效相似用户时，记录探针命中来源。
		if len(similarUserIds) > 0 {
			observedSources = append(observedSources, recommendRecallProbeSimilarUser)
		}
	}
	if probeConfig.IsCollaborativeFilteringEnabled() && userId > 0 {
		limit := probeConfig.CollaborativeFiltering.ResolveLimit(defaultLimit)
		collaborativeFilteringResult, listErr := c.listCachedCollaborativeFilteringGoodsIds(ctx, userId, version, versionPublishedAt, limit, excludeGoodsIds)
		if listErr != nil {
			return nil, listErr
		}
		goodsIds := collaborativeFilteringResult.Ids
		probeContext["collaborativeFiltering"] = map[string]any{
			"enabled":          true,
			"joinCandidate":    probeConfig.CollaborativeFiltering.ShouldJoinCandidate(),
			"limit":            limit,
			"goodsIds":         goodsIds,
			"cacheReadContext": collaborativeFilteringResult.ReadContext,
		}
		// 读取到了有效协同过滤商品时，记录探针命中来源。
		if len(goodsIds) > 0 {
			observedSources = append(observedSources, recommendRecallProbeCollaborativeFiltering)
		}
	}
	if probeConfig.IsContentBasedEnabled() && goodsId > 0 {
		limit := probeConfig.ContentBased.ResolveLimit(defaultLimit)
		contentBasedResult, listErr := c.listCachedContentBasedGoodsIds(ctx, goodsId, version, versionPublishedAt, limit, excludeGoodsIds)
		if listErr != nil {
			return nil, listErr
		}
		goodsIds := contentBasedResult.Ids
		probeContext["contentBased"] = map[string]any{
			"enabled":          true,
			"joinCandidate":    probeConfig.ContentBased.ShouldJoinCandidate(),
			"limit":            limit,
			"goodsIds":         goodsIds,
			"cacheReadContext": contentBasedResult.ReadContext,
		}
		// 读取到了有效内容相似商品时，记录探针命中来源。
		if len(goodsIds) > 0 {
			observedSources = append(observedSources, recommendRecallProbeContentBased)
		}
	}
	probeContext["observedSources"] = recommendCore.DedupeStrings(observedSources)
	return probeContext, nil
}

// loadRecommendStageScores 读取模型精排与 LLM 重排阶段得分。
func (c *RecommendRequestCase) loadRecommendStageScores(
	ctx context.Context,
	sceneStrategyContext *recommendDomain.SceneStrategyContext,
	actor *recommendDomain.Actor,
	request *recommendDomain.GoodsRequest,
	candidateGoodsIds []int64,
	candidateGoodsList []*app.GoodsInfo,
) (recommendOnlineRank.StageScoreSet, map[string]any, error) {
	stageScores := recommendOnlineRank.StageScoreSet{
		RankerScores: map[int64]float64{},
		LlmScores:    map[int64]float64{},
	}
	cacheReadContext := make(map[string]any, 2)
	// 当前没有场景策略或没有候选商品时，不继续读取排序阶段缓存。
	if sceneStrategyContext == nil || len(candidateGoodsIds) == 0 {
		return stageScores, cacheReadContext, nil
	}

	rankerResult, err := c.loadCachedRankerScores(ctx, actor, sceneStrategyContext, candidateGoodsIds)
	if err != nil {
		return stageScores, nil, err
	}
	stageScores.RankerScores = rankerResult.Scores
	cacheReadContext = recommendCache.MergeScoreReadContext(cacheReadContext, rankerResult)

	llmResult, err := c.loadCachedLlmRerankScores(ctx, actor, request, sceneStrategyContext, candidateGoodsIds, candidateGoodsList)
	if err != nil {
		return stageScores, nil, err
	}
	stageScores.LlmScores = llmResult.Scores
	cacheReadContext = recommendCache.MergeScoreReadContext(cacheReadContext, llmResult)
	return stageScores, cacheReadContext, nil
}

// loadCachedRankerScores 读取模型精排阶段缓存分数。
func (c *RecommendRequestCase) loadCachedRankerScores(
	ctx context.Context,
	actor *recommendDomain.Actor,
	sceneStrategyContext *recommendDomain.SceneStrategyContext,
	candidateGoodsIds []int64,
) (*recommendDomain.CacheScoreReadResult, error) {
	if sceneStrategyContext == nil || sceneStrategyContext.Config == nil || sceneStrategyContext.Config.Ranker == nil {
		result := recommendCache.NewScoreReadResult(
			recommendCacheHitRanker,
			recommendCache.Ranker,
			"",
			recommendCache.DefaultVersion,
			time.Time{},
			int64(len(candidateGoodsIds)),
		)
		result.ReadContext["skipped"] = true
		result.ReadContext["skipReason"] = "not_configured"
		return result, nil
	}
	// 当前模型精排配置未启用时，不继续读取缓存分数。
	if !sceneStrategyContext.Config.Ranker.IsEnabled() {
		result := recommendCache.NewScoreReadResult(
			recommendCacheHitRanker,
			recommendCache.Ranker,
			"",
			sceneStrategyContext.EffectiveVersion,
			sceneStrategyContext.VersionPublishedAt,
			int64(len(candidateGoodsIds)),
		)
		result.ReadContext["skipped"] = true
		result.ReadContext["skipReason"] = "disabled"
		return result, nil
	}

	subset := recommendCache.RankerSubset(
		sceneStrategyContext.Scene,
		actor.ResolveCacheActorType(),
		actor.ResolveCacheActorId(),
		sceneStrategyContext.EffectiveVersion,
	)
	result, err := c.loadCachedScoreMap(
		ctx,
		recommendCache.Ranker,
		subset,
		recommendCacheHitRanker,
		sceneStrategyContext.EffectiveVersion,
		sceneStrategyContext.VersionPublishedAt,
		candidateGoodsIds,
	)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// loadCachedLlmRerankScores 读取 LLM 二次重排阶段缓存分数。
func (c *RecommendRequestCase) loadCachedLlmRerankScores(
	ctx context.Context,
	actor *recommendDomain.Actor,
	request *recommendDomain.GoodsRequest,
	sceneStrategyContext *recommendDomain.SceneStrategyContext,
	candidateGoodsIds []int64,
	candidateGoodsList []*app.GoodsInfo,
) (*recommendDomain.CacheScoreReadResult, error) {
	if sceneStrategyContext == nil || sceneStrategyContext.Config == nil || sceneStrategyContext.Config.LlmRerank == nil {
		result := recommendCache.NewScoreReadResult(
			recommendCacheHitLlmRerank,
			recommendCache.LlmRerank,
			"",
			recommendCache.DefaultVersion,
			time.Time{},
			int64(len(candidateGoodsIds)),
		)
		result.ReadContext["skipped"] = true
		result.ReadContext["skipReason"] = "not_configured"
		return result, nil
	}
	// 当前 LLM 重排配置未启用时，不继续读取缓存分数。
	if !sceneStrategyContext.Config.LlmRerank.IsEnabled() {
		result := recommendCache.NewScoreReadResult(
			recommendCacheHitLlmRerank,
			recommendCache.LlmRerank,
			"",
			sceneStrategyContext.EffectiveVersion,
			sceneStrategyContext.VersionPublishedAt,
			int64(len(candidateGoodsIds)),
		)
		result.ReadContext["skipped"] = true
		result.ReadContext["skipReason"] = "disabled"
		return result, nil
	}

	requestHash := recommendOnlineRank.BuildRerankRequestHash(
		request,
		actor,
		sceneStrategyContext.Config.LlmRerank,
		candidateGoodsIds,
		sceneStrategyContext.Config.LlmRerank.ResolveTopN(int64(len(candidateGoodsIds))),
	)
	subset := recommendCache.LlmRerankSubset(
		sceneStrategyContext.Scene,
		actor.ResolveCacheActorType(),
		actor.ResolveCacheActorId(),
		requestHash,
		sceneStrategyContext.EffectiveVersion,
	)
	result, err := c.loadCachedScoreMap(
		ctx,
		recommendCache.LlmRerank,
		subset,
		recommendCacheHitLlmRerank,
		sceneStrategyContext.EffectiveVersion,
		sceneStrategyContext.VersionPublishedAt,
		candidateGoodsIds,
	)
	if err != nil {
		return nil, err
	}
	result.ReadContext["requestHash"] = requestHash
	// 当前已经命中缓存或没有在线执行器时，不继续触发在线 LLM 重排。
	if result.ReadContext["hit"] == true || c.recommendLlmRerankService == nil {
		return result, nil
	}
	// 在线执行器未配置 API Key 时，直接保留缓存 miss 结果，不阻断主推荐链路。
	if !c.recommendLlmRerankService.IsConfigured() {
		result.ReadContext["liveSkipped"] = true
		result.ReadContext["liveSkipReason"] = "service_not_configured"
		return result, nil
	}
	liveResult, liveErr := c.recommendLlmRerankService.Rerank(ctx, recommendLlmRerank.Request{
		Strategy:           sceneStrategyContext.Config.LlmRerank,
		Actor:              actor,
		GoodsRequest:       request,
		CandidateGoodsIds:  candidateGoodsIds,
		CandidateGoodsList: candidateGoodsList,
	})
	// 在线调用失败时，只记录调试上下文并继续走无分数降级。
	if liveErr != nil {
		result.ReadContext["liveError"] = liveErr.Error()
		log.Errorf("recommend llm rerank live error request_hash=%s err=%v", requestHash, liveErr)
		return result, nil
	}
	// 在线执行器返回空结果时，不继续写缓存，直接保留 miss 结果。
	if liveResult == nil || len(liveResult.Scores) == 0 {
		result.ReadContext["liveSkipped"] = true
		result.ReadContext["liveSkipReason"] = "empty_live_scores"
		if liveResult != nil && len(liveResult.DebugContext) > 0 {
			result.ReadContext["liveContext"] = liveResult.DebugContext
		}
		return result, nil
	}

	result.Scores = liveResult.Scores
	result.ReadContext["returnedCount"] = len(liveResult.Scores)
	result.ReadContext["generated"] = true
	result.ReadContext["liveContext"] = liveResult.DebugContext
	result.ReadContext["liveProvider"] = "openai"
	// 在线补算出了有效分数时，明确记录当前阶段已经从 live fallback 成功降级恢复。
	if len(liveResult.Scores) > 0 {
		result.ReadContext["liveRecovered"] = true
	}
	cacheTTLSeconds := sceneStrategyContext.Config.LlmRerank.CacheTTLSeconds
	// 当前配置了有效 TTL 且存在缓存文档时，再把在线结果回写到版本缓存，避免每次都打外部模型。
	if cacheTTLSeconds > 0 && len(liveResult.Documents) > 0 {
		writeErr := recommendLlmRerank.WriteBackScores(
			ctx,
			c.recommendCacheStore,
			sceneStrategyContext.Scene,
			actor.ResolveCacheActorType(),
			actor.ResolveCacheActorId(),
			requestHash,
			sceneStrategyContext.EffectiveVersion,
			liveResult.Documents,
			time.Duration(cacheTTLSeconds)*time.Second,
		)
		if writeErr != nil {
			result.ReadContext["writeBackError"] = writeErr.Error()
			log.Errorf("recommend llm rerank write cache error request_hash=%s err=%v", requestHash, writeErr)
			return result, nil
		}
		result.ReadContext["writeBack"] = true
		result.ReadContext["cacheTTLSeconds"] = cacheTTLSeconds
	}
	return result, nil
}

// listCachedInt64Ids 读取指定缓存子集合中的编号列表。
func (c *RecommendRequestCase) listCachedInt64Ids(
	ctx context.Context,
	collection string,
	subset string,
	hitSource string,
	version string,
	versionPublishedAt time.Time,
	limit int64,
	excludeIds []int64,
) (*recommendDomain.CacheReadResult, error) {
	return recommendCache.ReadInt64Ids(
		ctx,
		c.recommendCacheStore,
		collection,
		subset,
		hitSource,
		version,
		versionPublishedAt,
		limit,
		excludeIds,
	)
}

// loadCachedScoreMap 读取指定缓存子集合中的分数映射。
func (c *RecommendRequestCase) loadCachedScoreMap(
	ctx context.Context,
	collection string,
	subset string,
	hitSource string,
	version string,
	versionPublishedAt time.Time,
	candidateGoodsIds []int64,
) (*recommendDomain.CacheScoreReadResult, error) {
	return recommendCache.ReadScoreMap(
		ctx,
		c.recommendCacheStore,
		collection,
		subset,
		hitSource,
		version,
		versionPublishedAt,
		candidateGoodsIds,
	)
}

// listCachedGoodsIds 读取指定缓存子集合中的商品编号列表。
func (c *RecommendRequestCase) listCachedGoodsIds(
	ctx context.Context,
	collection string,
	subset string,
	hitSource string,
	version string,
	versionPublishedAt time.Time,
	limit int64,
	excludeGoodsIds []int64,
) (*recommendDomain.CacheReadResult, error) {
	return recommendCache.ReadInt64Ids(
		ctx,
		c.recommendCacheStore,
		collection,
		subset,
		hitSource,
		version,
		versionPublishedAt,
		limit,
		excludeGoodsIds,
	)
}

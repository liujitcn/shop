package local

import (
	"context"

	_const "shop/pkg/const"

	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/recommend/dto"
)

type localPlan struct {
	providerName ProviderName
	scoreWeight  localScoreWeight
}

// ChainReceiver 表示本地推荐责任链接收器。
type ChainReceiver struct {
	recommend       *Recommend
	contextReceiver *ContextReceiver
	hot             *HotReceiver
	explore         *ExploreReceiver
}

// NewChainReceiver 创建本地推荐责任链接收器。
func NewChainReceiver(
	recommend *Recommend,
	contextReceiver *ContextReceiver,
	hot *HotReceiver,
	explore *ExploreReceiver,
) *ChainReceiver {
	return &ChainReceiver{
		recommend:       recommend,
		contextReceiver: contextReceiver,
		hot:             hot,
		explore:         explore,
	}
}

// Enabled 判断当前本地推荐责任链接收器是否可用。
func (r *ChainReceiver) Enabled() bool {
	return r != nil && r.recommend != nil && r.recommend.Enabled()
}

// ExecutePlan 按场景执行单一的本地推荐策略。
func (r *ChainReceiver) ExecutePlan(
	ctx context.Context,
	scene commonv1.RecommendScene,
	actor *dto.RecommendActor,
	goodsID, requestID int64,
	contextGoodsIDs []int64,
	pageNum, pageSize int64,
) (*dto.GoodsResult, error) {
	result := &dto.GoodsResult{
		GoodsIDs: []int64{},
		Strategy: commonv1.RecommendStrategy(_const.RECOMMEND_STRATEGY_LOCAL),
		Trace:    make([]*dto.GoodsTrace, 0),
	}
	// 本地推荐链路未启用时，直接返回空结果。
	if !r.Enabled() {
		return result, nil
	}
	// 页码或每页条数非法时，直接返回空结果。
	if pageNum <= 0 || pageSize <= 0 {
		return result, nil
	}

	normalizedContextGoodsIDs := append([]int64(nil), contextGoodsIDs...)
	// 上下文商品为空但显式传入了锚点商品时，统一将锚点商品补为本地上下文。
	if len(normalizedContextGoodsIDs) == 0 && goodsID > 0 {
		normalizedContextGoodsIDs = append(normalizedContextGoodsIDs, goodsID)
	}

	plan := r.buildRecommendPlan(scene, actor, normalizedContextGoodsIDs)
	// 当前场景没有可执行的本地推荐器时，直接返回空结果。
	if plan.providerName == "" {
		return result, nil
	}

	providers := map[ProviderName]func(ctx context.Context) ([]int64, int64, error){
		CONTEXT_CATEGORY_7D: func(ctx context.Context) ([]int64, int64, error) {
			return r.contextReceiver.GetGoodsPage(ctx, normalizedContextGoodsIDs, 7, plan.scoreWeight, pageNum, pageSize)
		},
		CONTEXT_CATEGORY_30D: func(ctx context.Context) ([]int64, int64, error) {
			return r.contextReceiver.GetGoodsPage(ctx, normalizedContextGoodsIDs, 30, plan.scoreWeight, pageNum, pageSize)
		},
		NON_PERSONALIZED_HOT_7D: func(ctx context.Context) ([]int64, int64, error) {
			return r.hot.GetGoodsPage(ctx, normalizedContextGoodsIDs, 7, plan.scoreWeight, pageNum, pageSize)
		},
		NON_PERSONALIZED_HOT_30D: func(ctx context.Context) ([]int64, int64, error) {
			return r.hot.GetGoodsPage(ctx, normalizedContextGoodsIDs, 30, plan.scoreWeight, pageNum, pageSize)
		},
		EXPLORE_ALL_GOODS: func(ctx context.Context) ([]int64, int64, error) {
			return r.explore.GetGoodsPage(ctx, scene, requestID, normalizedContextGoodsIDs, pageNum, pageSize)
		},
	}
	execute, ok := providers[plan.providerName]
	// 当前 provider 未注册时，记录轨迹后返回空结果。
	if !ok {
		result.Trace = append(result.Trace, &dto.GoodsTrace{
			ProviderName: string(plan.providerName),
			ErrorMsg:     "推荐提供方未注册",
		})
		return result, nil
	}

	goodsIDs, total, err := execute(ctx)
	trace := &dto.GoodsTrace{
		ProviderName: string(plan.providerName),
		ResultCount:  len(goodsIDs),
		Hit:          err == nil && len(goodsIDs) > 0,
	}
	// 当前 provider 执行失败时，记录轨迹后直接返回错误。
	if err != nil {
		trace.ErrorMsg = err.Error()
		result.Trace = append(result.Trace, trace)
		return result, err
	}

	result.Trace = append(result.Trace, trace)
	result.Total = total
	// 当前 provider 没有命中结果时，直接返回空页结果和总数。
	if len(goodsIDs) == 0 {
		return result, nil
	}

	result.GoodsIDs = goodsIDs
	result.ProviderName = string(plan.providerName)
	return result, nil
}

// buildRecommendPlan 按场景构建单一的本地推荐方案。
func (r *ChainReceiver) buildRecommendPlan(
	scene commonv1.RecommendScene,
	actor *dto.RecommendActor,
	contextGoodsIDs []int64,
) *localPlan {
	isLogin := actor != nil && actor.IsUser()
	hasContext := len(contextGoodsIDs) > 0
	plan := &localPlan{
		scoreWeight: localScoreWeight{
			viewWeight:     1,
			collectWeight:  3,
			cartWeight:     4,
			orderWeight:    5,
			payWeight:      6,
			payGoodsWeight: 2,
		},
	}

	// 不同推荐场景只选择一个最合适的本地推荐器，避免多候选池混排带来的复杂度和额外开销。
	switch scene {
	case commonv1.RecommendScene(_const.RECOMMEND_SCENE_HOME):
		// 首页优先保证商品广覆盖；登录且有上下文时，再优先走类目相关推荐。
		if isLogin && hasContext {
			plan.providerName = CONTEXT_CATEGORY_30D
		} else {
			plan.providerName = EXPLORE_ALL_GOODS
		}
	case commonv1.RecommendScene(_const.RECOMMEND_SCENE_GOODS_DETAIL):
		plan.scoreWeight = localScoreWeight{
			viewWeight:     1,
			collectWeight:  2,
			cartWeight:     3,
			orderWeight:    5,
			payWeight:      6,
			payGoodsWeight: 2,
		}
		// 商品详情场景优先围绕当前商品做同类目推荐；缺少上下文时再回退到全站热度。
		if hasContext {
			plan.providerName = CONTEXT_CATEGORY_7D
		} else {
			plan.providerName = NON_PERSONALIZED_HOT_30D
		}
	case commonv1.RecommendScene(_const.RECOMMEND_SCENE_CART):
		plan.scoreWeight = localScoreWeight{
			viewWeight:     1,
			collectWeight:  2,
			cartWeight:     5,
			orderWeight:    6,
			payWeight:      7,
			payGoodsWeight: 3,
		}
		// 购物车场景优先围绕当前待购商品补充同类推荐；没有上下文时再使用近期热销补位。
		if hasContext {
			plan.providerName = CONTEXT_CATEGORY_7D
		} else {
			plan.providerName = NON_PERSONALIZED_HOT_7D
		}
	case commonv1.RecommendScene(_const.RECOMMEND_SCENE_PROFILE):
		plan.scoreWeight = localScoreWeight{
			viewWeight:     1,
			collectWeight:  4,
			cartWeight:     4,
			orderWeight:    5,
			payWeight:      6,
			payGoodsWeight: 2,
		}
		// 个人中心对登录用户优先补兴趣相关商品；缺少画像时回退到全量探索，尽量提升商品曝光覆盖。
		if isLogin && hasContext {
			plan.providerName = CONTEXT_CATEGORY_30D
		} else {
			plan.providerName = EXPLORE_ALL_GOODS
		}
	case commonv1.RecommendScene(_const.RECOMMEND_SCENE_ORDER_DETAIL), commonv1.RecommendScene(_const.RECOMMEND_SCENE_ORDER_PAID):
		plan.scoreWeight = localScoreWeight{
			viewWeight:     1,
			collectWeight:  2,
			cartWeight:     5,
			orderWeight:    6,
			payWeight:      7,
			payGoodsWeight: 3,
		}
		// 订单相关场景优先围绕已购商品继续推荐；缺少订单商品上下文时回退到近期热销。
		if hasContext {
			plan.providerName = CONTEXT_CATEGORY_7D
		} else {
			plan.providerName = NON_PERSONALIZED_HOT_7D
		}
	default:
		// 通用兜底场景只要存在上下文就优先走类目推荐，否则回退到全量探索。
		if hasContext {
			plan.providerName = CONTEXT_CATEGORY_30D
		} else {
			plan.providerName = EXPLORE_ALL_GOODS
		}
	}
	return plan
}

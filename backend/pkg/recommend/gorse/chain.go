package gorse

import (
	"context"

	_const "shop/pkg/const"

	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/recommend/dto"
)

// ProviderName 表示 Gorse 推荐 provider 标识。
type ProviderName string

const (
	// GET_RECOMMEND 表示登录用户个性化推荐，对应 Gorse 推荐的 GetRecommend API。
	GET_RECOMMEND ProviderName = "recommend"
	// USER_TO_USER_SIMILAR_USERS 表示命名 user-to-user/similar_users 推荐器。
	USER_TO_USER_SIMILAR_USERS ProviderName = "user_to_user.similar_users"
	// SESSION 表示会话级推荐，对应 Gorse 推荐的 SessionRecommend API。
	SESSION ProviderName = "session"
	// NEIGHBORS 表示相邻商品推荐，对应 Gorse 推荐的 GetNeighbors API。
	NEIGHBORS ProviderName = "neighbors"
	// ITEM_TO_ITEM_GOODS_RELATION 表示命名 item-to-item/goods_relation 推荐器。
	ITEM_TO_ITEM_GOODS_RELATION ProviderName = "item_to_item.goods_relation"
	// NON_PERSONALIZED_HOT_30D 表示命名 non-personalized/hot_30d 推荐器。
	NON_PERSONALIZED_HOT_30D ProviderName = "non_personalized.hot_30d"
	// NON_PERSONALIZED_HOT_7D 表示命名 non-personalized/hot_7d 推荐器。
	NON_PERSONALIZED_HOT_7D ProviderName = "non_personalized.hot_7d"
	// NON_PERSONALIZED_HOT_PAY_30D 表示命名 non-personalized/hot_pay_30d 推荐器。
	NON_PERSONALIZED_HOT_PAY_30D ProviderName = "non_personalized.hot_pay_30d"
	// LATEST 表示最新商品推荐，对应 Gorse 推荐的 GetLatestItems API。
	LATEST ProviderName = "latest"
)

// ChainReceiver 表示 Gorse 推荐责任链接收器。
type ChainReceiver struct {
	recommend *Recommend
	user      *UserReceiver
	session   *SessionReceiver
	named     *NamedReceiver
}

// NewChainReceiver 创建 Gorse 推荐责任链接收器。
func NewChainReceiver(recommend *Recommend, user *UserReceiver, session *SessionReceiver, named *NamedReceiver) *ChainReceiver {
	return &ChainReceiver{
		recommend: recommend,
		user:      user,
		session:   session,
		named:     named,
	}
}

// Enabled 判断当前 Gorse 推荐责任链接收器是否可用。
func (r *ChainReceiver) Enabled() bool {
	return r.recommend.Enabled()
}

// ExecutePlan 按场景组装步骤并执行 Gorse 推荐责任链。
func (r *ChainReceiver) ExecutePlan(
	ctx context.Context,
	scene commonv1.RecommendScene,
	actor *dto.RecommendActor,
	goodsID int64,
	contextGoodsIDs []int64,
	pageNum, pageSize int64,
) (*dto.GoodsResult, error) {
	result := &dto.GoodsResult{
		GoodsIDs: []int64{},
		Strategy: commonv1.RecommendStrategy(_const.RECOMMEND_STRATEGY_REMOTE),
		Trace:    make([]*dto.GoodsTrace, 0),
	}
	// 责任链接收器未启用时，直接返回空结果，交由业务侧继续走本地兜底。
	if !r.Enabled() {
		return result, nil
	}

	chain := r.buildRecommendChain(scene, actor)
	// 推荐责任链为空时，直接返回空结果，交由业务侧继续走本地兜底。
	if len(chain) == 0 {
		return result, nil
	}

	providers := r.buildProviders(actor, goodsID, contextGoodsIDs, pageNum, pageSize)
	for _, providerName := range chain {

		execute, ok := providers[providerName]
		// 当前 provider 未注册时，记录轨迹后继续后续步骤。
		if !ok {
			result.Trace = append(result.Trace, &dto.GoodsTrace{
				ProviderName: string(providerName),
				ErrorMsg:     "provider not registered",
			})
			continue
		}

		goodsIDs, total, err := execute(ctx)
		trace := &dto.GoodsTrace{
			ProviderName: string(providerName),
			ResultCount:  len(goodsIDs),
			Hit:          err == nil && len(goodsIDs) > 0,
		}
		// 当前提供方执行失败时，记录轨迹后继续尝试下一个链路节点。
		if err != nil {
			trace.ErrorMsg = err.Error()
			result.Trace = append(result.Trace, trace)
			continue
		}

		result.Trace = append(result.Trace, trace)
		// 当前提供方没有命中推荐结果时，继续执行后续链路节点。
		if len(goodsIDs) == 0 {
			continue
		}

		result.GoodsIDs = goodsIDs
		result.Total = total
		result.ProviderName = string(providerName)
		return result, nil
	}
	return result, nil
}

// buildRecommendChain 按场景构建Gorse 推荐责任链。
func (r *ChainReceiver) buildRecommendChain(scene commonv1.RecommendScene, actor *dto.RecommendActor) []ProviderName {
	isLogin := actor.IsUser()
	steps := make([]ProviderName, 0, 6)
	switch scene {
	case commonv1.RecommendScene(_const.RECOMMEND_SCENE_HOME):
		// 首页
		// 个性化推荐 -> 相似用户推荐 -> 会话推荐 -> 30 天热门商品 -> 最新商品
		if isLogin {
			steps = append(steps, GET_RECOMMEND)
			steps = append(steps, USER_TO_USER_SIMILAR_USERS)
		}
		// 会话推荐 -> 30 天热门商品 -> 最新商品
		steps = append(steps, SESSION)
		steps = append(steps, NON_PERSONALIZED_HOT_30D)
		steps = append(steps, LATEST)
	case commonv1.RecommendScene(_const.RECOMMEND_SCENE_GOODS_DETAIL):
		// 商品详情
		// 相邻商品推荐 -> 商品关联推荐 -> 会话推荐 -> 30 天热门商品 -> 最新商品
		steps = append(steps, NEIGHBORS)
		steps = append(steps, ITEM_TO_ITEM_GOODS_RELATION)
		steps = append(steps, SESSION)
		steps = append(steps, NON_PERSONALIZED_HOT_30D)
		steps = append(steps, LATEST)
	case commonv1.RecommendScene(_const.RECOMMEND_SCENE_CART):
		// 购物车
		// 会话推荐 -> 30 天热门商品 -> 最新商品
		if isLogin {
			steps = append(steps, SESSION)
		} else {
			// 7 天热门商品 -> 30 天热门商品 -> 最新商品
			steps = append(steps, NON_PERSONALIZED_HOT_7D)
		}
		steps = append(steps, NON_PERSONALIZED_HOT_30D)
		steps = append(steps, LATEST)
	case commonv1.RecommendScene(_const.RECOMMEND_SCENE_PROFILE):
		// 个人中心
		// 个性化推荐 -> 相似用户推荐 -> 会话推荐 -> 30 天热门商品 -> 最新商品
		if isLogin {
			steps = append(steps, GET_RECOMMEND)
			steps = append(steps, USER_TO_USER_SIMILAR_USERS)
			steps = append(steps, SESSION)
		} else {
			// 7天热门商品 -> 30 天热门商品 -> 最新商品
			steps = append(steps, NON_PERSONALIZED_HOT_7D)
		}
		steps = append(steps, NON_PERSONALIZED_HOT_30D)
		steps = append(steps, LATEST)
	case commonv1.RecommendScene(_const.RECOMMEND_SCENE_ORDER_DETAIL):
		// 订单详情
		// 商品关联推荐 -> 相邻商品推荐 -> 会话推荐 -> 30 天支付热门商品 -> 30 天热门商品 -> 最新商品。
		steps = append(steps, ITEM_TO_ITEM_GOODS_RELATION)
		steps = append(steps, NEIGHBORS)
		steps = append(steps, SESSION)
		steps = append(steps, NON_PERSONALIZED_HOT_PAY_30D)
		steps = append(steps, NON_PERSONALIZED_HOT_30D)
		steps = append(steps, LATEST)
	case commonv1.RecommendScene(_const.RECOMMEND_SCENE_ORDER_PAID):
		// 支付成功页
		// 会话推荐 -> 商品关联推荐 -> 30 天支付热门商品 -> 30 天热门商品 -> 最新商品。
		steps = append(steps, SESSION)
		steps = append(steps, ITEM_TO_ITEM_GOODS_RELATION)
		steps = append(steps, NON_PERSONALIZED_HOT_PAY_30D)
		steps = append(steps, NON_PERSONALIZED_HOT_30D)
		steps = append(steps, LATEST)
	default:
		// 兜底
		// 个性化推荐 -> 相似用户推荐 -> 会话推荐 -> 30 天热门商品 -> 最新商品。
		if isLogin {
			// 登录用户仍然先尝试个性化召回，保证通用场景也能优先命中画像相关推荐。
			steps = append(steps, GET_RECOMMEND)
			steps = append(steps, USER_TO_USER_SIMILAR_USERS)
		}
		// 会话推荐 -> 30 天热门商品 -> 最新商品。
		steps = append(steps, SESSION)
		steps = append(steps, NON_PERSONALIZED_HOT_30D)
		steps = append(steps, LATEST)
	}

	return steps
}

// buildProviders 构建Gorse 推荐 provider 注册表。
func (r *ChainReceiver) buildProviders(
	actor *dto.RecommendActor,
	goodsID int64,
	contextGoodsIDs []int64,
	pageNum, pageSize int64,
) map[ProviderName]func(ctx context.Context) ([]int64, int64, error) {
	return map[ProviderName]func(ctx context.Context) ([]int64, int64, error){
		GET_RECOMMEND: func(ctx context.Context) ([]int64, int64, error) {
			return r.user.GetGoodsIDs(ctx, actor, pageNum, pageSize)
		},
		USER_TO_USER_SIMILAR_USERS: func(ctx context.Context) ([]int64, int64, error) {
			return r.named.GetUserToUserGoodsIDs(ctx, "similar_users", actor, pageNum, pageSize)
		},
		SESSION: func(ctx context.Context) ([]int64, int64, error) {
			return r.session.GetGoodsIDs(ctx, contextGoodsIDs, pageNum, pageSize)
		},
		NEIGHBORS: func(ctx context.Context) ([]int64, int64, error) {
			anchorGoodsID := r.recommend.resolveAnchorGoodsID(goodsID, contextGoodsIDs)
			return r.named.GetNeighborsGoodsIDs(ctx, anchorGoodsID, pageNum, pageSize)
		},
		ITEM_TO_ITEM_GOODS_RELATION: func(ctx context.Context) ([]int64, int64, error) {
			anchorGoodsID := r.recommend.resolveAnchorGoodsID(goodsID, contextGoodsIDs)
			return r.named.GetItemToItemGoodsIDs(ctx, "goods_relation", anchorGoodsID, pageNum, pageSize)
		},
		NON_PERSONALIZED_HOT_30D: func(ctx context.Context) ([]int64, int64, error) {
			return r.named.GetNonPersonalizedGoodsIDs(ctx, "hot_30d", pageNum, pageSize)
		},
		NON_PERSONALIZED_HOT_7D: func(ctx context.Context) ([]int64, int64, error) {
			return r.named.GetNonPersonalizedGoodsIDs(ctx, "hot_7d", pageNum, pageSize)
		},
		NON_PERSONALIZED_HOT_PAY_30D: func(ctx context.Context) ([]int64, int64, error) {
			return r.named.GetNonPersonalizedGoodsIDs(ctx, "hot_pay_30d", pageNum, pageSize)
		},
		LATEST: func(ctx context.Context) ([]int64, int64, error) {
			return r.session.GetLatestGoodsIDs(ctx, pageNum, pageSize)
		},
	}
}

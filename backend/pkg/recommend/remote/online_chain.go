package remote

import (
	"context"

	"shop/api/gen/go/common"
	"shop/pkg/recommend/dto"
)

// RemoteProviderName 表示在线推荐 provider 标识。
type RemoteProviderName string

const (
	// GetRecommend 表示登录用户个性化推荐，对应 Gorse 的 GetRecommend API。
	GetRecommend RemoteProviderName = "recommend"
	// UserToUserSimilarUsers 表示命名 user-to-user/similar_users 推荐器。
	UserToUserSimilarUsers RemoteProviderName = "user_to_user.similar_users"
	// Session 表示会话级推荐，对应 Gorse 的 SessionRecommend API。
	Session RemoteProviderName = "session"
	// Neighbors 表示相邻商品推荐，对应 Gorse 的 GetNeighbors API。
	Neighbors RemoteProviderName = "neighbors"
	// ItemToItemGoodsRelation 表示命名 item-to-item/goods_relation 推荐器。
	ItemToItemGoodsRelation RemoteProviderName = "item_to_item.goods_relation"
	// NonPersonalizedHot30d 表示命名 non-personalized/hot_30d 推荐器。
	NonPersonalizedHot30d RemoteProviderName = "non_personalized.hot_30d"
	// NonPersonalizedHot7d 表示命名 non-personalized/hot_7d 推荐器。
	NonPersonalizedHot7d RemoteProviderName = "non_personalized.hot_7d"
	// NonPersonalizedHotPay30d 表示命名 non-personalized/hot_pay_30d 推荐器。
	NonPersonalizedHotPay30d RemoteProviderName = "non_personalized.hot_pay_30d"
	// Latest 表示最新商品推荐，对应 Gorse 的 GetLatestItems API。
	Latest RemoteProviderName = "latest"
)

// OnlineChainReceiver 表示在线推荐责任链接收器。
type OnlineChainReceiver struct {
	recommend     *Recommend
	onlineUser    *OnlineUserReceiver
	onlineSession *OnlineSessionReceiver
	onlineNamed   *OnlineNamedReceiver
}

// NewOnlineChainReceiver 创建在线推荐责任链接收器。
func NewOnlineChainReceiver(recommend *Recommend, onlineUser *OnlineUserReceiver, onlineSession *OnlineSessionReceiver, onlineNamed *OnlineNamedReceiver) *OnlineChainReceiver {
	return &OnlineChainReceiver{
		recommend:     recommend,
		onlineUser:    onlineUser,
		onlineSession: onlineSession,
		onlineNamed:   onlineNamed,
	}
}

// Enabled 判断当前在线推荐责任链接收器是否可用。
func (r *OnlineChainReceiver) Enabled() bool {
	return r.recommend.Enabled()
}

// ExecuteOnlinePlan 按场景组装步骤并执行在线推荐责任链。
func (r *OnlineChainReceiver) ExecuteOnlinePlan(
	ctx context.Context,
	scene common.RecommendScene,
	actor *dto.RecommendActor,
	goodsId int64,
	contextGoodsIds []int64,
	pageNum, pageSize int64,
) (*dto.GoodsResult, error) {
	result := &dto.GoodsResult{
		GoodsIds: []int64{},
		Strategy: dto.RemoteStrategy,
		Trace:    make([]*dto.GoodsTrace, 0),
	}
	// 责任链接收器未启用时，直接返回空结果，交由业务侧继续走本地兜底。
	if !r.Enabled() {
		return result, nil
	}

	chain := r.buildOnlineRecommendChain(scene, actor)
	// 推荐责任链为空时，直接返回空结果，交由业务侧继续走本地兜底。
	if len(chain) == 0 {
		return result, nil
	}

	providers := r.buildProviders(actor, goodsId, contextGoodsIds, pageNum, pageSize)
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

		goodsIds, total, err := execute(ctx)
		trace := &dto.GoodsTrace{
			ProviderName: string(providerName),
			ResultCount:  len(goodsIds),
			Hit:          err == nil && len(goodsIds) > 0,
		}
		// 当前提供方执行失败时，记录轨迹后继续尝试下一个链路节点。
		if err != nil {
			trace.ErrorMsg = err.Error()
			result.Trace = append(result.Trace, trace)
			continue
		}

		result.Trace = append(result.Trace, trace)
		// 当前提供方没有命中推荐结果时，继续执行后续链路节点。
		if len(goodsIds) == 0 {
			continue
		}

		result.GoodsIds = goodsIds
		result.Total = total
		result.ProviderName = string(providerName)
		return result, nil
	}
	return result, nil
}

// buildOnlineRecommendChain 按场景构建在线推荐责任链。
func (r *OnlineChainReceiver) buildOnlineRecommendChain(scene common.RecommendScene, actor *dto.RecommendActor) []RemoteProviderName {
	// 推荐主体缺失或主体编号非法时，当前请求无法走推荐系统推荐。
	if !actor.IsValid() {
		return []RemoteProviderName{}
	}

	isLogin := actor.IsUser()
	steps := make([]RemoteProviderName, 0, 6)
	switch scene {
	case common.RecommendScene_HOME:
		// 首页
		// 个性化推荐 -> 相似用户推荐 -> 会话推荐 -> 30 天热门商品 -> 最新商品
		if isLogin {
			steps = append(steps, GetRecommend)
			steps = append(steps, UserToUserSimilarUsers)
		}
		// 会话推荐 -> 30 天热门商品 -> 最新商品
		steps = append(steps, Session)
		steps = append(steps, NonPersonalizedHot30d)
		steps = append(steps, Latest)
	case common.RecommendScene_GOODS_DETAIL:
		// 商品详情
		// 相邻商品推荐 -> 商品关联推荐 -> 会话推荐 -> 30 天热门商品 -> 最新商品
		steps = append(steps, Neighbors)
		steps = append(steps, ItemToItemGoodsRelation)
		steps = append(steps, Session)
		steps = append(steps, NonPersonalizedHot30d)
		steps = append(steps, Latest)
	case common.RecommendScene_CART:
		// 购物车
		// 会话推荐 -> 30 天热门商品 -> 最新商品
		if isLogin {
			steps = append(steps, Session)
		} else {
			// 7 天热门商品 -> 30 天热门商品 -> 最新商品
			steps = append(steps, NonPersonalizedHot7d)
		}
		steps = append(steps, NonPersonalizedHot30d)
		steps = append(steps, Latest)
	case common.RecommendScene_PROFILE:
		// 个人中心
		// 个性化推荐 -> 相似用户推荐 -> 会话推荐 -> 30 天热门商品 -> 最新商品
		if isLogin {
			steps = append(steps, GetRecommend)
			steps = append(steps, UserToUserSimilarUsers)
			steps = append(steps, Session)
		} else {
			// 7天热门商品 -> 30 天热门商品 -> 最新商品
			steps = append(steps, NonPersonalizedHot7d)
		}
		steps = append(steps, NonPersonalizedHot30d)
		steps = append(steps, Latest)
	case common.RecommendScene_ORDER_DETAIL:
		// 订单详情
		// 商品关联推荐 -> 相邻商品推荐 -> 会话推荐 -> 30 天支付热门商品 -> 30 天热门商品 -> 最新商品。
		steps = append(steps, ItemToItemGoodsRelation)
		steps = append(steps, Neighbors)
		steps = append(steps, Session)
		steps = append(steps, NonPersonalizedHotPay30d)
		steps = append(steps, NonPersonalizedHot30d)
		steps = append(steps, Latest)
	case common.RecommendScene_ORDER_PAID:
		// 支付成功页
		// 会话推荐 -> 商品关联推荐 -> 30 天支付热门商品 -> 30 天热门商品 -> 最新商品。
		steps = append(steps, Session)
		steps = append(steps, ItemToItemGoodsRelation)
		steps = append(steps, NonPersonalizedHotPay30d)
		steps = append(steps, NonPersonalizedHot30d)
		steps = append(steps, Latest)
	default:
		// 兜底
		// 个性化推荐 -> 相似用户推荐 -> 会话推荐 -> 30 天热门商品 -> 最新商品。
		if isLogin {
			// 登录用户仍然先尝试个性化召回，保证通用场景也能优先命中画像相关推荐。
			steps = append(steps, GetRecommend)
			steps = append(steps, UserToUserSimilarUsers)
		}
		// 会话推荐 -> 30 天热门商品 -> 最新商品。
		steps = append(steps, Session)
		steps = append(steps, NonPersonalizedHot30d)
		steps = append(steps, Latest)
	}

	return steps
}

// buildProviders 构建在线推荐 provider 注册表。
func (r *OnlineChainReceiver) buildProviders(
	actor *dto.RecommendActor,
	goodsId int64,
	contextGoodsIds []int64,
	pageNum, pageSize int64,
) map[RemoteProviderName]func(ctx context.Context) ([]int64, int64, error) {
	return map[RemoteProviderName]func(ctx context.Context) ([]int64, int64, error){
		GetRecommend: func(ctx context.Context) ([]int64, int64, error) {
			return r.onlineUser.GetGoodsIds(ctx, actor, pageNum, pageSize)
		},
		UserToUserSimilarUsers: func(ctx context.Context) ([]int64, int64, error) {
			return r.onlineNamed.GetUserToUserGoodsIds(ctx, "similar_users", actor, pageNum, pageSize)
		},
		Session: func(ctx context.Context) ([]int64, int64, error) {
			return r.onlineSession.GetGoodsIds(ctx, contextGoodsIds, pageNum, pageSize)
		},
		Neighbors: func(ctx context.Context) ([]int64, int64, error) {
			anchorGoodsId := r.recommend.resolveAnchorGoodsId(goodsId, contextGoodsIds)
			return r.onlineNamed.GetNeighborsGoodsIds(ctx, anchorGoodsId, pageNum, pageSize)
		},
		ItemToItemGoodsRelation: func(ctx context.Context) ([]int64, int64, error) {
			anchorGoodsId := r.recommend.resolveAnchorGoodsId(goodsId, contextGoodsIds)
			return r.onlineNamed.GetItemToItemGoodsIds(ctx, "goods_relation", anchorGoodsId, pageNum, pageSize)
		},
		NonPersonalizedHot30d: func(ctx context.Context) ([]int64, int64, error) {
			return r.onlineNamed.GetNonPersonalizedGoodsIds(ctx, "hot_30d", pageNum, pageSize)
		},
		NonPersonalizedHot7d: func(ctx context.Context) ([]int64, int64, error) {
			return r.onlineNamed.GetNonPersonalizedGoodsIds(ctx, "hot_7d", pageNum, pageSize)
		},
		NonPersonalizedHotPay30d: func(ctx context.Context) ([]int64, int64, error) {
			return r.onlineNamed.GetNonPersonalizedGoodsIds(ctx, "hot_pay_30d", pageNum, pageSize)
		},
		Latest: func(ctx context.Context) ([]int64, int64, error) {
			return r.onlineSession.GetLatestGoodsIds(ctx, pageNum, pageSize)
		},
	}
}

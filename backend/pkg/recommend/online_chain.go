package recommend

import (
	"context"

	"shop/api/gen/go/app"
	"shop/api/gen/go/common"
)

// OnlineProviderName 表示在线推荐 provider 标识。
type OnlineProviderName string

const (
	// OnlineProviderGetRecommend 表示登录用户个性化推荐，对应 Gorse 的 GetRecommend API。
	OnlineProviderGetRecommend OnlineProviderName = "recommend"
	// OnlineProviderUserToUserSimilarUsers 表示命名 user-to-user/similar_users 推荐器。
	OnlineProviderUserToUserSimilarUsers OnlineProviderName = "user_to_user.similar_users"
	// OnlineProviderSession 表示会话级推荐，对应 Gorse 的 SessionRecommend API。
	OnlineProviderSession OnlineProviderName = "session"
	// OnlineProviderNeighbors 表示相邻商品推荐，对应 Gorse 的 GetNeighbors API。
	OnlineProviderNeighbors OnlineProviderName = "neighbors"
	// OnlineProviderItemToItemGoodsRelation 表示命名 item-to-item/goods_relation 推荐器。
	OnlineProviderItemToItemGoodsRelation OnlineProviderName = "item_to_item.goods_relation"
	// OnlineProviderNonPersonalizedHot30d 表示命名 non-personalized/hot_30d 推荐器。
	OnlineProviderNonPersonalizedHot30d OnlineProviderName = "non_personalized.hot_30d"
	// OnlineProviderNonPersonalizedHot7d 表示命名 non-personalized/hot_7d 推荐器。
	OnlineProviderNonPersonalizedHot7d OnlineProviderName = "non_personalized.hot_7d"
	// OnlineProviderNonPersonalizedHotPay30d 表示命名 non-personalized/hot_pay_30d 推荐器。
	OnlineProviderNonPersonalizedHotPay30d OnlineProviderName = "non_personalized.hot_pay_30d"
	// OnlineProviderLatest 表示最新商品推荐，对应 Gorse 的 GetLatestItems API。
	OnlineProviderLatest OnlineProviderName = "latest"
)

// OnlineRecommendStep 表示在线推荐责任链中的一个步骤。
type OnlineRecommendStep struct {
	ProviderName OnlineProviderName
}

// OnlineRecommendTrace 表示责任链执行轨迹。
type OnlineRecommendTrace struct {
	ProviderName OnlineProviderName
	ResultCount  int
	Hit          bool
	ErrorMsg     string
}

// OnlineRecommendResult 表示在线推荐执行结果。
type OnlineRecommendResult struct {
	GoodsIds     []int64
	Total        int64
	ProviderName OnlineProviderName
	Trace        []*OnlineRecommendTrace
}

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

// validateOnlineRecommendStep 校验在线推荐步骤是否合法。
func validateOnlineRecommendStep(step *OnlineRecommendStep) string {
	// 空步骤由上游直接过滤，这里只兜底防御非法输入。
	if step == nil {
		return "step is nil"
	}
	// provider 未配置时，当前步骤无法定位到具体推荐能力。
	if step.ProviderName == "" {
		return "provider name is empty"
	}
	return ""
}

// ExecuteOnlinePlan 按场景组装步骤并执行在线推荐责任链。
func (r *OnlineChainReceiver) ExecuteOnlinePlan(
	ctx context.Context,
	scene common.RecommendScene,
	actor *app.RecommendActor,
	goodsId int64,
	contextGoodsIds []int64,
	pageNum, pageSize int64,
) (*OnlineRecommendResult, error) {
	result := &OnlineRecommendResult{
		GoodsIds: []int64{},
		Trace:    make([]*OnlineRecommendTrace, 0),
	}
	// 责任链接收器未启用时，直接返回空结果，交由业务侧继续走本地兜底。
	if !r.Enabled() {
		return result, nil
	}

	steps := r.buildOnlineRecommendSteps(scene, actor)
	// 推荐步骤为空时，直接返回空结果，交由业务侧继续走本地兜底。
	if len(steps) == 0 {
		return result, nil
	}
	// 页码非法时，统一回退到第 1 页。
	if pageNum <= 0 {
		pageNum = 1
	}
	// 每页条数非法时，统一回退到 10 条。
	if pageSize <= 0 {
		pageSize = 10
	}

	providers := r.buildProviders(actor, goodsId, contextGoodsIds, pageNum, pageSize)
	for _, step := range steps {
		// 当前步骤为空时，直接忽略，避免单个空配置阻塞整条责任链。
		if step == nil {
			continue
		}
		if validationMsg := validateOnlineRecommendStep(step); validationMsg != "" {
			result.Trace = append(result.Trace, &OnlineRecommendTrace{
				ProviderName: step.ProviderName,
				ErrorMsg:     validationMsg,
			})
			continue
		}

		execute, ok := providers[step.ProviderName]
		// 当前 provider 未注册时，记录轨迹后继续后续步骤。
		if !ok {
			result.Trace = append(result.Trace, &OnlineRecommendTrace{
				ProviderName: step.ProviderName,
				ErrorMsg:     "provider not registered",
			})
			continue
		}

		goodsIds, total, err := execute(ctx, step)
		trace := &OnlineRecommendTrace{
			ProviderName: step.ProviderName,
			ResultCount:  len(goodsIds),
			Hit:          err == nil && len(goodsIds) > 0,
		}
		// 当前步骤执行失败时，记录轨迹后继续尝试下一个步骤。
		if err != nil {
			trace.ErrorMsg = err.Error()
			result.Trace = append(result.Trace, trace)
			continue
		}

		result.Trace = append(result.Trace, trace)
		// 当前步骤没有命中推荐结果时，继续执行后续步骤。
		if len(goodsIds) == 0 {
			continue
		}

		result.GoodsIds = goodsIds
		result.Total = total
		result.ProviderName = step.ProviderName
		return result, nil
	}
	return result, nil
}

// buildOnlineRecommendSteps 按场景构建在线推荐步骤。
func (r *OnlineChainReceiver) buildOnlineRecommendSteps(scene common.RecommendScene, actor *app.RecommendActor) []*OnlineRecommendStep {
	// 推荐主体缺失或主体编号非法时，当前请求无法走推荐系统推荐。
	if actor == nil || actor.GetActorId() <= 0 {
		return []*OnlineRecommendStep{}
	}

	isLogin := actor.GetActorType() == common.RecommendActorType_USER
	steps := make([]*OnlineRecommendStep, 0, 6)
	switch scene {
	case common.RecommendScene_HOME:
		// 首页登录态优先走用户画像推荐，未登录则优先走会话推荐。
		if isLogin {
			steps = append(steps, &OnlineRecommendStep{ProviderName: OnlineProviderGetRecommend})
			steps = append(steps, &OnlineRecommendStep{ProviderName: OnlineProviderUserToUserSimilarUsers})
			steps = append(steps, &OnlineRecommendStep{ProviderName: OnlineProviderSession})
		} else {
			steps = append(steps, &OnlineRecommendStep{ProviderName: OnlineProviderSession})
		}
		steps = append(steps, &OnlineRecommendStep{ProviderName: OnlineProviderNonPersonalizedHot30d})
		steps = append(steps, &OnlineRecommendStep{ProviderName: OnlineProviderLatest})
	case common.RecommendScene_GOODS_DETAIL:
		// 商品详情优先围绕当前商品做相似推荐，再回退到会话和热门兜底。
		steps = append(steps, &OnlineRecommendStep{ProviderName: OnlineProviderNeighbors})
		steps = append(steps, &OnlineRecommendStep{ProviderName: OnlineProviderItemToItemGoodsRelation})
		steps = append(steps, &OnlineRecommendStep{ProviderName: OnlineProviderSession})
		steps = append(steps, &OnlineRecommendStep{ProviderName: OnlineProviderNonPersonalizedHot30d})
		steps = append(steps, &OnlineRecommendStep{ProviderName: OnlineProviderLatest})
	case common.RecommendScene_CART:
		// 购物车登录态优先走购物篮推荐，游客态优先走近期热门。
		if isLogin {
			steps = append(steps, &OnlineRecommendStep{ProviderName: OnlineProviderSession})
			steps = append(steps, &OnlineRecommendStep{ProviderName: OnlineProviderNonPersonalizedHot30d})
		} else {
			steps = append(steps, &OnlineRecommendStep{ProviderName: OnlineProviderNonPersonalizedHot7d})
			steps = append(steps, &OnlineRecommendStep{ProviderName: OnlineProviderNonPersonalizedHot30d})
		}
		steps = append(steps, &OnlineRecommendStep{ProviderName: OnlineProviderLatest})
	case common.RecommendScene_PROFILE:
		// 个人中心登录态优先走用户画像推荐，游客态保持热门语义。
		if isLogin {
			steps = append(steps, &OnlineRecommendStep{ProviderName: OnlineProviderGetRecommend})
			steps = append(steps, &OnlineRecommendStep{ProviderName: OnlineProviderUserToUserSimilarUsers})
			steps = append(steps, &OnlineRecommendStep{ProviderName: OnlineProviderSession})
			steps = append(steps, &OnlineRecommendStep{ProviderName: OnlineProviderNonPersonalizedHot30d})
		} else {
			steps = append(steps, &OnlineRecommendStep{ProviderName: OnlineProviderNonPersonalizedHot7d})
			steps = append(steps, &OnlineRecommendStep{ProviderName: OnlineProviderNonPersonalizedHot30d})
		}
		steps = append(steps, &OnlineRecommendStep{ProviderName: OnlineProviderLatest})
	case common.RecommendScene_ORDER_DETAIL:
		// 订单详情优先走 also-buy 语义，再回退到相似商品、会话和购买导向热榜。
		steps = append(steps, &OnlineRecommendStep{ProviderName: OnlineProviderItemToItemGoodsRelation})
		steps = append(steps, &OnlineRecommendStep{ProviderName: OnlineProviderNeighbors})
		steps = append(steps, &OnlineRecommendStep{ProviderName: OnlineProviderSession})
		steps = append(steps, &OnlineRecommendStep{ProviderName: OnlineProviderNonPersonalizedHotPay30d})
		steps = append(steps, &OnlineRecommendStep{ProviderName: OnlineProviderNonPersonalizedHot30d})
		steps = append(steps, &OnlineRecommendStep{ProviderName: OnlineProviderLatest})
	case common.RecommendScene_ORDER_PAID:
		// 支付成功页优先走订单商品会话推荐，再回退到 also-buy 和支付导向热榜。
		steps = append(steps, &OnlineRecommendStep{ProviderName: OnlineProviderSession})
		steps = append(steps, &OnlineRecommendStep{ProviderName: OnlineProviderItemToItemGoodsRelation})
		steps = append(steps, &OnlineRecommendStep{ProviderName: OnlineProviderNonPersonalizedHotPay30d})
		steps = append(steps, &OnlineRecommendStep{ProviderName: OnlineProviderNonPersonalizedHot30d})
		steps = append(steps, &OnlineRecommendStep{ProviderName: OnlineProviderLatest})
	default:
		// 其余场景统一回退到通用 feed 链路，避免出现空步骤直接跳过在线推荐。
		if isLogin {
			steps = append(steps, &OnlineRecommendStep{ProviderName: OnlineProviderGetRecommend})
			steps = append(steps, &OnlineRecommendStep{ProviderName: OnlineProviderUserToUserSimilarUsers})
			steps = append(steps, &OnlineRecommendStep{ProviderName: OnlineProviderSession})
		} else {
			steps = append(steps, &OnlineRecommendStep{ProviderName: OnlineProviderSession})
		}
		steps = append(steps, &OnlineRecommendStep{ProviderName: OnlineProviderNonPersonalizedHot30d})
		steps = append(steps, &OnlineRecommendStep{ProviderName: OnlineProviderLatest})
	}

	return steps
}

// buildProviders 构建在线推荐 provider 注册表。
func (r *OnlineChainReceiver) buildProviders(
	actor *app.RecommendActor,
	goodsId int64,
	contextGoodsIds []int64,
	pageNum, pageSize int64,
) map[OnlineProviderName]func(ctx context.Context, step *OnlineRecommendStep) ([]int64, int64, error) {
	return map[OnlineProviderName]func(ctx context.Context, step *OnlineRecommendStep) ([]int64, int64, error){
		OnlineProviderGetRecommend: func(ctx context.Context, _ *OnlineRecommendStep) ([]int64, int64, error) {
			return r.onlineUser.GetGoodsIds(ctx, actor, pageNum, pageSize)
		},
		OnlineProviderUserToUserSimilarUsers: func(ctx context.Context, _ *OnlineRecommendStep) ([]int64, int64, error) {
			return r.onlineNamed.GetUserToUserGoodsIds(ctx, "similar_users", actor, pageNum, pageSize)
		},
		OnlineProviderSession: func(ctx context.Context, _ *OnlineRecommendStep) ([]int64, int64, error) {
			return r.onlineSession.GetGoodsIds(ctx, contextGoodsIds, pageNum, pageSize)
		},
		OnlineProviderNeighbors: func(ctx context.Context, _ *OnlineRecommendStep) ([]int64, int64, error) {
			anchorGoodsId := r.recommend.resolveAnchorGoodsId(goodsId, contextGoodsIds)
			return r.onlineNamed.GetNeighborsGoodsIds(ctx, anchorGoodsId, pageNum, pageSize)
		},
		OnlineProviderItemToItemGoodsRelation: func(ctx context.Context, _ *OnlineRecommendStep) ([]int64, int64, error) {
			anchorGoodsId := r.recommend.resolveAnchorGoodsId(goodsId, contextGoodsIds)
			return r.onlineNamed.GetItemToItemGoodsIds(ctx, "goods_relation", anchorGoodsId, pageNum, pageSize)
		},
		OnlineProviderNonPersonalizedHot30d: func(ctx context.Context, _ *OnlineRecommendStep) ([]int64, int64, error) {
			return r.onlineNamed.GetNonPersonalizedGoodsIds(ctx, "hot_30d", pageNum, pageSize)
		},
		OnlineProviderNonPersonalizedHot7d: func(ctx context.Context, _ *OnlineRecommendStep) ([]int64, int64, error) {
			return r.onlineNamed.GetNonPersonalizedGoodsIds(ctx, "hot_7d", pageNum, pageSize)
		},
		OnlineProviderNonPersonalizedHotPay30d: func(ctx context.Context, _ *OnlineRecommendStep) ([]int64, int64, error) {
			return r.onlineNamed.GetNonPersonalizedGoodsIds(ctx, "hot_pay_30d", pageNum, pageSize)
		},
		OnlineProviderLatest: func(ctx context.Context, _ *OnlineRecommendStep) ([]int64, int64, error) {
			return r.onlineSession.GetLatestGoodsIds(ctx, pageNum, pageSize)
		},
	}
}

package recommend

import (
	"context"

	"shop/api/gen/go/app"
)

// OnlineProviderName 表示在线推荐 provider 标识。
type OnlineProviderName string

const (
	// OnlineProviderGetRecommend 表示登录用户个性化推荐，对应 Gorse 的 GetRecommend API。
	OnlineProviderGetRecommend OnlineProviderName = "recommend"
	// OnlineProviderSession 表示会话级推荐，对应 Gorse 的 SessionRecommend API。
	OnlineProviderSession OnlineProviderName = "session"
	// OnlineProviderNeighbors 表示相邻商品推荐，对应 Gorse 的 GetNeighbors API。
	OnlineProviderNeighbors OnlineProviderName = "neighbors"
	// OnlineProviderItemToItem 表示命名 item-to-item 推荐器，对应 Gorse 的 /api/item-to-item/{name}/{item-id} API。
	OnlineProviderItemToItem OnlineProviderName = "item_to_item"
	// OnlineProviderNonPersonalized 表示命名非个性化推荐器，对应 Gorse 的 /api/non-personalized/{name} API。
	OnlineProviderNonPersonalized OnlineProviderName = "non_personalized"
	// OnlineProviderLatest 表示最新商品推荐，对应 Gorse 的 GetLatestItems API。
	OnlineProviderLatest OnlineProviderName = "latest"
)

// OnlineRecommendStep 表示在线推荐责任链中的一个步骤。
type OnlineRecommendStep struct {
	ProviderName    OnlineProviderName
	RecommenderName string
}

// OnlineRecommendPlan 表示一条在线推荐执行链。
type OnlineRecommendPlan struct {
	Steps []*OnlineRecommendStep
}

// OnlineRecommendRequest 表示在线推荐执行入参。
type OnlineRecommendRequest struct {
	Actor           *app.RecommendActor
	GoodsId         int64
	ContextGoodsIds []int64
	PageNum         int64
	PageSize        int64
}

// OnlineRecommendTrace 表示责任链执行轨迹。
type OnlineRecommendTrace struct {
	ProviderName    OnlineProviderName
	RecommenderName string
	ResultCount     int
	Hit             bool
	ErrorMsg        string
}

// OnlineRecommendResult 表示在线推荐执行结果。
type OnlineRecommendResult struct {
	GoodsIds        []int64
	Total           int64
	ProviderName    OnlineProviderName
	RecommenderName string
	Trace           []*OnlineRecommendTrace
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

// onlineRecommendProvider 表示单个在线推荐 provider。
type onlineRecommendProvider interface {
	Name() OnlineProviderName
	Execute(ctx context.Context, req *OnlineRecommendRequest, step *OnlineRecommendStep) ([]int64, int64, error)
}

type onlineProviderFunc struct {
	providerName OnlineProviderName
	execute      func(ctx context.Context, req *OnlineRecommendRequest, step *OnlineRecommendStep) ([]int64, int64, error)
}

// Name 返回当前 provider 标识。
func (p *onlineProviderFunc) Name() OnlineProviderName {
	return p.providerName
}

// Execute 执行当前 provider 逻辑。
func (p *onlineProviderFunc) Execute(ctx context.Context, req *OnlineRecommendRequest, step *OnlineRecommendStep) ([]int64, int64, error) {
	// 执行函数未注入时，直接返回空结果，交由责任链继续处理下一个 provider。
	if p == nil || p.execute == nil {
		return []int64{}, 0, nil
	}
	return p.execute(ctx, req, step)
}

// ExecuteOnlinePlan 按责任链顺序执行在线推荐计划。
func (r *OnlineChainReceiver) ExecuteOnlinePlan(ctx context.Context, plan *OnlineRecommendPlan, req *OnlineRecommendRequest) (*OnlineRecommendResult, error) {
	result := &OnlineRecommendResult{
		GoodsIds: []int64{},
		Trace:    make([]*OnlineRecommendTrace, 0),
	}
	// 责任链接收器未启用时，直接返回空结果，交由业务侧继续走本地兜底。
	if !r.Enabled() {
		return result, nil
	}
	// 推荐计划为空或没有步骤时，直接返回空结果，交由业务侧继续走本地兜底。
	if plan == nil || len(plan.Steps) == 0 {
		return result, nil
	}
	if req == nil {
		req = &OnlineRecommendRequest{}
	}
	// 页码非法时，统一回退到第 1 页。
	if req.PageNum <= 0 {
		req.PageNum = 1
	}
	// 每页条数非法时，统一回退到 10 条。
	if req.PageSize <= 0 {
		req.PageSize = 10
	}

	providers := r.buildProviders()
	for _, step := range plan.Steps {
		// 当前步骤为空时，直接忽略，避免单个空配置阻塞整条责任链。
		if step == nil {
			continue
		}
		provider, ok := providers[step.ProviderName]
		// 当前 provider 未注册时，记录轨迹后继续后续步骤。
		if !ok {
			result.Trace = append(result.Trace, &OnlineRecommendTrace{
				ProviderName:    step.ProviderName,
				RecommenderName: step.RecommenderName,
				ErrorMsg:        "provider not registered",
			})
			continue
		}

		goodsIds, total, err := provider.Execute(ctx, req, step)
		trace := &OnlineRecommendTrace{
			ProviderName:    step.ProviderName,
			RecommenderName: step.RecommenderName,
			ResultCount:     len(goodsIds),
			Hit:             err == nil && len(goodsIds) > 0,
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
		result.RecommenderName = step.RecommenderName
		return result, nil
	}
	return result, nil
}

// buildProviders 构建在线推荐 provider 注册表。
func (r *OnlineChainReceiver) buildProviders() map[OnlineProviderName]onlineRecommendProvider {
	return map[OnlineProviderName]onlineRecommendProvider{
		OnlineProviderGetRecommend: &onlineProviderFunc{
			providerName: OnlineProviderGetRecommend,
			execute: func(ctx context.Context, req *OnlineRecommendRequest, _ *OnlineRecommendStep) ([]int64, int64, error) {
				return r.onlineUser.GetGoodsIds(ctx, req.Actor, req.PageNum, req.PageSize)
			},
		},
		OnlineProviderSession: &onlineProviderFunc{
			providerName: OnlineProviderSession,
			execute: func(ctx context.Context, req *OnlineRecommendRequest, _ *OnlineRecommendStep) ([]int64, int64, error) {
				return r.onlineSession.GetGoodsIds(ctx, req.ContextGoodsIds, req.PageNum, req.PageSize)
			},
		},
		OnlineProviderNeighbors: &onlineProviderFunc{
			providerName: OnlineProviderNeighbors,
			execute: func(ctx context.Context, req *OnlineRecommendRequest, _ *OnlineRecommendStep) ([]int64, int64, error) {
				anchorGoodsId := r.recommend.resolveAnchorGoodsId(req.GoodsId, req.ContextGoodsIds)
				return r.onlineNamed.GetNeighborsGoodsIds(ctx, anchorGoodsId, req.PageNum, req.PageSize)
			},
		},
		OnlineProviderItemToItem: &onlineProviderFunc{
			providerName: OnlineProviderItemToItem,
			execute: func(ctx context.Context, req *OnlineRecommendRequest, step *OnlineRecommendStep) ([]int64, int64, error) {
				anchorGoodsId := r.recommend.resolveAnchorGoodsId(req.GoodsId, req.ContextGoodsIds)
				return r.onlineNamed.GetItemToItemGoodsIds(ctx, step.RecommenderName, anchorGoodsId, req.PageNum, req.PageSize)
			},
		},
		OnlineProviderNonPersonalized: &onlineProviderFunc{
			providerName: OnlineProviderNonPersonalized,
			execute: func(ctx context.Context, req *OnlineRecommendRequest, step *OnlineRecommendStep) ([]int64, int64, error) {
				return r.onlineNamed.GetNonPersonalizedGoodsIds(ctx, step.RecommenderName, req.PageNum, req.PageSize)
			},
		},
		OnlineProviderLatest: &onlineProviderFunc{
			providerName: OnlineProviderLatest,
			execute: func(ctx context.Context, req *OnlineRecommendRequest, _ *OnlineRecommendStep) ([]int64, int64, error) {
				return r.onlineSession.GetLatestGoodsIds(ctx, req.PageNum, req.PageSize)
			},
		},
	}
}

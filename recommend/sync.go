package recommend

import (
	"context"
	"recommend/internal/engine"
)

// Recommend 执行场景化推荐并返回排序后的商品结果。
func (r *Recommend) Recommend(ctx context.Context, request RecommendRequest) (*RecommendResult, error) {
	request = r.normalizeRecommendRequest(request)
	return engine.Recommend(ctx, r.dependencies, r.config, request)
}

// SyncExposure 在曝光事实落库后更新运行态惩罚与追踪数据。
func (r *Recommend) SyncExposure(ctx context.Context, request ExposureSyncRequest) error {
	return engine.SyncExposure(ctx, r.dependencies, r.config, request)
}

// SyncBehavior 在行为事实落库后更新会话态与惩罚态。
func (r *Recommend) SyncBehavior(ctx context.Context, request BehaviorSyncRequest) error {
	return engine.SyncBehavior(ctx, r.dependencies, r.config, request)
}

// SyncActorBind 在匿名主体绑定成功后归并运行态数据。
func (r *Recommend) SyncActorBind(ctx context.Context, request ActorBindRequest) error {
	return engine.SyncActorBind(ctx, r.dependencies, r.config, request)
}

// normalizeRecommendRequest 归一化在线推荐请求。
func (r *Recommend) normalizeRecommendRequest(request RecommendRequest) RecommendRequest {
	if request.Pager.PageNum <= 0 {
		request.Pager.PageNum = r.config.Query.DefaultPageNum
	}
	if request.Pager.PageSize <= 0 {
		request.Pager.PageSize = r.config.Query.DefaultPageSize
	}
	// 未显式开启 explain 时，允许实例级默认配置补齐。
	if !request.Explain && r.config.Query.DefaultExplain {
		request.Explain = true
	}
	return request
}

package recommend

import (
	"context"
	"recommend/internal/engine"
)

// Recommend 执行场景化推荐并返回排序后的商品结果。
func Recommend(ctx context.Context, dependencies Dependencies, request RecommendRequest) (*RecommendResult, error) {
	return engine.Recommend(ctx, dependencies, request)
}

// SyncExposure 在曝光事实落库后更新运行态惩罚与追踪数据。
func SyncExposure(_ context.Context, _ Dependencies, _ ExposureSyncRequest) error {
	return ErrNotImplemented
}

// SyncBehavior 在行为事实落库后更新会话态与惩罚态。
func SyncBehavior(_ context.Context, _ Dependencies, _ BehaviorSyncRequest) error {
	return ErrNotImplemented
}

// SyncActorBind 在匿名主体绑定成功后归并运行态数据。
func SyncActorBind(_ context.Context, _ Dependencies, _ ActorBindRequest) error {
	return ErrNotImplemented
}

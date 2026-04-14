package engine

import (
	"context"
	"recommend/internal/core"
	"recommend/internal/materialize"
)

// BuildNonPersonalized 构建最新商品、场景热销和全站热销候选池。
func BuildNonPersonalized(ctx context.Context, dependencies core.Dependencies, config core.ServiceConfig, request core.BuildNonPersonalizedRequest) (*core.BuildResult, error) {
	return materialize.BuildNonPersonalized(ctx, dependencies, config, request)
}

// BuildUserCandidate 构建用户商品偏好和类目偏好候选池。
func BuildUserCandidate(ctx context.Context, dependencies core.Dependencies, _ core.ServiceConfig, request core.BuildUserCandidateRequest) (*core.BuildResult, error) {
	return materialize.BuildUserCandidate(ctx, dependencies, request)
}

// BuildGoodsRelation 构建商品关联候选池。
func BuildGoodsRelation(ctx context.Context, dependencies core.Dependencies, _ core.ServiceConfig, request core.BuildGoodsRelationRequest) (*core.BuildResult, error) {
	return materialize.BuildGoodsRelation(ctx, dependencies, request)
}

// BuildUserToUser 构建相似用户召回所需的邻居用户池和商品候选池。
func BuildUserToUser(ctx context.Context, dependencies core.Dependencies, _ core.ServiceConfig, request core.BuildUserToUserRequest) (*core.BuildResult, error) {
	return materialize.BuildUserToUser(ctx, dependencies, request)
}

// BuildCollaborative 构建协同过滤候选池。
func BuildCollaborative(ctx context.Context, dependencies core.Dependencies, _ core.ServiceConfig, request core.BuildCollaborativeRequest) (*core.BuildResult, error) {
	return materialize.BuildCollaborative(ctx, dependencies, request)
}

// BuildExternal 构建活动池、营销池、人工池等外部推荐池。
func BuildExternal(ctx context.Context, dependencies core.Dependencies, _ core.ServiceConfig, request core.BuildExternalRequest) (*core.BuildResult, error) {
	return materialize.BuildExternal(ctx, dependencies, request)
}

// BuildVector 构建向量召回池。
func BuildVector(ctx context.Context, dependencies core.Dependencies, config core.ServiceConfig, request core.BuildVectorRequest) (*core.BuildResult, error) {
	return materialize.BuildVector(ctx, dependencies, config, request)
}

// TrainRanking 训练学习排序模型。
func TrainRanking(ctx context.Context, dependencies core.Dependencies, config core.ServiceConfig, request core.TrainRankingRequest) (*core.BuildResult, error) {
	return materialize.TrainRanking(ctx, dependencies, config, request)
}

// Rebuild 按统一入口执行离线池重建，并可选执行离线评估。
func Rebuild(ctx context.Context, dependencies core.Dependencies, config core.ServiceConfig, request core.RebuildRequest) (*core.RebuildResult, error) {
	return materialize.Rebuild(ctx, dependencies, config, request)
}

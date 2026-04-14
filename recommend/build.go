package recommend

import (
	"context"
	"recommend/internal/engine"
)

// BuildNonPersonalized 构建最新商品、场景热销和全站热销候选池。
func (r *Recommend) BuildNonPersonalized(ctx context.Context, request BuildNonPersonalizedRequest) (*BuildResult, error) {
	request = r.normalizeBuildNonPersonalizedRequest(request)
	return engine.BuildNonPersonalized(ctx, r.dependencies, r.config, request)
}

// BuildUserCandidate 构建用户商品偏好和类目偏好候选池。
func (r *Recommend) BuildUserCandidate(ctx context.Context, request BuildUserCandidateRequest) (*BuildResult, error) {
	request = r.normalizeBuildUserCandidateRequest(request)
	return engine.BuildUserCandidate(ctx, r.dependencies, r.config, request)
}

// BuildGoodsRelation 构建商品关联候选池。
func (r *Recommend) BuildGoodsRelation(ctx context.Context, request BuildGoodsRelationRequest) (*BuildResult, error) {
	request = r.normalizeBuildGoodsRelationRequest(request)
	return engine.BuildGoodsRelation(ctx, r.dependencies, r.config, request)
}

// BuildUserToUser 构建相似用户召回所需的邻居用户池和商品候选池。
func (r *Recommend) BuildUserToUser(ctx context.Context, request BuildUserToUserRequest) (*BuildResult, error) {
	request = r.normalizeBuildUserToUserRequest(request)
	return engine.BuildUserToUser(ctx, r.dependencies, r.config, request)
}

// BuildCollaborative 构建协同过滤候选池。
func (r *Recommend) BuildCollaborative(ctx context.Context, request BuildCollaborativeRequest) (*BuildResult, error) {
	request = r.normalizeBuildCollaborativeRequest(request)
	return engine.BuildCollaborative(ctx, r.dependencies, r.config, request)
}

// BuildExternal 构建活动池、营销池、人工池等外部推荐池。
func (r *Recommend) BuildExternal(ctx context.Context, request BuildExternalRequest) (*BuildResult, error) {
	request = r.normalizeBuildExternalRequest(request)
	return engine.BuildExternal(ctx, r.dependencies, r.config, request)
}

// BuildVector 构建向量召回池。
func (r *Recommend) BuildVector(ctx context.Context, request BuildVectorRequest) (*BuildResult, error) {
	request = r.normalizeBuildVectorRequest(request)
	return engine.BuildVector(ctx, r.dependencies, r.config, request)
}

// TrainRanking 训练学习排序模型。
func (r *Recommend) TrainRanking(ctx context.Context, request TrainRankingRequest) (*BuildResult, error) {
	request = r.normalizeTrainRankingRequest(request)
	return engine.TrainRanking(ctx, r.dependencies, r.config, request)
}

// Rebuild 按统一入口执行离线池重建，并可选触发离线评估。
func (r *Recommend) Rebuild(ctx context.Context, request RebuildRequest) (*RebuildResult, error) {
	return engine.Rebuild(ctx, r.dependencies, r.config, r.normalizeRebuildRequest(request))
}

// normalizeBuildNonPersonalizedRequest 归一化匿名通用池构建请求。
func (r *Recommend) normalizeBuildNonPersonalizedRequest(request BuildNonPersonalizedRequest) BuildNonPersonalizedRequest {
	if request.Limit <= 0 {
		request.Limit = r.config.Materialize.DefaultLimit
	}
	if len(request.Scenes) == 0 {
		request.Scenes = cloneScenes(r.config.Materialize.DefaultScenes)
	}
	return request
}

// normalizeBuildUserCandidateRequest 归一化用户候选池构建请求。
func (r *Recommend) normalizeBuildUserCandidateRequest(request BuildUserCandidateRequest) BuildUserCandidateRequest {
	if request.Limit <= 0 {
		request.Limit = r.config.Materialize.DefaultLimit
	}
	return request
}

// normalizeBuildGoodsRelationRequest 归一化商品关联池构建请求。
func (r *Recommend) normalizeBuildGoodsRelationRequest(request BuildGoodsRelationRequest) BuildGoodsRelationRequest {
	if request.Limit <= 0 {
		request.Limit = r.config.Materialize.DefaultLimit
	}
	return request
}

// normalizeBuildUserToUserRequest 归一化相似用户池构建请求。
func (r *Recommend) normalizeBuildUserToUserRequest(request BuildUserToUserRequest) BuildUserToUserRequest {
	if request.NeighborLimit <= 0 {
		request.NeighborLimit = r.config.Materialize.DefaultNeighborLimit
	}
	if request.Limit <= 0 {
		request.Limit = r.config.Materialize.DefaultLimit
	}
	return request
}

// normalizeBuildCollaborativeRequest 归一化协同过滤池构建请求。
func (r *Recommend) normalizeBuildCollaborativeRequest(request BuildCollaborativeRequest) BuildCollaborativeRequest {
	if request.Limit <= 0 {
		request.Limit = r.config.Materialize.DefaultLimit
	}
	return request
}

// normalizeBuildExternalRequest 归一化外部池构建请求。
func (r *Recommend) normalizeBuildExternalRequest(request BuildExternalRequest) BuildExternalRequest {
	if request.Limit <= 0 {
		request.Limit = r.config.Materialize.DefaultLimit
	}
	if len(request.Scenes) == 0 {
		request.Scenes = cloneScenes(r.config.Materialize.DefaultScenes)
	}
	return request
}

// normalizeBuildVectorRequest 归一化向量召回池构建请求。
func (r *Recommend) normalizeBuildVectorRequest(request BuildVectorRequest) BuildVectorRequest {
	if request.Limit <= 0 {
		request.Limit = r.config.Vector.RecallLimit
	}
	if request.Limit <= 0 {
		request.Limit = r.config.Materialize.DefaultLimit
	}
	if len(request.Scenes) == 0 {
		request.Scenes = cloneScenes(r.config.Materialize.DefaultScenes)
	}
	return request
}

// normalizeTrainRankingRequest 归一化学习排序训练请求。
func (r *Recommend) normalizeTrainRankingRequest(request TrainRankingRequest) TrainRankingRequest {
	if len(request.Scenes) == 0 {
		request.Scenes = cloneScenes(r.config.Materialize.DefaultScenes)
	}
	return request
}

// normalizeRebuildRequest 归一化一键重建请求。
func (r *Recommend) normalizeRebuildRequest(request RebuildRequest) RebuildRequest {
	if request.Limit <= 0 {
		request.Limit = r.config.Materialize.DefaultLimit
	}
	if request.NeighborLimit <= 0 {
		request.NeighborLimit = r.config.Materialize.DefaultNeighborLimit
	}
	if request.TopK <= 0 {
		request.TopK = r.config.Evaluate.DefaultTopK
	}
	if len(request.Scenes) == 0 {
		request.Scenes = cloneScenes(r.config.Materialize.DefaultScenes)
	}
	// 调用方未显式指定是否评估时，沿用实例默认配置。
	if !request.EvaluateAfterBuild && r.config.Materialize.EnableEvaluateAfterRebuild {
		request.EvaluateAfterBuild = true
	}
	return request
}

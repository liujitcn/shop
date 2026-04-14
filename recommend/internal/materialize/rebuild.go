package materialize

import (
	"context"
	"errors"
	"recommend/internal/core"
	evaluatex "recommend/internal/evaluate"
)

// Rebuild 按统一入口执行离线池重建，并可选执行离线评估。
func Rebuild(ctx context.Context, dependencies core.Dependencies, config core.ServiceConfig, request core.RebuildRequest) (*core.RebuildResult, error) {
	result := &core.RebuildResult{
		Builds: make([]core.BuildResult, 0, 8),
	}

	request = normalizeRebuildRequest(config, request)
	scopes := resolveRebuildScopes(request, config)

	if scopes.includeNonPersonalized {
		buildResult, err := BuildNonPersonalized(ctx, dependencies, config, core.BuildNonPersonalizedRequest{
			Scenes:   request.Scenes,
			StatDate: request.StatDate,
			Limit:    request.Limit,
		})
		if err != nil {
			return nil, err
		}
		result.Builds = append(result.Builds, *buildResult)
	}

	if scopes.includeUserCandidate && len(request.UserIds) > 0 {
		buildResult, err := BuildUserCandidate(ctx, dependencies, core.BuildUserCandidateRequest{
			UserIds: request.UserIds,
			Limit:   request.Limit,
		})
		if err != nil {
			return nil, err
		}
		result.Builds = append(result.Builds, *buildResult)
	}

	if scopes.includeGoodsRelation && len(request.GoodsIds) > 0 {
		buildResult, err := BuildGoodsRelation(ctx, dependencies, core.BuildGoodsRelationRequest{
			GoodsIds: request.GoodsIds,
			Limit:    request.Limit,
		})
		if err != nil {
			return nil, err
		}
		result.Builds = append(result.Builds, *buildResult)
	}

	if scopes.includeUserToUser && len(request.UserIds) > 0 {
		buildResult, err := BuildUserToUser(ctx, dependencies, core.BuildUserToUserRequest{
			UserIds:       request.UserIds,
			NeighborLimit: request.NeighborLimit,
			Limit:         request.Limit,
		})
		if err != nil {
			return nil, err
		}
		result.Builds = append(result.Builds, *buildResult)
	}

	if scopes.includeCollaborative && len(request.UserIds) > 0 {
		buildResult, err := BuildCollaborative(ctx, dependencies, core.BuildCollaborativeRequest{
			UserIds: request.UserIds,
			Limit:   request.Limit,
		})
		if err != nil {
			return nil, err
		}
		result.Builds = append(result.Builds, *buildResult)
	}

	if scopes.includeExternal && len(request.Strategies) > 0 {
		buildResult, err := BuildExternal(ctx, dependencies, core.BuildExternalRequest{
			Scenes:     request.Scenes,
			Strategies: request.Strategies,
			ActorType:  request.ActorType,
			ActorIds:   request.ActorIds,
			Limit:      request.Limit,
		})
		if err != nil {
			return nil, err
		}
		result.Builds = append(result.Builds, *buildResult)
	}

	if scopes.includeVector && (len(request.UserIds) > 0 || len(request.GoodsIds) > 0) {
		if dependencies.Vector == nil {
			if request.IncludeVector {
				return nil, errors.New("recommend: 向量数据源未配置")
			}
		} else {
			buildResult, err := BuildVector(ctx, dependencies, config, core.BuildVectorRequest{
				Scenes:   request.Scenes,
				UserIds:  request.UserIds,
				GoodsIds: request.GoodsIds,
				Limit:    request.Limit,
			})
			if err != nil {
				return nil, err
			}
			result.Builds = append(result.Builds, *buildResult)
		}
	}

	if scopes.includeTraining {
		if dependencies.Recommend == nil || dependencies.Cache == nil {
			if request.IncludeTraining {
				return nil, errors.New("recommend: 训练依赖未完整配置")
			}
		} else {
			buildResult, err := TrainRanking(ctx, dependencies, config, core.TrainRankingRequest{
				Scenes:   request.Scenes,
				StatDate: request.StatDate,
			})
			if err != nil {
				return nil, err
			}
			result.Builds = append(result.Builds, *buildResult)
		}
	}

	if request.EvaluateAfterBuild {
		evaluation, err := evaluatex.EvaluateOffline(ctx, dependencies, core.EvaluateRequest{
			Scenes:   request.Scenes,
			StatDate: request.StatDate,
			TopK:     request.TopK,
		})
		if err != nil {
			return nil, err
		}
		result.Evaluation = evaluation
	}

	return result, nil
}

// rebuildScopes 表示一键重建时实际启用的构建动作集合。
type rebuildScopes struct {
	includeNonPersonalized bool
	includeUserCandidate   bool
	includeGoodsRelation   bool
	includeUserToUser      bool
	includeCollaborative   bool
	includeExternal        bool
	includeVector          bool
	includeTraining        bool
}

// normalizeRebuildRequest 归一化重建请求。
func normalizeRebuildRequest(config core.ServiceConfig, request core.RebuildRequest) core.RebuildRequest {
	if request.Limit <= 0 {
		request.Limit = config.Materialize.DefaultLimit
	}
	if request.NeighborLimit <= 0 {
		request.NeighborLimit = config.Materialize.DefaultNeighborLimit
	}
	if request.TopK <= 0 {
		request.TopK = config.Evaluate.DefaultTopK
	}
	if len(request.Scenes) == 0 {
		request.Scenes = append([]core.Scene(nil), config.Materialize.DefaultScenes...)
	}
	// 调用方未显式要求评估时，沿用实例默认配置。
	if !request.EvaluateAfterBuild && config.Materialize.EnableEvaluateAfterRebuild {
		request.EvaluateAfterBuild = true
	}
	return request
}

// resolveRebuildScopes 解析重建请求中的构建动作集合。
func resolveRebuildScopes(request core.RebuildRequest, config core.ServiceConfig) rebuildScopes {
	scopes := rebuildScopes{
		includeNonPersonalized: request.IncludeNonPersonalized,
		includeUserCandidate:   request.IncludeUserCandidate,
		includeGoodsRelation:   request.IncludeGoodsRelation,
		includeUserToUser:      request.IncludeUserToUser,
		includeCollaborative:   request.IncludeCollaborative,
		includeExternal:        request.IncludeExternal,
		includeVector:          request.IncludeVector,
		includeTraining:        request.IncludeTraining,
	}
	isAnyScopeSelected := scopes.includeNonPersonalized ||
		scopes.includeUserCandidate ||
		scopes.includeGoodsRelation ||
		scopes.includeUserToUser ||
		scopes.includeCollaborative ||
		scopes.includeExternal ||
		scopes.includeVector ||
		scopes.includeTraining
	if isAnyScopeSelected {
		return scopes
	}

	return rebuildScopes{
		includeNonPersonalized: true,
		includeUserCandidate:   true,
		includeGoodsRelation:   true,
		includeUserToUser:      true,
		includeCollaborative:   true,
		includeExternal:        true,
		includeVector:          config.Vector.Enabled,
		includeTraining:        config.Training.EnableOptimization || config.Ranking.Mode == core.RankingModeFm,
	}
}

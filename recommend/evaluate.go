package recommend

import (
	"context"
	"recommend/internal/engine"
)

// EvaluateOffline 基于请求、曝光和行为事实执行离线评估。
func (r *Recommend) EvaluateOffline(ctx context.Context, request EvaluateRequest) (*EvaluateResult, error) {
	request = r.normalizeEvaluateRequest(request)
	return engine.EvaluateOffline(ctx, r.dependencies, r.config, request)
}

// normalizeEvaluateRequest 归一化离线评估请求。
func (r *Recommend) normalizeEvaluateRequest(request EvaluateRequest) EvaluateRequest {
	if request.TopK <= 0 {
		request.TopK = r.config.Evaluate.DefaultTopK
	}
	if len(request.Scenes) == 0 {
		request.Scenes = cloneScenes(r.config.Materialize.DefaultScenes)
	}
	return request
}

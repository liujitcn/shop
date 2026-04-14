package engine

import (
	"context"
	"recommend/internal/core"
	evaluatex "recommend/internal/evaluate"
)

// EvaluateOffline 基于请求、曝光和行为事实执行离线评估。
func EvaluateOffline(ctx context.Context, dependencies core.Dependencies, _ core.ServiceConfig, request core.EvaluateRequest) (*core.EvaluateResult, error) {
	return evaluatex.EvaluateOffline(ctx, dependencies, request)
}

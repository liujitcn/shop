package recommend

import (
	"context"
	"recommend/internal/engine"
)

// Explain 查询已持久化的推荐追踪结果。
func (r *Recommend) Explain(ctx context.Context, request ExplainRequest) (*ExplainResult, error) {
	return engine.Explain(ctx, r.dependencies, r.config, request)
}

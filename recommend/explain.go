package recommend

import "context"

// Explain 查询已持久化的推荐追踪结果。
func Explain(_ context.Context, _ Dependencies, _ ExplainRequest) (*ExplainResult, error) {
	return nil, ErrNotImplemented
}

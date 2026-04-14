package recommend

import "context"

// EvaluateOffline 基于请求、曝光和行为事实执行离线评估。
func EvaluateOffline(_ context.Context, _ Dependencies, _ EvaluateRequest) (*EvaluateResult, error) {
	return nil, ErrNotImplemented
}

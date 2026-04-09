package recall

import (
	"context"

	recommendcore "shop/pkg/recommend/core"
)

// Provider 定义单个召回器需要实现的最小接口。
type Provider interface {
	// Name 返回召回器标识。
	Name() string
	// Recall 执行一次候选召回。
	Recall(ctx context.Context, req *recommendcore.RecallRequest) ([]*recommendcore.Candidate, error)
}

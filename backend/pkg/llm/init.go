package llm

import "github.com/google/wire"

// ProviderSet 汇总大模型客户端依赖注入提供者。
var ProviderSet = wire.NewSet(
	NewClient,
)

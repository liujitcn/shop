package agent

import (
	"shop/pkg/agent/assistant"
	"shop/pkg/agent/comment"
	"shop/pkg/agent/provider"
	"shop/pkg/agent/stream"

	"github.com/google/wire"
)

// ProviderSet 汇总 Agent 能力层依赖。
var ProviderSet = wire.NewSet(
	provider.NewChatClient,
	comment.NewRuntime,
	assistant.NewRuntime,
	stream.NewPublisher,
)

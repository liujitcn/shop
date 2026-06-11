package agent

import (
	"shop/pkg/agent/assistant"
	"shop/pkg/agent/comment"
	einoModel "shop/pkg/agent/eino/model"
	einoStructured "shop/pkg/agent/eino/structured"

	"github.com/google/wire"
)

// ProviderSet 汇总 Agent 能力层依赖。
var ProviderSet = wire.NewSet(
	einoModel.NewChatClient,
	einoModel.NewResponsesClient,
	einoStructured.NewRunner,
	comment.NewRuntime,
	assistant.NewRuntime,
)

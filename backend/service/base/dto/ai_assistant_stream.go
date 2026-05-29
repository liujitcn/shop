package dto

import basev1 "shop/api/gen/go/base/v1"

// AiAssistantStreamEvent 表示聊天专用 SSE 事件名称。
type AiAssistantStreamEvent string

const (
	// AiAssistantStreamEventDelta 表示助手回复增量。
	AiAssistantStreamEventDelta AiAssistantStreamEvent = "delta"
	// AiAssistantStreamEventFinish 表示助手回复完成。
	AiAssistantStreamEventFinish AiAssistantStreamEvent = "finish"
	// AiAssistantStreamEventError 表示助手回复异常。
	AiAssistantStreamEventError AiAssistantStreamEvent = "error"
)

// AiAssistantStreamPayload 表示聊天专用 SSE 事件负载。
type AiAssistantStreamPayload struct {
	SessionID string                       `json:"session_id"`
	MessageID string                       `json:"message_id"`
	Delta     string                       `json:"delta,omitempty"`
	Messages  []*basev1.AiAssistantMessage `json:"messages,omitempty"`
	Session   *basev1.AiAssistantSession   `json:"session,omitempty"`
}

// AiAssistantStreamEmitter 定义聊天专用 SSE 事件发送能力。
type AiAssistantStreamEmitter interface {
	EmitAiAssistantStream(event AiAssistantStreamEvent, payload AiAssistantStreamPayload) error
}

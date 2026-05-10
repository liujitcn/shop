package dto

import "encoding/json"

const (
	// AiAssistantToolStatusSuccess 表示工具执行成功。
	AiAssistantToolStatusSuccess = "success"
	// AiAssistantToolStatusFailed 表示工具执行失败。
	AiAssistantToolStatusFailed = "failed"

	// AiAssistantConfirmStatusPending 表示确认卡待处理。
	AiAssistantConfirmStatusPending = "pending"
	// AiAssistantConfirmStatusApproved 表示确认卡已确认。
	AiAssistantConfirmStatusApproved = "approved"
	// AiAssistantConfirmStatusRejected 表示确认卡已拒绝。
	AiAssistantConfirmStatusRejected = "rejected"
	// AiAssistantConfirmStatusFailed 表示确认卡执行失败。
	AiAssistantConfirmStatusFailed = "failed"
)

// AiAssistantMessagePayload 表示落库消息正文结构。
type AiAssistantMessagePayload struct {
	// Content 表示展示给前端的正文内容。
	Content string `json:"content"`
	// ReplySource 表示回复来源。
	ReplySource string `json:"reply_source"`
	// Model 表示本次回复使用的模型名称。
	Model string `json:"model"`
	// Fallback 表示是否为降级回复。
	Fallback bool `json:"fallback"`
	// FallbackReason 表示降级原因。
	FallbackReason string `json:"fallback_reason"`
	// Confirm 表示确认卡状态。
	Confirm *AiAssistantConfirmState `json:"confirm,omitempty"`
}

// AiAssistantConfirmState 表示确认卡在消息中的状态快照。
type AiAssistantConfirmState struct {
	// Status 表示确认状态。
	Status string `json:"status"`
	// Action 表示确认动作编码。
	Action string `json:"action"`
	// Summary 表示确认动作摘要。
	Summary string `json:"summary"`
	// Payload 表示执行确认动作所需载荷。
	Payload json.RawMessage `json:"payload,omitempty"`
	// FormSchema 表示前端确认时需要采集的表单结构。
	FormSchema []AiAssistantConfirmFormField `json:"form_schema,omitempty"`
}

// AiAssistantConfirmRequest 表示运行时生成的确认请求。
type AiAssistantConfirmRequest struct {
	// Title 表示确认卡标题。
	Title string `json:"title"`
	// Lines 表示确认卡文案。
	Lines []string `json:"lines"`
	// Action 表示确认动作编码。
	Action string `json:"action"`
	// Summary 表示确认动作摘要。
	Summary string `json:"summary"`
	// Payload 表示后续确认执行所需载荷。
	Payload json.RawMessage `json:"payload,omitempty"`
	// FormSchema 表示前端确认时需要采集的表单结构。
	FormSchema []AiAssistantConfirmFormField `json:"form_schema,omitempty"`
}

// AiAssistantConfirmFormField 表示确认卡附带的表单字段。
type AiAssistantConfirmFormField struct {
	// Prop 表示字段键名。
	Prop string `json:"prop"`
	// Label 表示字段标题。
	Label string `json:"label"`
	// Placeholder 表示输入提示。
	Placeholder string `json:"placeholder"`
	// Required 表示是否必填。
	Required bool `json:"required"`
}

// AiAssistantToolResult 表示单个工具执行结果。
type AiAssistantToolResult struct {
	// Name 表示工具名称。
	Name string `json:"name"`
	// Status 表示工具执行状态。
	Status string `json:"status"`
	// Elapsed 表示工具耗时。
	Elapsed string `json:"elapsed"`
	// Input 表示工具入参摘要。
	Input string `json:"input"`
	// Summary 表示工具结果摘要。
	Summary string `json:"summary"`
	// ErrorMessage 表示工具失败原因。
	ErrorMessage string `json:"error_message"`
	// Output 表示工具原始输出摘要。
	Output string `json:"output"`
}

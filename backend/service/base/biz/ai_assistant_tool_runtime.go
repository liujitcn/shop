package biz

import (
	"context"

	"shop/pkg/llm"
	baseDTO "shop/service/base/dto"
)

// AiAssistantToolRuntime 定义 AI 助手工具编排运行时。
type AiAssistantToolRuntime interface {
	// RunToolCalls 根据当前输入决定是否执行工具，并返回补充上下文与工具记录。
	RunToolCalls(ctx context.Context, input AiAssistantToolRuntimeInput) (*AiAssistantToolRuntimeResult, error)
	// ExecuteConfirm 根据确认卡动作执行后续操作。
	ExecuteConfirm(ctx context.Context, input AiAssistantConfirmRuntimeInput) (*AiAssistantConfirmRuntimeResult, error)
}

// AiAssistantToolRuntimeInput 表示工具编排输入。
type AiAssistantToolRuntimeInput struct {
	Scene        string
	Terminal     string
	SessionTitle string
	Content      string
}

// AiAssistantToolRuntimeResult 表示工具编排输出。
type AiAssistantToolRuntimeResult struct {
	PromptAugment string
	Tools         []llm.AiAssistantToolCall
	Confirm       *baseDTO.AiAssistantConfirmRequest
}

// AiAssistantConfirmRuntimeInput 表示确认动作运行时输入。
type AiAssistantConfirmRuntimeInput struct {
	SessionID string
	MessageID string
	Action    string
	Confirm   *baseDTO.AiAssistantConfirmState
	FormJSON  string
}

// AiAssistantConfirmRuntimeResult 表示确认动作执行结果。
type AiAssistantConfirmRuntimeResult struct {
	Status  string
	Summary string
	Reply   string
	Tools   []llm.AiAssistantToolCall
}

type noopAiAssistantToolRuntime struct{}

// NewNoopAiAssistantToolRuntime 创建空工具编排实现。
func NewNoopAiAssistantToolRuntime() AiAssistantToolRuntime {
	return &noopAiAssistantToolRuntime{}
}

// RunToolCalls 返回空结果。
func (r *noopAiAssistantToolRuntime) RunToolCalls(context.Context, AiAssistantToolRuntimeInput) (*AiAssistantToolRuntimeResult, error) {
	return &AiAssistantToolRuntimeResult{}, nil
}

// ExecuteConfirm 返回空确认执行结果。
func (r *noopAiAssistantToolRuntime) ExecuteConfirm(context.Context, AiAssistantConfirmRuntimeInput) (*AiAssistantConfirmRuntimeResult, error) {
	return &AiAssistantConfirmRuntimeResult{
		Status: baseDTO.AiAssistantConfirmStatusFailed,
	}, nil
}

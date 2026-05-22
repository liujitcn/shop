package comment

import (
	"context"
	"fmt"
	"strings"

	configv1 "shop/api/gen/go/config/v1"
	"shop/pkg/agent/provider"

	"github.com/go-kratos/blades"
	"github.com/google/jsonschema-go/jsonschema"
)

// Runtime 封装评论生成式智能体能力。
type Runtime struct {
	// client 统一承接底层模型提供商，审核和摘要共用同一个聊天模型入口。
	client *provider.ChatClient
	// commentReviewInstruction 是评价审核的系统提示词，来自配置中心，缺失时禁止发起审核调用。
	commentReviewInstruction string
	// commentAIInstruction 是评价摘要的系统提示词，和审核提示词分开，避免两个任务互相污染输出格式。
	commentAIInstruction string
}

// NewRuntime 创建评论智能体运行时。
func NewRuntime(client *provider.ChatClient, prompt *configv1.Prompt) *Runtime {
	runtime := &Runtime{client: client}
	// prompt 可能在本地开发或未配置 AI 时为空；此时保留空指令，让实际调用处返回明确错误。
	if prompt != nil {
		runtime.commentReviewInstruction = strings.TrimSpace(prompt.GetCommentReview())
		runtime.commentAIInstruction = strings.TrimSpace(prompt.GetCommentAi())
	}
	return runtime
}

// Enabled 判断评论智能体是否可用。
func (r *Runtime) Enabled() bool {
	// 这里只判断模型客户端是否具备可调用提供商；提示词是否配置由具体结构化任务校验，方便区分“模型未配置”和“提示词未配置”。
	return r != nil && r.client != nil && r.client.ModelProvider != nil
}

// Model 返回评论智能体当前使用的模型名称。
func (r *Runtime) Model() string {
	if !r.Enabled() {
		return ""
	}
	return r.client.Name()
}

// generateStructured 按 JSON Schema 调用大模型并反序列化结构化结果。
func (r *Runtime) generateStructured(
	ctx context.Context,
	instruction string,
	parts []any,
	schema *jsonschema.Schema,
	out any,
) error {
	// 模型客户端未初始化时，调用方无法继续发起大模型请求。
	if !r.Enabled() {
		return fmt.Errorf("agent chat client is not configured")
	}
	// 结构化任务必须配置系统提示词，避免用空规则调用大模型。
	if strings.TrimSpace(instruction) == "" {
		return fmt.Errorf("agent instruction is empty")
	}
	// 输出目标为空时，无法承载结构化响应。
	if out == nil {
		return fmt.Errorf("agent structured output is nil")
	}

	// 所有评论智能体任务都要求模型按 JSON Schema 输出，减少业务层再猜字段含义的成本。
	response, err := r.client.Generate(ctx, &blades.ModelRequest{
		Instruction:  blades.SystemMessage(instruction),
		Messages:     []*blades.Message{blades.UserMessage(parts...)},
		OutputSchema: schema,
	})
	if err != nil {
		return fmt.Errorf("request agent structured output: %w", err)
	}
	// 服务商返回空消息时，无法解析结构化结果。
	if response == nil || response.Message == nil {
		return fmt.Errorf("agent structured response is empty")
	}

	content := strings.TrimSpace(response.Message.Text())
	// 模型未返回 JSON 文本时，直接返回错误供调用方重试或降级。
	if content == "" {
		return fmt.Errorf("agent structured response content is empty")
	}
	err = decodeStructuredContent(content, out)
	if err != nil {
		return fmt.Errorf("decode agent structured response: %w", err)
	}
	return nil
}

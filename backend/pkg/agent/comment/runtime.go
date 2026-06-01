package comment

import (
	"context"
	"encoding/json"
	"fmt"

	"shop/pkg/agent/provider"

	"github.com/cloudwego/eino/schema"
	"github.com/google/jsonschema-go/jsonschema"
)

const (
	// commentReviewInstruction 是评价审核结构化输出的标准提示词。
	commentReviewInstruction = `你是电商评价审核与标签生成助手。请同时审核评价文本和图片是否适合公开展示，并提取评价标签。
审核拒绝范围包括：色情低俗、暴力血腥、违法违禁、政治敏感、辱骂攻击、广告引流、二维码或联系方式、明显无关图片、侵犯隐私等。
正常商品体验、物流包装、尺码质量、使用感受可以通过。
如果审核通过，approved 必须为 true，textRisk 和 imageRisk 必须为 false，riskReason 必须为空字符串。
如果审核不通过，riskReason 必须具体说明违规类别、命中文本片段或图片序号、判定依据，例如“图片1疑似色情低俗：出现裸露身体部位，不适合公开展示”；不要只写“内容安全风险”“审核不通过”等泛化原因。
只返回符合 JSON Schema 的 JSON，不要输出解释。`

	// commentSummaryInstruction 是评价摘要结构化输出的标准提示词。
	commentSummaryInstruction = `你是电商评价摘要助手。请基于已审核通过的商品评价，生成商品详情摘要和评价列表摘要。
摘要必须客观、简短，不能编造评价中没有出现的事实；每条摘要使用标签和内容表达，商品详情摘要只返回一条，评价列表摘要可返回多条。
只返回符合 JSON Schema 的 JSON，不要输出解释。`
)

// Runtime 封装评论生成式智能体能力。
type Runtime struct {
	// client 统一承接底层模型提供商，审核和摘要共用同一个聊天模型入口。
	client *provider.ChatClient
}

// NewRuntime 创建评论智能体运行时。
func NewRuntime(client *provider.ChatClient) *Runtime {
	return &Runtime{client: client}
}

// Enabled 判断评论智能体是否可用。
func (r *Runtime) Enabled() bool {
	return r != nil && r.client != nil && r.client.BaseChatModel != nil
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
	parts []schema.MessageInputPart,
	outputSchema *jsonschema.Schema,
	out any,
) error {
	// 模型客户端未初始化时，调用方无法继续发起大模型请求。
	if !r.Enabled() {
		return fmt.Errorf("agent chat client is not configured")
	}
	// 结构化任务必须配置系统提示词，避免用空规则调用大模型。
	if instruction == "" {
		return fmt.Errorf("agent instruction is empty")
	}
	// 输出目标为空时，无法承载结构化响应。
	if out == nil {
		return fmt.Errorf("agent structured output is nil")
	}

	messages := []*schema.Message{
		schema.SystemMessage(instruction + "\n\n" + structuredOutputSchemaPrompt(outputSchema)),
		{
			Role:                  schema.User,
			UserInputMultiContent: parts,
		},
	}
	response, err := r.client.Generate(ctx, messages)
	if err != nil {
		return fmt.Errorf("request agent structured output: %w", err)
	}
	// 服务商返回空消息时，无法解析结构化结果。
	if response == nil {
		return fmt.Errorf("agent structured response is empty")
	}

	content := response.Content
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

// structuredOutputSchemaPrompt 构造结构化输出的 JSON Schema 文本约束。
func structuredOutputSchemaPrompt(outputSchema *jsonschema.Schema) string {
	if outputSchema == nil {
		return "只返回一个合法 JSON 对象，不要使用 Markdown 代码块。"
	}
	raw, err := json.Marshal(outputSchema)
	if err != nil {
		return "只返回一个合法 JSON 对象，不要使用 Markdown 代码块。"
	}
	return "请严格按以下 JSON Schema 返回一个合法 JSON 对象，不要输出 Markdown 代码块或额外说明：\n" + string(raw)
}

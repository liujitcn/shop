package comment

import (
	"context"
	"fmt"

	einoStructured "shop/pkg/agent/eino/structured"
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
	// structured 统一承接评论审核和摘要的结构化模型调用。
	structured *einoStructured.Runner
}

// NewRuntime 创建评论智能体运行时。
func NewRuntime(structured *einoStructured.Runner) *Runtime {
	return &Runtime{structured: structured}
}

// Enabled 判断评论智能体是否可用。
func (r *Runtime) Enabled() bool {
	return r != nil && r.structured != nil && r.structured.Enabled()
}

// Model 返回评论智能体当前使用的模型名称。
func (r *Runtime) Model() string {
	if !r.Enabled() {
		return ""
	}
	return r.structured.Model()
}

// generateStructured 按 JSON Schema 调用大模型并反序列化结构化结果。
func (r *Runtime) generateStructured(
	ctx context.Context,
	instruction string,
	parts []*einoStructured.Part,
	outputSchema *einoStructured.Schema,
	out any,
) error {
	if !r.Enabled() {
		return fmt.Errorf("agent chat client is not configured")
	}
	return r.structured.Generate(ctx, instruction, parts, outputSchema, out)
}

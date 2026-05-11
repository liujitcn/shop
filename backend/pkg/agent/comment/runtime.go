package comment

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	configv1 "shop/api/gen/go/config/v1"
	"shop/pkg/agent/provider"

	"github.com/go-kratos/blades"
	"github.com/google/jsonschema-go/jsonschema"
)

// Runtime 封装评论生成式智能体能力。
type Runtime struct {
	client                   *provider.ChatClient
	commentReviewInstruction string
	commentAIInstruction     string
}

// NewRuntime 创建评论智能体运行时。
func NewRuntime(client *provider.ChatClient, prompt *configv1.Prompt) *Runtime {
	runtime := &Runtime{client: client}
	if prompt != nil {
		runtime.commentReviewInstruction = strings.TrimSpace(prompt.GetCommentReview())
		runtime.commentAIInstruction = strings.TrimSpace(prompt.GetCommentAi())
	}
	return runtime
}

// Enabled 判断评论智能体是否可用。
func (r *Runtime) Enabled() bool {
	return r != nil && r.client != nil && r.client.Enabled()
}

// Model 返回评论智能体当前使用的模型名称。
func (r *Runtime) Model() string {
	if r == nil || r.client == nil {
		return ""
	}
	return r.client.Model()
}

// ReviewComment 审核评论内容。
func (r *Runtime) ReviewComment(ctx context.Context, req ReviewRequest) (*ReviewResult, error) {
	content := strings.TrimSpace(req.Content)
	imageURLs := cleanStringList(req.ImageURLs)
	imageData := cleanReviewImageData(req.ImageData)
	// 文本和图片都为空时，视为无公开风险，调用方可结合评分直接处理。
	if content == "" && len(imageURLs) == 0 && len(imageData) == 0 {
		return &ReviewResult{Approved: true}, nil
	}

	parts := make([]any, 0, len(imageURLs)+len(imageData)+1)
	payload := map[string]any{
		"goodsName":  strings.TrimSpace(req.GoodsName),
		"skuDesc":    strings.TrimSpace(req.SKUDesc),
		"content":    content,
		"imageCount": len(imageURLs) + len(imageData),
	}
	var err error
	var rawPayload []byte
	rawPayload, err = json.Marshal(payload)
	if err != nil {
		parts = append(parts, "请审核以下商品评价图文，并返回审核结果和标签。")
	} else {
		parts = append(parts, "请审核以下商品评价图文，并返回审核结果和标签：\n"+string(rawPayload))
	}
	for _, imageURL := range imageURLs {
		parts = append(parts, blades.FilePart{
			Name:     "comment-image",
			URI:      imageURL,
			MIMEType: imageMIMEType(imageURL),
		})
	}
	for _, image := range imageData {
		parts = append(parts, blades.DataPart{
			Name:     image.Name,
			Bytes:    image.Bytes,
			MIMEType: imageDataMIMEType(image.MIMEType, image.Name),
		})
	}

	var schema *jsonschema.Schema
	schema, err = jsonschema.For[ReviewResult](nil)
	if err != nil {
		return nil, fmt.Errorf("build comment review schema: %w", err)
	}
	result := &ReviewResult{}
	err = r.generateStructured(ctx, r.commentReviewInstruction, parts, schema, result)
	if err != nil {
		return nil, err
	}
	r.normalizeReviewResult(result)
	if reviewNeedsConcreteReason(result) {
		retryParts := append([]any(nil), parts...)
		retryParts = append(retryParts, "上一次审核结果缺少具体不通过原因。请重新审核：如果不通过，riskReason 必须说明违规类别、命中的文本片段或图片序号、具体判定依据，例如“图片1疑似色情低俗：出现裸露身体部位，不适合公开展示”。不要只写“内容安全风险”或“审核不通过”。")
		err = r.generateStructured(ctx, r.commentReviewInstruction, retryParts, schema, result)
		if err != nil {
			return nil, err
		}
		r.normalizeReviewResult(result)
	}
	if reviewNeedsConcreteReason(result) {
		return nil, fmt.Errorf("agent review rejected without concrete reason")
	}
	return result, nil
}

// GenerateCommentAi 生成商品评价 AI 摘要数据。
func (r *Runtime) GenerateCommentAi(ctx context.Context, req CommentAiRequest) (*CommentAiResult, error) {
	// 没有审核通过的评价时，不调用大模型生成空摘要。
	if len(req.Comments) == 0 {
		return &CommentAiResult{}, nil
	}

	var err error
	var schema *jsonschema.Schema
	schema, err = jsonschema.For[CommentAiResult](nil)
	if err != nil {
		return nil, fmt.Errorf("build comment ai schema: %w", err)
	}
	result := &CommentAiResult{}
	req.GoodsName = strings.TrimSpace(req.GoodsName)
	var rawPayload []byte
	rawPayload, err = json.Marshal(req)
	commentAIPrompt := "请基于已审核通过的商品评价生成评价 AI 摘要。"
	if err == nil {
		commentAIPrompt = "请基于已审核通过的商品评价生成评价 AI 摘要：\n" + string(rawPayload)
	}
	err = r.generateStructured(ctx, r.commentAIInstruction, []any{commentAIPrompt}, schema, result)
	if err != nil {
		return nil, err
	}
	result.Overview.Content = limitCommentAiContentItems(result.Overview.Content, 1, "AI 总结")
	result.List.Content = limitCommentAiContentItems(result.List.Content, 4, "")
	return result, nil
}

// normalizeReviewResult 规范化评论审核结果。
func (r *Runtime) normalizeReviewResult(result *ReviewResult) {
	if result == nil {
		return
	}
	result.RiskReason = strings.TrimSpace(result.RiskReason)
	result.Tags = cleanStringList(result.Tags)
	// 模型可能返回过多标签，最多保留 5 个用于前台展示。
	if len(result.Tags) > 5 {
		result.Tags = append([]string(nil), result.Tags[:5]...)
	}
	// 任一风险命中时，强制将公开展示结果置为不通过，避免模型字段互相矛盾。
	if result.TextRisk || result.ImageRisk {
		result.Approved = false
	}
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

	response, err := r.client.Provider().Generate(ctx, &blades.ModelRequest{
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
	if content != "" {
		start := strings.IndexAny(content, "{[")
		end := strings.LastIndexAny(content, "}]")
		// 找到 JSON 起止位置时，剔除模型可能额外输出的围栏和解释文字。
		if start >= 0 && end >= start {
			content = strings.TrimSpace(content[start : end+1])
		}
	}
	// 模型未返回 JSON 文本时，直接返回错误供调用方重试或降级。
	if content == "" {
		return fmt.Errorf("agent structured response content is empty")
	}
	err = json.Unmarshal([]byte(content), out)
	if err != nil {
		return fmt.Errorf("decode agent structured response: %w", err)
	}
	return nil
}

// reviewNeedsConcreteReason 判断拒绝结果是否缺少具体原因。
func reviewNeedsConcreteReason(result *ReviewResult) bool {
	if result == nil || result.Approved {
		return false
	}
	return !hasConcreteReviewReason(result.RiskReason)
}

// imageDataMIMEType 按图片字节元信息推断 MIME 类型。
func imageDataMIMEType(rawMIMEType string, name string) blades.MIMEType {
	cleanMIMEType := strings.ToLower(strings.TrimSpace(rawMIMEType))
	// 调用方已经提供图片 MIME 类型时，优先使用该值。
	if strings.HasPrefix(cleanMIMEType, "image/png") {
		return blades.MIMEImagePNG
	}
	// 调用方已经提供图片 MIME 类型时，优先使用该值。
	if strings.HasPrefix(cleanMIMEType, "image/webp") {
		return blades.MIMEImageWEBP
	}
	// 调用方已经提供图片 MIME 类型时，优先使用该值。
	if strings.HasPrefix(cleanMIMEType, "image/jpeg") || strings.HasPrefix(cleanMIMEType, "image/jpg") {
		return blades.MIMEImageJPEG
	}
	return imageMIMEType(name)
}

// cleanReviewImageData 清理评论审核图片字节列表。
func cleanReviewImageData(values []ReviewImageData) []ReviewImageData {
	result := make([]ReviewImageData, 0, len(values))
	for index, value := range values {
		// 图片字节为空时无法参与多模态审核。
		if len(value.Bytes) == 0 {
			continue
		}
		value.Name = strings.TrimSpace(value.Name)
		if value.Name == "" {
			value.Name = fmt.Sprintf("comment-image-%d", index+1)
		}
		value.MIMEType = strings.TrimSpace(value.MIMEType)
		result = append(result, value)
	}
	return result
}

// hasConcreteReviewReason 判断审核原因是否包含具体违规线索。
func hasConcreteReviewReason(reason string) bool {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return false
	}
	genericReasons := []string{
		"审核不通过",
		"LLM审核不通过",
		"内容安全风险",
		"评价内容未通过内容安全审核",
		"评价文本命中内容安全风险",
		"评价图片命中内容安全风险",
	}
	for _, genericReason := range genericReasons {
		// 完全等于泛化文案时，不能作为可解释拒绝原因。
		if strings.EqualFold(reason, genericReason) {
			return false
		}
	}
	evidenceKeywords := []string{
		"色情", "低俗", "裸露", "暴力", "血腥", "违法", "违禁", "政治", "辱骂", "攻击", "广告", "引流", "二维码", "联系方式", "隐私", "无关",
		"文本", "片段", "图片", "第", "疑似", "出现", "包含", "涉及", "命中",
	}
	for _, keyword := range evidenceKeywords {
		if strings.Contains(reason, keyword) {
			return true
		}
	}
	return len([]rune(reason)) >= 12
}

// imageMIMEType 按图片地址推断 MIME 类型。
func imageMIMEType(rawURL string) blades.MIMEType {
	lowerURL := strings.ToLower(strings.TrimSpace(rawURL))
	// data URL 自带 MIME 类型时，优先按前缀判断具体图片格式。
	if strings.HasPrefix(lowerURL, "data:image/png") {
		return blades.MIMEImagePNG
	}
	// data URL 自带 MIME 类型时，优先按前缀判断具体图片格式。
	if strings.HasPrefix(lowerURL, "data:image/webp") {
		return blades.MIMEImageWEBP
	}
	// data URL 自带 MIME 类型时，优先按前缀判断具体图片格式。
	if strings.HasPrefix(lowerURL, "data:image/jpeg") || strings.HasPrefix(lowerURL, "data:image/jpg") {
		return blades.MIMEImageJPEG
	}

	urlWithoutQuery := lowerURL
	queryIndex := strings.Index(urlWithoutQuery, "?")
	// URL 携带查询参数时，先剔除查询部分再判断扩展名。
	if queryIndex >= 0 {
		urlWithoutQuery = urlWithoutQuery[:queryIndex]
	}
	// 按常见图片扩展名推断 MIME 类型，未知格式默认按 JPEG 处理。
	switch {
	case strings.HasSuffix(urlWithoutQuery, ".png"):
		return blades.MIMEImagePNG
	case strings.HasSuffix(urlWithoutQuery, ".webp"):
		return blades.MIMEImageWEBP
	default:
		return blades.MIMEImageJPEG
	}
}

// cleanStringList 清理字符串列表并去重。
func cleanStringList(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		cleanValue := strings.TrimSpace(value)
		// 清理后为空或已经出现过的值不再保留。
		if cleanValue == "" {
			continue
		}
		// 已经保留过的值不再重复追加。
		if _, ok := seen[cleanValue]; ok {
			continue
		}
		seen[cleanValue] = struct{}{}
		result = append(result, cleanValue)
	}
	return result
}

// limitCommentAiContentItems 清理并限制 AI 摘要内容项数量。
func limitCommentAiContentItems(values []CommentAiContentItem, limit int, defaultLabel string) []CommentAiContentItem {
	// 限制小于等于 0 时，直接返回空列表。
	if limit <= 0 {
		return []CommentAiContentItem{}
	}
	result := make([]CommentAiContentItem, 0, len(values))
	for _, value := range values {
		value.Label = strings.TrimSpace(value.Label)
		value.Content = strings.TrimSpace(value.Content)
		// 摘要内容为空时，不进入最终摘要。
		if value.Content == "" {
			continue
		}
		// 商品详情摘要标签固定兜底，避免模型遗漏标签导致前端展示异常。
		if value.Label == "" {
			value.Label = defaultLabel
		}
		result = append(result, value)
		// 已达到模块上限时，停止继续追加。
		if len(result) >= limit {
			break
		}
	}
	return result
}

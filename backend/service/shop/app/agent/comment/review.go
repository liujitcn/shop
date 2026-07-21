package comment

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"strings"

	"shop/pkg/agent/eino/message"
	einoStructured "shop/pkg/agent/eino/structured"
)

const (
	// commentReviewTagLimit 表示审核阶段最多保留的商品体验标签数量。
	commentReviewTagLimit = 5
	// commentReviewDefaultImageName 表示审核图片缺少名称时使用的默认前缀。
	commentReviewDefaultImageName = "comment-image"
)

// HasConcreteReviewReason 判断审核原因是否包含具体违规线索。
func HasConcreteReviewReason(reason string) bool {
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
	// 这些文案只能说明“失败了”，不能说明为什么失败，不能作为最终审核结论保存给运营。
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
	// 命中风险类别、证据位置或判定动作类关键词时，认为该原因已经具备人工复核价值。
	for _, keyword := range evidenceKeywords {
		if strings.Contains(reason, keyword) {
			return true
		}
	}
	// 较长的非泛化原因通常已包含一定解释信息，避免过度要求固定关键词导致正常拒绝结论被反复重试。
	return len([]rune(reason)) >= 12
}

// ReviewComment 审核评论内容。
func (r *Runtime) ReviewComment(ctx context.Context, req ReviewRequest) (*ReviewResult, error) {
	content := req.Content
	imageURLs := cleanStringList(req.ImageURLs)
	imageData := cleanReviewImageData(req.ImageData)
	existingTags := cleanStringList(req.ExistingTags)
	// 文本和图片都为空时，视为无公开风险，调用方可结合评分直接处理。
	if content == "" && len(imageURLs) == 0 && len(imageData) == 0 {
		return &ReviewResult{Approved: true}, nil
	}

	// 审核输入同时支持公网图片地址和本地图片字节；文本 payload 放在第一段，图片作为多模态附件追加。
	parts := make([]*einoStructured.Part, 0, len(imageURLs)+len(imageData)+2)
	payload := map[string]any{
		"goodsName":    req.GoodsName,
		"skuDesc":      req.SKUDesc,
		"content":      content,
		"imageCount":   len(imageURLs) + len(imageData),
		"existingTags": existingTags,
	}
	rawPayload, err := json.Marshal(payload)
	// payload 序列化失败时仍保留兜底提示词，让模型至少能基于图片完成审核。
	if err != nil {
		parts = append(parts, einoStructured.TextPart("请审核以下商品评价图文，并返回审核结果和标签。"))
	} else {
		parts = append(parts, einoStructured.TextPart("请审核以下商品评价图文，并返回审核结果和标签：\n"+string(rawPayload)))
	}
	if len(existingTags) > 0 {
		parts = append(parts, einoStructured.TextPart("标签生成规则：tags 必须优先从 existingTags 中选择并原样返回；只有评价语义确实无法归入任何已有标签时，才允许生成新的短标签。"))
	}
	for _, imageURL := range imageURLs {
		parts = append(parts, message.ImageURLPart(imageURL))
	}
	for _, image := range imageData {
		parts = append(parts, message.ImageDataPart(image.Bytes, reviewImageDataMIMEType(image.MIMEType, image.Name)))
	}

	var outputSchema *einoStructured.Schema
	outputSchema, err = cachedReviewResultSchema()
	if err != nil {
		return nil, fmt.Errorf("build comment review schema: %w", err)
	}
	result := &ReviewResult{}
	err = r.generateStructured(ctx, commentReviewInstruction, parts, outputSchema, result)
	if err != nil {
		return nil, err
	}
	r.normalizeReviewResult(result)
	// 拒绝结果必须能解释给运营和用户看；模型只返回泛化原因时，再带着原始输入追问一次。
	if reviewNeedsConcreteReason(result) {
		retryParts := append([]*einoStructured.Part(nil), parts...)
		retryParts = append(retryParts, einoStructured.TextPart("上一次审核结果缺少清晰结论或具体不通过原因。请重新审核：如果可以公开展示，approved 必须为 true，textRisk 和 imageRisk 必须为 false，riskReason 必须为空字符串；如果不通过，approved 必须为 false，riskReason 必须说明违规类别、命中的文本片段或图片序号、具体判定依据，例如“图片1疑似色情低俗：出现裸露身体部位，不适合公开展示”。不要只写“内容安全风险”或“审核不通过”。"))
		err = r.generateStructured(ctx, commentReviewInstruction, retryParts, outputSchema, result)
		if err != nil {
			return nil, err
		}
		r.normalizeReviewResult(result)
	}
	completeMissingSafeReviewVerdict(result)
	return result, nil
}

// normalizeReviewResult 规范化评论审核结果。
func (r *Runtime) normalizeReviewResult(result *ReviewResult) {
	if result == nil {
		return
	}
	result.Tags = cleanStringList(result.Tags)
	// 模型可能返回过多标签，最多保留固定数量用于前台展示。
	if len(result.Tags) > commentReviewTagLimit {
		result.Tags = append([]string(nil), result.Tags[:commentReviewTagLimit]...)
	}
	// 任一风险命中时，强制将公开展示结果置为不通过，避免模型字段互相矛盾。
	if result.TextRisk || result.ImageRisk {
		result.Approved = false
	}
}

// cleanReviewImageData 清理评论审核图片字节列表。
func cleanReviewImageData(values []ReviewImageData) []ReviewImageData {
	result := make([]ReviewImageData, 0, len(values))
	for index, value := range values {
		// 图片字节为空时无法参与多模态审核。
		if len(value.Bytes) == 0 {
			continue
		}
		if value.Name == "" {
			value.Name = fmt.Sprintf("%s-%d", commentReviewDefaultImageName, index+1)
		}
		// MIMEType 后续会参与图片格式推断，保留调用方传入的具体类型。
		result = append(result, value)
	}
	return result
}

// reviewImageMIMEType 按图片地址推断 MIME 类型。
func reviewImageMIMEType(rawURL string) string {
	lowerURL := strings.ToLower(rawURL)
	// data URL 自带 MIME 类型时，优先按前缀判断具体图片格式。
	switch {
	case strings.HasPrefix(lowerURL, "data:image/png"):
		return "image/png"
	case strings.HasPrefix(lowerURL, "data:image/webp"):
		return "image/webp"
	case strings.HasPrefix(lowerURL, "data:image/jpeg") || strings.HasPrefix(lowerURL, "data:image/jpg"):
		return "image/jpeg"
	}

	cleanPath := lowerURL
	parsedURL, err := url.Parse(lowerURL)
	if err == nil && parsedURL.Path != "" {
		cleanPath = parsedURL.Path
	} else {
		suffixIndex := strings.IndexAny(cleanPath, "?#")
		// 无法按标准 URL 解析时，仍剔除查询参数和片段后再判断扩展名。
		if suffixIndex >= 0 {
			cleanPath = cleanPath[:suffixIndex]
		}
	}
	// 按常见图片扩展名推断 MIME 类型，未知格式默认按 JPEG 处理。
	switch path.Ext(cleanPath) {
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	default:
		return "image/jpeg"
	}
}

// reviewImageDataMIMEType 按图片字节元信息推断 MIME 类型。
func reviewImageDataMIMEType(rawMIMEType string, name string) string {
	cleanMIMEType := strings.ToLower(rawMIMEType)
	// 调用方已经提供图片 MIME 类型时，优先使用该值；不识别时再按文件名兜底推断。
	switch {
	case strings.HasPrefix(cleanMIMEType, "image/png"):
		return "image/png"
	case strings.HasPrefix(cleanMIMEType, "image/webp"):
		return "image/webp"
	case strings.HasPrefix(cleanMIMEType, "image/jpeg") || strings.HasPrefix(cleanMIMEType, "image/jpg"):
		return "image/jpeg"
	}
	return reviewImageMIMEType(name)
}

// reviewNeedsConcreteReason 判断拒绝结果是否缺少具体原因。
func reviewNeedsConcreteReason(result *ReviewResult) bool {
	if result == nil || result.Approved {
		return false
	}
	return !HasConcreteReviewReason(result.RiskReason)
}

// completeMissingSafeReviewVerdict 补全只返回标签但没有风险信号的审核结论。
func completeMissingSafeReviewVerdict(result *ReviewResult) {
	if result == nil || result.Approved || result.TextRisk || result.ImageRisk || result.RiskReason != "" || len(result.Tags) == 0 {
		return
	}
	if result.approvedSet || result.textRiskSet || result.imageRiskSet {
		return
	}
	// 部分兼容模型会只返回 tags 而遗漏 approved，已二次追问后仍无风险证据时按通过处理，避免正常评价被误记为审核异常。
	result.Approved = true
}

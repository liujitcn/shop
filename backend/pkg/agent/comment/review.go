package comment

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/go-kratos/blades"
	"github.com/google/jsonschema-go/jsonschema"
)

const (
	// commentReviewTagLimit 表示审核阶段最多保留的商品体验标签数量。
	commentReviewTagLimit = 5
	// commentReviewDefaultImageName 表示审核图片缺少名称时使用的默认前缀。
	commentReviewDefaultImageName = "comment-image"
)

// ReviewComment 审核评论内容。
func (r *Runtime) ReviewComment(ctx context.Context, req ReviewRequest) (*ReviewResult, error) {
	content := strings.TrimSpace(req.Content)
	imageURLs := cleanStringList(req.ImageURLs)
	imageData := cleanReviewImageData(req.ImageData)
	// 文本和图片都为空时，视为无公开风险，调用方可结合评分直接处理。
	if content == "" && len(imageURLs) == 0 && len(imageData) == 0 {
		return &ReviewResult{Approved: true}, nil
	}

	// 审核输入同时支持公网图片地址和本地图片字节；文本 payload 放在第一段，图片作为多模态附件追加。
	parts := make([]any, 0, len(imageURLs)+len(imageData)+1)
	payload := map[string]any{
		"goodsName":  strings.TrimSpace(req.GoodsName),
		"skuDesc":    strings.TrimSpace(req.SKUDesc),
		"content":    content,
		"imageCount": len(imageURLs) + len(imageData),
	}
	rawPayload, err := json.Marshal(payload)
	// payload 序列化失败时仍保留兜底提示词，让模型至少能基于图片完成审核。
	if err != nil {
		parts = append(parts, "请审核以下商品评价图文，并返回审核结果和标签。")
	} else {
		parts = append(parts, "请审核以下商品评价图文，并返回审核结果和标签：\n"+string(rawPayload))
	}
	for _, imageURL := range imageURLs {
		parts = append(parts, blades.FilePart{
			Name:     commentReviewDefaultImageName,
			URI:      imageURL,
			MIMEType: reviewImageMIMEType(imageURL),
		})
	}
	for _, image := range imageData {
		parts = append(parts, blades.DataPart{
			Name:     image.Name,
			Bytes:    image.Bytes,
			MIMEType: reviewImageDataMIMEType(image.MIMEType, image.Name),
		})
	}

	var schema *jsonschema.Schema
	schema, err = cachedReviewResultSchema()
	if err != nil {
		return nil, fmt.Errorf("build comment review schema: %w", err)
	}
	result := &ReviewResult{}
	err = r.generateStructured(ctx, r.commentReviewInstruction, parts, schema, result)
	if err != nil {
		return nil, err
	}
	r.normalizeReviewResult(result)
	// 拒绝结果必须能解释给运营和用户看；模型只返回泛化原因时，再带着原始输入追问一次。
	if reviewNeedsConcreteReason(result) {
		retryParts := append([]any(nil), parts...)
		retryParts = append(retryParts, "上一次审核结果缺少具体不通过原因。请重新审核：如果不通过，riskReason 必须说明违规类别、命中的文本片段或图片序号、具体判定依据，例如“图片1疑似色情低俗：出现裸露身体部位，不适合公开展示”。不要只写“内容安全风险”或“审核不通过”。")
		err = r.generateStructured(ctx, r.commentReviewInstruction, retryParts, schema, result)
		if err != nil {
			return nil, err
		}
		r.normalizeReviewResult(result)
	}
	return result, nil
}

// normalizeReviewResult 规范化评论审核结果。
func (r *Runtime) normalizeReviewResult(result *ReviewResult) {
	if result == nil {
		return
	}
	result.RiskReason = strings.TrimSpace(result.RiskReason)
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

// reviewNeedsConcreteReason 判断拒绝结果是否缺少具体原因。
func reviewNeedsConcreteReason(result *ReviewResult) bool {
	if result == nil || result.Approved {
		return false
	}
	return !HasConcreteReviewReason(result.RiskReason)
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
			value.Name = fmt.Sprintf("%s-%d", commentReviewDefaultImageName, index+1)
		}
		// MIMEType 后续会参与图片格式推断，只清理边界空白，不改写调用方传入的具体类型。
		value.MIMEType = strings.TrimSpace(value.MIMEType)
		result = append(result, value)
	}
	return result
}

// reviewImageDataMIMEType 按图片字节元信息推断 MIME 类型。
func reviewImageDataMIMEType(rawMIMEType string, name string) blades.MIMEType {
	cleanMIMEType := strings.ToLower(strings.TrimSpace(rawMIMEType))
	// 调用方已经提供图片 MIME 类型时，优先使用该值；不识别时再按文件名兜底推断。
	switch {
	case strings.HasPrefix(cleanMIMEType, "image/png"):
		return blades.MIMEImagePNG
	case strings.HasPrefix(cleanMIMEType, "image/webp"):
		return blades.MIMEImageWEBP
	case strings.HasPrefix(cleanMIMEType, "image/jpeg") || strings.HasPrefix(cleanMIMEType, "image/jpg"):
		return blades.MIMEImageJPEG
	}
	return reviewImageMIMEType(name)
}

// reviewImageMIMEType 按图片地址推断 MIME 类型。
func reviewImageMIMEType(rawURL string) blades.MIMEType {
	lowerURL := strings.ToLower(strings.TrimSpace(rawURL))
	// data URL 自带 MIME 类型时，优先按前缀判断具体图片格式。
	switch {
	case strings.HasPrefix(lowerURL, "data:image/png"):
		return blades.MIMEImagePNG
	case strings.HasPrefix(lowerURL, "data:image/webp"):
		return blades.MIMEImageWEBP
	case strings.HasPrefix(lowerURL, "data:image/jpeg") || strings.HasPrefix(lowerURL, "data:image/jpg"):
		return blades.MIMEImageJPEG
	}

	cleanPath := lowerURL
	parsedURL, parseErr := url.Parse(lowerURL)
	if parseErr == nil && parsedURL.Path != "" {
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
		return blades.MIMEImagePNG
	case ".webp":
		return blades.MIMEImageWEBP
	default:
		return blades.MIMEImageJPEG
	}
}

// HasConcreteReviewReason 判断审核原因是否包含具体违规线索。
func HasConcreteReviewReason(reason string) bool {
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

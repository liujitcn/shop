package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	configv1 "shop/api/gen/go/config/v1"

	"github.com/go-kratos/blades"
	openaiProvider "github.com/go-kratos/blades/contrib/openai"
	"github.com/google/jsonschema-go/jsonschema"
	bootstrapConfigv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
)

// Client 表示商城评价场景的大模型客户端。
type Client struct {
	provider                 blades.ModelProvider
	commentReviewInstruction string
	commentAIInstruction     string
}

// NewClient 创建大模型客户端。
func NewClient(bootstrapCfg *bootstrapConfigv1.Client_Llm, cfg *configv1.Prompt) *Client {
	client := &Client{}

	// 启动配置缺失或关键字段不完整时，保持客户端关闭状态。
	if bootstrapCfg != nil &&
		strings.TrimSpace(bootstrapCfg.GetBaseUrl()) != "" &&
		strings.TrimSpace(bootstrapCfg.GetApiKey()) != "" &&
		strings.TrimSpace(bootstrapCfg.GetModel()) != "" {
		providerConfig := openaiProvider.Config{
			BaseURL:          strings.TrimRight(strings.TrimSpace(bootstrapCfg.GetBaseUrl()), "/"),
			APIKey:           strings.TrimSpace(bootstrapCfg.GetApiKey()),
			Seed:             bootstrapCfg.GetSeed(),
			MaxOutputTokens:  bootstrapCfg.GetMaxOutputTokens(),
			FrequencyPenalty: bootstrapCfg.GetFrequencyPenalty(),
			PresencePenalty:  bootstrapCfg.GetPresencePenalty(),
			Temperature:      bootstrapCfg.GetTemperature(),
			TopP:             bootstrapCfg.GetTopP(),
			StopSequences:    bootstrapCfg.GetStopSequences(),
		}
		client.provider = openaiProvider.NewModel(strings.TrimSpace(bootstrapCfg.GetModel()), providerConfig)
	}

	// 提示词只从商城业务配置读取，不在客户端内保留默认业务文案。
	if cfg != nil {
		client.commentReviewInstruction = strings.TrimSpace(cfg.GetCommentReview())
		client.commentAIInstruction = strings.TrimSpace(cfg.GetCommentAi())
	}
	return client
}

// Enabled 判断大模型客户端是否可用。
func (c *Client) Enabled() bool {
	return c != nil && c.provider != nil
}

// Model 返回当前大模型名称。
func (c *Client) Model() string {
	// 客户端未初始化时，调用方无法获取模型名称，统一返回空字符串。
	if !c.Enabled() {
		return ""
	}
	return c.provider.Name()
}

// ReviewComment 审核评论图文并生成标签。
func (c *Client) ReviewComment(ctx context.Context, req CommentReviewRequest) (*CommentReviewResult, error) {
	content := strings.TrimSpace(req.Content)
	imageURLs := cleanStringList(req.ImageURLs)
	// 文本和图片都为空时，视为无公开风险，调用方可结合评分直接处理。
	if content == "" && len(imageURLs) == 0 {
		return &CommentReviewResult{Approved: true}, nil
	}

	parts := make([]any, 0, len(imageURLs)+1)
	payload := map[string]any{
		"goodsName":  strings.TrimSpace(req.GoodsName),
		"skuDesc":    strings.TrimSpace(req.SKUDesc),
		"content":    content,
		"imageCount": len(imageURLs),
	}
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		parts = append(parts, "请审核以下商品评价图文，并返回审核结果和标签。")
	} else {
		parts = append(parts, "请审核以下商品评价图文，并返回审核结果和标签：\n"+string(rawPayload))
	}
	for _, imageURL := range imageURLs {
		parts = append(parts, blades.FilePart{
			Name:     "comment-image",
			URI:      imageURL,
			MIMEType: imageMimeType(imageURL),
		})
	}

	var schema *jsonschema.Schema
	schema, err = jsonschema.For[CommentReviewResult](nil)
	if err != nil {
		return nil, fmt.Errorf("build comment review schema: %w", err)
	}
	result := &CommentReviewResult{}
	err = c.generateStructured(ctx, c.commentReviewInstruction, parts, schema, result)
	if err != nil {
		return nil, err
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
	return result, nil
}

// GenerateCommentAi 生成商品评价 AI 摘要数据。
func (c *Client) GenerateCommentAi(ctx context.Context, req CommentAiRequest) (*CommentAiResult, error) {
	// 没有审核通过的评价时，不调用大模型生成空摘要。
	if len(req.Comments) == 0 {
		return &CommentAiResult{}, nil
	}

	schema, err := jsonschema.For[CommentAiResult](nil)
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
	err = c.generateStructured(ctx, c.commentAIInstruction, []any{commentAIPrompt}, schema, result)
	if err != nil {
		return nil, err
	}
	result.Overview.Content = limitCommentAiContentItems(result.Overview.Content, 1, "AI 总结")
	result.List.Content = limitCommentAiContentItems(result.List.Content, 4, "")
	return result, nil
}

// generateStructured 按 JSON Schema 调用大模型并反序列化结构化结果。
func (c *Client) generateStructured(
	ctx context.Context,
	instruction string,
	parts []any,
	schema *jsonschema.Schema,
	out any,
) error {
	// 客户端配置不完整时，调用方无法继续发起大模型请求。
	if !c.Enabled() {
		return fmt.Errorf("llm client is not configured")
	}
	// 结构化任务必须配置系统提示词，避免用空规则调用大模型。
	if strings.TrimSpace(instruction) == "" {
		return fmt.Errorf("llm instruction is empty")
	}
	// 输出目标为空时，无法承载结构化响应。
	if out == nil {
		return fmt.Errorf("llm structured output is nil")
	}

	response, err := c.provider.Generate(ctx, &blades.ModelRequest{
		Instruction:  blades.SystemMessage(instruction),
		Messages:     []*blades.Message{blades.UserMessage(parts...)},
		OutputSchema: schema,
	})
	if err != nil {
		return fmt.Errorf("request llm structured output: %w", err)
	}
	// 服务商返回空消息时，无法解析结构化结果。
	if response == nil || response.Message == nil {
		return fmt.Errorf("llm structured response is empty")
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
		return fmt.Errorf("llm structured response content is empty")
	}
	err = json.Unmarshal([]byte(content), out)
	if err != nil {
		return fmt.Errorf("decode llm structured response: %w", err)
	}
	return nil
}

// imageMimeType 按图片地址推断 MIME 类型。
func imageMimeType(rawURL string) blades.MIMEType {
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

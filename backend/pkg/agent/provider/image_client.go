package provider

import (
	"strings"

	"shop/pkg/agent/sub2api"

	"github.com/go-kratos/blades"
	bootstrapConfigv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
)

const defaultImageModel = "gpt-image-2"

// ImageGenerateOptions 表示单次图片生成可覆盖的模型参数。
type ImageGenerateOptions struct {
	// Model 模型名称。
	Model string
	// Background 图片背景模式。
	Background string
	// Size 图片尺寸。
	Size string
	// Quality 图片质量。
	Quality string
	// ResponseFormat 响应格式。
	ResponseFormat string
	// OutputFormat 输出图片格式。
	OutputFormat string
	// Style 图片风格。
	Style string
	// N 生成图片数量。
	N int64
}

// ImageClient 表示 AI 图片生成模型客户端。
type ImageClient struct {
	baseURL     string
	apiKey      string
	extraFields map[string]any
}

// NewImageClient 创建 AI 图片生成模型客户端。
func NewImageClient(bootstrapCfg *bootstrapConfigv1.Client_Llm) *ImageClient {
	client := &ImageClient{}
	if bootstrapCfg == nil {
		return client
	}
	client.baseURL = strings.TrimRight(strings.TrimSpace(bootstrapCfg.GetBaseUrl()), "/")
	client.apiKey = strings.TrimSpace(bootstrapCfg.GetApiKey())
	client.extraFields = llmExtraFields(bootstrapCfg)
	return client
}

// Enabled 判断图片生成客户端是否可用。
func (c *ImageClient) Enabled() bool {
	return c != nil && c.baseURL != "" && c.apiKey != ""
}

// DefaultModel 返回图片生成默认模型名称。
func (c *ImageClient) DefaultModel() string {
	return defaultImageModel
}

// Provider 按单次请求参数构造底层图片模型提供者。
func (c *ImageClient) Provider(opts ImageGenerateOptions) blades.ModelProvider {
	if !c.Enabled() {
		return nil
	}
	model := strings.TrimSpace(opts.Model)
	if model == "" {
		model = defaultImageModel
	}
	return sub2api.NewImage(model, sub2api.ImageConfig{
		BaseURL:        c.baseURL,
		APIKey:         c.apiKey,
		Background:     strings.TrimSpace(opts.Background),
		Size:           strings.TrimSpace(opts.Size),
		Quality:        strings.TrimSpace(opts.Quality),
		ResponseFormat: strings.TrimSpace(opts.ResponseFormat),
		OutputFormat:   strings.TrimSpace(opts.OutputFormat),
		Style:          strings.TrimSpace(opts.Style),
		N:              opts.N,
		PartialImages:  2,
		ExtraFields:    c.extraFields,
	})
}

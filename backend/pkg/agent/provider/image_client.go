package provider

import (
	"strings"

	"github.com/go-kratos/blades"
	openaiProvider "github.com/go-kratos/blades/contrib/openai"
	bootstrapConfigv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
)

const defaultImageModel = "gpt-image-2"

// ImageGenerateOptions 表示单次图片生成可覆盖的模型参数。
type ImageGenerateOptions struct {
	Model          string
	Background     string
	Size           string
	Quality        string
	ResponseFormat string
	OutputFormat   string
	Style          string
	N              int64
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

// Provider 按单次请求参数构造底层图片模型提供者。
func (c *ImageClient) Provider(opts ImageGenerateOptions) blades.ModelProvider {
	if !c.Enabled() {
		return nil
	}
	model := strings.TrimSpace(opts.Model)
	if model == "" {
		model = defaultImageModel
	}
	return openaiProvider.NewImage(model, openaiProvider.ImageConfig{
		BaseURL:        c.baseURL,
		APIKey:         c.apiKey,
		Background:     strings.TrimSpace(opts.Background),
		Size:           strings.TrimSpace(opts.Size),
		Quality:        strings.TrimSpace(opts.Quality),
		ResponseFormat: strings.TrimSpace(opts.ResponseFormat),
		OutputFormat:   strings.TrimSpace(opts.OutputFormat),
		Style:          strings.TrimSpace(opts.Style),
		N:              opts.N,
		ExtraFields:    cloneExtraFields(c.extraFields),
	})
}

// DefaultModel 返回图片生成默认模型名称。
func (c *ImageClient) DefaultModel() string {
	return defaultImageModel
}

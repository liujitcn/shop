package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/go-kratos/blades"
	openaiProvider "github.com/go-kratos/blades/contrib/openai"
	bootstrapConfigv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	openaiapi "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
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
	model       string
	extraFields map[string]any
	client      openaiapi.Client
}

// NewImageClient 创建 AI 图片生成模型客户端。
func NewImageClient(bootstrapCfg *bootstrapConfigv1.Client_Llm) *ImageClient {
	client := &ImageClient{}
	if bootstrapCfg == nil {
		return client
	}
	client.baseURL = strings.TrimRight(strings.TrimSpace(bootstrapCfg.GetBaseUrl()), "/")
	client.apiKey = strings.TrimSpace(bootstrapCfg.GetApiKey())
	client.model = strings.TrimSpace(bootstrapCfg.GetModel())
	client.extraFields = llmExtraFields(bootstrapCfg)
	if client.Enabled() {
		client.client = openaiapi.NewClient(
			option.WithBaseURL(client.baseURL),
			option.WithAPIKey(client.apiKey),
			option.WithMaxRetries(0),
		)
	}
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

// ImageResponseTask 表示 Responses 图片生成后台任务。
type ImageResponseTask struct {
	// ID OpenAI Responses 响应编号。
	ID string
	// Status Responses 任务状态。
	Status string
	// CreatedAt Responses 创建时间戳。
	CreatedAt int64
	// Images 已完成的图片结果。
	Images []ImageResponseResult
}

// ImageResponseResult 表示 Responses 图片生成结果。
type ImageResponseResult struct {
	// ID 图片生成调用编号。
	ID string
	// Result 图片 base64 内容。
	Result string
	// Status 图片生成调用状态。
	Status string
}

// CreateResponseImageTask 创建 Responses 后台图片生成任务。
func (c *ImageClient) CreateResponseImageTask(ctx context.Context, prompt string, opts ImageGenerateOptions) (*ImageResponseTask, error) {
	if !c.Enabled() {
		return nil, errors.New("AI图片客户端未配置")
	}
	params := c.toResponseImageParams(prompt, opts)
	response, err := c.client.Responses.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("create responses image task failed: %w", err)
	}
	return imageResponseTaskFromResponse(response), nil
}

// GetResponseImageTask 查询 Responses 后台图片生成任务。
func (c *ImageClient) GetResponseImageTask(ctx context.Context, responseID string) (*ImageResponseTask, error) {
	if !c.Enabled() {
		return nil, errors.New("AI图片客户端未配置")
	}
	responseID = strings.TrimSpace(responseID)
	if responseID == "" {
		return nil, errors.New("responses image task id is empty")
	}
	response, err := c.client.Responses.Get(ctx, responseID, responses.ResponseGetParams{})
	if err != nil {
		return nil, fmt.Errorf("get responses image task failed: %w", err)
	}
	return imageResponseTaskFromResponse(response), nil
}

// toResponseImageParams 构建 Responses 图片生成参数。
func (c *ImageClient) toResponseImageParams(prompt string, opts ImageGenerateOptions) responses.ResponseNewParams {
	model := strings.TrimSpace(opts.Model)
	if model == "" {
		model = defaultImageModel
	}
	responseModel := strings.TrimSpace(c.model)
	if responseModel == "" {
		responseModel = "gpt-4.1-mini"
	}
	imageTool := responses.ToolImageGenerationParam{
		Model:        model,
		Background:   strings.TrimSpace(opts.Background),
		Size:         strings.TrimSpace(opts.Size),
		Quality:      strings.TrimSpace(opts.Quality),
		OutputFormat: strings.TrimSpace(opts.OutputFormat),
	}
	params := responses.ResponseNewParams{
		Model:      responseModel,
		Input:      responses.ResponseNewParamsInputUnion{OfString: param.NewOpt(strings.TrimSpace(prompt))},
		Background: param.NewOpt(true),
		Store:      param.NewOpt(true),
		Tools: []responses.ToolUnionParam{
			{OfImageGeneration: &imageTool},
		},
		ToolChoice: responses.ResponseNewParamsToolChoiceUnion{
			OfHostedTool: &responses.ToolChoiceTypesParam{
				Type: responses.ToolChoiceTypesTypeImageGeneration,
			},
		},
	}
	extraFields := cloneExtraFields(c.extraFields)
	if len(extraFields) > 0 {
		params.SetExtraFields(extraFields)
	}
	return params
}

// imageResponseTaskFromResponse 将 Responses 响应转换成图片任务。
func imageResponseTaskFromResponse(response *responses.Response) *ImageResponseTask {
	if response == nil {
		return nil
	}
	task := &ImageResponseTask{
		ID:        strings.TrimSpace(response.ID),
		Status:    string(response.Status),
		CreatedAt: int64(response.CreatedAt),
	}
	for _, output := range response.Output {
		if output.Type != "image_generation_call" {
			continue
		}
		result := strings.TrimSpace(output.Result)
		task.Images = append(task.Images, ImageResponseResult{
			ID:     strings.TrimSpace(output.ID),
			Result: result,
			Status: strings.TrimSpace(output.Status),
		})
	}
	return task
}

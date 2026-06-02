package openai

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	sdkopenai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/shared"
)

// ChatCompletionsConfig 表示 OpenAI Chat Completions 模型配置。
type ChatCompletionsConfig struct {
	// BaseURL OpenAI 兼容基础地址，通常以 /v1 结尾。
	BaseURL string
	// APIKey OpenAI API Key。
	APIKey string
	// MaxOutputTokens 最大输出 token 数。
	MaxOutputTokens int64
	// Temperature 采样温度。
	Temperature float64
	// TopP 核采样参数。
	TopP float64
	// ExtraFields 额外透传字段。
	ExtraFields map[string]any
	// ReasoningEffort 推理强度。
	ReasoningEffort shared.ReasoningEffort
	// RequestOptions 请求级 OpenAI SDK 选项。
	RequestOptions []option.RequestOption
}

// chatCompletionsModel 使用 OpenAI Chat Completions API 实现 Eino 模型提供者。
type chatCompletionsModel struct {
	model  string
	config ChatCompletionsConfig
	client sdkopenai.Client
}

// NewChatCompletions 创建 OpenAI Chat Completions 模型提供者。
func NewChatCompletions(modelName string, config ChatCompletionsConfig) model.BaseChatModel {
	opts := make([]option.RequestOption, 0, len(config.RequestOptions)+2)
	opts = append(opts, config.RequestOptions...)
	if baseURL := config.BaseURL; baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	if apiKey := config.APIKey; apiKey != "" {
		opts = append(opts, option.WithAPIKey(apiKey))
	}
	return &chatCompletionsModel{
		model:  modelName,
		config: config,
		client: sdkopenai.NewClient(opts...),
	}
}

// Generate 执行非流式 Chat Completions 请求。
func (m *chatCompletionsModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	params, err := m.buildChatCompletionsParams(input, opts...)
	if err != nil {
		return nil, err
	}
	response, err := m.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, err
	}
	return chatCompletionToMessage(response)
}

// Stream 执行流式 Chat Completions 请求。
func (m *chatCompletionsModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	params, err := m.buildChatCompletionsParams(input, opts...)
	if err != nil {
		return nil, err
	}
	reader, writer := schema.Pipe[*schema.Message](8)
	go m.streamChatCompletions(ctx, params, writer)
	return reader, nil
}

// buildChatCompletionsParams 将 Eino 请求转换成 OpenAI Chat Completions 请求参数。
func (m *chatCompletionsModel) buildChatCompletionsParams(input []*schema.Message, opts ...model.Option) (sdkopenai.ChatCompletionNewParams, error) {
	options := model.GetCommonOptions(&model.Options{}, opts...)
	params := sdkopenai.ChatCompletionNewParams{
		Model:    shared.ChatModel(m.model),
		Messages: chatCompletionMessages(input),
		Store:    param.NewOpt(false),
	}
	if len(params.Messages) == 0 {
		return sdkopenai.ChatCompletionNewParams{}, errors.New("chat completions request messages is empty")
	}
	if options.Model != nil && *options.Model != "" {
		params.Model = shared.ChatModel(*options.Model)
	}
	if len(options.Tools) > 0 {
		chatTools, err := toolsToChatCompletionTools(options.Tools)
		if err != nil {
			return sdkopenai.ChatCompletionNewParams{}, err
		}
		params.Tools = chatTools
	}
	applyChatCompletionToolChoice(&params, options)
	if m.config.MaxOutputTokens > 0 {
		params.MaxCompletionTokens = param.NewOpt(m.config.MaxOutputTokens)
	}
	if options.MaxTokens != nil && *options.MaxTokens > 0 {
		params.MaxCompletionTokens = param.NewOpt(int64(*options.MaxTokens))
	}
	if m.config.Temperature > 0 {
		params.Temperature = param.NewOpt(m.config.Temperature)
	}
	if options.Temperature != nil && *options.Temperature > 0 {
		params.Temperature = param.NewOpt(float64(*options.Temperature))
	}
	if m.config.TopP > 0 {
		params.TopP = param.NewOpt(m.config.TopP)
	}
	if options.TopP != nil && *options.TopP > 0 {
		params.TopP = param.NewOpt(float64(*options.TopP))
	}
	if m.config.ReasoningEffort != "" {
		params.ReasoningEffort = m.config.ReasoningEffort
	}
	if len(options.Stop) > 0 {
		params.Stop = sdkopenai.ChatCompletionNewParamsStopUnion{OfStringArray: options.Stop}
	}
	if len(m.config.ExtraFields) > 0 {
		params.SetExtraFields(m.config.ExtraFields)
	}
	return params, nil
}

// chatCompletionMessages 将 Eino 消息转换为 Chat Completions messages。
func chatCompletionMessages(input []*schema.Message) []sdkopenai.ChatCompletionMessageParamUnion {
	messages := make([]sdkopenai.ChatCompletionMessageParamUnion, 0, len(input))
	for _, message := range input {
		if message == nil {
			continue
		}
		switch message.Role {
		case schema.System:
			if text := messageText(message); text != "" {
				messages = append(messages, sdkopenai.SystemMessage(text))
			}
		case schema.Assistant:
			messages = append(messages, chatCompletionAssistantMessage(message))
		case schema.Tool:
			if message.ToolCallID != "" {
				messages = append(messages, sdkopenai.ToolMessage(message.Content, message.ToolCallID))
			}
		default:
			if content := chatCompletionUserContent(message); len(content) > 0 {
				messages = append(messages, sdkopenai.UserMessage(content))
			}
		}
	}
	return messages
}

// chatCompletionAssistantMessage 转换助手消息，保留可能存在的工具调用上下文。
func chatCompletionAssistantMessage(message *schema.Message) sdkopenai.ChatCompletionMessageParamUnion {
	assistant := sdkopenai.ChatCompletionAssistantMessageParam{}
	if text := messageText(message); text != "" {
		assistant.Content.OfString = param.NewOpt(text)
	}
	for _, toolCall := range message.ToolCalls {
		if toolCall.ID == "" || toolCall.Function.Name == "" {
			continue
		}
		assistant.ToolCalls = append(assistant.ToolCalls, sdkopenai.ChatCompletionMessageToolCallUnionParam{
			OfFunction: &sdkopenai.ChatCompletionMessageFunctionToolCallParam{
				ID: toolCall.ID,
				Function: sdkopenai.ChatCompletionMessageFunctionToolCallFunctionParam{
					Name:      toolCall.Function.Name,
					Arguments: toolCall.Function.Arguments,
				},
			},
		})
	}
	return sdkopenai.ChatCompletionMessageParamUnion{OfAssistant: &assistant}
}

// chatCompletionUserContent 转换用户文本和多模态输入片段。
func chatCompletionUserContent(message *schema.Message) []sdkopenai.ChatCompletionContentPartUnionParam {
	parts := make([]sdkopenai.ChatCompletionContentPartUnionParam, 0, 1+len(message.UserInputMultiContent))
	if message.Content != "" {
		parts = append(parts, sdkopenai.TextContentPart(message.Content))
	}
	for _, part := range message.UserInputMultiContent {
		switch part.Type {
		case schema.ChatMessagePartTypeText:
			if part.Text != "" {
				parts = append(parts, sdkopenai.TextContentPart(part.Text))
			}
		case schema.ChatMessagePartTypeImageURL:
			imageURL := inputImageURL(part.Image)
			if imageURL == "" {
				continue
			}
			parts = append(parts, sdkopenai.ImageContentPart(sdkopenai.ChatCompletionContentPartImageImageURLParam{
				URL:    imageURL,
				Detail: "auto",
			}))
		}
	}
	return parts
}

// toolsToChatCompletionTools 将 Eino 工具转换成 Chat Completions 工具。
func toolsToChatCompletionTools(toolList []*schema.ToolInfo) ([]sdkopenai.ChatCompletionToolUnionParam, error) {
	if len(toolList) == 0 {
		return nil, nil
	}
	result := make([]sdkopenai.ChatCompletionToolUnionParam, 0, len(toolList))
	for _, item := range toolList {
		if item == nil || item.Name == "" {
			continue
		}
		parameters := map[string]any{}
		if item.ParamsOneOf != nil {
			schemaValue, err := item.ParamsOneOf.ToJSONSchema()
			if err != nil {
				return nil, err
			}
			if schemaValue != nil {
				raw, err := json.Marshal(schemaValue)
				if err != nil {
					return nil, err
				}
				if err = json.Unmarshal(raw, &parameters); err != nil {
					return nil, err
				}
			}
		}
		fn := shared.FunctionDefinitionParam{
			Name:       item.Name,
			Parameters: shared.FunctionParameters(parameters),
			Strict:     param.NewOpt(false),
		}
		if description := item.Desc; description != "" {
			fn.Description = param.NewOpt(description)
		}
		result = append(result, sdkopenai.ChatCompletionToolUnionParam{
			OfFunction: &sdkopenai.ChatCompletionFunctionToolParam{Function: fn},
		})
	}
	return result, nil
}

// applyChatCompletionToolChoice 应用 Eino 的工具选择策略。
func applyChatCompletionToolChoice(params *sdkopenai.ChatCompletionNewParams, options *model.Options) {
	if params == nil || options == nil {
		return
	}
	if options.ToolChoice == nil {
		if len(params.Tools) > 0 {
			params.ToolChoice = sdkopenai.ChatCompletionToolChoiceOptionUnionParam{
				OfAuto: param.NewOpt(string(sdkopenai.ChatCompletionToolChoiceOptionAutoAuto)),
			}
		}
		return
	}
	switch *options.ToolChoice {
	case schema.ToolChoiceForbidden:
		params.ToolChoice = sdkopenai.ChatCompletionToolChoiceOptionUnionParam{
			OfAuto: param.NewOpt(string(sdkopenai.ChatCompletionToolChoiceOptionAutoNone)),
		}
	case schema.ToolChoiceForced:
		params.ToolChoice = sdkopenai.ChatCompletionToolChoiceOptionUnionParam{
			OfAuto: param.NewOpt(string(sdkopenai.ChatCompletionToolChoiceOptionAutoRequired)),
		}
	default:
		params.ToolChoice = sdkopenai.ChatCompletionToolChoiceOptionUnionParam{
			OfAuto: param.NewOpt(string(sdkopenai.ChatCompletionToolChoiceOptionAutoAuto)),
		}
	}
}

// chatCompletionToMessage 将 Chat Completions 响应转换成 Eino 消息。
func chatCompletionToMessage(response *sdkopenai.ChatCompletion) (*schema.Message, error) {
	if response == nil {
		return nil, errors.New("chat completions api returned empty response")
	}
	if len(response.Choices) == 0 {
		return nil, errors.New("chat completions api returned empty choices")
	}
	choice := response.Choices[0]
	message := &schema.Message{
		Role:    schema.Assistant,
		Content: choice.Message.Content,
		ResponseMeta: &schema.ResponseMeta{
			FinishReason: choice.FinishReason,
			Usage:        chatCompletionTokenUsage(response.Usage),
		},
		Extra: map[string]any{"response_id": response.ID},
	}
	for _, item := range choice.Message.ToolCalls {
		if item.Function.Name == "" {
			continue
		}
		message.ToolCalls = append(message.ToolCalls, schema.ToolCall{
			ID: item.ID,
			Function: schema.FunctionCall{
				Name:      item.Function.Name,
				Arguments: item.Function.Arguments,
			},
		})
	}
	return message, nil
}

// chatCompletionTokenUsage 转换 Chat Completions token 用量。
func chatCompletionTokenUsage(usage sdkopenai.CompletionUsage) *schema.TokenUsage {
	return &schema.TokenUsage{
		PromptTokens:     int(usage.PromptTokens),
		CompletionTokens: int(usage.CompletionTokens),
		TotalTokens:      int(usage.TotalTokens),
		PromptTokenDetails: schema.PromptTokenDetails{
			CachedTokens: int(usage.PromptTokensDetails.CachedTokens),
		},
		CompletionTokensDetails: schema.CompletionTokensDetails{
			ReasoningTokens: int(usage.CompletionTokensDetails.ReasoningTokens),
		},
	}
}

// streamChatCompletions 将 Chat Completions 流式事件转换成 Eino 响应。
func (m *chatCompletionsModel) streamChatCompletions(ctx context.Context, params sdkopenai.ChatCompletionNewParams, writer *schema.StreamWriter[*schema.Message]) {
	defer writer.Close()
	stream := m.client.Chat.Completions.NewStreaming(ctx, params)
	defer func() {
		_ = stream.Close()
	}()

	var content strings.Builder
	var accumulator sdkopenai.ChatCompletionAccumulator
	for stream.Next() {
		chunk := stream.Current()
		if !accumulator.AddChunk(chunk) {
			writer.Send(nil, errors.New("chat completions stream chunk mismatch"))
			return
		}
		for _, choice := range chunk.Choices {
			if choice.Delta.Content == "" {
				continue
			}
			content.WriteString(choice.Delta.Content)
			if writer.Send(&schema.Message{Role: schema.Assistant, Content: choice.Delta.Content}, nil) {
				return
			}
		}
	}
	if err := stream.Err(); err != nil {
		writer.Send(nil, err)
		return
	}
	message, err := chatCompletionToMessage(&accumulator.ChatCompletion)
	if err != nil && content.Len() > 0 {
		message = &schema.Message{Role: schema.Assistant, Content: content.String()}
		err = nil
	}
	markStreamFinal(message)
	writer.Send(message, err)
}

// _ 保证类型满足 Eino 模型接口。
var _ model.BaseChatModel = (*chatCompletionsModel)(nil)

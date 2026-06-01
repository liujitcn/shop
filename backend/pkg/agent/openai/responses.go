package openai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	sdkopenai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
	sdkresponses "github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"
)

// ResponsesConfig 表示 OpenAI Responses 模型配置。
type ResponsesConfig struct {
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

// responsesModel 使用 OpenAI Responses API 实现 Eino 模型提供者。
type responsesModel struct {
	model  string
	config ResponsesConfig
	client sdkopenai.Client
}

// NewResponses 创建 OpenAI Responses 模型提供者。
func NewResponses(modelName string, config ResponsesConfig) model.BaseChatModel {
	opts := make([]option.RequestOption, 0, len(config.RequestOptions)+2)
	opts = append(opts, config.RequestOptions...)
	if baseURL := config.BaseURL; baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	if apiKey := config.APIKey; apiKey != "" {
		opts = append(opts, option.WithAPIKey(apiKey))
	}
	return &responsesModel{
		model:  modelName,
		config: config,
		client: sdkopenai.NewClient(opts...),
	}
}

// Generate 执行非流式 Responses 请求。
func (m *responsesModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	params, err := m.buildResponsesParams(input, opts...)
	if err != nil {
		return nil, err
	}
	response, err := m.client.Responses.New(ctx, params)
	if err != nil {
		return nil, err
	}
	return responseToMessage(response, "")
}

// Stream 执行流式 Responses 请求。
func (m *responsesModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	params, err := m.buildResponsesParams(input, opts...)
	if err != nil {
		return nil, err
	}
	reader, writer := schema.Pipe[*schema.Message](8)
	go m.streamResponses(ctx, params, writer)
	return reader, nil
}

// buildResponsesParams 将 Eino 请求转换成 OpenAI Responses 请求参数。
func (m *responsesModel) buildResponsesParams(input []*schema.Message, opts ...model.Option) (sdkresponses.ResponseNewParams, error) {
	options := model.GetCommonOptions(&model.Options{}, opts...)
	params := sdkresponses.ResponseNewParams{
		Model: m.model,
		Input: sdkresponses.ResponseNewParamsInputUnion{
			OfInputItemList: responsesInputItems(input),
		},
		Store: param.NewOpt(false),
		Instructions: param.NewOpt(
			responsesInstructions(input),
		),
	}
	if options.Model != nil && *options.Model != "" {
		params.Model = *options.Model
	}
	if len(options.Tools) > 0 {
		responseTools, err := toolsToResponsesTools(options.Tools)
		if err != nil {
			return sdkresponses.ResponseNewParams{}, err
		}
		params.Tools = responseTools
	}
	if len(params.Tools) > 0 {
		params.ToolChoice = sdkresponses.ResponseNewParamsToolChoiceUnion{OfToolChoiceMode: param.NewOpt(sdkresponses.ToolChoiceOptionsAuto)}
	}
	if len(params.Tools) == 0 && !hasFunctionToolContext(input) {
		params.Tools = []sdkresponses.ToolUnionParam{sdkresponses.ToolParamOfWebSearch(sdkresponses.WebSearchToolTypeWebSearch)}
		params.Include = []sdkresponses.ResponseIncludable{sdkresponses.ResponseIncludableWebSearchCallActionSources}
		params.ToolChoice = sdkresponses.ResponseNewParamsToolChoiceUnion{OfToolChoiceMode: param.NewOpt(sdkresponses.ToolChoiceOptionsAuto)}
	}
	if m.config.MaxOutputTokens > 0 {
		params.MaxOutputTokens = param.NewOpt(m.config.MaxOutputTokens)
	}
	if options.MaxTokens != nil && *options.MaxTokens > 0 {
		params.MaxOutputTokens = param.NewOpt(int64(*options.MaxTokens))
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
		params.Reasoning = shared.ReasoningParam{
			Effort: m.config.ReasoningEffort,
		}
	}
	if len(m.config.ExtraFields) > 0 {
		params.SetExtraFields(m.config.ExtraFields)
	}
	return params, nil
}

// hasFunctionToolContext 判断当前请求是否已经处在内部 function tool 回填阶段。
func hasFunctionToolContext(input []*schema.Message) bool {
	for _, message := range input {
		if message == nil {
			continue
		}
		if message.Role == schema.Tool {
			return true
		}
		if len(message.ToolCalls) > 0 {
			return true
		}
	}
	return false
}

// responsesInputItems 将 Eino 消息转换为 Responses input。
func responsesInputItems(input []*schema.Message) sdkresponses.ResponseInputParam {
	items := make(sdkresponses.ResponseInputParam, 0, len(input))
	for _, message := range input {
		if message == nil || message.Role == schema.System {
			continue
		}
		if message.Role == schema.Tool {
			if message.ToolCallID != "" {
				items = append(items, sdkresponses.ResponseInputItemParamOfFunctionCallOutput(message.ToolCallID, message.Content))
			}
			continue
		}
		if message.Role == schema.Assistant {
			if len(message.ToolCalls) > 0 {
				for _, toolCall := range message.ToolCalls {
					items = append(items, sdkresponses.ResponseInputItemParamOfFunctionCall(toolCall.Function.Arguments, toolCall.ID, toolCall.Function.Name))
				}
				continue
			}
			text := messageText(message)
			if text == "" {
				continue
			}
			items = append(items, sdkresponses.ResponseInputItemParamOfOutputMessage(
				[]sdkresponses.ResponseOutputMessageContentUnionParam{{OfOutputText: &sdkresponses.ResponseOutputTextParam{Text: text}}},
				normalizedMessageID(""),
				sdkresponses.ResponseOutputMessageStatusCompleted,
			))
			continue
		}
		content := responsesContentParts(message)
		if len(content) == 0 {
			continue
		}
		role := "user"
		if message.Role == schema.System {
			role = "system"
		}
		items = append(items, sdkresponses.ResponseInputItemParamOfInputMessage(content, role))
	}
	return items
}

// responsesContentParts 将消息片段转换为 Responses content。
func responsesContentParts(message *schema.Message) sdkresponses.ResponseInputMessageContentListParam {
	if message == nil {
		return nil
	}
	parts := make(sdkresponses.ResponseInputMessageContentListParam, 0, 1+len(message.UserInputMultiContent))
	if message.Content != "" {
		parts = append(parts, sdkresponses.ResponseInputContentParamOfInputText(message.Content))
	}
	for _, part := range message.UserInputMultiContent {
		switch part.Type {
		case schema.ChatMessagePartTypeText:
			if part.Text != "" {
				parts = append(parts, sdkresponses.ResponseInputContentParamOfInputText(part.Text))
			}
		case schema.ChatMessagePartTypeImageURL:
			imageURL := inputImageURL(part.Image)
			if imageURL == "" {
				continue
			}
			inputImage := sdkresponses.ResponseInputContentParamOfInputImage(sdkresponses.ResponseInputImageDetailAuto)
			inputImage.OfInputImage.ImageURL = param.NewOpt(imageURL)
			parts = append(parts, inputImage)
		}
	}
	return parts
}

// inputImageURL 转换 Eino 图片输入为 Responses 可识别的 URL 或 data URL。
func inputImageURL(image *schema.MessageInputImage) string {
	if image == nil {
		return ""
	}
	if image.URL != nil && *image.URL != "" {
		return *image.URL
	}
	if image.Base64Data == nil || *image.Base64Data == "" {
		return ""
	}
	mimeType := image.MIMEType
	if mimeType == "" {
		mimeType = "image/png"
	}
	return fmt.Sprintf("data:%s;base64,%s", mimeType, *image.Base64Data)
}

// messageText 提取 Eino 消息中的文本内容。
func messageText(message *schema.Message) string {
	if message == nil {
		return ""
	}
	parts := make([]string, 0, 1+len(message.AssistantGenMultiContent)+len(message.UserInputMultiContent))
	if message.Content != "" {
		parts = append(parts, message.Content)
	}
	for _, part := range message.AssistantGenMultiContent {
		if part.Type == schema.ChatMessagePartTypeText && part.Text != "" {
			parts = append(parts, part.Text)
		}
	}
	for _, part := range message.UserInputMultiContent {
		if part.Type == schema.ChatMessagePartTypeText && part.Text != "" {
			parts = append(parts, part.Text)
		}
	}
	return strings.Join(parts, "\n")
}

// normalizedMessageID 返回符合 Responses 历史输出消息格式的 ID。
func normalizedMessageID(raw string) string {
	itemID := raw
	if itemID == "" {
		return fmt.Sprintf("msg_%d", time.Now().UnixNano())
	}
	if strings.HasPrefix(itemID, "msg_") {
		return itemID
	}
	return "msg_" + strings.ReplaceAll(itemID, "-", "")
}

// responsesInstructions 返回 Responses 请求的系统提示词。
func responsesInstructions(input []*schema.Message) string {
	instructions := "你是一个通用 AI 聊天助手。"
	for _, message := range input {
		if message != nil && message.Role == schema.System {
			if text := messageText(message); text != "" {
				return text
			}
		}
	}
	return instructions
}

// toolsToResponsesTools 将 Eino 工具转换成 Responses 工具。
func toolsToResponsesTools(toolList []*schema.ToolInfo) ([]sdkresponses.ToolUnionParam, error) {
	if len(toolList) == 0 {
		return nil, nil
	}
	result := make([]sdkresponses.ToolUnionParam, 0, len(toolList))
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
		fn := sdkresponses.FunctionToolParam{
			Name:       item.Name,
			Parameters: parameters,
			Strict:     param.NewOpt(false),
		}
		if description := item.Desc; description != "" {
			fn.Description = param.NewOpt(description)
		}
		result = append(result, sdkresponses.ToolUnionParam{OfFunction: &fn})
	}
	return result, nil
}

// responseToMessage 将 Responses 响应转换成 Eino 消息。
func responseToMessage(response *sdkresponses.Response, fallbackText string) (*schema.Message, error) {
	if response == nil && fallbackText == "" {
		return nil, errors.New("responses api returned empty response")
	}
	if err := responseError(response); err != nil {
		return nil, err
	}
	content := fallbackText
	message := &schema.Message{Role: schema.Assistant}
	if response != nil {
		if text := response.OutputText(); text != "" {
			content = text
		}
		message.ResponseMeta = &schema.ResponseMeta{
			Usage: &schema.TokenUsage{
				PromptTokens:     int(response.Usage.InputTokens),
				CompletionTokens: int(response.Usage.OutputTokens),
				TotalTokens:      int(response.Usage.TotalTokens),
			},
		}
		message.Extra = map[string]any{"response_id": response.ID}
	}
	message.Content = content
	for _, item := range responseToolCalls(response) {
		message.ToolCalls = append(message.ToolCalls, item)
	}
	return message, nil
}

// responseError 收敛 Responses 失败信息。
func responseError(response *sdkresponses.Response) error {
	if response == nil {
		return nil
	}
	if message := response.Error.Message; message != "" {
		return errors.New(message)
	}
	if reason := response.IncompleteDetails.Reason; reason != "" && response.Status == sdkresponses.ResponseStatusIncomplete {
		return fmt.Errorf("responses api incomplete: %s", reason)
	}
	if response.Status == sdkresponses.ResponseStatusFailed {
		return errors.New("responses api failed")
	}
	return nil
}

// responseToolCalls 提取 Responses 输出中的工具调用。
func responseToolCalls(response *sdkresponses.Response) []schema.ToolCall {
	if response == nil {
		return nil
	}
	toolCalls := make([]schema.ToolCall, 0)
	for _, item := range response.Output {
		if item.Type != "function_call" {
			continue
		}
		call := item.AsFunctionCall()
		toolCalls = append(toolCalls, schema.ToolCall{
			ID: call.CallID,
			Function: schema.FunctionCall{
				Name:      call.Name,
				Arguments: call.Arguments,
			},
		})
	}
	return toolCalls
}

// streamResponses 将 Responses 流式事件转换成 Eino 响应。
func (m *responsesModel) streamResponses(ctx context.Context, params sdkresponses.ResponseNewParams, writer *schema.StreamWriter[*schema.Message]) {
	defer writer.Close()
	stream := m.client.Responses.NewStreaming(ctx, params)
	defer func() {
		_ = stream.Close()
	}()

	var content strings.Builder
	var finalResponse sdkresponses.Response
	var hasFinalResponse bool
	for stream.Next() {
		event := stream.Current()
		switch item := event.AsAny().(type) {
		case sdkresponses.ResponseTextDeltaEvent:
			if item.Delta == "" {
				continue
			}
			content.WriteString(item.Delta)
			if writer.Send(&schema.Message{Role: schema.Assistant, Content: item.Delta}, nil) {
				return
			}
		case sdkresponses.ResponseCompletedEvent:
			finalResponse = item.Response
			hasFinalResponse = true
		case sdkresponses.ResponseIncompleteEvent:
			writer.Send(nil, responseError(&item.Response))
			return
		case sdkresponses.ResponseFailedEvent:
			writer.Send(nil, responseError(&item.Response))
			return
		case sdkresponses.ResponseErrorEvent:
			writer.Send(nil, responseEventError(item))
			return
		}
	}
	if err := stream.Err(); err != nil {
		writer.Send(nil, err)
		return
	}
	if hasFinalResponse {
		message, err := responseToMessage(&finalResponse, content.String())
		markStreamFinal(message)
		writer.Send(message, err)
		return
	}
	message, err := responseToMessage(nil, content.String())
	markStreamFinal(message)
	writer.Send(message, err)
}

// markStreamFinal 标记流式响应中的最终完整消息，避免调用方重复推送完整正文。
func markStreamFinal(message *schema.Message) {
	if message == nil {
		return
	}
	if message.Extra == nil {
		message.Extra = map[string]any{}
	}
	message.Extra["stream_final"] = true
}

// responseEventError 将 Responses 流式错误事件转换为普通错误。
func responseEventError(event sdkresponses.ResponseErrorEvent) error {
	message := event.Message
	if message == "" {
		message = "responses api stream error"
	}
	if code := event.Code; code != "" {
		return fmt.Errorf("%s: %s", code, message)
	}
	return errors.New(message)
}

// _ 保证类型满足 Eino 模型接口。
var _ model.BaseChatModel = (*responsesModel)(nil)

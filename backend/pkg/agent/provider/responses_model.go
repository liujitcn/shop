package provider

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/go-kratos/blades"
	"github.com/go-kratos/kratos/v2/log"
	openaiapi "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"
)

// ResponsesConfig 表示 OpenAI Responses 模型配置。
type ResponsesConfig struct {
	// BaseURL OpenAI 兼容接口基础地址。
	BaseURL string
	// APIKey OpenAI 兼容接口密钥。
	APIKey string
	// MaxOutputTokens 最大输出 token 数。
	MaxOutputTokens int64
	// Temperature 采样温度。
	Temperature float64
	// TopP 核采样参数。
	TopP float64
	// ExtraFields 额外透传字段。
	ExtraFields map[string]any
	// RequestOptions OpenAI SDK 请求选项。
	RequestOptions []option.RequestOption
	// ReasoningEffort 推理强度。
	ReasoningEffort shared.ReasoningEffort
}

// ResponsesModel 使用 OpenAI Responses API 实现 Blades 模型提供者。
type ResponsesModel struct {
	model  string
	config ResponsesConfig
	client openaiapi.Client
}

// NewResponsesModel 创建 OpenAI Responses 模型提供者。
func NewResponsesModel(model string, config ResponsesConfig) blades.ModelProvider {
	opts := config.RequestOptions
	if config.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(config.BaseURL))
	}
	if config.APIKey != "" {
		opts = append(opts, option.WithAPIKey(config.APIKey))
	}
	return &ResponsesModel{
		model:  model,
		config: config,
		client: openaiapi.NewClient(opts...),
	}
}

// Name 返回模型名称。
func (m *ResponsesModel) Name() string {
	return m.model
}

// Generate 执行非流式 Responses 请求。
func (m *ResponsesModel) Generate(ctx context.Context, req *blades.ModelRequest) (*blades.ModelResponse, error) {
	return m.streamResponses(ctx, req, nil)
}

// NewStreaming 执行流式 Responses 请求。
func (m *ResponsesModel) NewStreaming(ctx context.Context, req *blades.ModelRequest) blades.Generator[*blades.ModelResponse, error] {
	return func(yield func(*blades.ModelResponse, error) bool) {
		_, err := m.streamResponses(ctx, req, yield)
		if err != nil {
			yield(nil, err)
		}
	}
}

// streamResponses 使用流式 Responses 完成一次请求，兼容 sub2api/Codex 上游。
func (m *ResponsesModel) streamResponses(ctx context.Context, req *blades.ModelRequest, yield func(*blades.ModelResponse, error) bool) (*blades.ModelResponse, error) {
	var err error
	var params responses.ResponseNewParams
	params, err = m.toResponseNewParams(req)
	if err != nil {
		return nil, err
	}
	stream := m.client.Responses.NewStreaming(ctx, params)
	defer func() {
		if closeErr := stream.Close(); closeErr != nil {
			log.Warnf("关闭 Responses 流失败：%v", closeErr)
		}
	}()
	var finalResponse *responses.Response
	var content strings.Builder
	for stream.Next() {
		event := stream.Current()
		switch value := event.AsAny().(type) {
		case responses.ResponseTextDeltaEvent:
			if value.Delta == "" {
				continue
			}
			content.WriteString(value.Delta)
			if yield == nil {
				continue
			}
			message := blades.NewAssistantMessage(blades.StatusIncomplete)
			message.Parts = append(message.Parts, blades.TextPart{Text: value.Delta})
			if !yield(&blades.ModelResponse{Message: message}, nil) {
				return &blades.ModelResponse{Message: message}, nil
			}
		case responses.ResponseCompletedEvent:
			finalResponse = &value.Response
		case responses.ResponseFailedEvent:
			return nil, responseError(value.Response)
		case responses.ResponseErrorEvent:
			return nil, errors.New(value.Message)
		}
	}
	err = stream.Err()
	if err != nil {
		return nil, err
	}
	var finalModelResponse *blades.ModelResponse
	finalModelResponse, err = m.responseToModelResponse(finalResponse, content.String())
	if err != nil {
		return nil, err
	}
	if yield != nil {
		yield(finalModelResponse, nil)
	}
	return finalModelResponse, nil
}

// toResponseNewParams 将 Blades 请求转换为 Responses 请求参数。
func (m *ResponsesModel) toResponseNewParams(req *blades.ModelRequest) (responses.ResponseNewParams, error) {
	logUnsupportedTools(req)
	params := responses.ResponseNewParams{
		Model: m.model,
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: toInputItems(req),
		},
		Store: param.NewOpt(false),
		Tools: []responses.ToolUnionParam{
			responses.ToolParamOfWebSearch(responses.WebSearchToolTypeWebSearch),
		},
		ToolChoice: responses.ResponseNewParamsToolChoiceUnion{
			OfToolChoiceMode: param.NewOpt(responses.ToolChoiceOptionsAuto),
		},
		Include: []responses.ResponseIncludable{
			responses.ResponseIncludableWebSearchCallActionSources,
		},
	}
	if m.config.MaxOutputTokens > 0 {
		params.MaxOutputTokens = param.NewOpt(m.config.MaxOutputTokens)
	}
	if m.config.Temperature > 0 {
		params.Temperature = param.NewOpt(m.config.Temperature)
	}
	if m.config.TopP > 0 {
		params.TopP = param.NewOpt(m.config.TopP)
	}
	if strings.TrimSpace(string(m.config.ReasoningEffort)) != "" {
		params.Reasoning = shared.ReasoningParam{
			Effort: m.config.ReasoningEffort,
		}
	}
	if extraFields := cloneExtraFields(m.config.ExtraFields); len(extraFields) > 0 {
		params.SetExtraFields(extraFields)
	}
	instructions := "你是一个通用 AI 聊天助手。"
	if req != nil && req.Instruction != nil {
		if text := strings.TrimSpace(messageText(req.Instruction)); text != "" {
			instructions = text
		}
	}
	params.Instructions = param.NewOpt(instructions)
	return params, nil
}

// toInputItems 将上下文消息转换为 Responses 输入项。
func toInputItems(req *blades.ModelRequest) responses.ResponseInputParam {
	if req == nil {
		return nil
	}
	inputItems := make(responses.ResponseInputParam, 0, len(req.Messages))
	for _, message := range req.Messages {
		if message == nil {
			continue
		}
		text := strings.TrimSpace(messageText(message))
		content := inputContentParts(message)
		if len(content) == 0 {
			continue
		}
		switch message.Role {
		case blades.RoleAssistant:
			if text == "" {
				continue
			}
			outputContent := []responses.ResponseOutputMessageContentUnionParam{
				{
					OfOutputText: &responses.ResponseOutputTextParam{
						Text:        text,
						Annotations: []responses.ResponseOutputTextAnnotationUnionParam{},
					},
				},
			}
			itemID := strings.TrimSpace(message.ID)
			if itemID == "" {
				itemID = "msg_" + blades.NewMessageID()
			}
			if !strings.HasPrefix(itemID, "msg_") {
				itemID = "msg_" + strings.ReplaceAll(itemID, "-", "")
			}
			inputItems = append(inputItems, responses.ResponseInputItemParamOfOutputMessage(outputContent, itemID, responses.ResponseOutputMessageStatusCompleted))
		default:
			role := string(responses.ResponseInputMessageItemRoleUser)
			if message.Role == blades.RoleSystem {
				role = string(responses.ResponseInputMessageItemRoleSystem)
			}
			inputItems = append(inputItems, responses.ResponseInputItemUnionParam{
				OfInputMessage: &responses.ResponseInputItemMessageParam{
					Content: content,
					Role:    role,
					Type:    string(responses.ResponseInputMessageItemTypeMessage),
				},
			})
		}
	}
	return inputItems
}

// responseToModelResponse 将 Responses 响应转换为 Blades 响应。
func (m *ResponsesModel) responseToModelResponse(response *responses.Response, fallbackText string) (*blades.ModelResponse, error) {
	if response == nil && strings.TrimSpace(fallbackText) == "" {
		return nil, fmt.Errorf("responses api returned empty response")
	}
	var err error
	if response != nil {
		err = responseError(*response)
		if err != nil {
			return nil, err
		}
	}
	message := blades.NewAssistantMessage(blades.StatusCompleted)
	content := strings.TrimSpace(fallbackText)
	if response != nil && strings.TrimSpace(response.OutputText()) != "" {
		content = strings.TrimSpace(response.OutputText())
	}
	if content != "" {
		message.Parts = append(message.Parts, blades.TextPart{Text: content})
	}
	if response != nil {
		message.TokenUsage = blades.TokenUsage{
			InputTokens:  response.Usage.InputTokens,
			OutputTokens: response.Usage.OutputTokens,
			TotalTokens:  response.Usage.TotalTokens,
		}
		message.Metadata = map[string]any{
			"response_id": response.ID,
		}
	}
	return &blades.ModelResponse{Message: message}, nil
}

// responseError 收敛 Responses 调用失败信息。
func responseError(response responses.Response) error {
	if strings.TrimSpace(response.Error.Message) != "" {
		return errors.New(response.Error.Message)
	}
	if strings.TrimSpace(response.IncompleteDetails.Reason) != "" {
		return fmt.Errorf("responses api incomplete: %s", response.IncompleteDetails.Reason)
	}
	return nil
}

// inputContentParts 将 Blades 消息转换为 Responses 多模态输入片段。
func inputContentParts(message *blades.Message) responses.ResponseInputMessageContentListParam {
	if message == nil {
		return nil
	}
	parts := make(responses.ResponseInputMessageContentListParam, 0, len(message.Parts))
	for _, part := range message.Parts {
		switch value := part.(type) {
		case blades.TextPart:
			if text := strings.TrimSpace(value.Text); text != "" {
				parts = append(parts, responses.ResponseInputContentParamOfInputText(text))
			}
		case blades.FilePart:
			if strings.HasPrefix(strings.ToLower(strings.TrimSpace(string(value.MIMEType))), "image/") && strings.TrimSpace(value.URI) != "" {
				imagePart := responses.ResponseInputContentParamOfInputImage(responses.ResponseInputImageDetailAuto)
				imagePart.OfInputImage.ImageURL = param.NewOpt(strings.TrimSpace(value.URI))
				parts = append(parts, imagePart)
				continue
			}
			if text := strings.TrimSpace(filePartText(value)); text != "" {
				parts = append(parts, responses.ResponseInputContentParamOfInputText(text))
			}
		case blades.DataPart:
			if strings.HasPrefix(strings.ToLower(strings.TrimSpace(string(value.MIMEType))), "image/") && len(value.Bytes) > 0 {
				imagePart := responses.ResponseInputContentParamOfInputImage(responses.ResponseInputImageDetailAuto)
				imagePart.OfInputImage.ImageURL = param.NewOpt(fmt.Sprintf(
					"data:%s;base64,%s",
					strings.TrimSpace(string(value.MIMEType)),
					base64.StdEncoding.EncodeToString(value.Bytes),
				))
				parts = append(parts, imagePart)
				continue
			}
			if len(value.Bytes) > 0 {
				parts = append(parts, responses.ResponseInputContentParamOfInputText(
					fmt.Sprintf("附件《%s》为 %s 文件，大小约 %d 字节。", strings.TrimSpace(value.Name), value.MIMEType, len(value.Bytes)),
				))
			}
		case blades.ToolPart:
			raw, err := json.Marshal(value)
			if err != nil {
				continue
			}
			if text := strings.TrimSpace(string(raw)); text != "" {
				parts = append(parts, responses.ResponseInputContentParamOfInputText(text))
			}
		}
	}
	return parts
}

// filePartText 将暂不支持的文件引用降级为文本描述。
func filePartText(part blades.FilePart) string {
	if strings.TrimSpace(part.URI) == "" {
		return ""
	}
	return fmt.Sprintf("附件《%s》地址：%s，类型：%s。", strings.TrimSpace(part.Name), strings.TrimSpace(part.URI), part.MIMEType)
}

// partText 提取 Responses 当前实现可安全转发的文本内容。
func partText(part blades.Part) string {
	switch value := part.(type) {
	case blades.TextPart:
		return value.Text
	case blades.FilePart:
		return filePartText(value)
	case blades.DataPart:
		if len(value.Bytes) == 0 {
			return ""
		}
		return fmt.Sprintf("附件《%s》为 %s 文件，大小约 %d 字节。", strings.TrimSpace(value.Name), value.MIMEType, len(value.Bytes))
	case blades.ToolPart:
		raw, err := json.Marshal(value)
		if err != nil {
			return ""
		}
		return string(raw)
	default:
		return ""
	}
}

// messageText 提取消息文本内容。
func messageText(message *blades.Message) string {
	if message == nil {
		return ""
	}
	parts := make([]string, 0, len(message.Parts))
	for _, part := range message.Parts {
		text := strings.TrimSpace(partText(part))
		if text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n")
}

// logUnsupportedTools 记录 Responses 纯聊天模式忽略的 Blades 自定义工具。
func logUnsupportedTools(req *blades.ModelRequest) {
	if req == nil || len(req.Tools) == 0 {
		return
	}
	log.Warnf("Responses 纯聊天模式忽略 %d 个 Blades 自定义工具", len(req.Tools))
}

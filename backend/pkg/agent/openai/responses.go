package openai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/go-kratos/blades"
	"github.com/go-kratos/blades/tools"
	"github.com/google/jsonschema-go/jsonschema"
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

// responsesModel 使用 OpenAI Responses API 实现 Blades 模型提供者。
type responsesModel struct {
	model  string
	config ResponsesConfig
	client sdkopenai.Client
}

// NewResponses 创建 OpenAI Responses 模型提供者。
func NewResponses(model string, config ResponsesConfig) blades.ModelProvider {
	opts := make([]option.RequestOption, 0, len(config.RequestOptions)+2)
	opts = append(opts, config.RequestOptions...)
	if baseURL := strings.TrimSpace(config.BaseURL); baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	if apiKey := strings.TrimSpace(config.APIKey); apiKey != "" {
		opts = append(opts, option.WithAPIKey(apiKey))
	}
	return &responsesModel{
		model:  strings.TrimSpace(model),
		config: config,
		client: sdkopenai.NewClient(opts...),
	}
}

// Name 返回模型名称。
func (m *responsesModel) Name() string {
	return m.model
}

// Generate 执行非流式 Responses 请求。
func (m *responsesModel) Generate(ctx context.Context, req *blades.ModelRequest) (*blades.ModelResponse, error) {
	params, err := m.buildResponsesParams(req)
	if err != nil {
		return nil, err
	}
	response, err := m.client.Responses.New(ctx, params)
	if err != nil {
		return nil, err
	}
	return responseToModelResponse(response, "")
}

// NewStreaming 执行流式 Responses 请求。
func (m *responsesModel) NewStreaming(ctx context.Context, req *blades.ModelRequest) blades.Generator[*blades.ModelResponse, error] {
	return func(yield func(*blades.ModelResponse, error) bool) {
		response, err := m.streamResponses(ctx, req, yield)
		if err != nil {
			yield(nil, err)
			return
		}
		if response != nil {
			yield(response, nil)
		}
	}
}

// streamResponses 将 Responses 流式事件转换成 Blades 响应。
func (m *responsesModel) streamResponses(ctx context.Context, req *blades.ModelRequest, yield func(*blades.ModelResponse, error) bool) (*blades.ModelResponse, error) {
	params, err := m.buildResponsesParams(req)
	if err != nil {
		return nil, err
	}
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
			if yield == nil {
				continue
			}
			message := blades.NewAssistantMessage(blades.StatusIncomplete)
			message.Parts = append(message.Parts, blades.TextPart{Text: item.Delta})
			if !yield(&blades.ModelResponse{Message: message}, nil) {
				return nil, nil
			}
		case sdkresponses.ResponseCompletedEvent:
			finalResponse = item.Response
			hasFinalResponse = true
		case sdkresponses.ResponseIncompleteEvent:
			return nil, responseError(&item.Response)
		case sdkresponses.ResponseFailedEvent:
			return nil, responseError(&item.Response)
		case sdkresponses.ResponseErrorEvent:
			return nil, responseEventError(item)
		}
	}
	if err = stream.Err(); err != nil {
		return nil, err
	}
	if hasFinalResponse {
		return responseToModelResponse(&finalResponse, content.String())
	}
	return responseToModelResponse(nil, content.String())
}

// buildResponsesParams 将 Blades 请求转换成 OpenAI Responses 请求参数。
func (m *responsesModel) buildResponsesParams(req *blades.ModelRequest) (sdkresponses.ResponseNewParams, error) {
	params := sdkresponses.ResponseNewParams{
		Model: m.model,
		Input: sdkresponses.ResponseNewParamsInputUnion{
			OfInputItemList: responsesInputItems(req),
		},
		Store:      param.NewOpt(false),
		Include:    []sdkresponses.ResponseIncludable{sdkresponses.ResponseIncludableWebSearchCallActionSources},
		ToolChoice: sdkresponses.ResponseNewParamsToolChoiceUnion{OfToolChoiceMode: param.NewOpt(sdkresponses.ToolChoiceOptionsAuto)},
		Tools:      []sdkresponses.ToolUnionParam{sdkresponses.ToolParamOfWebSearch(sdkresponses.WebSearchToolTypeWebSearch)},
		Instructions: param.NewOpt(
			responsesInstructions(req),
		),
	}
	if req != nil && len(req.Tools) > 0 {
		responseTools, err := toolsToResponsesTools(req.Tools)
		if err != nil {
			return sdkresponses.ResponseNewParams{}, err
		}
		params.Tools = append(params.Tools, responseTools...)
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
	if m.config.ReasoningEffort != "" {
		params.Reasoning = shared.ReasoningParam{
			Effort: m.config.ReasoningEffort,
		}
	}
	if req != nil && req.OutputSchema != nil {
		schema, err := schemaToMap(req.OutputSchema)
		if err != nil {
			return sdkresponses.ResponseNewParams{}, err
		}
		params.Text = sdkresponses.ResponseTextConfigParam{
			Format: responseFormatFromSchema(req.OutputSchema, schema),
		}
	}
	if len(m.config.ExtraFields) > 0 {
		params.SetExtraFields(m.config.ExtraFields)
	}
	return params, nil
}

// responsesInstructions 返回 Responses 请求的系统提示词。
func responsesInstructions(req *blades.ModelRequest) string {
	instructions := "你是一个通用 AI 聊天助手。"
	if req == nil || req.Instruction == nil {
		return instructions
	}
	if text := strings.TrimSpace(messageText(req.Instruction)); text != "" {
		return text
	}
	return instructions
}

// responsesInputItems 将 Blades 消息转换为 Responses input。
func responsesInputItems(req *blades.ModelRequest) sdkresponses.ResponseInputParam {
	if req == nil {
		return nil
	}
	items := make(sdkresponses.ResponseInputParam, 0, len(req.Messages))
	for _, message := range req.Messages {
		if message == nil {
			continue
		}
		content := responsesContentParts(message)
		if len(content) == 0 {
			continue
		}
		if message.Role == blades.RoleAssistant {
			text := strings.TrimSpace(messageText(message))
			if text == "" {
				continue
			}
			itemID := normalizedMessageID(message.ID)
			outputText := sdkresponses.ResponseOutputTextParam{Text: text}
			items = append(items, sdkresponses.ResponseInputItemParamOfOutputMessage(
				[]sdkresponses.ResponseOutputMessageContentUnionParam{{OfOutputText: &outputText}},
				itemID,
				sdkresponses.ResponseOutputMessageStatusCompleted,
			))
			continue
		}
		role := "user"
		if message.Role == blades.RoleSystem {
			role = "system"
		}
		items = append(items, sdkresponses.ResponseInputItemParamOfInputMessage(content, role))
	}
	return items
}

// responsesContentParts 将消息片段转换为 Responses content。
func responsesContentParts(message *blades.Message) sdkresponses.ResponseInputMessageContentListParam {
	if message == nil {
		return nil
	}
	parts := make(sdkresponses.ResponseInputMessageContentListParam, 0, len(message.Parts))
	for _, part := range message.Parts {
		switch value := part.(type) {
		case blades.TextPart:
			if text := strings.TrimSpace(value.Text); text != "" {
				parts = append(parts, sdkresponses.ResponseInputContentParamOfInputText(text))
			}
		case blades.FilePart:
			if value.MIMEType.Type() == "image" && strings.TrimSpace(value.URI) != "" {
				inputImage := sdkresponses.ResponseInputContentParamOfInputImage(sdkresponses.ResponseInputImageDetailAuto)
				inputImage.OfInputImage.ImageURL = param.NewOpt(strings.TrimSpace(value.URI))
				parts = append(parts, inputImage)
				continue
			}
			if text := strings.TrimSpace(partText(value)); text != "" {
				parts = append(parts, sdkresponses.ResponseInputContentParamOfInputText(text))
			}
		case blades.DataPart:
			if value.MIMEType.Type() == "image" && len(value.Bytes) > 0 {
				inputImage := sdkresponses.ResponseInputContentParamOfInputImage(sdkresponses.ResponseInputImageDetailAuto)
				inputImage.OfInputImage.ImageURL = param.NewOpt(dataURLFromPart(value))
				parts = append(parts, inputImage)
				continue
			}
			if text := strings.TrimSpace(partText(value)); text != "" {
				parts = append(parts, sdkresponses.ResponseInputContentParamOfInputText(text))
			}
		case blades.ToolPart:
			raw, err := json.Marshal(value)
			if err != nil {
				continue
			}
			parts = append(parts, sdkresponses.ResponseInputContentParamOfInputText(string(raw)))
		}
	}
	return parts
}

// normalizedMessageID 返回符合 Responses 历史输出消息格式的 ID。
func normalizedMessageID(raw string) string {
	itemID := strings.TrimSpace(raw)
	if itemID == "" {
		return "msg_" + blades.NewMessageID()
	}
	if strings.HasPrefix(itemID, "msg_") {
		return itemID
	}
	return "msg_" + strings.ReplaceAll(itemID, "-", "")
}

// toolsToResponsesTools 将 Blades 工具转换成 Responses 工具。
func toolsToResponsesTools(toolList []tools.Tool) ([]sdkresponses.ToolUnionParam, error) {
	if len(toolList) == 0 {
		return nil, nil
	}
	result := make([]sdkresponses.ToolUnionParam, 0, len(toolList))
	for _, item := range toolList {
		if item == nil || strings.TrimSpace(item.Name()) == "" {
			continue
		}
		parameters := map[string]any{}
		if item.InputSchema() != nil {
			schema, err := schemaToMap(item.InputSchema())
			if err != nil {
				return nil, err
			}
			parameters = schema
		}
		fn := sdkresponses.FunctionToolParam{
			Name:       item.Name(),
			Parameters: parameters,
			Strict:     param.NewOpt(false),
		}
		if description := strings.TrimSpace(item.Description()); description != "" {
			fn.Description = param.NewOpt(description)
		}
		result = append(result, sdkresponses.ToolUnionParam{OfFunction: &fn})
	}
	return result, nil
}

// responseFormatFromSchema 将 Blades 输出 Schema 转换为 Responses 文本格式约束。
func responseFormatFromSchema(schema *jsonschema.Schema, schemaMap map[string]any) sdkresponses.ResponseFormatTextConfigUnionParam {
	name := "structured_outputs"
	if title := strings.TrimSpace(schema.Title); title != "" {
		name = title
	}
	jsonSchema := sdkresponses.ResponseFormatTextJSONSchemaConfigParam{
		Name:   name,
		Schema: schemaMap,
		Strict: param.NewOpt(true),
	}
	if description := strings.TrimSpace(schema.Description); description != "" {
		jsonSchema.Description = param.NewOpt(description)
	}
	return sdkresponses.ResponseFormatTextConfigUnionParam{OfJSONSchema: &jsonSchema}
}

// responseToModelResponse 将 Responses 响应转换成 Blades 响应。
func responseToModelResponse(response *sdkresponses.Response, fallbackText string) (*blades.ModelResponse, error) {
	if response == nil && strings.TrimSpace(fallbackText) == "" {
		return nil, errors.New("responses api returned empty response")
	}
	if err := responseError(response); err != nil {
		return nil, err
	}
	message := blades.NewAssistantMessage(blades.StatusCompleted)
	content := strings.TrimSpace(fallbackText)
	if response != nil {
		if text := strings.TrimSpace(response.OutputText()); text != "" {
			content = text
		}
		message.TokenUsage = blades.TokenUsage{
			InputTokens:  response.Usage.InputTokens,
			OutputTokens: response.Usage.OutputTokens,
			TotalTokens:  response.Usage.TotalTokens,
		}
		message.Metadata["response_id"] = response.ID
	}
	if content != "" {
		message.Parts = append(message.Parts, blades.TextPart{Text: content})
	}
	for _, item := range responseToolParts(response) {
		message.Role = blades.RoleTool
		message.Parts = append(message.Parts, item)
	}
	return &blades.ModelResponse{Message: message}, nil
}

// responseToolParts 提取 Responses 输出中的工具调用。
func responseToolParts(response *sdkresponses.Response) []blades.ToolPart {
	if response == nil {
		return nil
	}
	parts := make([]blades.ToolPart, 0)
	for _, item := range response.Output {
		if item.Type != "function_call" {
			continue
		}
		parts = append(parts, blades.ToolPart{
			ID:      item.CallID,
			Name:    item.Name,
			Request: item.Arguments,
		})
	}
	return parts
}

// responseEventError 将 Responses 流式错误事件转换为普通错误。
func responseEventError(event sdkresponses.ResponseErrorEvent) error {
	message := strings.TrimSpace(event.Message)
	if message == "" {
		message = "responses api stream error"
	}
	if code := strings.TrimSpace(event.Code); code != "" {
		return fmt.Errorf("%s: %s", code, message)
	}
	return errors.New(message)
}

// responseError 收敛 Responses 失败信息。
func responseError(response *sdkresponses.Response) error {
	if response == nil {
		return nil
	}
	if message := strings.TrimSpace(response.Error.Message); message != "" {
		return errors.New(message)
	}
	if reason := strings.TrimSpace(response.IncompleteDetails.Reason); reason != "" && response.Status == sdkresponses.ResponseStatusIncomplete {
		return fmt.Errorf("responses api incomplete: %s", reason)
	}
	if response.Status == sdkresponses.ResponseStatusFailed {
		return errors.New("responses api failed")
	}
	return nil
}

// messageText 提取 Blades 消息中的可转发文本内容。
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

// partText 将暂不支持的附件降级成文本描述。
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

// filePartText 将文件引用转成文本描述。
func filePartText(part blades.FilePart) string {
	if strings.TrimSpace(part.URI) == "" {
		return ""
	}
	return fmt.Sprintf("附件《%s》地址：%s，类型：%s。", strings.TrimSpace(part.Name), strings.TrimSpace(part.URI), part.MIMEType)
}

// dataURLFromPart 将图片字节转换成 data URL。
func dataURLFromPart(part blades.DataPart) string {
	if len(part.Bytes) == 0 {
		return ""
	}
	mimeType := strings.TrimSpace(string(part.MIMEType))
	if mimeType == "" {
		mimeType = string(blades.MIMEImagePNG)
	}
	return fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(part.Bytes))
}

// schemaToMap 将 JSON Schema 转为普通 map，便于写入 SDK 参数。
func schemaToMap(schema *jsonschema.Schema) (map[string]any, error) {
	if schema == nil {
		return nil, nil
	}
	raw, err := json.Marshal(schema)
	if err != nil {
		return nil, err
	}
	var result map[string]any
	if err = json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	return result, nil
}

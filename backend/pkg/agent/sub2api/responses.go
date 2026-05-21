package sub2api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/go-kratos/blades"
	"github.com/go-kratos/kratos/v2/log"
)

// ResponsesConfig 表示 sub2api Responses 模型配置。
type ResponsesConfig struct {
	// BaseURL sub2api OpenAI 兼容基础地址，通常以 /v1 结尾。
	BaseURL string
	// APIKey sub2api 分配的 API Key。
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
	ReasoningEffort string
}

// responsesModel 使用 sub2api /responses 实现 Blades 模型提供者。
type responsesModel struct {
	model  string
	config ResponsesConfig
	client *apiClient
}

// NewResponses 创建 sub2api Responses 模型提供者。
func NewResponses(model string, config ResponsesConfig) blades.ModelProvider {
	return &responsesModel{
		model:  strings.TrimSpace(model),
		config: config,
		client: newAPIClient(config.BaseURL, config.APIKey),
	}
}

// Name 返回模型名称。
func (m *responsesModel) Name() string {
	return m.model
}

// Generate 执行非流式 Responses 请求。
func (m *responsesModel) Generate(ctx context.Context, req *blades.ModelRequest) (*blades.ModelResponse, error) {
	return m.streamResponses(ctx, req, nil)
}

// NewStreaming 执行流式 Responses 请求。
func (m *responsesModel) NewStreaming(ctx context.Context, req *blades.ModelRequest) blades.Generator[*blades.ModelResponse, error] {
	return func(yield func(*blades.ModelResponse, error) bool) {
		var err error
		_, err = m.streamResponses(ctx, req, yield)
		if err != nil {
			yield(nil, err)
		}
	}
}

// streamResponses 使用 SSE Responses 请求完成一次模型调用。
func (m *responsesModel) streamResponses(ctx context.Context, req *blades.ModelRequest, yield func(*blades.ModelResponse, error) bool) (*blades.ModelResponse, error) {
	var err error
	var body map[string]any
	body, err = m.buildResponsesRequest(req, true)
	if err != nil {
		return nil, err
	}
	var stream *sseStream
	stream, err = m.client.doSSE(ctx, endpointResponses, body)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err = stream.Close(); err != nil {
			log.Warnf("关闭 sub2api Responses 流失败：%v", err)
		}
	}()

	var finalResponse *responsesAPIResponse
	var finalModelResponse *blades.ModelResponse
	var content strings.Builder
	var event *sseEvent
	for {
		event, err = stream.Next()
		if err != nil {
			if errorsIsEOF(err) {
				break
			}
			return nil, err
		}
		if event == nil || event.IsDone() {
			break
		}
		var streamEvent responsesStreamEvent
		err = event.DecodeJSON(&streamEvent)
		if err != nil {
			return nil, err
		}
		switch streamEvent.Type {
		case "response.output_text.delta":
			if streamEvent.Delta == "" {
				continue
			}
			content.WriteString(streamEvent.Delta)
			if yield == nil {
				continue
			}
			message := blades.NewAssistantMessage(blades.StatusIncomplete)
			message.Parts = append(message.Parts, blades.TextPart{Text: streamEvent.Delta})
			if !yield(&blades.ModelResponse{Message: message}, nil) {
				return &blades.ModelResponse{Message: message}, nil
			}
		case "response.completed":
			finalResponse = streamEvent.Response
		case "response.incomplete", "response.failed":
			if streamEvent.Response != nil {
				return nil, responsesError(streamEvent.Response)
			}
			return nil, fmt.Errorf("responses api %s", streamEvent.Type)
		case "error":
			return nil, errors.New(strings.TrimSpace(streamEvent.Message))
		}
	}
	finalModelResponse, err = responsesToModelResponse(finalResponse, content.String())
	if err != nil {
		return nil, err
	}
	if yield != nil {
		yield(finalModelResponse, nil)
	}
	return finalModelResponse, nil
}

// buildResponsesRequest 将 Blades 请求转换为 sub2api Responses 请求体。
func (m *responsesModel) buildResponsesRequest(req *blades.ModelRequest, stream bool) (map[string]any, error) {
	var err error
	var tools []map[string]any
	if req != nil {
		tools, err = toolsToResponsesTools(req.Tools)
		if err != nil {
			return nil, err
		}
	}
	body := map[string]any{
		"model":  m.model,
		"input":  responsesInputItems(req),
		"store":  false,
		"stream": stream,
		"tools": []map[string]any{
			{"type": "web_search"},
		},
		"tool_choice": "auto",
		"include":     []string{"web_search_call.action.sources"},
	}
	if len(tools) > 0 {
		body["tools"] = append(body["tools"].([]map[string]any), tools...)
	}
	if m.config.MaxOutputTokens > 0 {
		body["max_output_tokens"] = m.config.MaxOutputTokens
	}
	if m.config.Temperature > 0 {
		body["temperature"] = m.config.Temperature
	}
	if m.config.TopP > 0 {
		body["top_p"] = m.config.TopP
	}
	if effort := strings.TrimSpace(m.config.ReasoningEffort); effort != "" {
		body["reasoning"] = map[string]any{"effort": effort}
	}
	instructions := "你是一个通用 AI 聊天助手。"
	if req != nil && req.Instruction != nil {
		if text := strings.TrimSpace(messageText(req.Instruction)); text != "" {
			instructions = text
		}
	}
	body["instructions"] = instructions
	mergeExtraFields(body, m.config.ExtraFields)
	return body, nil
}

// responsesInputItems 将 Blades 消息转换为 Responses input。
func responsesInputItems(req *blades.ModelRequest) []map[string]any {
	if req == nil {
		return nil
	}
	items := make([]map[string]any, 0, len(req.Messages))
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
			itemID := strings.TrimSpace(message.ID)
			if itemID == "" {
				itemID = "msg_" + blades.NewMessageID()
			}
			if !strings.HasPrefix(itemID, "msg_") {
				itemID = "msg_" + strings.ReplaceAll(itemID, "-", "")
			}
			items = append(items, map[string]any{
				"type":   "message",
				"id":     itemID,
				"role":   "assistant",
				"status": "completed",
				"content": []map[string]any{
					{"type": "output_text", "text": text},
				},
			})
			continue
		}
		role := "user"
		if message.Role == blades.RoleSystem {
			role = "system"
		}
		items = append(items, map[string]any{
			"type":    "message",
			"role":    role,
			"content": content,
		})
	}
	return items
}

// responsesContentParts 将消息片段转换为 Responses content。
func responsesContentParts(message *blades.Message) []map[string]any {
	if message == nil {
		return nil
	}
	var err error
	parts := make([]map[string]any, 0, len(message.Parts))
	for _, part := range message.Parts {
		switch value := part.(type) {
		case blades.TextPart:
			if text := strings.TrimSpace(value.Text); text != "" {
				parts = append(parts, map[string]any{"type": "input_text", "text": text})
			}
		case blades.FilePart:
			if value.MIMEType.Type() == "image" && strings.TrimSpace(value.URI) != "" {
				parts = append(parts, map[string]any{
					"type":      "input_image",
					"image_url": strings.TrimSpace(value.URI),
					"detail":    "auto",
				})
				continue
			}
			if text := strings.TrimSpace(filePartText(value)); text != "" {
				parts = append(parts, map[string]any{"type": "input_text", "text": text})
			}
		case blades.DataPart:
			if value.MIMEType.Type() == "image" && len(value.Bytes) > 0 {
				parts = append(parts, map[string]any{
					"type":      "input_image",
					"image_url": dataURLFromPart(value),
					"detail":    "auto",
				})
				continue
			}
			if text := strings.TrimSpace(partText(value)); text != "" {
				parts = append(parts, map[string]any{"type": "input_text", "text": text})
			}
		case blades.ToolPart:
			var raw []byte
			raw, err = json.Marshal(value)
			if err != nil {
				continue
			}
			parts = append(parts, map[string]any{"type": "input_text", "text": string(raw)})
		}
	}
	return parts
}

type responsesStreamEvent struct {
	Type     string                `json:"type"`
	Delta    string                `json:"delta"`
	Message  string                `json:"message"`
	Response *responsesAPIResponse `json:"response"`
}

type responsesAPIResponse struct {
	ID                string                     `json:"id"`
	Status            string                     `json:"status"`
	CreatedAt         int64                      `json:"created_at"`
	Output            []responsesOutputItem      `json:"output"`
	Usage             responsesUsage             `json:"usage"`
	Error             responsesAPIError          `json:"error"`
	IncompleteDetails responsesIncompleteDetails `json:"incomplete_details"`
}

type responsesOutputItem struct {
	Type          string                 `json:"type"`
	Content       []responsesContentPart `json:"content"`
	Result        string                 `json:"result"`
	ID            string                 `json:"id"`
	Status        string                 `json:"status"`
	CallID        string                 `json:"call_id"`
	Name          string                 `json:"name"`
	Arguments     string                 `json:"arguments"`
	RevisedPrompt string                 `json:"revised_prompt"`
	OutputFormat  string                 `json:"output_format"`
	Size          string                 `json:"size"`
	Quality       string                 `json:"quality"`
	Background    string                 `json:"background"`
}

type responsesContentPart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type responsesUsage struct {
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
	TotalTokens  int64 `json:"total_tokens"`
}

type responsesAPIError struct {
	Message string `json:"message"`
}

type responsesIncompleteDetails struct {
	Reason string `json:"reason"`
}

// responsesToModelResponse 将 Responses 响应转换成 Blades 响应。
func responsesToModelResponse(response *responsesAPIResponse, fallbackText string) (*blades.ModelResponse, error) {
	if response == nil && strings.TrimSpace(fallbackText) == "" {
		return nil, errors.New("responses api returned empty response")
	}
	var err error
	if response != nil {
		err = responsesError(response)
		if err != nil {
			return nil, err
		}
	}
	message := blades.NewAssistantMessage(blades.StatusCompleted)
	content := strings.TrimSpace(fallbackText)
	if response != nil {
		if text := strings.TrimSpace(responseOutputText(response.Output)); text != "" {
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
	return &blades.ModelResponse{Message: message}, nil
}

// responseOutputText 汇总 Responses output_text。
func responseOutputText(outputs []responsesOutputItem) string {
	var builder strings.Builder
	for _, item := range outputs {
		if item.Type != "message" {
			continue
		}
		for _, content := range item.Content {
			if content.Type == "output_text" && content.Text != "" {
				builder.WriteString(content.Text)
			}
		}
	}
	return builder.String()
}

// responsesError 收敛 Responses 失败信息。
func responsesError(response *responsesAPIResponse) error {
	if response == nil {
		return nil
	}
	if strings.TrimSpace(response.Error.Message) != "" {
		return errors.New(response.Error.Message)
	}
	if strings.TrimSpace(response.IncompleteDetails.Reason) != "" && response.Status == "incomplete" {
		return fmt.Errorf("responses api incomplete: %s", response.IncompleteDetails.Reason)
	}
	if response.Status == "failed" {
		return errors.New("responses api failed")
	}
	return nil
}

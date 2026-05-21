package sub2api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"sort"
	"strings"

	"github.com/go-kratos/blades"
)

// ChatConfig 表示 sub2api Chat Completions 模型配置。
type ChatConfig struct {
	// BaseURL sub2api OpenAI 兼容基础地址，通常以 /v1 结尾。
	BaseURL string
	// APIKey sub2api 分配的 API Key。
	APIKey string
	// Seed 随机种子。
	Seed int64
	// MaxOutputTokens 最大输出 token 数。
	MaxOutputTokens int64
	// FrequencyPenalty 频率惩罚。
	FrequencyPenalty float64
	// PresencePenalty 存在惩罚。
	PresencePenalty float64
	// Temperature 采样温度。
	Temperature float64
	// TopP 核采样参数。
	TopP float64
	// StopSequences 停止序列。
	StopSequences []string
	// ExtraFields 额外透传字段。
	ExtraFields map[string]any
	// ReasoningEffort 推理强度。
	ReasoningEffort string
}

// chatModel 使用 sub2api /chat/completions 实现 Blades 模型提供者。
type chatModel struct {
	model  string
	config ChatConfig
	client *apiClient
}

// NewChat 创建 sub2api Chat Completions 模型提供者。
func NewChat(model string, config ChatConfig) blades.ModelProvider {
	return &chatModel{
		model:  strings.TrimSpace(model),
		config: config,
		client: newAPIClient(config.BaseURL, config.APIKey),
	}
}

// Name 返回模型名称。
func (m *chatModel) Name() string {
	return m.model
}

// Generate 执行非流式 Chat Completions 请求。
func (m *chatModel) Generate(ctx context.Context, req *blades.ModelRequest) (*blades.ModelResponse, error) {
	var err error
	var body map[string]any
	body, err = m.buildChatRequest(req, false)
	if err != nil {
		return nil, err
	}
	var respBody []byte
	respBody, err = m.client.doJSON(ctx, endpointChatCompletions, body)
	if err != nil {
		return nil, err
	}
	var response chatCompletionResponse
	err = json.Unmarshal(respBody, &response)
	if err != nil {
		return nil, err
	}
	return chatCompletionToModelResponse(&response)
}

// NewStreaming 执行流式 Chat Completions 请求。
func (m *chatModel) NewStreaming(ctx context.Context, req *blades.ModelRequest) blades.Generator[*blades.ModelResponse, error] {
	return func(yield func(*blades.ModelResponse, error) bool) {
		var err error
		var body map[string]any
		body, err = m.buildChatRequest(req, true)
		if err != nil {
			yield(nil, err)
			return
		}
		var stream *sseStream
		stream, err = m.client.doSSE(ctx, endpointChatCompletions, body)
		if err != nil {
			yield(nil, err)
			return
		}
		defer func() {
			_ = stream.Close()
		}()

		acc := newChatStreamAccumulator()
		var event *sseEvent
		for {
			event, err = stream.Next()
			if err != nil {
				if errorsIsEOF(err) {
					break
				}
				yield(nil, err)
				return
			}
			if event == nil || event.IsDone() {
				break
			}
			var chunk chatCompletionChunk
			if err = event.DecodeJSON(&chunk); err != nil {
				yield(nil, err)
				return
			}
			delta := acc.addChunk(chunk)
			if delta == nil || len(delta.Message.Parts) == 0 {
				continue
			}
			if !yield(delta, nil) {
				return
			}
		}
		yield(acc.response(), nil)
	}
}

// buildChatRequest 将 Blades 请求转换为 Chat Completions 请求体。
func (m *chatModel) buildChatRequest(req *blades.ModelRequest, stream bool) (map[string]any, error) {
	var err error
	var tools []map[string]any
	if req != nil {
		tools, err = toolsToChatTools(req.Tools)
		if err != nil {
			return nil, err
		}
	}
	var messages []map[string]any
	messages, err = chatMessagesFromRequest(req)
	if err != nil {
		return nil, err
	}
	body := map[string]any{
		"model":    m.model,
		"messages": messages,
	}
	if stream {
		body["stream"] = true
		body["stream_options"] = map[string]any{"include_usage": true}
	}
	if len(tools) > 0 {
		body["tools"] = tools
	}
	if m.config.Seed > 0 {
		body["seed"] = m.config.Seed
	}
	if m.config.MaxOutputTokens > 0 {
		body["max_completion_tokens"] = m.config.MaxOutputTokens
	}
	if m.config.FrequencyPenalty > 0 {
		body["frequency_penalty"] = m.config.FrequencyPenalty
	}
	if m.config.PresencePenalty > 0 {
		body["presence_penalty"] = m.config.PresencePenalty
	}
	if m.config.Temperature > 0 {
		body["temperature"] = m.config.Temperature
	}
	if m.config.TopP > 0 {
		body["top_p"] = m.config.TopP
	}
	if len(m.config.StopSequences) > 0 {
		body["stop"] = append([]string(nil), m.config.StopSequences...)
	}
	if strings.TrimSpace(m.config.ReasoningEffort) != "" {
		body["reasoning_effort"] = strings.TrimSpace(m.config.ReasoningEffort)
	}
	if req != nil && req.OutputSchema != nil {
		var schema any
		schema, err = schemaToAny(req.OutputSchema)
		if err != nil {
			return nil, err
		}
		name := strings.TrimSpace(req.OutputSchema.Title)
		if name == "" {
			name = "structured_outputs"
		}
		jsonSchema := map[string]any{
			"name":   name,
			"schema": schema,
			"strict": true,
		}
		if description := strings.TrimSpace(req.OutputSchema.Description); description != "" {
			jsonSchema["description"] = description
		}
		body["response_format"] = map[string]any{
			"type":        "json_schema",
			"json_schema": jsonSchema,
		}
	}
	mergeExtraFields(body, m.config.ExtraFields)
	return body, nil
}

// chatMessagesFromRequest 构造 Chat Completions messages。
func chatMessagesFromRequest(req *blades.ModelRequest) ([]map[string]any, error) {
	if req == nil {
		return nil, nil
	}
	messages := make([]map[string]any, 0, len(req.Messages)+1)
	if req.Instruction != nil {
		if content := chatTextContent(req.Instruction); len(content) > 0 {
			messages = append(messages, map[string]any{"role": "system", "content": content})
		}
	}
	for _, message := range req.Messages {
		if message == nil {
			continue
		}
		switch message.Role {
		case blades.RoleSystem:
			if content := chatTextContent(message); len(content) > 0 {
				messages = append(messages, map[string]any{"role": "system", "content": content})
			}
		case blades.RoleAssistant:
			messages = append(messages, chatAssistantMessage(message))
		case blades.RoleTool:
			messages = append(messages, chatToolMessages(message)...)
		default:
			content := chatContentParts(message)
			if len(content) == 0 {
				continue
			}
			messages = append(messages, map[string]any{"role": "user", "content": content})
		}
	}
	return messages, nil
}

// chatTextContent 提取系统或助手文本内容。
func chatTextContent(message *blades.Message) string {
	return strings.TrimSpace(messageText(message))
}

// chatAssistantMessage 构造 assistant 消息。
func chatAssistantMessage(message *blades.Message) map[string]any {
	result := map[string]any{
		"role":    "assistant",
		"content": chatTextContent(message),
	}
	toolCalls := make([]map[string]any, 0)
	for _, part := range message.Parts {
		toolPart, ok := part.(blades.ToolPart)
		if !ok || strings.TrimSpace(toolPart.Name) == "" {
			continue
		}
		id := strings.TrimSpace(toolPart.ID)
		if id == "" {
			id = "call_" + blades.NewMessageID()
		}
		args := strings.TrimSpace(toolPart.Request)
		if args == "" {
			args = "{}"
		}
		toolCalls = append(toolCalls, map[string]any{
			"id":   id,
			"type": "function",
			"function": map[string]any{
				"name":      toolPart.Name,
				"arguments": args,
			},
		})
	}
	if len(toolCalls) > 0 {
		result["tool_calls"] = toolCalls
	}
	return result
}

// chatToolMessages 构造工具调用和工具结果消息。
func chatToolMessages(message *blades.Message) []map[string]any {
	messages := make([]map[string]any, 0, len(message.Parts)*2)
	for _, part := range message.Parts {
		toolPart, ok := part.(blades.ToolPart)
		if !ok || strings.TrimSpace(toolPart.Name) == "" {
			continue
		}
		id := strings.TrimSpace(toolPart.ID)
		if id == "" {
			id = "call_" + blades.NewMessageID()
		}
		args := strings.TrimSpace(toolPart.Request)
		if args == "" {
			args = "{}"
		}
		messages = append(messages, map[string]any{
			"role": "assistant",
			"tool_calls": []map[string]any{
				{
					"id":   id,
					"type": "function",
					"function": map[string]any{
						"name":      toolPart.Name,
						"arguments": args,
					},
				},
			},
		})
		if strings.TrimSpace(toolPart.Response) != "" {
			messages = append(messages, map[string]any{
				"role":         "tool",
				"tool_call_id": id,
				"content":      toolPart.Response,
			})
		}
	}
	return messages
}

// chatContentParts 构造多模态用户消息内容。
func chatContentParts(message *blades.Message) []map[string]any {
	if message == nil {
		return nil
	}
	var err error
	parts := make([]map[string]any, 0, len(message.Parts))
	for _, part := range message.Parts {
		switch value := part.(type) {
		case blades.TextPart:
			if text := strings.TrimSpace(value.Text); text != "" {
				parts = append(parts, map[string]any{"type": "text", "text": text})
			}
		case blades.FilePart:
			if value.MIMEType.Type() == "image" && strings.TrimSpace(value.URI) != "" {
				parts = append(parts, map[string]any{
					"type":      "image_url",
					"image_url": map[string]any{"url": strings.TrimSpace(value.URI)},
				})
				continue
			}
			if text := strings.TrimSpace(filePartText(value)); text != "" {
				parts = append(parts, map[string]any{"type": "text", "text": text})
			}
		case blades.DataPart:
			if value.MIMEType.Type() == "image" && len(value.Bytes) > 0 {
				parts = append(parts, map[string]any{
					"type":      "image_url",
					"image_url": map[string]any{"url": dataURLFromPart(value)},
				})
				continue
			}
			if text := strings.TrimSpace(partText(value)); text != "" {
				parts = append(parts, map[string]any{"type": "text", "text": text})
			}
		case blades.ToolPart:
			var raw []byte
			raw, err = json.Marshal(value)
			if err != nil {
				continue
			}
			parts = append(parts, map[string]any{"type": "text", "text": string(raw)})
		}
	}
	return parts
}

type chatCompletionResponse struct {
	ID      string                 `json:"id"`
	Usage   chatUsage              `json:"usage"`
	Choices []chatCompletionChoice `json:"choices"`
}

type chatCompletionChoice struct {
	FinishReason string      `json:"finish_reason"`
	Message      chatMessage `json:"message"`
}

type chatMessage struct {
	Content   string         `json:"content"`
	Audio     chatAudio      `json:"audio"`
	ToolCalls []chatToolCall `json:"tool_calls"`
}

type chatAudio struct {
	Data string `json:"data"`
}

type chatToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function chatToolFunction `json:"function"`
}

type chatToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type chatUsage struct {
	PromptTokens     int64 `json:"prompt_tokens"`
	CompletionTokens int64 `json:"completion_tokens"`
	TotalTokens      int64 `json:"total_tokens"`
}

// chatCompletionToModelResponse 转换非流式响应。
func chatCompletionToModelResponse(response *chatCompletionResponse) (*blades.ModelResponse, error) {
	var err error
	message := blades.NewAssistantMessage(blades.StatusCompleted)
	if response != nil {
		message.TokenUsage = blades.TokenUsage{
			InputTokens:  response.Usage.PromptTokens,
			OutputTokens: response.Usage.CompletionTokens,
			TotalTokens:  response.Usage.TotalTokens,
		}
		if strings.TrimSpace(response.ID) != "" {
			message.Metadata["response_id"] = strings.TrimSpace(response.ID)
		}
	}
	if response == nil {
		return &blades.ModelResponse{Message: message}, nil
	}
	for _, choice := range response.Choices {
		if strings.TrimSpace(choice.Message.Content) != "" {
			message.Parts = append(message.Parts, blades.TextPart{Text: choice.Message.Content})
		}
		if strings.TrimSpace(choice.Message.Audio.Data) != "" {
			var bytes []byte
			bytes, err = base64.StdEncoding.DecodeString(choice.Message.Audio.Data)
			if err != nil {
				return nil, err
			}
			message.Parts = append(message.Parts, blades.DataPart{Bytes: bytes})
		}
		if strings.TrimSpace(choice.FinishReason) != "" {
			message.FinishReason = choice.FinishReason
		}
		for _, call := range choice.Message.ToolCalls {
			message.Role = blades.RoleTool
			message.Parts = append(message.Parts, blades.ToolPart{
				ID:      call.ID,
				Name:    call.Function.Name,
				Request: call.Function.Arguments,
			})
		}
	}
	return &blades.ModelResponse{Message: message}, nil
}

type chatCompletionChunk struct {
	ID      string            `json:"id"`
	Usage   chatUsage         `json:"usage"`
	Choices []chatChunkChoice `json:"choices"`
}

type chatChunkChoice struct {
	FinishReason string    `json:"finish_reason"`
	Delta        chatDelta `json:"delta"`
}

type chatDelta struct {
	Content   string              `json:"content"`
	ToolCalls []chatDeltaToolCall `json:"tool_calls"`
}

type chatDeltaToolCall struct {
	ID       string           `json:"id"`
	Index    int              `json:"index"`
	Type     string           `json:"type"`
	Function chatToolFunction `json:"function"`
}

type chatStreamAccumulator struct {
	id           string
	content      strings.Builder
	usage        chatUsage
	finishReason string
	toolCalls    map[int]chatToolCall
}

// newChatStreamAccumulator 创建 Chat 流式累积器。
func newChatStreamAccumulator() *chatStreamAccumulator {
	return &chatStreamAccumulator{toolCalls: make(map[int]chatToolCall)}
}

// addChunk 累积流式分片并返回当前增量。
func (a *chatStreamAccumulator) addChunk(chunk chatCompletionChunk) *blades.ModelResponse {
	if strings.TrimSpace(chunk.ID) != "" {
		a.id = strings.TrimSpace(chunk.ID)
	}
	if chunk.Usage.TotalTokens > 0 || chunk.Usage.PromptTokens > 0 || chunk.Usage.CompletionTokens > 0 {
		a.usage = chunk.Usage
	}
	message := blades.NewAssistantMessage(blades.StatusIncomplete)
	for _, choice := range chunk.Choices {
		if strings.TrimSpace(choice.FinishReason) != "" {
			a.finishReason = choice.FinishReason
			message.FinishReason = choice.FinishReason
		}
		if choice.Delta.Content != "" {
			a.content.WriteString(choice.Delta.Content)
			message.Parts = append(message.Parts, blades.TextPart{Text: choice.Delta.Content})
		}
		for _, call := range choice.Delta.ToolCalls {
			current := a.toolCalls[call.Index]
			if call.ID != "" {
				current.ID = call.ID
			}
			if call.Function.Name != "" {
				current.Function.Name = call.Function.Name
			}
			if call.Function.Arguments != "" {
				current.Function.Arguments += call.Function.Arguments
			}
			a.toolCalls[call.Index] = current
			message.Role = blades.RoleTool
			message.Parts = append(message.Parts, blades.ToolPart{
				ID:      current.ID,
				Name:    current.Function.Name,
				Request: call.Function.Arguments,
			})
		}
	}
	if len(message.Parts) == 0 && message.FinishReason == "" {
		return nil
	}
	return &blades.ModelResponse{Message: message}
}

// response 返回流式最终响应。
func (a *chatStreamAccumulator) response() *blades.ModelResponse {
	message := blades.NewAssistantMessage(blades.StatusCompleted)
	message.TokenUsage = blades.TokenUsage{
		InputTokens:  a.usage.PromptTokens,
		OutputTokens: a.usage.CompletionTokens,
		TotalTokens:  a.usage.TotalTokens,
	}
	if a.id != "" {
		message.Metadata["response_id"] = a.id
	}
	if a.finishReason != "" {
		message.FinishReason = a.finishReason
	}
	if text := a.content.String(); text != "" {
		message.Parts = append(message.Parts, blades.TextPart{Text: text})
	}
	indices := make([]int, 0, len(a.toolCalls))
	for index := range a.toolCalls {
		indices = append(indices, index)
	}
	sort.Ints(indices)
	for _, index := range indices {
		call := a.toolCalls[index]
		message.Role = blades.RoleTool
		message.Parts = append(message.Parts, blades.ToolPart{
			ID:      call.ID,
			Name:    call.Function.Name,
			Request: call.Function.Arguments,
		})
	}
	return &blades.ModelResponse{Message: message}
}

// errorsIsEOF 判断 SSE 读取是否正常结束。
func errorsIsEOF(err error) bool {
	return errors.Is(err, io.EOF)
}

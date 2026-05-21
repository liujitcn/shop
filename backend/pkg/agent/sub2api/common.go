package sub2api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	stdhttp "net/http"
	"strings"

	"github.com/go-kratos/blades"
	"github.com/go-kratos/blades/tools"
	"github.com/google/jsonschema-go/jsonschema"
	_http "github.com/liujitcn/go-utils/http"
)

const (
	endpointChatCompletions = "chat/completions"
	endpointResponses       = "responses"
	endpointImageGeneration = "images/generations"
)

type sseEvent = _http.SSEEvent
type sseStream = _http.SSEStream

// apiClient 封装 sub2api OpenAI 兼容接口请求能力。
type apiClient struct {
	client *_http.Client
}

// newAPIClient 创建 sub2api HTTP 客户端。
func newAPIClient(baseURL string, apiKey string) *apiClient {
	options := make([]_http.ClientOption, 0, 3)
	if normalized := normalizeBaseURL(baseURL); normalized != "" {
		options = append(options, _http.WithBaseURL(normalized))
	}
	apiKey = strings.TrimSpace(apiKey)
	if apiKey != "" {
		options = append(options, _http.WithDefaultHeader("Authorization", "Bearer "+apiKey))
	}
	return &apiClient{
		client: _http.NewClient(options...),
	}
}

// doJSON 发送 JSON 请求并返回完整响应体。
func (c *apiClient) doJSON(ctx context.Context, target string, body any) ([]byte, error) {
	if c == nil || c.client == nil {
		return nil, errors.New("sub2api client is not configured")
	}
	var resp *_http.Response
	var err error
	resp, err = c.client.Do(
		stdhttp.MethodPost,
		target,
		_http.WithContext(ctx),
		_http.WithJSONBody(body),
	)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < stdhttp.StatusOK || resp.StatusCode >= stdhttp.StatusMultipleChoices {
		return nil, apiErrorFromBody("sub2api request failed", resp.StatusCode, resp.Body)
	}
	return resp.Body, nil
}

// doSSE 发送 SSE 请求并返回事件流。
func (c *apiClient) doSSE(ctx context.Context, target string, body any) (*_http.SSEStream, error) {
	if c == nil || c.client == nil {
		return nil, errors.New("sub2api client is not configured")
	}
	var stream *_http.SSEStream
	var err error
	stream, err = c.client.DoSSE(
		stdhttp.MethodPost,
		target,
		_http.WithContext(ctx),
		_http.WithJSONBody(body),
	)
	if err != nil {
		return nil, err
	}
	if stream.StatusCode < stdhttp.StatusOK || stream.StatusCode >= stdhttp.StatusMultipleChoices {
		defer func() {
			_ = stream.Close()
		}()
		var bodyBytes []byte
		bodyBytes, err = io.ReadAll(io.LimitReader(stream.Body, 2<<20))
		if err != nil {
			return nil, err
		}
		return nil, apiErrorFromBody("sub2api stream request failed", stream.StatusCode, bodyBytes)
	}
	return stream, nil
}

// normalizeBaseURL 规范化基础地址，保证相对路径会拼在 /v1/ 后。
func normalizeBaseURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	return strings.TrimRight(raw, "/") + "/"
}

// apiErrorFromBody 提取 OpenAI 兼容错误响应。
func apiErrorFromBody(prefix string, statusCode int, body []byte) error {
	bodyText := strings.TrimSpace(string(body))
	var payload struct {
		Error struct {
			Code    string `json:"code"`
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
		Message string `json:"message"`
	}
	if len(body) > 0 && json.Unmarshal(body, &payload) == nil {
		if message := strings.TrimSpace(payload.Error.Message); message != "" {
			return fmt.Errorf("%s: status=%d code=%s type=%s message=%s", prefix, statusCode, payload.Error.Code, payload.Error.Type, message)
		}
		if message := strings.TrimSpace(payload.Message); message != "" {
			return fmt.Errorf("%s: status=%d message=%s", prefix, statusCode, message)
		}
	}
	if bodyText == "" {
		return fmt.Errorf("%s: status=%d", prefix, statusCode)
	}
	return fmt.Errorf("%s: status=%d body=%s", prefix, statusCode, bodyText)
}

// cloneExtraFields 复制扩展字段，避免污染调用方配置。
func cloneExtraFields(extraFields map[string]any) map[string]any {
	if len(extraFields) == 0 {
		return nil
	}
	result := make(map[string]any, len(extraFields))
	for key, value := range extraFields {
		result[key] = value
	}
	return result
}

// mergeExtraFields 将扩展字段透传到请求体顶层。
func mergeExtraFields(body map[string]any, extraFields map[string]any) {
	for key, value := range cloneExtraFields(extraFields) {
		if strings.TrimSpace(key) == "" {
			continue
		}
		body[key] = value
	}
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

// schemaToAny 将 JSON Schema 转为普通 map，便于写入请求体。
func schemaToAny(schema *jsonschema.Schema) (any, error) {
	if schema == nil {
		return nil, nil
	}
	var err error
	var raw []byte
	raw, err = json.Marshal(schema)
	if err != nil {
		return nil, err
	}
	var result any
	err = json.Unmarshal(raw, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// toolsToChatTools 将 Blades 工具转换成 Chat Completions 工具。
func toolsToChatTools(toolList []tools.Tool) ([]map[string]any, error) {
	if len(toolList) == 0 {
		return nil, nil
	}
	var err error
	result := make([]map[string]any, 0, len(toolList))
	for _, item := range toolList {
		if item == nil || strings.TrimSpace(item.Name()) == "" {
			continue
		}
		fn := map[string]any{
			"name": item.Name(),
		}
		if description := strings.TrimSpace(item.Description()); description != "" {
			fn["description"] = description
		}
		if item.InputSchema() != nil {
			var parameters any
			parameters, err = schemaToAny(item.InputSchema())
			if err != nil {
				return nil, err
			}
			fn["parameters"] = parameters
		}
		result = append(result, map[string]any{
			"type":     "function",
			"function": fn,
		})
	}
	return result, nil
}

// toolsToResponsesTools 将 Blades 工具转换成 Responses 工具。
func toolsToResponsesTools(toolList []tools.Tool) ([]map[string]any, error) {
	if len(toolList) == 0 {
		return nil, nil
	}
	var err error
	result := make([]map[string]any, 0, len(toolList))
	for _, item := range toolList {
		if item == nil || strings.TrimSpace(item.Name()) == "" {
			continue
		}
		tool := map[string]any{
			"type": "function",
			"name": item.Name(),
		}
		if description := strings.TrimSpace(item.Description()); description != "" {
			tool["description"] = description
		}
		if item.InputSchema() != nil {
			var parameters any
			parameters, err = schemaToAny(item.InputSchema())
			if err != nil {
				return nil, err
			}
			tool["parameters"] = parameters
		}
		result = append(result, tool)
	}
	return result, nil
}

// decodeDataURL 解析 data URL 图片内容。
func decodeDataURL(raw string) ([]byte, blades.MIMEType, bool, error) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(strings.ToLower(raw), "data:") {
		return nil, "", false, nil
	}
	header, data, ok := strings.Cut(raw, ",")
	if !ok {
		return nil, "", false, fmt.Errorf("invalid data url")
	}
	mimeType := strings.TrimPrefix(header, "data:")
	if index := strings.Index(mimeType, ";"); index >= 0 {
		mimeType = mimeType[:index]
	}
	if strings.TrimSpace(mimeType) == "" {
		mimeType = string(blades.MIMEImagePNG)
	}
	bytes, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, "", false, err
	}
	return bytes, blades.MIMEType(mimeType), true, nil
}

// imageMimeType 按输出格式推断图片 MIME 类型。
func imageMimeType(format string) blades.MIMEType {
	format = strings.ToLower(strings.TrimSpace(format))
	switch format {
	case "jpg", "jpeg", "image/jpeg":
		return blades.MIMEImageJPEG
	case "webp", "image/webp":
		return blades.MIMEImageWEBP
	default:
		return blades.MIMEImagePNG
	}
}

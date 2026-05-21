package sub2api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-kratos/blades"
)

// ImageConfig 表示 sub2api 图片生成模型配置。
type ImageConfig struct {
	// BaseURL sub2api OpenAI 兼容基础地址，通常以 /v1 结尾。
	BaseURL string
	// APIKey sub2api 分配的 API Key。
	APIKey string
	// Background 图片背景模式。
	Background string
	// Size 图片尺寸。
	Size string
	// Quality 图片质量。
	Quality string
	// ResponseFormat 图片响应格式，默认 b64_json。
	ResponseFormat string
	// OutputFormat 图片输出格式。
	OutputFormat string
	// Moderation 内容审核强度。
	Moderation string
	// Style 图片风格。
	Style string
	// User 终端用户标识。
	User string
	// N 生成图片数量。
	N int64
	// PartialImages 流式预览图片数量，透传给 sub2api。
	PartialImages int64
	// OutputCompression 输出压缩质量。
	OutputCompression int64
	// ExtraFields 额外透传字段。
	ExtraFields map[string]any
}

// imageModel 使用 sub2api /images/generations 实现 Blades 图片模型提供者。
type imageModel struct {
	model  string
	config ImageConfig
	client *apiClient
}

// NewImage 创建 sub2api 图片生成模型提供者。
func NewImage(model string, config ImageConfig) blades.ModelProvider {
	return &imageModel{
		model:  strings.TrimSpace(model),
		config: config,
		client: newAPIClient(config.BaseURL, config.APIKey),
	}
}

// Name 返回模型名称。
func (m *imageModel) Name() string {
	return m.model
}

// Generate 执行非流式图片生成请求。
func (m *imageModel) Generate(ctx context.Context, req *blades.ModelRequest) (*blades.ModelResponse, error) {
	body := m.buildImageRequest(req)
	var err error
	var respBody []byte
	respBody, err = m.client.doJSON(ctx, endpointImageGeneration, body)
	if err != nil {
		return nil, err
	}
	var response imageGenerationResponse
	err = json.Unmarshal(respBody, &response)
	if err != nil {
		return nil, err
	}
	return m.imageResponseToModelResponse(&response)
}

// NewStreaming 包装 Generate，提供统一的 Blades 流式接口。
func (m *imageModel) NewStreaming(ctx context.Context, req *blades.ModelRequest) blades.Generator[*blades.ModelResponse, error] {
	return func(yield func(*blades.ModelResponse, error) bool) {
		var err error
		var response *blades.ModelResponse
		response, err = m.Generate(ctx, req)
		if err != nil {
			yield(nil, err)
			return
		}
		yield(response, nil)
	}
}

// buildImageRequest 将 Blades 请求转换为 Images API 请求体。
func (m *imageModel) buildImageRequest(req *blades.ModelRequest) map[string]any {
	body := map[string]any{
		"model":  m.model,
		"prompt": imagePromptFromMessages(req),
	}
	responseFormat := strings.TrimSpace(m.config.ResponseFormat)
	if responseFormat == "" {
		responseFormat = "b64_json"
	}
	body["response_format"] = responseFormat
	if m.config.N > 0 {
		body["n"] = m.config.N
	}
	if value := strings.TrimSpace(m.config.Background); value != "" {
		body["background"] = value
	}
	if value := strings.TrimSpace(m.config.Size); value != "" {
		body["size"] = value
	}
	if value := strings.TrimSpace(m.config.Quality); value != "" {
		body["quality"] = value
	}
	if value := strings.TrimSpace(m.config.OutputFormat); value != "" {
		body["output_format"] = value
	}
	if value := strings.TrimSpace(m.config.Moderation); value != "" {
		body["moderation"] = value
	}
	if value := strings.TrimSpace(m.config.Style); value != "" {
		body["style"] = value
	}
	if value := strings.TrimSpace(m.config.User); value != "" {
		body["user"] = value
	}
	if m.config.PartialImages > 0 {
		body["partial_images"] = m.config.PartialImages
	}
	if m.config.OutputCompression > 0 {
		body["output_compression"] = m.config.OutputCompression
	}
	mergeExtraFields(body, m.config.ExtraFields)
	return body
}

// imageResponseToModelResponse 将图片响应转换为 Blades 响应。
func (m *imageModel) imageResponseToModelResponse(response *imageGenerationResponse) (*blades.ModelResponse, error) {
	if response == nil {
		return nil, errors.New("images api returned empty response")
	}
	var err error
	err = response.imageError()
	if err != nil {
		return nil, err
	}

	results := response.imageResults()
	if len(results) == 0 {
		return nil, errors.New("images api result is empty")
	}

	createdAt := response.createdAt()
	status := response.status()
	message := blades.NewAssistantMessage(blades.StatusCompleted)
	if status != "" && status != "completed" {
		message.Status = blades.StatusIncomplete
	}
	message.Metadata["response_id"] = response.responseID()
	message.Metadata["response_status"] = status
	message.Metadata["response_created"] = createdAt
	message.Metadata["created"] = createdAt
	message.Metadata["size"] = firstNonEmpty(response.Size, m.config.Size)
	message.Metadata["quality"] = firstNonEmpty(response.Quality, m.config.Quality)
	message.Metadata["background"] = firstNonEmpty(response.Background, m.config.Background)
	message.Metadata["output_format"] = firstNonEmpty(response.OutputFormat, m.config.OutputFormat)

	for index, result := range results {
		err = appendImageResultPart(message, result, index, firstNonEmpty(response.OutputFormat, m.config.OutputFormat))
		if err != nil {
			return nil, err
		}
	}
	if len(message.Parts) == 0 {
		return nil, errors.New("images api result is empty")
	}
	return &blades.ModelResponse{Message: message}, nil
}

type imageGenerationResponse struct {
	ID                string                     `json:"id"`
	Type              string                     `json:"type"`
	Status            string                     `json:"status"`
	Created           int64                      `json:"created"`
	CreatedAt         int64                      `json:"created_at"`
	Size              string                     `json:"size"`
	Quality           string                     `json:"quality"`
	Background        string                     `json:"background"`
	OutputFormat      string                     `json:"output_format"`
	B64JSON           string                     `json:"b64_json"`
	URL               string                     `json:"url"`
	RevisedPrompt     string                     `json:"revised_prompt"`
	Data              []imageGenerationData      `json:"data"`
	Output            []responsesOutputItem      `json:"output"`
	Response          *responsesAPIResponse      `json:"response"`
	Item              *responsesOutputItem       `json:"item"`
	PartialImageB64   string                     `json:"partial_image_b64"`
	PartialImageIndex int64                      `json:"partial_image_index"`
	Error             responsesAPIError          `json:"error"`
	IncompleteDetails responsesIncompleteDetails `json:"incomplete_details"`
}

type imageGenerationData struct {
	B64JSON       string `json:"b64_json"`
	URL           string `json:"url"`
	RevisedPrompt string `json:"revised_prompt"`
}

type imageResult struct {
	ID            string
	B64JSON       string
	URL           string
	RevisedPrompt string
	OutputFormat  string
	Size          string
	Quality       string
	Background    string
	Status        string
}

// imageError 收敛图片响应中的错误信息。
func (r *imageGenerationResponse) imageError() error {
	if r == nil {
		return nil
	}
	if strings.TrimSpace(r.Error.Message) != "" {
		return errors.New(r.Error.Message)
	}
	if strings.TrimSpace(r.IncompleteDetails.Reason) != "" && r.Status == "incomplete" {
		return fmt.Errorf("images api incomplete: %s", r.IncompleteDetails.Reason)
	}
	if r.Response != nil {
		return responsesError(r.Response)
	}
	if r.Status == "failed" {
		return errors.New("images api failed")
	}
	return nil
}

// imageResults 提取 Images API 和 Responses image_generation 兼容结果。
func (r *imageGenerationResponse) imageResults() []imageResult {
	if r == nil {
		return nil
	}
	results := make([]imageResult, 0, len(r.Data)+len(r.Output)+1)
	for _, item := range r.Data {
		results = mergeImageResult(results, imageResult{
			B64JSON:       item.B64JSON,
			URL:           item.URL,
			RevisedPrompt: item.RevisedPrompt,
		})
	}
	if strings.TrimSpace(r.B64JSON) != "" || strings.TrimSpace(r.URL) != "" || strings.TrimSpace(r.PartialImageB64) != "" {
		results = mergeImageResult(results, imageResult{
			ID:            r.ID,
			B64JSON:       firstNonEmpty(r.B64JSON, r.PartialImageB64),
			URL:           r.URL,
			RevisedPrompt: r.RevisedPrompt,
			OutputFormat:  r.OutputFormat,
			Size:          r.Size,
			Quality:       r.Quality,
			Background:    r.Background,
			Status:        r.Status,
		})
	}
	if r.Item != nil {
		results = mergeImageResult(results, imageResultFromResponsesOutput(*r.Item))
	}
	for _, output := range r.Output {
		results = mergeImageResult(results, imageResultFromResponsesOutput(output))
	}
	if r.Response != nil {
		for _, output := range r.Response.Output {
			results = mergeImageResult(results, imageResultFromResponsesOutput(output))
		}
	}
	return results
}

// createdAt 返回响应创建时间，没有上游时间时使用当前时间。
func (r *imageGenerationResponse) createdAt() int64 {
	if r == nil {
		return time.Now().Unix()
	}
	if r.Response != nil && r.Response.CreatedAt > 0 {
		return r.Response.CreatedAt
	}
	if r.CreatedAt > 0 {
		return r.CreatedAt
	}
	if r.Created > 0 {
		return r.Created
	}
	return time.Now().Unix()
}

// status 返回图片生成状态。
func (r *imageGenerationResponse) status() string {
	if r == nil {
		return "completed"
	}
	if r.Response != nil && strings.TrimSpace(r.Response.Status) != "" {
		return strings.TrimSpace(r.Response.Status)
	}
	if strings.TrimSpace(r.Status) != "" {
		return strings.TrimSpace(r.Status)
	}
	return "completed"
}

// responseID 返回 Responses 或 Images 响应编号。
func (r *imageGenerationResponse) responseID() string {
	if r == nil {
		return ""
	}
	if r.Response != nil && strings.TrimSpace(r.Response.ID) != "" {
		return strings.TrimSpace(r.Response.ID)
	}
	return strings.TrimSpace(r.ID)
}

// imageResultFromResponsesOutput 从 Responses output item 提取图片结果。
func imageResultFromResponsesOutput(output responsesOutputItem) imageResult {
	if output.Type != "image_generation_call" {
		return imageResult{}
	}
	return imageResult{
		ID:            output.ID,
		B64JSON:       output.Result,
		RevisedPrompt: output.RevisedPrompt,
		OutputFormat:  output.OutputFormat,
		Size:          output.Size,
		Quality:       output.Quality,
		Background:    output.Background,
		Status:        output.Status,
	}
}

// mergeImageResult 合并图片结果，避免 partial 和 completed 重复。
func mergeImageResult(results []imageResult, next imageResult) []imageResult {
	if strings.TrimSpace(next.B64JSON) == "" && strings.TrimSpace(next.URL) == "" {
		return results
	}
	for index, current := range results {
		if strings.TrimSpace(next.ID) != "" && strings.TrimSpace(current.ID) == strings.TrimSpace(next.ID) {
			results[index] = mergeSingleImageResult(current, next)
			return results
		}
	}
	if len(results) == 1 && results[0].Status == "generating" {
		results[0] = mergeSingleImageResult(results[0], next)
		return results
	}
	return append(results, next)
}

// mergeSingleImageResult 合并单张图片的元数据。
func mergeSingleImageResult(current imageResult, next imageResult) imageResult {
	merged := current
	if strings.TrimSpace(next.ID) != "" {
		merged.ID = next.ID
	}
	if strings.TrimSpace(next.B64JSON) != "" {
		merged.B64JSON = next.B64JSON
	}
	if strings.TrimSpace(next.URL) != "" {
		merged.URL = next.URL
	}
	if strings.TrimSpace(next.RevisedPrompt) != "" {
		merged.RevisedPrompt = next.RevisedPrompt
	}
	if strings.TrimSpace(next.OutputFormat) != "" {
		merged.OutputFormat = next.OutputFormat
	}
	if strings.TrimSpace(next.Size) != "" {
		merged.Size = next.Size
	}
	if strings.TrimSpace(next.Quality) != "" {
		merged.Quality = next.Quality
	}
	if strings.TrimSpace(next.Background) != "" {
		merged.Background = next.Background
	}
	if strings.TrimSpace(next.Status) != "" {
		merged.Status = next.Status
	}
	return merged
}

// appendImageResultPart 将单张图片写入 Blades 消息片段。
func appendImageResultPart(message *blades.Message, result imageResult, index int, fallbackFormat string) error {
	if message == nil {
		return nil
	}
	var err error
	name := strings.TrimSpace(result.ID)
	if name == "" {
		name = fmt.Sprintf("image-%d", index+1)
	}
	outputFormat := firstNonEmpty(result.OutputFormat, fallbackFormat)
	mimeType := imageMimeType(outputFormat)
	if strings.TrimSpace(result.B64JSON) != "" {
		var bytes []byte
		var decodedMimeType blades.MIMEType
		bytes, decodedMimeType, err = decodeImageBase64(result.B64JSON, mimeType)
		if err != nil {
			return fmt.Errorf("sub2api/image: decode response: %w", err)
		}
		message.Parts = append(message.Parts, blades.DataPart{
			Name:     name,
			Bytes:    bytes,
			MIMEType: decodedMimeType,
		})
	}
	if strings.TrimSpace(result.URL) != "" {
		var bytes []byte
		var decodedMimeType blades.MIMEType
		var ok bool
		bytes, decodedMimeType, ok, err = decodeDataURL(result.URL)
		if err != nil {
			return fmt.Errorf("sub2api/image: decode response url: %w", err)
		}
		if ok {
			message.Parts = append(message.Parts, blades.DataPart{
				Name:     name,
				Bytes:    bytes,
				MIMEType: decodedMimeType,
			})
		} else {
			message.Parts = append(message.Parts, blades.FilePart{
				Name:     name,
				URI:      strings.TrimSpace(result.URL),
				MIMEType: mimeType,
			})
		}
	}
	if strings.TrimSpace(result.RevisedPrompt) != "" {
		message.Metadata[fmt.Sprintf("%s_revised_prompt_%d", name, index+1)] = strings.TrimSpace(result.RevisedPrompt)
	}
	if strings.TrimSpace(result.Size) != "" {
		message.Metadata[fmt.Sprintf("%s_size_%d", name, index+1)] = strings.TrimSpace(result.Size)
	}
	if strings.TrimSpace(result.Quality) != "" {
		message.Metadata[fmt.Sprintf("%s_quality_%d", name, index+1)] = strings.TrimSpace(result.Quality)
	}
	if strings.TrimSpace(result.Background) != "" {
		message.Metadata[fmt.Sprintf("%s_background_%d", name, index+1)] = strings.TrimSpace(result.Background)
	}
	if strings.TrimSpace(result.OutputFormat) != "" {
		message.Metadata[fmt.Sprintf("%s_output_format_%d", name, index+1)] = strings.TrimSpace(result.OutputFormat)
	}
	return nil
}

// decodeImageBase64 解码普通 base64 或 data URL 图片。
func decodeImageBase64(raw string, fallbackMimeType blades.MIMEType) ([]byte, blades.MIMEType, error) {
	var err error
	var bytes []byte
	var mimeType blades.MIMEType
	var ok bool
	bytes, mimeType, ok, err = decodeDataURL(raw)
	if ok || err != nil {
		return bytes, mimeType, err
	}
	normalized := strings.TrimSpace(raw)
	normalized = strings.TrimRight(normalized, "=")
	normalized += strings.Repeat("=", (4-len(normalized)%4)%4)
	bytes, err = base64.StdEncoding.DecodeString(normalized)
	if err != nil {
		return nil, "", err
	}
	if fallbackMimeType == "" {
		fallbackMimeType = blades.MIMEImagePNG
	}
	return bytes, fallbackMimeType, nil
}

// imagePromptFromMessages 合并图片生成提示词。
func imagePromptFromMessages(req *blades.ModelRequest) string {
	if req == nil {
		return ""
	}
	sections := make([]string, 0, len(req.Messages)+1)
	if req.Instruction != nil {
		if text := strings.TrimSpace(req.Instruction.Text()); text != "" {
			sections = append(sections, text)
		}
	}
	for _, message := range req.Messages {
		if message == nil {
			continue
		}
		if text := strings.TrimSpace(message.Text()); text != "" {
			sections = append(sections, text)
		}
	}
	return strings.Join(sections, "\n")
}

// firstNonEmpty 返回第一个非空字符串。
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

package biz

import (
	"context"
	"encoding/base64"
	"fmt"
	"path"
	"strings"
	"time"

	basev1 "shop/api/gen/go/base/v1"
	"shop/pkg/agent/provider"
	_const "shop/pkg/const"
	"shop/pkg/errorsx"

	"github.com/go-kratos/blades"
	"github.com/google/uuid"
	"github.com/liujitcn/kratos-kit/oss"
)

const (
	aiImageDefaultSize         = "1024x1024"
	aiImageDefaultQuality      = "auto"
	aiImageDefaultOutputFormat = "png"
	aiImageDefaultCount        = int64(1)
	aiImageMaxCount            = int64(4)
)

// AiImageCase 管理 AI 图片生成能力。
type AiImageCase struct {
	imageClient *provider.ImageClient
	chatClient  *provider.ChatClient
	oss         oss.OSS
}

// NewAiImageCase 创建 AI 图片业务实例。
func NewAiImageCase(imageClient *provider.ImageClient, chatClient *provider.ChatClient, oss oss.OSS) *AiImageCase {
	return &AiImageCase{
		imageClient: imageClient,
		chatClient:  chatClient,
		oss:         oss,
	}
}

// GenerateAiImage 生成 AI 图片并按需保存结果。
func (c *AiImageCase) GenerateAiImage(ctx context.Context, req *basev1.GenerateAiImageRequest) (*basev1.GenerateAiImageResponse, error) {
	originalPrompt := strings.TrimSpace(req.GetPrompt())
	if originalPrompt == "" {
		return nil, errorsx.InvalidArgument("图片提示词不能为空")
	}
	if c.imageClient == nil || !c.imageClient.Enabled() {
		return nil, errorsx.Internal("AI图片客户端未配置")
	}

	prompt := originalPrompt
	if req.GetPolishPrompt() {
		polishResponse, err := c.PolishAiImagePrompt(ctx, &basev1.PolishAiImagePromptRequest{
			Prompt: originalPrompt,
			Scene:  "商城后台图片生成",
		})
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(polishResponse.GetPrompt()) != "" {
			prompt = strings.TrimSpace(polishResponse.GetPrompt())
		}
	}

	model := strings.TrimSpace(req.GetModel())
	if model == "" {
		model = c.imageClient.DefaultModel()
	}
	requestID := newAiImageRequestID()
	outputFormat := normalizeAiImageOutputFormat(req.GetOutputFormat())
	providerOutputFormat := outputFormat
	if !isGPTImageModel(model) {
		providerOutputFormat = ""
	}
	providerInstance := c.imageClient.Provider(provider.ImageGenerateOptions{
		Model:          model,
		Background:     normalizeAiImageBackground(req.GetBackground(), model),
		Size:           normalizeAiImageSize(req.GetSize(), model),
		Quality:        normalizeAiImageQuality(req.GetQuality(), model),
		ResponseFormat: normalizeAiImageResponseFormat(req.GetResponseFormat(), model),
		OutputFormat:   providerOutputFormat,
		Style:          normalizeAiImageStyle(req.GetStyle(), model),
		N:              normalizeAiImageCount(req.GetN(), model),
	})
	if providerInstance == nil {
		return nil, errorsx.Internal("AI图片客户端未配置")
	}

	response, err := providerInstance.Generate(ctx, &blades.ModelRequest{
		Messages: []*blades.Message{
			blades.UserMessage(prompt),
		},
	})
	if err != nil {
		return nil, errorsx.Internal("AI图片生成失败").WithCause(err)
	}

	var images []*basev1.AiImage
	images, err = c.toAiImages(response, outputFormat, req.GetSaveOutput(), requestID)
	if err != nil {
		return nil, err
	}
	if len(images) == 0 {
		return nil, errorsx.Internal("AI图片生成结果为空")
	}

	return &basev1.GenerateAiImageResponse{
		Images:         images,
		Model:          model,
		Prompt:         prompt,
		Created:        readInt64Metadata(response, "created"),
		RequestId:      requestID,
		OriginalPrompt: originalPrompt,
	}, nil
}

// PolishAiImagePrompt 润色图片生成提示词。
func (c *AiImageCase) PolishAiImagePrompt(ctx context.Context, req *basev1.PolishAiImagePromptRequest) (*basev1.PolishAiImagePromptResponse, error) {
	originalPrompt := strings.TrimSpace(req.GetPrompt())
	if originalPrompt == "" {
		return nil, errorsx.InvalidArgument("图片提示词不能为空")
	}
	if c.chatClient == nil || !c.chatClient.Enabled() {
		return nil, errorsx.Internal("AI润色客户端未配置")
	}

	scene := strings.TrimSpace(req.GetScene())
	if scene == "" {
		scene = "商城商品图、活动素材或内容配图"
	}
	response, err := c.chatClient.Provider().Generate(ctx, &blades.ModelRequest{
		Instruction: blades.SystemMessage("你是专业的 AI 图片提示词策划。只输出一条中文图片生成提示词，不要解释，不要编号，不要使用 Markdown。"),
		Messages: []*blades.Message{
			blades.UserMessage(fmt.Sprintf(
				"请把下面的图片需求润色成适合文生图模型的中文提示词，控制在 80 到 160 个汉字，包含主体、场景、风格、光影、构图、细节和用途。使用场景：%s。原始需求：%s",
				scene,
				originalPrompt,
			)),
		},
	})
	if err != nil {
		return nil, errorsx.Internal("AI图片提示词润色失败").WithCause(err)
	}
	prompt := normalizeAiImagePromptText(messageText(response))
	if prompt == "" {
		return nil, errorsx.Internal("AI图片提示词润色结果为空")
	}

	return &basev1.PolishAiImagePromptResponse{
		Prompt:         prompt,
		OriginalPrompt: originalPrompt,
		Model:          c.chatClient.Model(),
	}, nil
}

// toAiImages 将 Blades 模型响应转换成接口图片结果。
func (c *AiImageCase) toAiImages(response *blades.ModelResponse, outputFormat string, saveOutput bool, requestID string) ([]*basev1.AiImage, error) {
	if response == nil || response.Message == nil {
		return nil, nil
	}
	images := make([]*basev1.AiImage, 0, len(response.Message.Parts))
	imageIndex := 0
	for _, part := range response.Message.Parts {
		switch value := part.(type) {
		case blades.DataPart:
			imageIndex++
			image, err := c.dataPartToAiImage(value, outputFormat, saveOutput, requestID)
			if err != nil {
				return nil, err
			}
			image.RevisedPrompt = readStringMetadata(response, fmt.Sprintf("image-%d_revised_prompt_%d", imageIndex, imageIndex))
			image.RequestId = requestID
			images = append(images, image)
		case blades.FilePart:
			imageIndex++
			images = append(images, &basev1.AiImage{
				Name:          value.Name,
				Url:           value.URI,
				MimeType:      string(value.MIMEType),
				RevisedPrompt: readStringMetadata(response, fmt.Sprintf("image-%d_revised_prompt_%d", imageIndex, imageIndex)),
				Saved:         false,
				RequestId:     requestID,
			})
		}
	}
	return images, nil
}

// dataPartToAiImage 将二进制图片结果转换为可展示图片。
func (c *AiImageCase) dataPartToAiImage(part blades.DataPart, outputFormat string, saveOutput bool, requestID string) (*basev1.AiImage, error) {
	mimeType := strings.TrimSpace(string(part.MIMEType))
	if mimeType == "" {
		mimeType = aiImageMimeType(outputFormat)
	}
	name := strings.TrimSpace(part.Name)
	if name == "" {
		name = fmt.Sprintf("image.%s", aiImageExtension(mimeType, outputFormat))
	}
	if path.Ext(name) == "" {
		name = fmt.Sprintf("%s.%s", name, aiImageExtension(mimeType, outputFormat))
	}

	image := &basev1.AiImage{
		Name:     name,
		MimeType: mimeType,
		Size:     int64(len(part.Bytes)),
	}
	if saveOutput && c.oss != nil {
		filePath := fmt.Sprintf("/%s/ai/images/%s/%s", _const.BASE_PATH, time.Now().Format("2006/01/02"), requestID)
		name = withAiImageRequestFileName(name)
		image.Name = name
		url, err := c.oss.UploadByByte(name, filePath, part.Bytes)
		if err != nil {
			return nil, errorsx.Internal("保存AI图片失败").WithCause(err)
		}
		image.Url = url
		image.Saved = true
		image.StoragePath = filePath
		return image, nil
	}

	image.Url = fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(part.Bytes))
	return image, nil
}

// messageText 提取模型回复文本。
func messageText(response *blades.ModelResponse) string {
	if response == nil || response.Message == nil {
		return ""
	}
	parts := make([]string, 0, len(response.Message.Parts))
	for _, part := range response.Message.Parts {
		if textPart, ok := part.(blades.TextPart); ok {
			text := strings.TrimSpace(textPart.Text)
			if text != "" {
				parts = append(parts, text)
			}
		}
	}
	return strings.Join(parts, "\n")
}

// normalizeAiImagePromptText 清理提示词润色结果中的多余格式。
func normalizeAiImagePromptText(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "` \n\r\t")
	value = strings.TrimPrefix(value, "提示词：")
	value = strings.TrimPrefix(value, "提示词:")
	lines := strings.Split(value, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "-")
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return strings.TrimSpace(value)
}

// newAiImageRequestID 生成图片批次编号。
func newAiImageRequestID() string {
	return strings.ReplaceAll(uuid.NewString(), "-", "")
}

// withAiImageRequestFileName 为保存文件名补充时间戳，降低同名覆盖概率。
func withAiImageRequestFileName(name string) string {
	ext := path.Ext(name)
	baseName := strings.TrimSuffix(name, ext)
	if strings.TrimSpace(baseName) == "" {
		baseName = "image"
	}
	if ext == "" {
		ext = ".png"
	}
	return fmt.Sprintf("%s-%d%s", baseName, time.Now().UnixNano(), ext)
}

// normalizeAiImageBackground 标准化图片背景模式。
func normalizeAiImageBackground(background string, model string) string {
	if !isGPTImageModel(model) {
		return ""
	}
	background = strings.TrimSpace(background)
	if background == "" {
		return "auto"
	}
	return background
}

// normalizeAiImageSize 按模型标准化图片尺寸。
func normalizeAiImageSize(size string, model string) string {
	size = strings.TrimSpace(size)
	normalizedModel := strings.ToLower(strings.TrimSpace(model))
	if size == "" {
		return aiImageDefaultSize
	}
	switch normalizedModel {
	case "dall-e-2":
		switch size {
		case "256x256", "512x512", "1024x1024":
			return size
		default:
			return aiImageDefaultSize
		}
	case "dall-e-3":
		switch size {
		case "1024x1024", "1792x1024", "1024x1792":
			return size
		case "1536x1024":
			return "1792x1024"
		case "1024x1536":
			return "1024x1792"
		default:
			return aiImageDefaultSize
		}
	}
	return size
}

// normalizeAiImageQuality 按模型标准化图片质量。
func normalizeAiImageQuality(quality string, model string) string {
	quality = strings.TrimSpace(quality)
	normalizedModel := strings.ToLower(strings.TrimSpace(model))
	if quality == "" {
		return aiImageDefaultQuality
	}
	switch normalizedModel {
	case "dall-e-2":
		return "standard"
	case "dall-e-3":
		if quality == "hd" || quality == "standard" {
			return quality
		}
		return "standard"
	}
	return quality
}

// normalizeAiImageOutputFormat 标准化图片输出格式。
func normalizeAiImageOutputFormat(format string) string {
	format = strings.ToLower(strings.TrimSpace(format))
	switch format {
	case "jpeg", "jpg":
		return "jpeg"
	case "webp":
		return "webp"
	default:
		return aiImageDefaultOutputFormat
	}
}

// normalizeAiImageResponseFormat 标准化图片响应格式。
func normalizeAiImageResponseFormat(format string, model string) string {
	format = strings.TrimSpace(format)
	if isGPTImageModel(model) {
		return ""
	}
	if format == "" {
		return "b64_json"
	}
	return format
}

// normalizeAiImageStyle 按模型标准化图片风格。
func normalizeAiImageStyle(style string, model string) string {
	if !strings.EqualFold(strings.TrimSpace(model), "dall-e-3") {
		return ""
	}
	return strings.TrimSpace(style)
}

// normalizeAiImageCount 按模型标准化图片生成数量。
func normalizeAiImageCount(count int64, model string) int64 {
	if strings.EqualFold(strings.TrimSpace(model), "dall-e-3") {
		return 1
	}
	if count <= 0 {
		return aiImageDefaultCount
	}
	if count > aiImageMaxCount {
		return aiImageMaxCount
	}
	return count
}

// isGPTImageModel 判断是否为 OpenAI gpt-image 系列模型。
func isGPTImageModel(model string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(model)), "gpt-image-")
}

// aiImageMimeType 按输出格式推断 MIME 类型。
func aiImageMimeType(format string) string {
	switch normalizeAiImageOutputFormat(format) {
	case "jpeg":
		return string(blades.MIMEImageJPEG)
	case "webp":
		return string(blades.MIMEImageWEBP)
	default:
		return string(blades.MIMEImagePNG)
	}
}

// aiImageExtension 按 MIME 类型推断文件扩展名。
func aiImageExtension(mimeType string, outputFormat string) string {
	switch strings.ToLower(strings.TrimSpace(mimeType)) {
	case string(blades.MIMEImageJPEG):
		return "jpg"
	case string(blades.MIMEImageWEBP):
		return "webp"
	case string(blades.MIMEImagePNG):
		return "png"
	default:
		return normalizeAiImageOutputFormat(outputFormat)
	}
}

// readStringMetadata 从模型响应元数据中读取字符串。
func readStringMetadata(response *blades.ModelResponse, key string) string {
	if response == nil || response.Message == nil || response.Message.Metadata == nil {
		return ""
	}
	value, ok := response.Message.Metadata[key]
	if !ok {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

// readInt64Metadata 从模型响应元数据中读取整数。
func readInt64Metadata(response *blades.ModelResponse, key string) int64 {
	if response == nil || response.Message == nil || response.Message.Metadata == nil {
		return 0
	}
	value := response.Message.Metadata[key]
	switch typed := value.(type) {
	case int64:
		return typed
	case int:
		return int64(typed)
	case float64:
		return int64(typed)
	default:
		return 0
	}
}

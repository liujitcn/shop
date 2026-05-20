package biz

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	basev1 "shop/api/gen/go/base/v1"
	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/agent/assistant"
	"shop/pkg/agent/provider"
	"shop/pkg/biz"
	_const "shop/pkg/const"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"shop/pkg/queue"

	"github.com/go-kratos/blades"
	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-kit/oss"
	queueData "github.com/liujitcn/kratos-kit/queue/data"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gen/field"
	"gorm.io/gorm"
)

const (
	aiImageDefaultSize         = "1024x1024"
	aiImageDefaultQuality      = "auto"
	aiImageDefaultOutputFormat = "png"
	aiImageDefaultCount        = int64(1)
	aiImageMaxCount            = int64(4)
	aiImagePollInterval        = 10 * time.Second
	aiImageMaxWait             = 30 * time.Minute
)

type aiImageParamsSnapshot struct {
	Prompt              string `json:"prompt"`
	Model               string `json:"model"`
	Size                string `json:"size"`
	Quality             string `json:"quality"`
	Style               string `json:"style"`
	Background          string `json:"background"`
	OutputFormat        string `json:"output_format"`
	ResponseFormat      string `json:"response_format"`
	N                   int64  `json:"n"`
	SaveOutput          bool   `json:"save_output"`
	PolishPrompt        bool   `json:"polish_prompt"`
	ResponseID          string `json:"response_id,omitempty"`
	ResponseStatus      string `json:"response_status,omitempty"`
	ResponseCreated     int64  `json:"response_created,omitempty"`
	ResponseSubmitted   bool   `json:"response_submitted,omitempty"`
	ResponsePolledAt    int64  `json:"response_polled_at,omitempty"`
	ResponseSubmittedAt int64  `json:"response_submitted_at,omitempty"`
}

// AiImageCase 管理 AI 图片生成能力。
type AiImageCase struct {
	*biz.BaseCase
	imageClient *provider.ImageClient
	chatClient  *provider.ChatClient
	oss         oss.OSS
	aiImageRepo *data.AiImageRepository
	mapper      *mapper.CopierMapper[basev1.AiImage, models.AiImage]
}

// NewAiImageCase 创建 AI 图片业务实例。
func NewAiImageCase(baseCase *biz.BaseCase, imageClient *provider.ImageClient, chatClient *provider.ChatClient, oss oss.OSS, aiImageRepo *data.AiImageRepository) *AiImageCase {
	imageMapper := mapper.NewCopierMapper[basev1.AiImage, models.AiImage]()
	imageMapper.AppendConverter(mapper.NewGenericTypeConverterPair(time.Time{}, (*timestamppb.Timestamp)(nil), timeToTimestamp, timestampToTime)[0])

	c := &AiImageCase{
		BaseCase:    baseCase,
		imageClient: imageClient,
		chatClient:  chatClient,
		oss:         oss,
		aiImageRepo: aiImageRepo,
		mapper:      imageMapper,
	}
	c.RegisterQueueConsumer(_const.AI_IMAGE_GENERATE, c.consumeAiImageGenerate)
	return c
}

// PageAiImages 分页查询当前用户的 AI 图片。
func (c *AiImageCase) PageAiImages(ctx context.Context, req *basev1.PageAiImagesRequest) (*basev1.PageAiImagesResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	query := c.aiImageRepo.Query(ctx).AiImage
	opts := make([]repository.QueryOption, 0, 6)
	opts = append(opts, repository.Where(query.CreatedBy.Eq(authInfo.UserId)))
	opts = append(opts, repository.Where(query.Terminal.Eq(assistant.NormalizeTerminal(req.GetTerminal()))))
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	keyword := strings.TrimSpace(req.GetKeyword())
	if keyword != "" {
		opts = append(opts, repository.Where(field.Or(
			query.Prompt.Like("%"+keyword+"%"),
			query.OriginalPrompt.Like("%"+keyword+"%"),
		)))
	}
	opts = append(opts, repository.Order(query.CreatedAt.Desc(), query.ID.Desc()))

	var (
		list  []*models.AiImage
		total int64
	)
	list, total, err = c.aiImageRepo.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	images := make([]*basev1.AiImage, 0, len(list))
	for _, item := range list {
		images = append(images, c.toImageDTO(item))
	}
	return &basev1.PageAiImagesResponse{Images: images, Total: int32(total)}, nil
}

// GetAiImage 查询当前用户的 AI 图片详情。
func (c *AiImageCase) GetAiImage(ctx context.Context, req *basev1.GetAiImageRequest) (*basev1.AiImage, error) {
	image, err := c.findCurrentUserImageByRawID(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return c.toImageDTO(image), nil
}

// DeleteAiImage 删除当前用户创建的 AI 图片。
func (c *AiImageCase) DeleteAiImage(ctx context.Context, req *basev1.DeleteAiImageRequest) (*emptypb.Empty, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	ids := _string.ConvertStringToInt64Array(req.GetIds())
	if len(ids) == 0 {
		return nil, errorsx.InvalidArgument("图片编号不能为空")
	}
	query := c.aiImageRepo.Query(ctx).AiImage
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.ID.In(ids...)))
	opts = append(opts, repository.Where(query.CreatedBy.Eq(authInfo.UserId)))
	if err = c.aiImageRepo.Delete(ctx, opts...); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// CreateAiImage 创建 AI 图片生成记录并异步投递队列。
func (c *AiImageCase) CreateAiImage(ctx context.Context, req *basev1.CreateAiImageRequest) (*emptypb.Empty, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	var image *models.AiImage
	image, err = c.buildAiImage(authInfo.UserId, req, false)
	if err != nil {
		return nil, err
	}
	query := c.aiImageRepo.Query(ctx).AiImage
	if err = query.WithContext(ctx).Omit(query.StartedAt, query.FinishedAt).Create(image); err != nil {
		return nil, err
	}
	queue.DispatchAiImageGenerate(image.ID)
	return &emptypb.Empty{}, nil
}

// RetryAiImage 重试失败或超时的 AI 图片生成记录。
func (c *AiImageCase) RetryAiImage(ctx context.Context, req *basev1.RetryAiImageRequest) (*emptypb.Empty, error) {
	image, err := c.findCurrentUserImageByRawID(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if image.Status != _const.AI_IMAGE_STATUS_FAILED && image.Status != _const.AI_IMAGE_STATUS_TIMEOUT {
		return nil, errorsx.StateConflict(
			"当前生成状态不允许重试",
			"ai_image",
			strconv.FormatInt(int64(image.Status), 10),
			fmt.Sprintf("%d|%d", _const.AI_IMAGE_STATUS_FAILED, _const.AI_IMAGE_STATUS_TIMEOUT),
		)
	}
	query := c.aiImageRepo.Query(ctx).AiImage
	_, err = query.WithContext(ctx).
		Where(query.ID.Eq(image.ID)).
		UpdateSimple(
			query.Status.Value(_const.AI_IMAGE_STATUS_PENDING),
			query.ErrorMessage.Value(""),
			query.ParamsJSON.Value(resetAiImageResponseTask(image.ParamsJSON)),
			query.StartedAt.Null(),
			query.FinishedAt.Null(),
		)
	if err != nil {
		return nil, err
	}
	queue.DispatchAiImageGenerate(image.ID)
	return &emptypb.Empty{}, nil
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

// consumeAiImageGenerate 消费 AI 图片生成队列。
func (c *AiImageCase) consumeAiImageGenerate(message queueData.Message) error {
	imageID, err := queue.DecodeQueueData[int64](message)
	if err != nil {
		return err
	}
	if imageID == nil || *imageID <= 0 {
		return nil
	}
	return c.generateAiImage(context.TODO(), *imageID)
}

// generateAiImage 执行单个 AI 图片生成轮询流程。
func (c *AiImageCase) generateAiImage(ctx context.Context, imageID int64) error {
	image, err := c.aiImageRepo.FindByID(ctx, imageID)
	if err != nil {
		return err
	}
	if image.Status != _const.AI_IMAGE_STATUS_PENDING && image.Status != _const.AI_IMAGE_STATUS_RUNNING {
		return nil
	}
	if image.Status == _const.AI_IMAGE_STATUS_PENDING {
		startedAt := time.Now()
		err = c.updateImageRunning(ctx, image.ID, startedAt)
		if err != nil {
			return err
		}
		image.Status = _const.AI_IMAGE_STATUS_RUNNING
		image.StartedAt = startedAt
	}

	generateCtx, cancel := context.WithTimeout(context.Background(), aiImagePollInterval)
	defer cancel()

	var response *basev1.AiImage
	var pending bool
	response, pending, err = c.generateAiImageResult(generateCtx, image)
	if err != nil {
		return c.markImageFailed(ctx, image.ID, err)
	}
	if pending {
		if image.StartedAt.IsZero() || time.Since(image.StartedAt) <= aiImageMaxWait {
			c.dispatchAiImagePoll(image.ID)
			return nil
		}
		return c.markImageFailed(ctx, image.ID, context.DeadlineExceeded)
	}
	return c.markImageSuccess(ctx, image.ID, response)
}

// buildAiImage 基于创建请求构建图片模型。
func (c *AiImageCase) buildAiImage(userID int64, req *basev1.CreateAiImageRequest, retry bool) (*models.AiImage, error) {
	originalPrompt := strings.TrimSpace(req.GetPrompt())
	if originalPrompt == "" {
		return nil, errorsx.InvalidArgument("图片提示词不能为空")
	}
	model := strings.TrimSpace(req.GetModel())
	if model == "" && c.imageClient != nil {
		model = c.imageClient.DefaultModel()
	}
	if model == "" {
		model = providerDefaultImageModel()
	}
	size := normalizeAiImageSize(req.GetSize(), model)
	quality := normalizeAiImageQuality(req.GetQuality(), model)
	background := normalizeAiImageBackground(req.GetBackground(), model)
	outputFormat := normalizeAiImageOutputFormat(req.GetOutputFormat())
	responseFormat := normalizeAiImageResponseFormat(req.GetResponseFormat(), model)
	style := normalizeAiImageStyle(req.GetStyle(), model)
	imageCount := normalizeAiImageCount(req.GetN(), model)
	paramsJSON, err := buildAiImageParamsJSON(originalPrompt, model, size, quality, style, background, outputFormat, responseFormat, imageCount, req.GetSaveOutput(), req.GetPolishPrompt())
	if err != nil {
		return nil, err
	}
	now := time.Now()
	image := &models.AiImage{
		CreatedBy:      userID,
		Terminal:       assistant.NormalizeTerminal(req.GetTerminal()),
		Prompt:         originalPrompt,
		OriginalPrompt: originalPrompt,
		Model:          model,
		Size:           size,
		Quality:        quality,
		Style:          style,
		Background:     background,
		OutputFormat:   outputFormat,
		ResponseFormat: responseFormat,
		ImageCount:     int32(imageCount),
		SaveOutput:     req.GetSaveOutput(),
		PolishPrompt:   req.GetPolishPrompt(),
		ParamsJSON:     paramsJSON,
		Status:         _const.AI_IMAGE_STATUS_PENDING,
		ImageUrlsJSON:  "[]",
		CreatedAt:      now,
	}
	if !retry && !req.GetSaveOutput() {
		image.SaveOutput = false
	}
	return image, nil
}

// generateAiImageResult 提交或查询 Responses 图片生成结果。
func (c *AiImageCase) generateAiImageResult(ctx context.Context, image *models.AiImage) (*basev1.AiImage, bool, error) {
	if c.imageClient == nil || !c.imageClient.Enabled() {
		return nil, false, errorsx.Internal("AI图片客户端未配置")
	}
	prompt := strings.TrimSpace(image.Prompt)
	var err error
	params := unmarshalAiImageParams(image.ParamsJSON)
	if image.PolishPrompt && strings.TrimSpace(params.ResponseID) == "" {
		var polishResponse *basev1.PolishAiImagePromptResponse
		polishResponse, err = c.PolishAiImagePrompt(ctx, &basev1.PolishAiImagePromptRequest{
			Prompt: image.OriginalPrompt,
			Scene:  "商城后台图片生成",
		})
		if err != nil {
			return nil, false, err
		}
		if strings.TrimSpace(polishResponse.GetPrompt()) != "" {
			prompt = strings.TrimSpace(polishResponse.GetPrompt())
		}
	}

	if strings.TrimSpace(params.ResponseID) == "" {
		task, createErr := c.imageClient.CreateResponseImageTask(ctx, prompt, provider.ImageGenerateOptions{
			Model:        image.Model,
			Background:   image.Background,
			Size:         normalizeResponsesAiImageSize(image.Size),
			Quality:      normalizeResponsesAiImageQuality(image.Quality),
			OutputFormat: image.OutputFormat,
			N:            1,
		})
		if createErr != nil {
			return nil, false, errorsx.Internal("AI图片生成任务创建失败").WithCause(createErr)
		}
		if task == nil || strings.TrimSpace(task.ID) == "" {
			return nil, false, errorsx.Internal("AI图片生成任务创建结果为空")
		}
		params.ResponseID = task.ID
		params.ResponseStatus = task.Status
		params.ResponseCreated = task.CreatedAt
		params.ResponseSubmitted = true
		params.ResponseSubmittedAt = time.Now().Unix()
		params.ResponsePolledAt = time.Now().Unix()
		if err = c.updateAiImageResponseTask(ctx, image.ID, prompt, params); err != nil {
			return nil, false, err
		}
		image.Prompt = prompt
		image.ParamsJSON = mustMarshalAiImageParams(params)
		return nil, true, nil
	}

	task, err := c.imageClient.GetResponseImageTask(ctx, params.ResponseID)
	if err != nil {
		return nil, false, errorsx.Internal("AI图片生成任务查询失败").WithCause(err)
	}
	params.ResponseStatus = task.Status
	params.ResponseCreated = task.CreatedAt
	params.ResponsePolledAt = time.Now().Unix()
	if err = c.updateAiImageResponseTask(ctx, image.ID, prompt, params); err != nil {
		return nil, false, err
	}
	switch task.Status {
	case "completed":
		images, convertErr := c.toAiImagesFromResponseTask(task, image.OutputFormat, image.SaveOutput)
		if convertErr != nil {
			return nil, false, convertErr
		}
		if len(images) == 0 {
			return nil, false, errorsx.Internal("AI图片生成结果为空")
		}
		return &basev1.AiImage{
			Id:             strconv.FormatInt(image.ID, 10),
			Prompt:         prompt,
			OriginalPrompt: image.OriginalPrompt,
			Model:          image.Model,
			Images:         images,
			Created:        task.CreatedAt,
		}, false, nil
	case "failed", "cancelled", "incomplete":
		return nil, false, errorsx.Internal("AI图片生成任务失败：" + task.Status)
	default:
		return nil, true, nil
	}
}

// updateImageRunning 将图片标记为生成中。
func (c *AiImageCase) updateImageRunning(ctx context.Context, imageID int64, startedAt time.Time) error {
	query := c.aiImageRepo.Query(ctx).AiImage
	_, err := query.WithContext(ctx).
		Where(query.ID.Eq(imageID), query.Status.Eq(_const.AI_IMAGE_STATUS_PENDING)).
		UpdateSimple(
			query.Status.Value(_const.AI_IMAGE_STATUS_RUNNING),
			query.StartedAt.Value(startedAt),
		)
	return err
}

// markImageSuccess 将生成成功结果写入图片记录。
func (c *AiImageCase) markImageSuccess(ctx context.Context, imageID int64, response *basev1.AiImage) error {
	now := time.Now()
	imageURLsJSON, err := marshalAiImageList(response.GetImages())
	if err != nil {
		return err
	}
	query := c.aiImageRepo.Query(ctx).AiImage
	_, err = query.WithContext(ctx).
		Where(query.ID.Eq(imageID)).
		UpdateSimple(
			query.Status.Value(_const.AI_IMAGE_STATUS_SUCCESS),
			query.Prompt.Value(response.GetPrompt()),
			query.ImageUrlsJSON.Value(imageURLsJSON),
			query.Created.Value(int32(response.GetCreated())),
			query.ErrorMessage.Value(""),
			query.FinishedAt.Value(now),
		)
	return err
}

// markImageFailed 将生成失败结果写入图片记录。
func (c *AiImageCase) markImageFailed(ctx context.Context, imageID int64, generateErr error) error {
	now := time.Now()
	status := _const.AI_IMAGE_STATUS_FAILED
	if errors.Is(generateErr, context.DeadlineExceeded) {
		status = _const.AI_IMAGE_STATUS_TIMEOUT
	}
	query := c.aiImageRepo.Query(ctx).AiImage
	_, err := query.WithContext(ctx).
		Where(query.ID.Eq(imageID)).
		UpdateSimple(
			query.Status.Value(status),
			query.ErrorMessage.Value(limitAiImageErrorMessage(generateErr)),
			query.RetryCount.Add(1),
			query.FinishedAt.Value(now),
		)
	return err
}

// dispatchAiImagePoll 延迟投递下一次图片生成状态查询。
func (c *AiImageCase) dispatchAiImagePoll(imageID int64) {
	time.AfterFunc(aiImagePollInterval, func() {
		queue.DispatchAiImageGenerate(imageID)
	})
}

// updateAiImageResponseTask 更新 Responses 图片生成任务快照。
func (c *AiImageCase) updateAiImageResponseTask(ctx context.Context, imageID int64, prompt string, params aiImageParamsSnapshot) error {
	paramsJSON, err := marshalAiImageParams(params)
	if err != nil {
		return err
	}
	query := c.aiImageRepo.Query(ctx).AiImage
	_, err = query.WithContext(ctx).
		Where(query.ID.Eq(imageID), query.Status.Eq(_const.AI_IMAGE_STATUS_RUNNING)).
		UpdateSimple(
			query.Prompt.Value(prompt),
			query.ParamsJSON.Value(paramsJSON),
			query.ErrorMessage.Value(""),
		)
	return err
}

// findCurrentUserImageByRawID 按当前用户与字符串图片编号查询记录。
func (c *AiImageCase) findCurrentUserImageByRawID(ctx context.Context, rawID string) (*models.AiImage, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	var imageID int64
	imageID, err = strconv.ParseInt(strings.TrimSpace(rawID), 10, 64)
	if err != nil || imageID <= 0 {
		return nil, errorsx.InvalidArgument("图片编号不能为空")
	}
	query := c.aiImageRepo.Query(ctx).AiImage
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.ID.Eq(imageID)))
	opts = append(opts, repository.Where(query.CreatedBy.Eq(authInfo.UserId)))
	var image *models.AiImage
	image, err = c.aiImageRepo.Find(ctx, opts...)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.ResourceNotFound("AI图片不存在")
		}
		return nil, err
	}
	return image, nil
}

// toImageDTO 将图片模型转换为接口响应。
func (c *AiImageCase) toImageDTO(image *models.AiImage) *basev1.AiImage {
	if image == nil {
		return nil
	}
	dto := c.mapper.ToDTO(image)
	dto.Id = strconv.FormatInt(image.ID, 10)
	dto.N = int64(image.ImageCount)
	dto.Status = basev1.AiImageStatus(image.Status)
	dto.Images = unmarshalAiImageList(image.ImageUrlsJSON)
	dto.Created = int64(image.Created)
	dto.Terminal = commonv1.Terminal(image.Terminal)
	dto.StartedAt = timeToTimestamp(image.StartedAt)
	dto.FinishedAt = timeToTimestamp(image.FinishedAt)
	dto.CreatedAt = timeToTimestamp(image.CreatedAt)
	return dto
}

// toAiImages 将 Blades 模型响应转换成接口图片结果。
func (c *AiImageCase) toAiImages(response *blades.ModelResponse, outputFormat string, saveOutput bool) ([]*basev1.AiImageResult, error) {
	if response == nil || response.Message == nil {
		return nil, nil
	}
	images := make([]*basev1.AiImageResult, 0, len(response.Message.Parts))
	imageIndex := 0
	var err error
	for _, part := range response.Message.Parts {
		switch value := part.(type) {
		case blades.DataPart:
			imageIndex++
			var image *basev1.AiImageResult
			image, err = c.dataPartToAiImage(value, outputFormat, saveOutput)
			if err != nil {
				return nil, err
			}
			image.RevisedPrompt = readStringMetadata(response, fmt.Sprintf("image-%d_revised_prompt_%d", imageIndex, imageIndex))
			images = append(images, image)
		case blades.FilePart:
			imageIndex++
			images = append(images, &basev1.AiImageResult{
				Name:          value.Name,
				Url:           value.URI,
				MimeType:      string(value.MIMEType),
				RevisedPrompt: readStringMetadata(response, fmt.Sprintf("image-%d_revised_prompt_%d", imageIndex, imageIndex)),
				Saved:         false,
			})
		}
	}
	return images, nil
}

// toAiImagesFromResponseTask 将 Responses 图片任务结果转换成接口图片结果。
func (c *AiImageCase) toAiImagesFromResponseTask(task *provider.ImageResponseTask, outputFormat string, saveOutput bool) ([]*basev1.AiImageResult, error) {
	if task == nil {
		return nil, nil
	}
	images := make([]*basev1.AiImageResult, 0, len(task.Images))
	for index, item := range task.Images {
		if strings.TrimSpace(item.Result) == "" {
			continue
		}
		rawBytes, err := base64.StdEncoding.DecodeString(item.Result)
		if err != nil {
			return nil, errorsx.Internal("解析AI图片生成结果失败").WithCause(err)
		}
		name := strings.TrimSpace(item.ID)
		if name == "" {
			name = fmt.Sprintf("image-%d", index+1)
		}
		image, err := c.dataPartToAiImage(blades.DataPart{
			Name:     name,
			MIMEType: blades.MIMEType(aiImageMimeType(outputFormat)),
			Bytes:    rawBytes,
		}, outputFormat, saveOutput)
		if err != nil {
			return nil, err
		}
		images = append(images, image)
	}
	return images, nil
}

// dataPartToAiImage 将二进制图片结果转换为可展示图片。
func (c *AiImageCase) dataPartToAiImage(part blades.DataPart, outputFormat string, saveOutput bool) (*basev1.AiImageResult, error) {
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

	image := &basev1.AiImageResult{
		Name:     name,
		MimeType: mimeType,
		Size:     int64(len(part.Bytes)),
	}
	if saveOutput && c.oss != nil {
		filePath := fmt.Sprintf("/%s/ai/images/%s", _const.BASE_PATH, time.Now().Format("2006/01/02"))
		name = withAiImageStorageFileName(name)
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

// buildAiImageParamsJSON 构建发送给图片模型的参数快照。
func buildAiImageParamsJSON(prompt string, model string, size string, quality string, style string, background string, outputFormat string, responseFormat string, imageCount int64, saveOutput bool, polishPrompt bool) (string, error) {
	params := aiImageParamsSnapshot{
		Prompt:         prompt,
		Model:          model,
		Size:           size,
		Quality:        quality,
		Style:          style,
		Background:     background,
		OutputFormat:   outputFormat,
		ResponseFormat: responseFormat,
		N:              imageCount,
		SaveOutput:     saveOutput,
		PolishPrompt:   polishPrompt,
	}
	return marshalAiImageParams(params)
}

// marshalAiImageParams 序列化图片生成参数快照。
func marshalAiImageParams(params aiImageParamsSnapshot) (string, error) {
	rawBody, err := json.Marshal(params)
	if err != nil {
		return "", errorsx.Internal("构建AI图片参数失败").WithCause(err)
	}
	return string(rawBody), nil
}

// mustMarshalAiImageParams 序列化图片生成参数快照，失败时返回空对象。
func mustMarshalAiImageParams(params aiImageParamsSnapshot) string {
	rawBody, err := json.Marshal(params)
	if err != nil {
		return "{}"
	}
	return string(rawBody)
}

// unmarshalAiImageParams 反序列化图片生成参数快照。
func unmarshalAiImageParams(rawValue string) aiImageParamsSnapshot {
	rawValue = strings.TrimSpace(rawValue)
	if rawValue == "" {
		return aiImageParamsSnapshot{}
	}
	var params aiImageParamsSnapshot
	if err := json.Unmarshal([]byte(rawValue), &params); err != nil {
		return aiImageParamsSnapshot{}
	}
	return params
}

// resetAiImageResponseTask 清除 Responses 任务信息，保留用户生成参数。
func resetAiImageResponseTask(rawValue string) string {
	params := unmarshalAiImageParams(rawValue)
	params.ResponseID = ""
	params.ResponseStatus = ""
	params.ResponseCreated = 0
	params.ResponseSubmitted = false
	params.ResponsePolledAt = 0
	params.ResponseSubmittedAt = 0
	return mustMarshalAiImageParams(params)
}

// marshalAiImageList 序列化图片列表。
func marshalAiImageList(images []*basev1.AiImageResult) (string, error) {
	if images == nil {
		return "[]", nil
	}
	rawBody, err := json.Marshal(images)
	if err != nil {
		return "", errorsx.Internal("序列化AI图片结果失败").WithCause(err)
	}
	return string(rawBody), nil
}

// unmarshalAiImageList 反序列化图片列表。
func unmarshalAiImageList(rawValue string) []*basev1.AiImageResult {
	rawValue = strings.TrimSpace(rawValue)
	if rawValue == "" {
		return nil
	}
	var images []*basev1.AiImageResult
	err := json.Unmarshal([]byte(rawValue), &images)
	if err != nil {
		return nil
	}
	return images
}

// limitAiImageErrorMessage 截断错误信息。
func limitAiImageErrorMessage(err error) string {
	message := strings.TrimSpace(fmt.Sprint(err))
	if message == "" {
		message = "AI图片生成失败"
	}
	const maxLength = 1000
	if len([]rune(message)) <= maxLength {
		return message
	}
	runes := []rune(message)
	return string(runes[:maxLength])
}

// timeToTimestamp 将非零时间转换为 protobuf 时间。
func timeToTimestamp(value time.Time) *timestamppb.Timestamp {
	if value.IsZero() {
		return nil
	}
	return timestamppb.New(value)
}

// timestampToTime 将 protobuf 时间转换为普通时间。
func timestampToTime(value *timestamppb.Timestamp) time.Time {
	if value == nil {
		return time.Time{}
	}
	return value.AsTime()
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

// withAiImageStorageFileName 为保存文件名补充时间戳，降低同名覆盖概率。
func withAiImageStorageFileName(name string) string {
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

// normalizeResponsesAiImageSize 标准化 Responses 图片工具尺寸。
func normalizeResponsesAiImageSize(size string) string {
	size = strings.TrimSpace(size)
	switch size {
	case "1024x1024", "1024x1536", "1536x1024", "auto":
		return size
	case "1792x1024":
		return "1536x1024"
	case "1024x1792":
		return "1024x1536"
	default:
		return "auto"
	}
}

// normalizeResponsesAiImageQuality 标准化 Responses 图片工具质量。
func normalizeResponsesAiImageQuality(quality string) string {
	quality = strings.TrimSpace(quality)
	switch quality {
	case "low", "medium", "high", "auto":
		return quality
	case "standard", "hd":
		return "auto"
	default:
		return aiImageDefaultQuality
	}
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

// providerDefaultImageModel 返回图片默认模型名称。
func providerDefaultImageModel() string {
	return "gpt-image-2"
}

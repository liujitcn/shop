package base

import (
	"context"

	basev1 "shop/api/gen/go/base/v1"
	"shop/pkg/errorsx"
	"shop/service/base/biz"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/grpc"
)

const _ = grpc.SupportPackageIsVersion7

// AiImageService AI 图片公共服务。
type AiImageService struct {
	basev1.UnimplementedAiImageServiceServer
	aiImageCase *biz.AiImageCase
}

// NewAiImageService 创建 AI 图片公共服务。
func NewAiImageService(aiImageCase *biz.AiImageCase) *AiImageService {
	return &AiImageService{
		aiImageCase: aiImageCase,
	}
}

// PageAiImageTasks 分页查询 AI 图片。
func (s *AiImageService) PageAiImageTasks(ctx context.Context, req *basev1.PageAiImageTasksRequest) (*basev1.PageAiImageTasksResponse, error) {
	res, err := s.aiImageCase.PageAiImageTasks(ctx, req)
	if err != nil {
		log.Errorf("PageAiImageTasks %v", err)
		return nil, errorsx.WrapInternal(err, "查询AI图片失败")
	}
	return res, nil
}

// GetAiImageTask 查询 AI 图片。
func (s *AiImageService) GetAiImageTask(ctx context.Context, req *basev1.GetAiImageTaskRequest) (*basev1.AiImageTask, error) {
	res, err := s.aiImageCase.GetAiImageTask(ctx, req)
	if err != nil {
		log.Errorf("GetAiImageTask %v", err)
		return nil, errorsx.WrapInternal(err, "查询AI图片失败")
	}
	return res, nil
}

// CreateAiImageTask 创建 AI 图片。
func (s *AiImageService) CreateAiImageTask(ctx context.Context, req *basev1.CreateAiImageTaskRequest) (*basev1.AiImageTask, error) {
	res, err := s.aiImageCase.CreateAiImageTask(ctx, req)
	if err != nil {
		log.Errorf("CreateAiImageTask %v", err)
		return nil, errorsx.WrapInternal(err, "创建AI图片失败")
	}
	return res, nil
}

// RetryAiImageTask 重试 AI 图片生成。
func (s *AiImageService) RetryAiImageTask(ctx context.Context, req *basev1.RetryAiImageTaskRequest) (*basev1.AiImageTask, error) {
	res, err := s.aiImageCase.RetryAiImageTask(ctx, req)
	if err != nil {
		log.Errorf("RetryAiImageTask %v", err)
		return nil, errorsx.WrapInternal(err, "重试AI图片失败")
	}
	return res, nil
}

// PolishAiImagePrompt 润色 AI 图片提示词。
func (s *AiImageService) PolishAiImagePrompt(ctx context.Context, req *basev1.PolishAiImagePromptRequest) (*basev1.PolishAiImagePromptResponse, error) {
	res, err := s.aiImageCase.PolishAiImagePrompt(ctx, req)
	if err != nil {
		log.Errorf("PolishAiImagePrompt %v", err)
		return nil, errorsx.WrapInternal(err, "润色AI图片提示词失败")
	}
	return res, nil
}

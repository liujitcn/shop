package base

import (
	"context"

	basev1 "shop/api/gen/go/base/v1"
	"shop/pkg/errorsx"
	"shop/service/base/biz"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
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

// PageAiImages 分页查询 AI 图片。
func (s *AiImageService) PageAiImages(ctx context.Context, req *basev1.PageAiImagesRequest) (*basev1.PageAiImagesResponse, error) {
	res, err := s.aiImageCase.PageAiImages(ctx, req)
	if err != nil {
		log.Errorf("PageAiImages %v", err)
		return nil, errorsx.WrapInternal(err, "查询AI图片失败")
	}
	return res, nil
}

// GetAiImage 查询 AI 图片。
func (s *AiImageService) GetAiImage(ctx context.Context, req *basev1.GetAiImageRequest) (*basev1.AiImage, error) {
	res, err := s.aiImageCase.GetAiImage(ctx, req)
	if err != nil {
		log.Errorf("GetAiImage %v", err)
		return nil, errorsx.WrapInternal(err, "查询AI图片失败")
	}
	return res, nil
}

// CreateAiImage 创建 AI 图片。
func (s *AiImageService) CreateAiImage(ctx context.Context, req *basev1.CreateAiImageRequest) (*basev1.AiImage, error) {
	res, err := s.aiImageCase.CreateAiImage(ctx, req)
	if err != nil {
		log.Errorf("CreateAiImage %v", err)
		return nil, errorsx.WrapInternal(err, "创建AI图片失败")
	}
	return res, nil
}

// DeleteAiImage 删除 AI 图片。
func (s *AiImageService) DeleteAiImage(ctx context.Context, req *basev1.DeleteAiImageRequest) (*emptypb.Empty, error) {
	res, err := s.aiImageCase.DeleteAiImage(ctx, req)
	if err != nil {
		log.Errorf("DeleteAiImage %v", err)
		return nil, errorsx.WrapInternal(err, "删除AI图片失败")
	}
	return res, nil
}

// RetryAiImage 重试 AI 图片生成。
func (s *AiImageService) RetryAiImage(ctx context.Context, req *basev1.RetryAiImageRequest) (*basev1.AiImage, error) {
	res, err := s.aiImageCase.RetryAiImage(ctx, req)
	if err != nil {
		log.Errorf("RetryAiImage %v", err)
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

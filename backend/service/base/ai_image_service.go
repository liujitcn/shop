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

// GenerateAiImage 生成 AI 图片。
func (s *AiImageService) GenerateAiImage(ctx context.Context, req *basev1.GenerateAiImageRequest) (*basev1.GenerateAiImageResponse, error) {
	res, err := s.aiImageCase.GenerateAiImage(ctx, req)
	if err != nil {
		log.Errorf("GenerateAiImage %v", err)
		return nil, errorsx.WrapInternal(err, "生成AI图片失败")
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

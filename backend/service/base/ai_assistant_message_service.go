package base

import (
	"context"

	basev1 "shop/api/gen/go/base/v1"
	"shop/pkg/errorsx"
	"shop/service/base/biz"
	"shop/service/base/dto"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/grpc"
)

const _ = grpc.SupportPackageIsVersion7

// AiAssistantMessageService AI 助手消息公共服务。
type AiAssistantMessageService struct {
	basev1.UnimplementedAiAssistantMessageServiceServer
	aiAssistantMessageCase *biz.AiAssistantMessageCase
}

// NewAiAssistantMessageService 创建 AI 助手消息公共服务。
func NewAiAssistantMessageService(aiAssistantMessageCase *biz.AiAssistantMessageCase) *AiAssistantMessageService {
	return &AiAssistantMessageService{
		aiAssistantMessageCase: aiAssistantMessageCase,
	}
}

// SendAiAssistantMessage 发送 AI 助手消息。
func (s *AiAssistantMessageService) SendAiAssistantMessage(ctx context.Context, req *basev1.SendAiAssistantMessageRequest) (*basev1.SendAiAssistantMessageResponse, error) {
	res, err := s.aiAssistantMessageCase.SendAiAssistantMessage(ctx, req)
	if err != nil {
		log.Errorf("SendAiAssistantMessage %v", err)
		return nil, errorsx.WrapInternal(err, "发送AI助手消息失败")
	}
	return res, nil
}

// DeleteAiAssistantMessage 删除 AI 助手消息。
func (s *AiAssistantMessageService) DeleteAiAssistantMessage(ctx context.Context, req *basev1.DeleteAiAssistantMessageRequest) (*basev1.DeleteAiAssistantMessageResponse, error) {
	err := s.aiAssistantMessageCase.DeleteAiAssistantMessage(ctx, req)
	if err != nil {
		log.Errorf("DeleteAiAssistantMessage %v", err)
		return nil, errorsx.WrapInternal(err, "删除AI助手消息失败")
	}
	return &basev1.DeleteAiAssistantMessageResponse{}, nil
}

// RetryAiAssistantUserMessage 重试失败的 AI 助手消息。
func (s *AiAssistantMessageService) RetryAiAssistantUserMessage(ctx context.Context, req *basev1.RetryAiAssistantUserMessageRequest) (*basev1.SendAiAssistantMessageResponse, error) {
	res, err := s.aiAssistantMessageCase.RetryAiAssistantUserMessage(ctx, req)
	if err != nil {
		log.Errorf("RetryAiAssistantUserMessage %v", err)
		return nil, errorsx.WrapInternal(err, "重试AI助手消息失败")
	}
	return res, nil
}

// RegenerateAiAssistantMessage 重新生成 AI 助手输出。
func (s *AiAssistantMessageService) RegenerateAiAssistantMessage(ctx context.Context, req *basev1.RegenerateAiAssistantMessageRequest) (*basev1.SendAiAssistantMessageResponse, error) {
	res, err := s.aiAssistantMessageCase.RegenerateAiAssistantMessage(ctx, req)
	if err != nil {
		log.Errorf("RegenerateAiAssistantMessage %v", err)
		return nil, errorsx.WrapInternal(err, "重新生成AI助手输出失败")
	}
	return res, nil
}

// StreamAiAssistantMessage 流式发送 AI 助手消息。
func (s *AiAssistantMessageService) StreamAiAssistantMessage(ctx context.Context, req *basev1.SendAiAssistantMessageRequest, emitter dto.AiAssistantStreamEmitter) error {
	err := s.aiAssistantMessageCase.StreamAiAssistantMessage(ctx, req, emitter)
	if err != nil {
		log.Errorf("StreamAiAssistantMessage %v", err)
		return errorsx.WrapInternal(err, "流式发送AI助手消息失败")
	}
	return nil
}

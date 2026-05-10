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

// AiAssistantService AI 助手公共服务。
type AiAssistantService struct {
	basev1.UnimplementedAiAssistantServiceServer
	aiAssistantCase *biz.AiAssistantCase
}

// NewAiAssistantService 创建 AI 助手公共服务。
func NewAiAssistantService(aiAssistantCase *biz.AiAssistantCase) *AiAssistantService {
	return &AiAssistantService{
		aiAssistantCase: aiAssistantCase,
	}
}

// ListAiAssistantSessions 查询 AI 助手会话列表。
func (s *AiAssistantService) ListAiAssistantSessions(ctx context.Context, req *basev1.ListAiAssistantSessionsRequest) (*basev1.ListAiAssistantSessionsResponse, error) {
	res, err := s.aiAssistantCase.ListAiAssistantSessions(ctx, req)
	if err != nil {
		log.Errorf("ListAiAssistantSessions %v", err)
		return nil, errorsx.WrapInternal(err, "查询AI助手会话失败")
	}
	return res, nil
}

// CreateAiAssistantSession 创建 AI 助手会话。
func (s *AiAssistantService) CreateAiAssistantSession(ctx context.Context, req *basev1.CreateAiAssistantSessionRequest) (*basev1.AiAssistantSession, error) {
	res, err := s.aiAssistantCase.CreateAiAssistantSession(ctx, req)
	if err != nil {
		log.Errorf("CreateAiAssistantSession %v", err)
		return nil, errorsx.WrapInternal(err, "创建AI助手会话失败")
	}
	return res, nil
}

// UpdateAiAssistantSession 更新 AI 助手会话。
func (s *AiAssistantService) UpdateAiAssistantSession(ctx context.Context, req *basev1.UpdateAiAssistantSessionRequest) (*basev1.AiAssistantSession, error) {
	res, err := s.aiAssistantCase.UpdateAiAssistantSession(ctx, req)
	if err != nil {
		log.Errorf("UpdateAiAssistantSession %v", err)
		return nil, errorsx.WrapInternal(err, "更新AI助手会话失败")
	}
	return res, nil
}

// DeleteAiAssistantSession 删除 AI 助手会话。
func (s *AiAssistantService) DeleteAiAssistantSession(ctx context.Context, req *basev1.DeleteAiAssistantSessionRequest) (*emptypb.Empty, error) {
	res, err := s.aiAssistantCase.DeleteAiAssistantSession(ctx, req)
	if err != nil {
		log.Errorf("DeleteAiAssistantSession %v", err)
		return nil, errorsx.WrapInternal(err, "删除AI助手会话失败")
	}
	return res, nil
}

// ListAiAssistantMessages 查询 AI 助手消息列表。
func (s *AiAssistantService) ListAiAssistantMessages(ctx context.Context, req *basev1.ListAiAssistantMessagesRequest) (*basev1.ListAiAssistantMessagesResponse, error) {
	res, err := s.aiAssistantCase.ListAiAssistantMessages(ctx, req)
	if err != nil {
		log.Errorf("ListAiAssistantMessages %v", err)
		return nil, errorsx.WrapInternal(err, "查询AI助手消息失败")
	}
	return res, nil
}

// SendAiAssistantMessage 发送 AI 助手消息。
func (s *AiAssistantService) SendAiAssistantMessage(ctx context.Context, req *basev1.SendAiAssistantMessageRequest) (*basev1.SendAiAssistantMessageResponse, error) {
	res, err := s.aiAssistantCase.SendAiAssistantMessage(ctx, req)
	if err != nil {
		log.Errorf("SendAiAssistantMessage %v", err)
		return nil, errorsx.WrapInternal(err, "发送AI助手消息失败")
	}
	return res, nil
}

// OperateAiAssistantConfirm 处理 AI 助手确认卡动作。
func (s *AiAssistantService) OperateAiAssistantConfirm(ctx context.Context, req *basev1.OperateAiAssistantConfirmRequest) (*basev1.OperateAiAssistantConfirmResponse, error) {
	res, err := s.aiAssistantCase.OperateAiAssistantConfirm(ctx, req)
	if err != nil {
		log.Errorf("OperateAiAssistantConfirm %v", err)
		return nil, errorsx.WrapInternal(err, "处理AI助手确认动作失败")
	}
	return res, nil
}

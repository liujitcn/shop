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

// AiAssistantService AI 助手公共服务。
type AiAssistantService struct {
	basev1.UnimplementedAiAssistantServiceServer
	aiAssistantSessionCase *biz.AiAssistantSessionCase
	aiAssistantMessageCase *biz.AiAssistantMessageCase
}

// NewAiAssistantService 创建 AI 助手公共服务。
func NewAiAssistantService(aiAssistantSessionCase *biz.AiAssistantSessionCase, aiAssistantMessageCase *biz.AiAssistantMessageCase) *AiAssistantService {
	return &AiAssistantService{
		aiAssistantSessionCase: aiAssistantSessionCase,
		aiAssistantMessageCase: aiAssistantMessageCase,
	}
}

// ListAiAssistantShortcuts 查询 AI 助手快捷入口列表。
func (s *AiAssistantService) ListAiAssistantShortcuts(ctx context.Context, req *basev1.ListAiAssistantShortcutsRequest) (*basev1.ListAiAssistantShortcutsResponse, error) {
	res, err := s.aiAssistantMessageCase.ListAiAssistantShortcuts(ctx, req)
	if err != nil {
		log.Errorf("ListAiAssistantShortcuts %v", err)
		return nil, errorsx.WrapInternal(err, "查询AI助手快捷入口失败")
	}
	return res, nil
}

// ListAiAssistantSessions 查询 AI 助手会话列表。
func (s *AiAssistantService) ListAiAssistantSessions(ctx context.Context, req *basev1.ListAiAssistantSessionsRequest) (*basev1.ListAiAssistantSessionsResponse, error) {
	res, err := s.aiAssistantSessionCase.ListAiAssistantSessions(ctx, req)
	if err != nil {
		log.Errorf("ListAiAssistantSessions %v", err)
		return nil, errorsx.WrapInternal(err, "查询AI助手会话失败")
	}
	return res, nil
}

// CreateAiAssistantSession 创建 AI 助手会话。
func (s *AiAssistantService) CreateAiAssistantSession(ctx context.Context, req *basev1.CreateAiAssistantSessionRequest) (*basev1.CreateAiAssistantSessionResponse, error) {
	res, err := s.aiAssistantSessionCase.CreateAiAssistantSession(ctx, req)
	if err != nil {
		log.Errorf("CreateAiAssistantSession %v", err)
		return nil, errorsx.WrapInternal(err, "创建AI助手会话失败")
	}
	return &basev1.CreateAiAssistantSessionResponse{Session: res}, nil
}

// UpdateAiAssistantSession 更新 AI 助手会话。
func (s *AiAssistantService) UpdateAiAssistantSession(ctx context.Context, req *basev1.UpdateAiAssistantSessionRequest) (*basev1.UpdateAiAssistantSessionResponse, error) {
	res, err := s.aiAssistantSessionCase.UpdateAiAssistantSession(ctx, req)
	if err != nil {
		log.Errorf("UpdateAiAssistantSession %v", err)
		return nil, errorsx.WrapInternal(err, "更新AI助手会话失败")
	}
	return &basev1.UpdateAiAssistantSessionResponse{Session: res}, nil
}

// DeleteAiAssistantSession 删除 AI 助手会话。
func (s *AiAssistantService) DeleteAiAssistantSession(ctx context.Context, req *basev1.DeleteAiAssistantSessionRequest) (*basev1.DeleteAiAssistantSessionResponse, error) {
	_, err := s.aiAssistantSessionCase.DeleteAiAssistantSession(ctx, req)
	if err != nil {
		log.Errorf("DeleteAiAssistantSession %v", err)
		return nil, errorsx.WrapInternal(err, "删除AI助手会话失败")
	}
	return &basev1.DeleteAiAssistantSessionResponse{}, nil
}

// ListAiAssistantMessages 查询 AI 助手消息列表。
func (s *AiAssistantService) ListAiAssistantMessages(ctx context.Context, req *basev1.ListAiAssistantMessagesRequest) (*basev1.ListAiAssistantMessagesResponse, error) {
	res, err := s.aiAssistantMessageCase.ListAiAssistantMessages(ctx, req)
	if err != nil {
		log.Errorf("ListAiAssistantMessages %v", err)
		return nil, errorsx.WrapInternal(err, "查询AI助手消息失败")
	}
	return res, nil
}

// CreateAiAssistantSessionBranch 从指定消息创建 AI 助手分支会话。
func (s *AiAssistantService) CreateAiAssistantSessionBranch(ctx context.Context, req *basev1.CreateAiAssistantSessionBranchRequest) (*basev1.CreateAiAssistantSessionBranchResponse, error) {
	res, err := s.aiAssistantSessionCase.CreateAiAssistantSessionBranch(ctx, req)
	if err != nil {
		log.Errorf("CreateAiAssistantSessionBranch %v", err)
		return nil, errorsx.WrapInternal(err, "创建AI助手分支会话失败")
	}
	return res, nil
}

package base

import (
	"context"
	"fmt"

	basev1 "shop/api/gen/go/base/v1"
	"shop/pkg/errorsx"
	"shop/service/base/biz"

	"github.com/go-kratos/kratos/v3/log"
	"google.golang.org/grpc"
)

const _ = grpc.SupportPackageIsVersion7

// AiService AI 助手公共服务。
type AiService struct {
	basev1.UnimplementedAiServiceServer
	aiSessionCase *biz.AiSessionCase
	aiMessageCase *biz.AiMessageCase
}

// NewAiService 创建 AI 助手公共服务。
func NewAiService(aiSessionCase *biz.AiSessionCase, aiMessageCase *biz.AiMessageCase) *AiService {
	return &AiService{
		aiSessionCase: aiSessionCase,
		aiMessageCase: aiMessageCase,
	}
}

// ListAiShortcut 查询 AI 助手快捷入口列表。
func (s *AiService) ListAiShortcut(ctx context.Context, req *basev1.ListAiShortcutRequest) (*basev1.ListAiShortcutResponse, error) {
	res, err := s.aiMessageCase.ListAiShortcut(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("ListAiShortcut %v", err))
		return nil, errorsx.WrapInternal(err, "查询AI助手快捷入口失败")
	}
	return res, nil
}

// ListAiSession 查询 AI 助手会话列表。
func (s *AiService) ListAiSession(ctx context.Context, req *basev1.ListAiSessionRequest) (*basev1.ListAiSessionResponse, error) {
	res, err := s.aiSessionCase.ListAiSession(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("ListAiSession %v", err))
		return nil, errorsx.WrapInternal(err, "查询AI助手会话失败")
	}
	return res, nil
}

// CreateAiSession 创建 AI 助手会话。
func (s *AiService) CreateAiSession(ctx context.Context, req *basev1.CreateAiSessionRequest) (*basev1.CreateAiSessionResponse, error) {
	res, err := s.aiSessionCase.CreateAiSession(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("CreateAiSession %v", err))
		return nil, errorsx.WrapInternal(err, "创建AI助手会话失败")
	}
	return &basev1.CreateAiSessionResponse{Session: res}, nil
}

// UpdateAiSession 更新 AI 助手会话。
func (s *AiService) UpdateAiSession(ctx context.Context, req *basev1.UpdateAiSessionRequest) (*basev1.UpdateAiSessionResponse, error) {
	res, err := s.aiSessionCase.UpdateAiSession(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("UpdateAiSession %v", err))
		return nil, errorsx.WrapInternal(err, "更新AI助手会话失败")
	}
	return &basev1.UpdateAiSessionResponse{Session: res}, nil
}

// DeleteAiSession 删除 AI 助手会话。
func (s *AiService) DeleteAiSession(ctx context.Context, req *basev1.DeleteAiSessionRequest) (*basev1.DeleteAiSessionResponse, error) {
	_, err := s.aiSessionCase.DeleteAiSession(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("DeleteAiSession %v", err))
		return nil, errorsx.WrapInternal(err, "删除AI助手会话失败")
	}
	return &basev1.DeleteAiSessionResponse{}, nil
}

// ListAiMessage 查询 AI 助手消息列表。
func (s *AiService) ListAiMessage(ctx context.Context, req *basev1.ListAiMessageRequest) (*basev1.ListAiMessageResponse, error) {
	res, err := s.aiMessageCase.ListAiMessage(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("ListAiMessage %v", err))
		return nil, errorsx.WrapInternal(err, "查询AI助手消息失败")
	}
	return res, nil
}

// CreateAiSessionBranch 从指定消息创建 AI 助手分支会话。
func (s *AiService) CreateAiSessionBranch(ctx context.Context, req *basev1.CreateAiSessionBranchRequest) (*basev1.CreateAiSessionBranchResponse, error) {
	res, err := s.aiSessionCase.CreateAiSessionBranch(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("CreateAiSessionBranch %v", err))
		return nil, errorsx.WrapInternal(err, "创建AI助手分支会话失败")
	}
	return res, nil
}

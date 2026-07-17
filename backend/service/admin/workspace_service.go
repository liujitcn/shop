package admin

import (
	"context"
	"fmt"

	adminv1 "shop/api/gen/go/admin/v1"
	"shop/pkg/errorsx"
	"shop/service/admin/biz"

	"github.com/go-kratos/kratos/v3/log"
	"google.golang.org/grpc"
)

const _ = grpc.SupportPackageIsVersion7

// WorkspaceService Admin 工作台服务。
type WorkspaceService struct {
	adminv1.UnimplementedWorkspaceServiceServer
	workspaceCase *biz.WorkspaceCase
}

// NewWorkspaceService 创建 Admin 工作台服务。
func NewWorkspaceService(workspaceCase *biz.WorkspaceCase) *WorkspaceService {
	return &WorkspaceService{
		workspaceCase: workspaceCase,
	}
}

// SummaryWorkspaceMetrics 查询工作台顶部指标。
func (s *WorkspaceService) SummaryWorkspaceMetrics(ctx context.Context, req *adminv1.SummaryWorkspaceMetricsRequest) (*adminv1.SummaryWorkspaceMetricsResponse, error) {
	res, err := s.workspaceCase.SummaryWorkspaceMetrics(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SummaryWorkspaceMetrics %v", err))
		return nil, errorsx.WrapInternal(err, "查询工作台顶部指标失败")
	}
	return res, nil
}

// SummaryWorkspaceTodo 查询工作台待处理事项。
func (s *WorkspaceService) SummaryWorkspaceTodo(ctx context.Context, req *adminv1.SummaryWorkspaceTodoRequest) (*adminv1.SummaryWorkspaceTodoResponse, error) {
	res, err := s.workspaceCase.SummaryWorkspaceTodo(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SummaryWorkspaceTodo %v", err))
		return nil, errorsx.WrapInternal(err, "查询工作台待处理事项失败")
	}
	return res, nil
}

// SummaryWorkspaceRisk 查询工作台风险提醒。
func (s *WorkspaceService) SummaryWorkspaceRisk(ctx context.Context, req *adminv1.SummaryWorkspaceRiskRequest) (*adminv1.SummaryWorkspaceRiskResponse, error) {
	res, err := s.workspaceCase.SummaryWorkspaceRisk(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SummaryWorkspaceRisk %v", err))
		return nil, errorsx.WrapInternal(err, "查询工作台风险提醒失败")
	}
	return res, nil
}

// SummaryWorkspaceReputation 查询工作台口碑洞察。
func (s *WorkspaceService) SummaryWorkspaceReputation(ctx context.Context, req *adminv1.SummaryWorkspaceReputationRequest) (*adminv1.SummaryWorkspaceReputationResponse, error) {
	res, err := s.workspaceCase.SummaryWorkspaceReputation(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SummaryWorkspaceReputation %v", err))
		return nil, errorsx.WrapInternal(err, "查询工作台口碑洞察失败")
	}
	return res, nil
}

// ListWorkspacePendingComment 查询工作台待审核评价。
func (s *WorkspaceService) ListWorkspacePendingComment(ctx context.Context, req *adminv1.ListWorkspacePendingCommentRequest) (*adminv1.ListWorkspacePendingCommentResponse, error) {
	res, err := s.workspaceCase.ListWorkspacePendingComment(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("ListWorkspacePendingComment %v", err))
		return nil, errorsx.WrapInternal(err, "查询工作台待审核评价失败")
	}
	return res, nil
}

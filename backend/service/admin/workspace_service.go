package admin

import (
	"context"

	adminApi "shop/api/gen/go/admin"
	"shop/pkg/errorsx"
	"shop/service/admin/biz"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/grpc"
)

const _ = grpc.SupportPackageIsVersion7

// WorkspaceService Admin 工作台服务。
type WorkspaceService struct {
	adminApi.UnimplementedWorkspaceServiceServer
	workspaceCase *biz.WorkspaceCase
}

// NewWorkspaceService 创建 Admin 工作台服务。
func NewWorkspaceService(workspaceCase *biz.WorkspaceCase) *WorkspaceService {
	return &WorkspaceService{
		workspaceCase: workspaceCase,
	}
}

// GetWorkspaceMetrics 查询工作台顶部指标。
func (s *WorkspaceService) GetWorkspaceMetrics(ctx context.Context, req *adminApi.WorkspaceMetricsRequest) (*adminApi.WorkspaceMetricsResponse, error) {
	res, err := s.workspaceCase.GetWorkspaceMetrics(ctx, req)
	if err != nil {
		log.Errorf("GetWorkspaceMetrics %v", err)
		return nil, errorsx.WrapInternal(err, "查询工作台顶部指标失败")
	}
	return res, nil
}

// GetWorkspaceTodoList 查询工作台待处理事项。
func (s *WorkspaceService) GetWorkspaceTodoList(ctx context.Context, req *adminApi.WorkspaceTodoListRequest) (*adminApi.WorkspaceTodoListResponse, error) {
	res, err := s.workspaceCase.GetWorkspaceTodoList(ctx, req)
	if err != nil {
		log.Errorf("GetWorkspaceTodoList %v", err)
		return nil, errorsx.WrapInternal(err, "查询工作台待处理事项失败")
	}
	return res, nil
}

// GetWorkspaceRiskList 查询工作台风险提醒。
func (s *WorkspaceService) GetWorkspaceRiskList(ctx context.Context, req *adminApi.WorkspaceRiskListRequest) (*adminApi.WorkspaceRiskListResponse, error) {
	res, err := s.workspaceCase.GetWorkspaceRiskList(ctx, req)
	if err != nil {
		log.Errorf("GetWorkspaceRiskList %v", err)
		return nil, errorsx.WrapInternal(err, "查询工作台风险提醒失败")
	}
	return res, nil
}

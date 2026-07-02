package admin

import (
	"context"
	"fmt"

	adminv1 "shop/api/gen/go/admin/v1"
	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/errorsx"
	"shop/service/admin/biz"

	"github.com/go-kratos/kratos/v3/log"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

const _ = grpc.SupportPackageIsVersion7

// BaseTenantService Admin租户管理服务。
type BaseTenantService struct {
	adminv1.UnimplementedBaseTenantServiceServer
	baseTenantCase *biz.BaseTenantCase
}

// NewBaseTenantService 创建Admin租户管理服务。
func NewBaseTenantService(baseTenantCase *biz.BaseTenantCase) *BaseTenantService {
	return &BaseTenantService{
		baseTenantCase: baseTenantCase,
	}
}

// OptionBaseTenants 查询租户下拉选择。
func (s *BaseTenantService) OptionBaseTenants(ctx context.Context, req *adminv1.OptionBaseTenantsRequest) (*commonv1.SelectOptionResponse, error) {
	list, err := s.baseTenantCase.OptionBaseTenants(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("OptionBaseTenants %v", err))
		return nil, errorsx.WrapInternal(err, "查询租户下拉选择失败")
	}
	return list, nil
}

// PageBaseTenants 查询租户分页列表。
func (s *BaseTenantService) PageBaseTenants(ctx context.Context, req *adminv1.PageBaseTenantsRequest) (*adminv1.PageBaseTenantsResponse, error) {
	page, err := s.baseTenantCase.PageBaseTenants(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBaseTenants %v", err))
		return nil, errorsx.WrapInternal(err, "查询租户分页列表失败")
	}
	return page, nil
}

// GetBaseTenant 查询租户。
func (s *BaseTenantService) GetBaseTenant(ctx context.Context, req *adminv1.GetBaseTenantRequest) (*adminv1.BaseTenantForm, error) {
	baseTenant, err := s.baseTenantCase.GetBaseTenant(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseTenant %v", err))
		return nil, errorsx.WrapInternal(err, "查询租户失败")
	}
	return baseTenant, nil
}

// CreateBaseTenant 创建租户。
func (s *BaseTenantService) CreateBaseTenant(ctx context.Context, req *adminv1.CreateBaseTenantRequest) (*emptypb.Empty, error) {
	err := s.baseTenantCase.CreateBaseTenant(ctx, req.GetBaseTenant())
	if err != nil {
		log.Error(fmt.Sprintf("CreateBaseTenant %v", err))
		return nil, errorsx.WrapInternal(err, "创建租户失败")
	}
	return new(emptypb.Empty), nil
}

// UpdateBaseTenant 更新租户。
func (s *BaseTenantService) UpdateBaseTenant(ctx context.Context, req *adminv1.UpdateBaseTenantRequest) (*emptypb.Empty, error) {
	err := s.baseTenantCase.UpdateBaseTenant(ctx, req.GetBaseTenant())
	if err != nil {
		log.Error(fmt.Sprintf("UpdateBaseTenant %v", err))
		return nil, errorsx.WrapInternal(err, "更新租户失败")
	}
	return new(emptypb.Empty), nil
}

// DeleteBaseTenant 删除租户。
func (s *BaseTenantService) DeleteBaseTenant(ctx context.Context, req *adminv1.DeleteBaseTenantRequest) (*emptypb.Empty, error) {
	err := s.baseTenantCase.DeleteBaseTenant(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("DeleteBaseTenant %v", err))
		return nil, errorsx.WrapInternal(err, "删除租户失败")
	}
	return new(emptypb.Empty), nil
}

// SetBaseTenantStatus 设置状态。
func (s *BaseTenantService) SetBaseTenantStatus(ctx context.Context, req *adminv1.SetBaseTenantStatusRequest) (*emptypb.Empty, error) {
	err := s.baseTenantCase.SetBaseTenantStatus(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SetBaseTenantStatus %v", err))
		return nil, errorsx.WrapInternal(err, "设置状态失败")
	}
	return new(emptypb.Empty), nil
}

package admin

import (
	"context"

	shopadminv1 "shop/api/gen/go/shop/admin/v1"
	"shop/pkg/errorsx"
	"shop/service/shop/admin/biz"

	"github.com/go-kratos/kratos/v3/log"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

const _ = grpc.SupportPackageIsVersion7

// TenantStoreService Admin租户门店服务。
type TenantStoreService struct {
	shopadminv1.UnimplementedTenantStoreServiceServer
	tenantStoreCase *biz.TenantStoreCase
}

// NewTenantStoreService 创建Admin租户门店服务。
func NewTenantStoreService(tenantStoreCase *biz.TenantStoreCase) *TenantStoreService {
	return &TenantStoreService{tenantStoreCase: tenantStoreCase}
}

// OptionTenantStore 查询租户门店下拉选项。
func (s *TenantStoreService) OptionTenantStore(ctx context.Context, req *shopadminv1.OptionTenantStoreRequest) (*shopadminv1.OptionTenantStoreResponse, error) {
	res, err := s.tenantStoreCase.OptionTenantStore(ctx, req)
	if err != nil {
		log.Error("OptionTenantStore", err)
		return nil, errorsx.WrapInternal(err, "查询租户门店下拉选项失败")
	}
	return res, nil
}

// TreeTenantStore 查询租户门店树形选项。
func (s *TenantStoreService) TreeTenantStore(ctx context.Context, req *shopadminv1.TreeTenantStoreRequest) (*shopadminv1.TreeTenantStoreResponse, error) {
	res, err := s.tenantStoreCase.TreeTenantStore(ctx, req)
	if err != nil {
		log.Error("TreeTenantStore", err)
		return nil, errorsx.WrapInternal(err, "查询租户门店树形选项失败")
	}
	return res, nil
}

// PageTenantStore 查询租户门店列表。
func (s *TenantStoreService) PageTenantStore(ctx context.Context, req *shopadminv1.PageTenantStoreRequest) (*shopadminv1.PageTenantStoreResponse, error) {
	res, err := s.tenantStoreCase.PageTenantStore(ctx, req)
	if err != nil {
		log.Error("PageTenantStore", err)
		return nil, errorsx.WrapInternal(err, "查询租户门店列表失败")
	}
	return res, nil
}

// GetTenantStore 查询租户门店。
func (s *TenantStoreService) GetTenantStore(ctx context.Context, req *shopadminv1.GetTenantStoreRequest) (*shopadminv1.TenantStoreForm, error) {
	res, err := s.tenantStoreCase.GetTenantStore(ctx, req.GetId())
	if err != nil {
		log.Error("GetTenantStore", err)
		return nil, errorsx.WrapInternal(err, "查询租户门店失败")
	}
	return res, nil
}

// CreateTenantStore 创建租户门店。
func (s *TenantStoreService) CreateTenantStore(ctx context.Context, req *shopadminv1.CreateTenantStoreRequest) (*emptypb.Empty, error) {
	tenantStore := req.GetTenantStore()
	if tenantStore == nil {
		return nil, errorsx.InvalidArgument("请填写门店信息")
	}
	err := s.tenantStoreCase.CreateTenantStore(ctx, tenantStore)
	if err != nil {
		log.Error("CreateTenantStore", err)
		return nil, errorsx.WrapInternal(err, "创建租户门店失败")
	}
	return new(emptypb.Empty), nil
}

// UpdateTenantStore 更新租户门店。
func (s *TenantStoreService) UpdateTenantStore(ctx context.Context, req *shopadminv1.UpdateTenantStoreRequest) (*emptypb.Empty, error) {
	tenantStore := req.GetTenantStore()
	if tenantStore == nil {
		return nil, errorsx.InvalidArgument("请填写门店信息")
	}
	tenantStore.Id = req.GetId()
	err := s.tenantStoreCase.UpdateTenantStore(ctx, tenantStore)
	if err != nil {
		log.Error("UpdateTenantStore", err)
		return nil, errorsx.WrapInternal(err, "更新租户门店失败")
	}
	return new(emptypb.Empty), nil
}

// DeleteTenantStore 删除租户门店。
func (s *TenantStoreService) DeleteTenantStore(ctx context.Context, req *shopadminv1.DeleteTenantStoreRequest) (*emptypb.Empty, error) {
	err := s.tenantStoreCase.DeleteTenantStore(ctx, req.GetIds())
	if err != nil {
		log.Error("DeleteTenantStore", err)
		return nil, errorsx.WrapInternal(err, "删除租户门店失败")
	}
	return new(emptypb.Empty), nil
}

// AuditTenantStore 审核租户门店。
func (s *TenantStoreService) AuditTenantStore(ctx context.Context, req *shopadminv1.AuditTenantStoreRequest) (*emptypb.Empty, error) {
	err := s.tenantStoreCase.AuditTenantStore(ctx, req)
	if err != nil {
		log.Error("AuditTenantStore", err)
		return nil, errorsx.WrapInternal(err, "审核租户门店失败")
	}
	return new(emptypb.Empty), nil
}

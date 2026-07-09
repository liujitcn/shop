package admin

import (
	"context"

	adminv1 "shop/api/gen/go/admin/v1"
	"shop/pkg/errorsx"
	"shop/service/admin/biz"

	"github.com/go-kratos/kratos/v3/log"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

const _ = grpc.SupportPackageIsVersion7

// TenantStoreService Admin租户门店服务。
type TenantStoreService struct {
	adminv1.UnimplementedTenantStoreServiceServer
	tenantStoreCase *biz.TenantStoreCase
}

// NewTenantStoreService 创建Admin租户门店服务。
func NewTenantStoreService(tenantStoreCase *biz.TenantStoreCase) *TenantStoreService {
	return &TenantStoreService{tenantStoreCase: tenantStoreCase}
}

// OptionTenantStores 查询租户门店下拉选项。
func (s *TenantStoreService) OptionTenantStores(ctx context.Context, req *adminv1.OptionTenantStoresRequest) (*adminv1.OptionTenantStoresResponse, error) {
	res, err := s.tenantStoreCase.OptionTenantStores(ctx, req)
	if err != nil {
		log.Error("OptionTenantStores", err)
		return nil, errorsx.WrapInternal(err, "查询租户门店下拉选项失败")
	}
	return res, nil
}

// TreeTenantStores 查询租户门店树形选项。
func (s *TenantStoreService) TreeTenantStores(ctx context.Context, req *adminv1.TreeTenantStoresRequest) (*adminv1.TreeTenantStoresResponse, error) {
	res, err := s.tenantStoreCase.TreeTenantStores(ctx, req)
	if err != nil {
		log.Error("TreeTenantStores", err)
		return nil, errorsx.WrapInternal(err, "查询租户门店树形选项失败")
	}
	return res, nil
}

// PageTenantStores 查询租户门店列表。
func (s *TenantStoreService) PageTenantStores(ctx context.Context, req *adminv1.PageTenantStoresRequest) (*adminv1.PageTenantStoresResponse, error) {
	res, err := s.tenantStoreCase.PageTenantStores(ctx, req)
	if err != nil {
		log.Error("PageTenantStores", err)
		return nil, errorsx.WrapInternal(err, "查询租户门店列表失败")
	}
	return res, nil
}

// GetTenantStore 查询租户门店。
func (s *TenantStoreService) GetTenantStore(ctx context.Context, req *adminv1.GetTenantStoreRequest) (*adminv1.TenantStoreForm, error) {
	res, err := s.tenantStoreCase.GetTenantStore(ctx, req.GetId())
	if err != nil {
		log.Error("GetTenantStore", err)
		return nil, errorsx.WrapInternal(err, "查询租户门店失败")
	}
	return res, nil
}

// CreateTenantStore 创建租户门店。
func (s *TenantStoreService) CreateTenantStore(ctx context.Context, req *adminv1.CreateTenantStoreRequest) (*emptypb.Empty, error) {
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
func (s *TenantStoreService) UpdateTenantStore(ctx context.Context, req *adminv1.UpdateTenantStoreRequest) (*emptypb.Empty, error) {
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
func (s *TenantStoreService) DeleteTenantStore(ctx context.Context, req *adminv1.DeleteTenantStoreRequest) (*emptypb.Empty, error) {
	err := s.tenantStoreCase.DeleteTenantStore(ctx, req.GetIds())
	if err != nil {
		log.Error("DeleteTenantStore", err)
		return nil, errorsx.WrapInternal(err, "删除租户门店失败")
	}
	return new(emptypb.Empty), nil
}

// AuditTenantStore 审核租户门店。
func (s *TenantStoreService) AuditTenantStore(ctx context.Context, req *adminv1.AuditTenantStoreRequest) (*emptypb.Empty, error) {
	err := s.tenantStoreCase.AuditTenantStore(ctx, req)
	if err != nil {
		log.Error("AuditTenantStore", err)
		return nil, errorsx.WrapInternal(err, "审核租户门店失败")
	}
	return new(emptypb.Empty), nil
}

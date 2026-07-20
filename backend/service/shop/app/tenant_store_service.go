package app

import (
	"context"

	shopappv1 "shop/api/gen/go/shop/app/v1"
	"shop/pkg/errorsx"
	"shop/service/shop/app/biz"

	"github.com/go-kratos/kratos/v3/log"
	"google.golang.org/grpc"
)

const _ = grpc.SupportPackageIsVersion7

// TenantStoreService App租户门店服务。
type TenantStoreService struct {
	shopappv1.UnimplementedTenantStoreServiceServer
	tenantStoreCase *biz.TenantStoreCase
}

// NewTenantStoreService 创建App租户门店服务。
func NewTenantStoreService(tenantStoreCase *biz.TenantStoreCase) *TenantStoreService {
	return &TenantStoreService{tenantStoreCase: tenantStoreCase}
}

// GetTenantStore 查询租户门店首页。
func (s *TenantStoreService) GetTenantStore(ctx context.Context, req *shopappv1.GetTenantStoreRequest) (*shopappv1.TenantStore, error) {
	res, err := s.tenantStoreCase.GetTenantStore(ctx, req.GetId())
	if err != nil {
		log.Error("GetTenantStore", err)
		return nil, errorsx.WrapInternal(err, "查询租户门店首页失败")
	}
	return res, nil
}

package biz

import (
	"context"

	shopappv1 "shop/api/gen/go/shop/app/v1"
	"shop/pkg/biz"
	_const "shop/service/shop/consts"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
)

// TenantStoreCase 商城端租户门店业务实例。
type TenantStoreCase struct {
	*biz.BaseCase
	*data.TenantStoreRepository
	mapper *mapper.CopierMapper[shopappv1.TenantStore, models.TenantStore]
}

// NewTenantStoreCase 创建商城端租户门店业务实例。
func NewTenantStoreCase(baseCase *biz.BaseCase, tenantStoreRepo *data.TenantStoreRepository) *TenantStoreCase {
	tenantStoreMapper := mapper.NewCopierMapper[shopappv1.TenantStore, models.TenantStore]()
	tenantStoreMapper.AppendConverters(mapper.NewJSONTypeConverter[[]string]().NewConverterPair())
	return &TenantStoreCase{
		BaseCase:              baseCase,
		TenantStoreRepository: tenantStoreRepo,
		mapper:                tenantStoreMapper,
	}
}

// GetTenantStore 查询可展示的租户门店首页资料。
func (c *TenantStoreCase) GetTenantStore(ctx context.Context, id int64) (*shopappv1.TenantStore, error) {
	query := c.Query(ctx).TenantStore
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.ID.Eq(id)))
	opts = append(opts, repository.Where(query.Status.Eq(_const.TENANT_STORE_STATUS_APPROVED)))
	tenantStore, err := c.Find(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return c.mapper.ToDTO(tenantStore), nil
}

// GetTenantStoreMapByIDs 查询门店。
func (c *TenantStoreCase) GetTenantStoreMapByIDs(ctx context.Context, ids []int64) (map[int64]*models.TenantStore, error) {
	tenantStores, err := c.ListByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	res := make(map[int64]*models.TenantStore, len(tenantStores))
	for _, item := range tenantStores {
		res[item.ID] = item
	}
	return res, nil
}

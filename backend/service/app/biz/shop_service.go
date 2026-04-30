package biz

import (
	"context"

	_const "shop/pkg/const"

	appv1 "shop/api/gen/go/app/v1"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
)

// ShopServiceCase 商城服务说明项业务处理对象
type ShopServiceCase struct {
	*biz.BaseCase
	*data.ShopServiceRepository
	mapper *mapper.CopierMapper[appv1.ShopService, models.ShopService]
}

// NewShopServiceCase 创建商城服务说明项业务处理对象
func NewShopServiceCase(baseCase *biz.BaseCase, shopServiceRepo *data.ShopServiceRepository) *ShopServiceCase {
	return &ShopServiceCase{
		BaseCase:              baseCase,
		ShopServiceRepository: shopServiceRepo,
		mapper:                mapper.NewCopierMapper[appv1.ShopService, models.ShopService](),
	}
}

// ListShopServices 查询商城服务列表
func (c *ShopServiceCase) ListShopServices(ctx context.Context) (*appv1.ListShopServicesResponse, error) {
	query := c.Query(ctx).ShopService
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Order(query.Sort.Asc()))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	opts = append(opts, repository.Where(query.Status.Eq(_const.STATUS_ENABLE)))
	all, err := c.ShopServiceRepository.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	list := make([]*appv1.ShopService, 0, len(all))
	for _, item := range all {
		list = append(list, c.mapper.ToDTO(item))
	}

	return &appv1.ListShopServicesResponse{
		ShopServices: list,
	}, nil
}

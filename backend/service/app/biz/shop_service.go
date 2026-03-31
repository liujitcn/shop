package biz

import (
	"context"

	"shop/api/gen/go/app"
	"shop/api/gen/go/common"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/gorm-kit/repo"
)

// ShopServiceCase 商城服务说明项业务处理对象
type ShopServiceCase struct {
	*biz.BaseCase
	*data.ShopServiceRepo
}

// NewShopServiceCase 创建商城服务说明项业务处理对象
func NewShopServiceCase(baseCase *biz.BaseCase, shopServiceRepo *data.ShopServiceRepo) *ShopServiceCase {
	return &ShopServiceCase{
		BaseCase:        baseCase,
		ShopServiceRepo: shopServiceRepo,
	}
}

// ListShopService 查询商城服务列表
func (c *ShopServiceCase) ListShopService(ctx context.Context) (*app.ListShopServiceResponse, error) {
	query := c.Query(ctx).ShopService
	opts := make([]repo.QueryOption, 0, 3)
	opts = append(opts, repo.Order(query.Sort.Asc()))
	opts = append(opts, repo.Order(query.UpdatedAt.Desc()))
	opts = append(opts, repo.Where(query.Status.Eq(int32(common.Status_ENABLE))))
	all, err := c.ShopServiceRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	list := make([]*app.ShopService, 0, len(all))
	for _, item := range all {
		list = append(list, c.convertToProto(ctx, item))
	}

	return &app.ListShopServiceResponse{
		List: list,
	}, nil
}

// 将商城服务模型转换为接口响应
func (c *ShopServiceCase) convertToProto(ctx context.Context, item *models.ShopService) *app.ShopService {
	return &app.ShopService{
		Label: item.Label,
		Value: item.Value,
	}
}

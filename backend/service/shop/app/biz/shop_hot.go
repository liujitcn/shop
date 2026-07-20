package biz

import (
	"context"

	_const "shop/pkg/const"

	shopappv1 "shop/api/gen/go/shop/app/v1"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
)

// ShopHotCase 热门推荐分组业务处理对象
type ShopHotCase struct {
	*biz.BaseCase
	*data.ShopHotRepository
	mapper *mapper.CopierMapper[shopappv1.ShopHot, models.ShopHot]
}

// NewShopHotCase 创建热门推荐分组业务处理对象
func NewShopHotCase(baseCase *biz.BaseCase, shopHotRepo *data.ShopHotRepository) *ShopHotCase {
	return &ShopHotCase{
		BaseCase:          baseCase,
		ShopHotRepository: shopHotRepo,
		mapper: func() *mapper.CopierMapper[shopappv1.ShopHot, models.ShopHot] {
			m := mapper.NewCopierMapper[shopappv1.ShopHot, models.ShopHot]()
			m.AppendConverters(mapper.NewJSONTypeConverter[[]string]().NewConverterPair())
			return m
		}(),
	}
}

// ListShopHot 查询热门推荐列表
func (c *ShopHotCase) ListShopHot(ctx context.Context) (*shopappv1.ListShopHotResponse, error) {
	query := c.Query(ctx).ShopHot
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Order(query.Sort.Asc()))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	opts = append(opts, repository.Where(query.Status.Eq(_const.STATUS_ENABLE)))
	all, err := c.ShopHotRepository.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	list := make([]*shopappv1.ShopHot, 0, len(all))
	for _, item := range all {
		list = append(list, c.mapper.ToDTO(item))
	}

	return &shopappv1.ListShopHotResponse{
		ShopHots: list,
	}, nil
}

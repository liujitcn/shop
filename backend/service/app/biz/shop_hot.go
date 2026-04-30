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

// ShopHotCase 热门推荐分组业务处理对象
type ShopHotCase struct {
	*biz.BaseCase
	*data.ShopHotRepository
	mapper *mapper.CopierMapper[appv1.ShopHot, models.ShopHot]
}

// NewShopHotCase 创建热门推荐分组业务处理对象
func NewShopHotCase(baseCase *biz.BaseCase, shopHotRepo *data.ShopHotRepository) *ShopHotCase {
	return &ShopHotCase{
		BaseCase:          baseCase,
		ShopHotRepository: shopHotRepo,
		mapper: func() *mapper.CopierMapper[appv1.ShopHot, models.ShopHot] {
			m := mapper.NewCopierMapper[appv1.ShopHot, models.ShopHot]()
			m.AppendConverters(mapper.NewJSONTypeConverter[[]string]().NewConverterPair())
			return m
		}(),
	}
}

// ListShopHots 查询热门推荐列表
func (c *ShopHotCase) ListShopHots(ctx context.Context) (*appv1.ListShopHotsResponse, error) {
	query := c.Query(ctx).ShopHot
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Order(query.Sort.Asc()))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	opts = append(opts, repository.Where(query.Status.Eq(_const.STATUS_ENABLE)))
	all, err := c.ShopHotRepository.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	list := make([]*appv1.ShopHot, 0, len(all))
	for _, item := range all {
		list = append(list, c.mapper.ToDTO(item))
	}

	return &appv1.ListShopHotsResponse{
		ShopHots: list,
	}, nil
}

// 按编号查询启用中的热门推荐分组
func (c *ShopHotCase) findByID(ctx context.Context, id int64) (*models.ShopHot, error) {
	query := c.Query(ctx).ShopHot
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.ID.Eq(id)))
	opts = append(opts, repository.Where(query.Status.Eq(_const.STATUS_ENABLE)))
	return c.Find(ctx, opts...)
}

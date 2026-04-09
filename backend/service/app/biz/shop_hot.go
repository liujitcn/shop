package biz

import (
	"context"

	"shop/api/gen/go/app"
	"shop/api/gen/go/common"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repo"
)

// ShopHotCase 热门推荐分组业务处理对象
type ShopHotCase struct {
	*biz.BaseCase
	*data.ShopHotRepo
	mapper *mapper.CopierMapper[app.ShopHot, models.ShopHot]
}

// NewShopHotCase 创建热门推荐分组业务处理对象
func NewShopHotCase(baseCase *biz.BaseCase, shopHotRepo *data.ShopHotRepo) *ShopHotCase {
	return &ShopHotCase{
		BaseCase:    baseCase,
		ShopHotRepo: shopHotRepo,
		mapper: func() *mapper.CopierMapper[app.ShopHot, models.ShopHot] {
			m := mapper.NewCopierMapper[app.ShopHot, models.ShopHot]()
			m.AppendConverters(mapper.NewJSONTypeConverter[[]string]().NewConverterPair())
			return m
		}(),
	}
}

// ListShopHot 查询热门推荐列表
func (c *ShopHotCase) ListShopHot(ctx context.Context) (*app.ListShopHotResponse, error) {
	query := c.Query(ctx).ShopHot
	opts := make([]repo.QueryOption, 0, 3)
	opts = append(opts, repo.Order(query.Sort.Asc()))
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))
	opts = append(opts, repo.Where(query.Status.Eq(int32(common.Status_ENABLE))))
	all, err := c.ShopHotRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	list := make([]*app.ShopHot, 0, len(all))
	for _, item := range all {
		list = append(list, c.mapper.ToDTO(item))
	}

	return &app.ListShopHotResponse{
		List: list,
	}, nil
}

// 按编号查询启用中的热门推荐分组
func (c *ShopHotCase) findById(ctx context.Context, id int64) (*models.ShopHot, error) {
	query := c.Query(ctx).ShopHot
	return c.Find(ctx,
		repo.Where(query.ID.Eq(id)),
		repo.Where(query.Status.Eq(int32(common.Status_ENABLE))),
	)
}

package biz

import (
	"context"

	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"shop/api/gen/go/app"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repo"
)

// GoodsPropCase 商品属性业务处理对象
type GoodsPropCase struct {
	*biz.BaseCase
	*data.GoodsPropRepo
	mapper *mapper.CopierMapper[app.GoodsInfoResponse_Prop, models.GoodsProp]
}

// NewGoodsPropCase 创建商品属性业务处理对象
func NewGoodsPropCase(baseCase *biz.BaseCase, goodsPropRepo *data.GoodsPropRepo) *GoodsPropCase {
	return &GoodsPropCase{
		BaseCase:      baseCase,
		GoodsPropRepo: goodsPropRepo,
		mapper:        mapper.NewCopierMapper[app.GoodsInfoResponse_Prop, models.GoodsProp](),
	}
}

// 查询商品属性列表
func (c *GoodsPropCase) listByGoodsId(ctx context.Context, goodsId int64) ([]*app.GoodsInfoResponse_Prop, error) {
	query := c.Query(ctx).GoodsProp
	opts := make([]repo.QueryOption, 0, 2)
	opts = append(opts, repo.Order(query.Sort.Asc()))
	opts = append(opts, repo.Where(query.GoodsID.Eq(goodsId)))
	all, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	list := make([]*app.GoodsInfoResponse_Prop, 0)
	for _, item := range all {
		list = append(list, c.mapper.ToDTO(item))
	}
	return list, nil
}

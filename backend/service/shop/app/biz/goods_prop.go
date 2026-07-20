package biz

import (
	"context"

	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	shopappv1 "shop/api/gen/go/shop/app/v1"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
)

// GoodsPropCase 商品属性业务处理对象
type GoodsPropCase struct {
	*biz.BaseCase
	*data.GoodsPropRepository
	mapper *mapper.CopierMapper[shopappv1.GoodsInfoResponse_Prop, models.GoodsProp]
}

// NewGoodsPropCase 创建商品属性业务处理对象
func NewGoodsPropCase(baseCase *biz.BaseCase, goodsPropRepo *data.GoodsPropRepository) *GoodsPropCase {
	return &GoodsPropCase{
		BaseCase:            baseCase,
		GoodsPropRepository: goodsPropRepo,
		mapper:              mapper.NewCopierMapper[shopappv1.GoodsInfoResponse_Prop, models.GoodsProp](),
	}
}

// 查询商品属性列表
func (c *GoodsPropCase) listByGoodsID(ctx context.Context, goodsID int64) ([]*shopappv1.GoodsInfoResponse_Prop, error) {
	query := c.Query(ctx).GoodsProp
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Order(query.Sort.Asc()))
	opts = append(opts, repository.Where(query.GoodsID.Eq(goodsID)))
	all, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	list := make([]*shopappv1.GoodsInfoResponse_Prop, 0)
	for _, item := range all {
		list = append(list, c.mapper.ToDTO(item))
	}
	return list, nil
}

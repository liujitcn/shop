package biz

import (
	"context"

	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"shop/api/gen/go/app"

	"github.com/liujitcn/gorm-kit/repo"
)

// GoodsPropCase 商品属性业务处理对象
type GoodsPropCase struct {
	*biz.BaseCase
	*data.GoodsPropRepo
}

// NewGoodsPropCase 创建商品属性业务处理对象
func NewGoodsPropCase(baseCase *biz.BaseCase, goodsPropRepo *data.GoodsPropRepo) *GoodsPropCase {
	return &GoodsPropCase{
		BaseCase:      baseCase,
		GoodsPropRepo: goodsPropRepo,
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
		list = append(list, c.convertToProto(item))
	}
	return list, nil
}

// 将商品属性模型转换为接口响应
func (c *GoodsPropCase) convertToProto(item *models.GoodsProp) *app.GoodsInfoResponse_Prop {
	res := &app.GoodsInfoResponse_Prop{
		Label: item.Label,
		Value: item.Value,
	}
	return res
}

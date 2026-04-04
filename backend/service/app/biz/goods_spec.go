package biz

import (
	"context"

	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"shop/api/gen/go/app"

	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repo"
)

// GoodsSpecCase 商品规格业务处理对象
type GoodsSpecCase struct {
	*biz.BaseCase
	*data.GoodsSpecRepo
}

// NewGoodsSpecCase 创建商品规格业务处理对象
func NewGoodsSpecCase(baseCase *biz.BaseCase, goodsSpecRepo *data.GoodsSpecRepo) *GoodsSpecCase {
	return &GoodsSpecCase{
		BaseCase:      baseCase,
		GoodsSpecRepo: goodsSpecRepo,
	}
}

// 查询商品下的全部规格列表
func (c *GoodsSpecCase) listByGoodsId(ctx context.Context, goodsId int64) ([]*app.GoodsInfoResponse_Spec, error) {
	query := c.Query(ctx).GoodsSpec
	opts := make([]repo.QueryOption, 0, 2)
	opts = append(opts, repo.Order(query.Sort.Asc()))
	opts = append(opts, repo.Where(query.GoodsID.Eq(goodsId)))
	all, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	list := make([]*app.GoodsInfoResponse_Spec, 0)
	for _, item := range all {
		list = append(list, c.convertToProto(item))
	}
	return list, nil
}

// 将商品规格模型转换为接口响应
func (c *GoodsSpecCase) convertToProto(item *models.GoodsSpec) *app.GoodsInfoResponse_Spec {
	itemList := make([]*app.GoodsInfoResponse_Spec_Item, 0)
	items := _string.ConvertJsonStringToStringArray(item.Item)
	for _, name := range items {
		itemList = append(itemList, &app.GoodsInfoResponse_Spec_Item{
			Name: name,
		})
	}

	res := &app.GoodsInfoResponse_Spec{
		Name: item.Name,
		Item: itemList,
	}
	return res
}

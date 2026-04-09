package biz

import (
	"context"

	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"shop/api/gen/go/app"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repo"
)

// GoodsSpecCase 商品规格业务处理对象
type GoodsSpecCase struct {
	*biz.BaseCase
	*data.GoodsSpecRepo
	mapper *mapper.CopierMapper[app.GoodsInfoResponse_Spec, models.GoodsSpec]
}

// NewGoodsSpecCase 创建商品规格业务处理对象
func NewGoodsSpecCase(baseCase *biz.BaseCase, goodsSpecRepo *data.GoodsSpecRepo) *GoodsSpecCase {
	return &GoodsSpecCase{
		BaseCase:      baseCase,
		GoodsSpecRepo: goodsSpecRepo,
		mapper:        mapper.NewCopierMapper[app.GoodsInfoResponse_Spec, models.GoodsSpec](),
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
		spec := c.mapper.ToDTO(item)
		itemList := make([]*app.GoodsInfoResponse_Spec_Item, 0)
		items := _string.ConvertJsonStringToStringArray(item.Item)
		for _, name := range items {
			itemList = append(itemList, &app.GoodsInfoResponse_Spec_Item{Name: name})
		}
		spec.Item = itemList
		list = append(list, spec)
	}
	return list, nil
}

package biz

import (
	"context"

	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	appv1 "shop/api/gen/go/app/v1"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
)

// GoodsSpecCase 商品规格业务处理对象
type GoodsSpecCase struct {
	*biz.BaseCase
	*data.GoodsSpecRepository
	mapper *mapper.CopierMapper[appv1.GoodsInfoResponse_Spec, models.GoodsSpec]
}

// NewGoodsSpecCase 创建商品规格业务处理对象
func NewGoodsSpecCase(baseCase *biz.BaseCase, goodsSpecRepo *data.GoodsSpecRepository) *GoodsSpecCase {
	return &GoodsSpecCase{
		BaseCase:            baseCase,
		GoodsSpecRepository: goodsSpecRepo,
		mapper:              mapper.NewCopierMapper[appv1.GoodsInfoResponse_Spec, models.GoodsSpec](),
	}
}

// 查询商品下的全部规格列表
func (c *GoodsSpecCase) listByGoodsID(ctx context.Context, goodsID int64) ([]*appv1.GoodsInfoResponse_Spec, error) {
	query := c.Query(ctx).GoodsSpec
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Order(query.Sort.Asc()))
	opts = append(opts, repository.Where(query.GoodsID.Eq(goodsID)))
	all, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	list := make([]*appv1.GoodsInfoResponse_Spec, 0)
	for _, item := range all {
		spec := c.mapper.ToDTO(item)
		items := _string.ConvertJsonStringToStringArray(item.Item)
		itemList := make([]*appv1.GoodsInfoResponse_Spec_Item, 0, len(items))
		seenItem := make(map[string]struct{}, len(items))
		for _, name := range items {
			// 同一规格维度内重复的选项只返回一次，避免前端 SKU 弹层出现重复按钮。
			if _, ok := seenItem[name]; ok {
				continue
			}
			seenItem[name] = struct{}{}
			itemList = append(itemList, &appv1.GoodsInfoResponse_Spec_Item{Name: name})
		}
		spec.Item = itemList
		list = append(list, spec)
	}
	return list, nil
}

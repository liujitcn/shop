package biz

import (
	"context"

	"shop/api/gen/go/admin"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repo"
)

// OrderGoodsCase 订单商品业务实例
type OrderGoodsCase struct {
	*biz.BaseCase
	*data.OrderGoodsRepo
	mapper *mapper.CopierMapper[admin.OrderGoods, models.OrderGoods]
}

// NewOrderGoodsCase 创建订单商品业务实例
func NewOrderGoodsCase(baseCase *biz.BaseCase, orderGoodsRepo *data.OrderGoodsRepo) *OrderGoodsCase {
	return &OrderGoodsCase{
		BaseCase:       baseCase,
		OrderGoodsRepo: orderGoodsRepo,
		mapper:         mapper.NewCopierMapper[admin.OrderGoods, models.OrderGoods](),
	}
}

// FindFromByOrderId 按订单查询商品信息
func (c *OrderGoodsCase) FindFromByOrderId(ctx context.Context, orderId int64) ([]*admin.OrderGoods, error) {
	query := c.Query(ctx).OrderGoods
	opts := make([]repo.QueryOption, 0, 2)
	opts = append(opts, repo.Order(query.SkuCode.Asc()))
	opts = append(opts, repo.Where(query.OrderID.Eq(orderId)))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	res := make([]*admin.OrderGoods, 0, len(list))
	for _, item := range list {
		orderGoods := c.mapper.ToDTO(item)
		orderGoods.SpecItem = _string.ConvertJsonStringToStringArray(item.SpecItem)
		res = append(res, orderGoods)
	}
	return res, nil
}

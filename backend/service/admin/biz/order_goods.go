package biz

import (
	"context"

	adminv1 "shop/api/gen/go/admin/v1"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
)

// OrderGoodsCase 订单商品业务实例
type OrderGoodsCase struct {
	*biz.BaseCase
	*data.OrderGoodsRepository
	mapper *mapper.CopierMapper[adminv1.OrderGoods, models.OrderGoods]
}

// NewOrderGoodsCase 创建订单商品业务实例
func NewOrderGoodsCase(baseCase *biz.BaseCase, orderGoodsRepo *data.OrderGoodsRepository) *OrderGoodsCase {
	return &OrderGoodsCase{
		BaseCase:             baseCase,
		OrderGoodsRepository: orderGoodsRepo,
		mapper:               mapper.NewCopierMapper[adminv1.OrderGoods, models.OrderGoods](),
	}
}

// FindFromByOrderID 按订单查询商品信息
func (c *OrderGoodsCase) FindFromByOrderID(ctx context.Context, orderID int64) ([]*adminv1.OrderGoods, error) {
	query := c.Query(ctx).OrderGoods
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Order(query.SKUCode.Asc()))
	opts = append(opts, repository.Where(query.OrderID.Eq(orderID)))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	res := make([]*adminv1.OrderGoods, 0, len(list))
	for _, item := range list {
		orderGoods := c.mapper.ToDTO(item)
		orderGoods.SpecItem = _string.ConvertJsonStringToStringArray(item.SpecItem)
		res = append(res, orderGoods)
	}
	return res, nil
}

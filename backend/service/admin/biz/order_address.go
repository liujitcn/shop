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

// OrderAddressCase 订单地址业务实例
type OrderAddressCase struct {
	*biz.BaseCase
	*data.OrderAddressRepository
	mapper *mapper.CopierMapper[adminv1.OrderAddress, models.OrderAddress]
}

// NewOrderAddressCase 创建订单地址业务实例
func NewOrderAddressCase(baseCase *biz.BaseCase, orderAddressRepo *data.OrderAddressRepository) *OrderAddressCase {
	return &OrderAddressCase{
		BaseCase:               baseCase,
		OrderAddressRepository: orderAddressRepo,
		mapper:                 mapper.NewCopierMapper[adminv1.OrderAddress, models.OrderAddress](),
	}
}

// FindFromByOrderID 按订单查询地址
func (c *OrderAddressCase) FindFromByOrderID(ctx context.Context, orderID int64) (*adminv1.OrderAddress, error) {
	query := c.Query(ctx).OrderAddress
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.OrderID.Eq(orderID)))
	item, err := c.Find(ctx, opts...)
	if err != nil {
		return nil, err
	}
	orderAddress := c.mapper.ToDTO(item)
	orderAddress.Address = _string.ConvertJsonStringToStringArray(item.Address)
	return orderAddress, nil
}

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

// OrderAddressCase 订单地址业务实例
type OrderAddressCase struct {
	*biz.BaseCase
	*data.OrderAddressRepo
	mapper *mapper.CopierMapper[admin.OrderAddress, models.OrderAddress]
}

// NewOrderAddressCase 创建订单地址业务实例
func NewOrderAddressCase(baseCase *biz.BaseCase, orderAddressRepo *data.OrderAddressRepo) *OrderAddressCase {
	return &OrderAddressCase{
		BaseCase:         baseCase,
		OrderAddressRepo: orderAddressRepo,
		mapper:           mapper.NewCopierMapper[admin.OrderAddress, models.OrderAddress](),
	}
}

// FindFromByOrderId 按订单查询地址
func (c *OrderAddressCase) FindFromByOrderId(ctx context.Context, orderId int64) (*admin.OrderAddress, error) {
	query := c.Query(ctx).OrderAddress
	item, err := c.Find(ctx, repo.Where(query.OrderID.Eq(orderId)))
	if err != nil {
		return nil, err
	}
	orderAddress := c.mapper.ToDTO(item)
	orderAddress.Address = _string.ConvertJsonStringToStringArray(item.Address)
	return orderAddress, nil
}

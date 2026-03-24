package biz

import (
	"context"
	"errors"

	"shop/api/gen/go/admin"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repo"
	"gorm.io/gorm"
)

// OrderCancelCase 订单取消业务实例
type OrderCancelCase struct {
	*biz.BaseCase
	*data.OrderCancelRepo
	mapper *mapper.CopierMapper[admin.OrderCancel, models.OrderCancel]
}

// NewOrderCancelCase 创建订单取消业务实例
func NewOrderCancelCase(baseCase *biz.BaseCase, orderCancelRepo *data.OrderCancelRepo) *OrderCancelCase {
	return &OrderCancelCase{
		BaseCase:        baseCase,
		OrderCancelRepo: orderCancelRepo,
		mapper:          mapper.NewCopierMapper[admin.OrderCancel, models.OrderCancel](),
	}
}

// FindFromByOrderId 按订单查询取消信息
func (c *OrderCancelCase) FindFromByOrderId(ctx context.Context, orderId int64) (*admin.OrderCancel, error) {
	query := c.Query(ctx).OrderCancel
	item, err := c.Find(ctx, repo.Where(query.OrderID.Eq(orderId)))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &admin.OrderCancel{}, nil
		}
		return nil, err
	}
	orderCancel := c.mapper.ToDTO(item)
	return orderCancel, nil
}

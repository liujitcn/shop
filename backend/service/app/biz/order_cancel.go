package biz

import (
	"context"

	"shop/pkg/biz"
	"shop/pkg/gen/data"

	_time "github.com/liujitcn/go-utils/time"
	"github.com/liujitcn/gorm-kit/repository"
)

// OrderCancelCase 订单取消记录业务处理对象
type OrderCancelCase struct {
	*biz.BaseCase
	*data.OrderCancelRepository
}

// NewOrderCancelCase 创建订单取消记录业务处理对象
func NewOrderCancelCase(baseCase *biz.BaseCase, orderCancelRepo *data.OrderCancelRepository,
) *OrderCancelCase {
	return &OrderCancelCase{
		BaseCase:              baseCase,
		OrderCancelRepository: orderCancelRepo,
	}
}

// findCancelTimeByOrderID 查询订单取消时间
func (c *OrderCancelCase) findCancelTimeByOrderID(ctx context.Context, orderID int64) (string, error) {
	query := c.Query(ctx).OrderCancel
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.OrderID.Eq(orderID)))
	orderCancel, err := c.Find(ctx, opts...)
	if err != nil {
		return "", err
	}
	return _time.TimeToTimeString(orderCancel.CreatedAt), nil
}

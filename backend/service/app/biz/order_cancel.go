package biz

import (
	"context"

	"shop/pkg/biz"
	"shop/pkg/gen/data"

	_time "github.com/liujitcn/go-utils/time"
	"github.com/liujitcn/gorm-kit/repo"
)

// OrderCancelCase 订单取消记录业务处理对象
type OrderCancelCase struct {
	*biz.BaseCase
	*data.OrderCancelRepo
}

// NewOrderCancelCase 创建订单取消记录业务处理对象
func NewOrderCancelCase(baseCase *biz.BaseCase, orderCancelRepo *data.OrderCancelRepo,
) *OrderCancelCase {
	return &OrderCancelCase{
		BaseCase:        baseCase,
		OrderCancelRepo: orderCancelRepo,
	}
}

// findCancelTimeByOrderId 查询订单取消时间
func (c *OrderCancelCase) findCancelTimeByOrderId(ctx context.Context, orderId int64) (string, error) {
	query := c.Query(ctx).OrderCancel
	opts := make([]repo.QueryOption, 0, 1)
	opts = append(opts, repo.Where(query.OrderID.Eq(orderId)))
	orderCancel, err := c.Find(ctx, opts...)
	if err != nil {
		return "", err
	}
	return _time.TimeToTimeString(orderCancel.CreatedAt), nil
}

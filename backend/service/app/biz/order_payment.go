package biz

import (
	"context"

	"shop/pkg/biz"
	"shop/pkg/gen/data"

	_time "github.com/liujitcn/go-utils/time"
	"github.com/liujitcn/gorm-kit/repo"
)

// OrderPaymentCase 订单支付记录业务处理对象
type OrderPaymentCase struct {
	*biz.BaseCase
	*data.OrderPaymentRepo
}

// NewOrderPaymentCase 创建订单支付记录业务处理对象
func NewOrderPaymentCase(baseCase *biz.BaseCase, orderPaymentRepo *data.OrderPaymentRepo,
) *OrderPaymentCase {
	return &OrderPaymentCase{
		BaseCase:         baseCase,
		OrderPaymentRepo: orderPaymentRepo,
	}
}

// findPaymentTimeByOrderId 查询订单支付时间
func (c *OrderPaymentCase) findPaymentTimeByOrderId(ctx context.Context, orderId int64) (string, error) {
	query := c.Query(ctx).OrderPayment
	opts := make([]repo.QueryOption, 0, 1)
	opts = append(opts, repo.Where(query.OrderID.Eq(orderId)))
	orderPayment, err := c.Find(ctx, opts...)
	if err != nil {
		return "", err
	}
	return _time.TimeToTimeString(orderPayment.SuccessTime), nil
}

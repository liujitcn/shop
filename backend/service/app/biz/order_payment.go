package biz

import (
	"context"

	"shop/pkg/biz"
	"shop/pkg/gen/data"

	_time "github.com/liujitcn/go-utils/time"
	"github.com/liujitcn/gorm-kit/repository"
)

// OrderPaymentCase 订单支付记录业务处理对象
type OrderPaymentCase struct {
	*biz.BaseCase
	*data.OrderPaymentRepository
}

// NewOrderPaymentCase 创建订单支付记录业务处理对象
func NewOrderPaymentCase(baseCase *biz.BaseCase, orderPaymentRepo *data.OrderPaymentRepository,
) *OrderPaymentCase {
	return &OrderPaymentCase{
		BaseCase:               baseCase,
		OrderPaymentRepository: orderPaymentRepo,
	}
}

// findPaymentTimeByOrderID 查询订单支付时间
func (c *OrderPaymentCase) findPaymentTimeByOrderID(ctx context.Context, orderID int64) (string, error) {
	query := c.Query(ctx).OrderPayment
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.OrderID.Eq(orderID)))
	orderPayment, err := c.Find(ctx, opts...)
	if err != nil {
		return "", err
	}
	return _time.TimeToTimeString(orderPayment.SuccessTime), nil
}

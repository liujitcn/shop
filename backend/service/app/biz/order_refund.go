package biz

import (
	"context"

	"shop/pkg/biz"
	"shop/pkg/gen/data"

	_time "github.com/liujitcn/go-utils/time"
	"github.com/liujitcn/gorm-kit/repo"
)

// OrderRefundCase 订单退款记录业务处理对象
type OrderRefundCase struct {
	*biz.BaseCase
	*data.OrderRefundRepo
}

// NewOrderRefundCase 创建订单退款记录业务处理对象
func NewOrderRefundCase(baseCase *biz.BaseCase, orderRefundRepo *data.OrderRefundRepo,
) *OrderRefundCase {
	return &OrderRefundCase{
		BaseCase:        baseCase,
		OrderRefundRepo: orderRefundRepo,
	}
}

// findRefundTimeByOrderId 查询订单退款时间
func (c *OrderRefundCase) findRefundTimeByOrderId(ctx context.Context, orderId int64) (string, error) {
	refundQuery := c.Query(ctx).OrderRefund
	orderRefund, err := c.Find(ctx,
		repo.Where(refundQuery.OrderID.Eq(orderId)),
	)
	if err != nil {
		return "", err
	}
	return _time.TimeToTimeString(orderRefund.SuccessTime), nil
}

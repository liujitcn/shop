package biz

import (
	"context"

	"shop/pkg/biz"
	"shop/pkg/gen/data"

	_time "github.com/liujitcn/go-utils/time"
	"github.com/liujitcn/gorm-kit/repository"
)

// OrderRefundCase 订单退款记录业务处理对象
type OrderRefundCase struct {
	*biz.BaseCase
	*data.OrderRefundRepository
}

// NewOrderRefundCase 创建订单退款记录业务处理对象
func NewOrderRefundCase(baseCase *biz.BaseCase, orderRefundRepo *data.OrderRefundRepository,
) *OrderRefundCase {
	return &OrderRefundCase{
		BaseCase:              baseCase,
		OrderRefundRepository: orderRefundRepo,
	}
}

// findRefundTimeByOrderID 查询订单退款时间
func (c *OrderRefundCase) findRefundTimeByOrderID(ctx context.Context, orderID int64) (string, error) {
	query := c.Query(ctx).OrderRefund
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.OrderID.Eq(orderID)))
	orderRefund, err := c.Find(ctx, opts...)
	if err != nil {
		return "", err
	}
	return _time.TimeToTimeString(orderRefund.SuccessTime), nil
}

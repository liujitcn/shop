package biz

import (
	"context"
	"time"

	shopappv1 "shop/api/gen/go/shop/app/v1"

	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	_time "github.com/liujitcn/go-utils/time"
	"github.com/liujitcn/gorm-kit/repository"
	"gorm.io/gorm"
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
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.OrderID.Eq(orderID)))
	opts = append(opts, repository.Where(query.RefundState.Eq(shopappv1.RefundResource_SUCCESS.String())))
	orderRefunds, err := c.List(ctx, opts...)
	if err != nil {
		return "", err
	}
	refundTime, ok := latestSuccessfulRefundTimes(orderRefunds)[orderID]
	if !ok {
		return "", gorm.ErrRecordNotFound
	}
	return _time.TimeToTimeString(refundTime), nil
}

// mapRefundTimeByOrderIDs 按订单编号批量查询退款成功时间映射。
func (c *OrderRefundCase) mapRefundTimeByOrderIDs(ctx context.Context, orderIDs []int64) (map[int64]string, error) {
	res := make(map[int64]string)
	// 没有退款订单时直接返回空映射，避免执行无意义查询。
	if len(orderIDs) == 0 {
		return res, nil
	}

	query := c.Query(ctx).OrderRefund
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.OrderID.In(orderIDs...)))
	opts = append(opts, repository.Where(query.RefundState.Eq(shopappv1.RefundResource_SUCCESS.String())))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	for orderID, refundTime := range latestSuccessfulRefundTimes(list) {
		res[orderID] = _time.TimeToTimeString(refundTime)
	}
	return res, nil
}

// latestSuccessfulRefundTimes 按订单编号筛选并保留最新退款成功时间。
func latestSuccessfulRefundTimes(orderRefunds []*models.OrderRefund) map[int64]time.Time {
	refundTimes := make(map[int64]time.Time)
	for _, orderRefund := range orderRefunds {
		if orderRefund.RefundState != shopappv1.RefundResource_SUCCESS.String() {
			continue
		}
		refundTime, ok := refundTimes[orderRefund.OrderID]
		if !ok || orderRefund.SuccessTime.After(refundTime) {
			refundTimes[orderRefund.OrderID] = orderRefund.SuccessTime
		}
	}
	return refundTimes
}

package biz

import (
	"context"

	adminv1 "shop/api/gen/go/admin/v1"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
)

// OrderRefundCase 订单退款业务实例
type OrderRefundCase struct {
	*biz.BaseCase
	*data.OrderRefundRepository
	mapper *mapper.CopierMapper[adminv1.OrderRefund, models.OrderRefund]
}

// NewOrderRefundCase 创建订单退款业务实例
func NewOrderRefundCase(baseCase *biz.BaseCase, orderRefundRepo *data.OrderRefundRepository) *OrderRefundCase {
	orderRefundMapper := mapper.NewCopierMapper[adminv1.OrderRefund, models.OrderRefund]()
	orderRefundMapper.AppendConverters(mapper.NewJSONTypeConverter[*adminv1.OrderRefund_Amount]().NewConverterPair())
	return &OrderRefundCase{
		BaseCase:              baseCase,
		OrderRefundRepository: orderRefundRepo,
		mapper:                orderRefundMapper,
	}
}

// FindFromByOrderID 按订单查询退款信息
func (c *OrderRefundCase) FindFromByOrderID(ctx context.Context, orderID int64) ([]*adminv1.OrderRefund, error) {
	query := c.Query(ctx).OrderRefund
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.OrderID.Eq(orderID)))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	res := make([]*adminv1.OrderRefund, 0, len(list))
	for _, item := range list {
		res = append(res, c.mapper.ToDTO(item))
	}
	return res, nil
}

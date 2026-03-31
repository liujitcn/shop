package biz

import (
	"context"

	"shop/api/gen/go/admin"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repo"
)

// OrderRefundCase 订单退款业务实例
type OrderRefundCase struct {
	*biz.BaseCase
	*data.OrderRefundRepo
	mapper *mapper.CopierMapper[admin.OrderRefund, models.OrderRefund]
}

// NewOrderRefundCase 创建订单退款业务实例
func NewOrderRefundCase(baseCase *biz.BaseCase, orderRefundRepo *data.OrderRefundRepo) *OrderRefundCase {
	orderRefundMapper := mapper.NewCopierMapper[admin.OrderRefund, models.OrderRefund]()
	orderRefundMapper.AppendConverters(mapper.NewJSONTypeConverter[*admin.OrderRefund_Amount]().NewConverterPair())
	return &OrderRefundCase{
		BaseCase:        baseCase,
		OrderRefundRepo: orderRefundRepo,
		mapper:          orderRefundMapper,
	}
}

// FindFromByOrderId 按订单查询退款信息
func (c *OrderRefundCase) FindFromByOrderId(ctx context.Context, orderId int64) ([]*admin.OrderRefund, error) {
	query := c.Query(ctx).OrderRefund
	opts := make([]repo.QueryOption, 0, 1)
	opts = append(opts, repo.Where(query.OrderID.Eq(orderId)))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	res := make([]*admin.OrderRefund, 0, len(list))
	for _, item := range list {
		res = append(res, c.mapper.ToDTO(item))
	}
	return res, nil
}

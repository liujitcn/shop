package biz

import (
	"context"
	"encoding/json"

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
	return &OrderRefundCase{
		BaseCase:        baseCase,
		OrderRefundRepo: orderRefundRepo,
		mapper:          mapper.NewCopierMapper[admin.OrderRefund, models.OrderRefund](),
	}
}

// FindFromByOrderId 按订单查询退款信息
func (c *OrderRefundCase) FindFromByOrderId(ctx context.Context, orderId int64) ([]*admin.OrderRefund, error) {
	query := c.Query(ctx).OrderRefund
	list, err := c.List(ctx, repo.Where(query.OrderID.Eq(orderId)))
	if err != nil {
		return nil, err
	}

	res := make([]*admin.OrderRefund, 0, len(list))
	for _, item := range list {
		var amount admin.OrderRefund_Amount
		_ = json.Unmarshal([]byte(item.Amount), &amount)
		orderRefund := c.mapper.ToDTO(item)
		orderRefund.Amount = &amount
		res = append(res, orderRefund)
	}
	return res, nil
}

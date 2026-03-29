package biz

import (
	"context"
	"errors"

	"shop/api/gen/go/admin"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repo"
	"gorm.io/gorm"
)

// OrderPaymentCase 订单支付业务实例
type OrderPaymentCase struct {
	*biz.BaseCase
	*data.OrderPaymentRepo
	mapper *mapper.CopierMapper[admin.OrderPayment, models.OrderPayment]
}

// NewOrderPaymentCase 创建订单支付业务实例
func NewOrderPaymentCase(baseCase *biz.BaseCase, orderPaymentRepo *data.OrderPaymentRepo) *OrderPaymentCase {
	orderPaymentMapper := mapper.NewCopierMapper[admin.OrderPayment, models.OrderPayment]()
	orderPaymentMapper.AppendConverters(mapper.NewJSONTypeConverter[*admin.OrderPayment_Payer]().NewConverterPair())
	orderPaymentMapper.AppendConverters(mapper.NewJSONTypeConverter[*admin.OrderPayment_Amount]().NewConverterPair())
	orderPaymentMapper.AppendConverters(mapper.NewJSONTypeConverter[*admin.OrderPayment_SceneInfo]().NewConverterPair())
	return &OrderPaymentCase{
		BaseCase:         baseCase,
		OrderPaymentRepo: orderPaymentRepo,
		mapper:           orderPaymentMapper,
	}
}

// FindFromByOrderId 按订单查询支付信息
func (c *OrderPaymentCase) FindFromByOrderId(ctx context.Context, orderId int64) (*admin.OrderPayment, error) {
	query := c.Query(ctx).OrderPayment
	item, err := c.Find(ctx, repo.Where(query.OrderID.Eq(orderId)))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &admin.OrderPayment{}, nil
		}
		return nil, err
	}

	return c.mapper.ToDTO(item), nil
}

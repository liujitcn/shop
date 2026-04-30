package biz

import (
	"context"
	"errors"

	adminv1 "shop/api/gen/go/admin/v1"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
	"gorm.io/gorm"
)

// OrderPaymentCase 订单支付业务实例
type OrderPaymentCase struct {
	*biz.BaseCase
	*data.OrderPaymentRepository
	mapper *mapper.CopierMapper[adminv1.OrderPayment, models.OrderPayment]
}

// NewOrderPaymentCase 创建订单支付业务实例
func NewOrderPaymentCase(baseCase *biz.BaseCase, orderPaymentRepo *data.OrderPaymentRepository) *OrderPaymentCase {
	orderPaymentMapper := mapper.NewCopierMapper[adminv1.OrderPayment, models.OrderPayment]()
	orderPaymentMapper.AppendConverters(mapper.NewJSONTypeConverter[*adminv1.OrderPayment_Payer]().NewConverterPair())
	orderPaymentMapper.AppendConverters(mapper.NewJSONTypeConverter[*adminv1.OrderPayment_Amount]().NewConverterPair())
	orderPaymentMapper.AppendConverters(mapper.NewJSONTypeConverter[*adminv1.OrderPayment_SceneInfo]().NewConverterPair())
	return &OrderPaymentCase{
		BaseCase:               baseCase,
		OrderPaymentRepository: orderPaymentRepo,
		mapper:                 orderPaymentMapper,
	}
}

// FindFromByOrderID 按订单查询支付信息
func (c *OrderPaymentCase) FindFromByOrderID(ctx context.Context, orderID int64) (*adminv1.OrderPayment, error) {
	query := c.Query(ctx).OrderPayment
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.OrderID.Eq(orderID)))
	item, err := c.Find(ctx, opts...)
	// 订单支付记录查询失败时，仅对“未找到”场景回退空对象。
	if err != nil {
		// 订单未生成支付记录时，返回空对象即可。
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &adminv1.OrderPayment{}, nil
		}
		// 其他查询错误直接返回，避免吞掉真实异常。
		return nil, err
	}

	return c.mapper.ToDTO(item), nil
}

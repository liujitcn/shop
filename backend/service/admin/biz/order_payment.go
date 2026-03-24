package biz

import (
	"context"
	"encoding/json"
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
	return &OrderPaymentCase{
		BaseCase:         baseCase,
		OrderPaymentRepo: orderPaymentRepo,
		mapper:           mapper.NewCopierMapper[admin.OrderPayment, models.OrderPayment](),
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

	var payer admin.OrderPayment_Payer
	var amount admin.OrderPayment_Amount
	var sceneInfo admin.OrderPayment_SceneInfo
	_ = json.Unmarshal([]byte(item.Payer), &payer)
	_ = json.Unmarshal([]byte(item.Amount), &amount)
	_ = json.Unmarshal([]byte(item.SceneInfo), &sceneInfo)

	orderPayment := c.mapper.ToDTO(item)
	orderPayment.Payer = &payer
	orderPayment.Amount = &amount
	orderPayment.SceneInfo = &sceneInfo
	return orderPayment, nil
}

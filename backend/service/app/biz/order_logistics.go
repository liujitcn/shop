package biz

import (
	"context"
	"encoding/json"

	"shop/api/gen/go/app"
	"shop/pkg/biz"
	"shop/pkg/gen/data"

	"github.com/liujitcn/gorm-kit/repo"
)

// OrderLogisticsCase 订单物流业务处理对象
type OrderLogisticsCase struct {
	*biz.BaseCase
	*data.OrderLogisticsRepo
}

// NewOrderLogisticsCase 创建订单物流业务处理对象
func NewOrderLogisticsCase(baseCase *biz.BaseCase, orderLogisticsRepo *data.OrderLogisticsRepo,
) *OrderLogisticsCase {
	return &OrderLogisticsCase{
		BaseCase:           baseCase,
		OrderLogisticsRepo: orderLogisticsRepo,
	}
}

// findByOrderId 按订单编号查询物流信息
func (c *OrderLogisticsCase) findByOrderId(ctx context.Context, orderId int64) (*app.OrderResponse_Logistics, error) {
	query := c.Query(ctx).OrderLogistics
	orderLogistics, err := c.Find(ctx,
		repo.Where(query.OrderID.Eq(orderId)),
	)
	if err != nil {
		return nil, err
	}
	detail := make([]*app.OrderResponse_Logistics_Detail, 0)
	// 物流轨迹以序列化字符串存储，这里反序列化成前端响应结构
	err = json.Unmarshal([]byte(orderLogistics.Detail), &detail)
	if err != nil {
		return nil, err
	}
	return &app.OrderResponse_Logistics{
		Name:    orderLogistics.Name,
		No:      orderLogistics.No,
		Contact: orderLogistics.Contact,
		Detail:  detail,
	}, nil
}

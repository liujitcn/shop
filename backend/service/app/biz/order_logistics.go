package biz

import (
	"context"
	"encoding/json"

	"shop/api/gen/go/app"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repo"
)

// OrderLogisticsCase 订单物流业务处理对象
type OrderLogisticsCase struct {
	*biz.BaseCase
	*data.OrderLogisticsRepo
	mapper *mapper.CopierMapper[app.OrderInfoResponse_Logistics, models.OrderLogistics]
}

// NewOrderLogisticsCase 创建订单物流业务处理对象
func NewOrderLogisticsCase(baseCase *biz.BaseCase, orderLogisticsRepo *data.OrderLogisticsRepo,
) *OrderLogisticsCase {
	return &OrderLogisticsCase{
		BaseCase:           baseCase,
		OrderLogisticsRepo: orderLogisticsRepo,
		mapper:             mapper.NewCopierMapper[app.OrderInfoResponse_Logistics, models.OrderLogistics](),
	}
}

// findByOrderId 按订单编号查询物流信息
func (c *OrderLogisticsCase) findByOrderId(ctx context.Context, orderId int64) (*app.OrderInfoResponse_Logistics, error) {
	query := c.Query(ctx).OrderLogistics
	orderLogistics, err := c.Find(ctx,
		repo.Where(query.OrderID.Eq(orderId)),
	)
	if err != nil {
		return nil, err
	}
	detail := make([]*app.OrderInfoResponse_Logistics_Detail, 0)
	// 物流轨迹以序列化字符串存储，这里反序列化成前端响应结构
	err = json.Unmarshal([]byte(orderLogistics.Detail), &detail)
	if err != nil {
		return nil, err
	}
	res := c.mapper.ToDTO(orderLogistics)
	res.Detail = detail
	return res, nil
}

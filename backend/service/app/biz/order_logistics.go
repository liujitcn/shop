package biz

import (
	"context"
	"encoding/json"

	appv1 "shop/api/gen/go/app/v1"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
)

// OrderLogisticsCase 订单物流业务处理对象
type OrderLogisticsCase struct {
	*biz.BaseCase
	*data.OrderLogisticsRepository
	mapper *mapper.CopierMapper[appv1.OrderInfoResponse_Logistics, models.OrderLogistics]
}

// NewOrderLogisticsCase 创建订单物流业务处理对象
func NewOrderLogisticsCase(baseCase *biz.BaseCase, orderLogisticsRepo *data.OrderLogisticsRepository,
) *OrderLogisticsCase {
	return &OrderLogisticsCase{
		BaseCase:                 baseCase,
		OrderLogisticsRepository: orderLogisticsRepo,
		mapper:                   mapper.NewCopierMapper[appv1.OrderInfoResponse_Logistics, models.OrderLogistics](),
	}
}

// findByOrderID 按订单编号查询物流信息
func (c *OrderLogisticsCase) findByOrderID(ctx context.Context, orderID int64) (*appv1.OrderInfoResponse_Logistics, error) {
	query := c.Query(ctx).OrderLogistics
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.OrderID.Eq(orderID)))
	orderLogistics, err := c.Find(ctx, opts...)
	if err != nil {
		return nil, err
	}
	detail := make([]*appv1.OrderInfoResponse_Logistics_Detail, 0)
	// 物流轨迹以序列化字符串存储，这里反序列化成前端响应结构
	err = json.Unmarshal([]byte(orderLogistics.Detail), &detail)
	if err != nil {
		return nil, err
	}
	res := c.mapper.ToDTO(orderLogistics)
	res.Detail = detail
	return res, nil
}

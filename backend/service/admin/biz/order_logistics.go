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

// OrderLogisticsCase 订单物流业务实例
type OrderLogisticsCase struct {
	*biz.BaseCase
	*data.OrderLogisticsRepo
	mapper *mapper.CopierMapper[admin.OrderLogistics, models.OrderLogistics]
}

// NewOrderLogisticsCase 创建订单物流业务实例
func NewOrderLogisticsCase(baseCase *biz.BaseCase, orderLogisticsRepo *data.OrderLogisticsRepo) *OrderLogisticsCase {
	orderLogisticsMapper := mapper.NewCopierMapper[admin.OrderLogistics, models.OrderLogistics]()
	orderLogisticsMapper.AppendConverters(mapper.NewJSONTypeConverter[[]*admin.OrderLogistics_Detail]().NewConverterPair())
	return &OrderLogisticsCase{
		BaseCase:           baseCase,
		OrderLogisticsRepo: orderLogisticsRepo,
		mapper:             orderLogisticsMapper,
	}
}

// FindFromByOrderId 按订单查询物流信息
func (c *OrderLogisticsCase) FindFromByOrderId(ctx context.Context, orderId int64) (*admin.OrderLogistics, error) {
	query := c.Query(ctx).OrderLogistics
	item, err := c.Find(ctx, repo.Where(query.OrderID.Eq(orderId)))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &admin.OrderLogistics{}, nil
		}
		return nil, err
	}

	return c.mapper.ToDTO(item), nil
}

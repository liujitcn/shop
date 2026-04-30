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

// OrderLogisticsCase 订单物流业务实例
type OrderLogisticsCase struct {
	*biz.BaseCase
	*data.OrderLogisticsRepository
	mapper *mapper.CopierMapper[adminv1.OrderLogistics, models.OrderLogistics]
}

// NewOrderLogisticsCase 创建订单物流业务实例
func NewOrderLogisticsCase(baseCase *biz.BaseCase, orderLogisticsRepo *data.OrderLogisticsRepository) *OrderLogisticsCase {
	orderLogisticsMapper := mapper.NewCopierMapper[adminv1.OrderLogistics, models.OrderLogistics]()
	orderLogisticsMapper.AppendConverters(mapper.NewJSONTypeConverter[[]*adminv1.OrderLogistics_Detail]().NewConverterPair())
	return &OrderLogisticsCase{
		BaseCase:                 baseCase,
		OrderLogisticsRepository: orderLogisticsRepo,
		mapper:                   orderLogisticsMapper,
	}
}

// FindFromByOrderID 按订单查询物流信息
func (c *OrderLogisticsCase) FindFromByOrderID(ctx context.Context, orderID int64) (*adminv1.OrderLogistics, error) {
	query := c.Query(ctx).OrderLogistics
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.OrderID.Eq(orderID)))
	item, err := c.Find(ctx, opts...)
	// 订单物流记录查询失败时，仅对“未找到”场景回退空对象。
	if err != nil {
		// 订单未生成物流记录时，返回空对象即可。
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &adminv1.OrderLogistics{}, nil
		}
		// 其他查询错误直接返回，避免吞掉真实异常。
		return nil, err
	}

	return c.mapper.ToDTO(item), nil
}

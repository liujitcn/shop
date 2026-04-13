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
	opts := make([]repo.QueryOption, 0, 1)
	opts = append(opts, repo.Where(query.OrderID.Eq(orderId)))
	item, err := c.Find(ctx, opts...)
	// 订单物流记录查询失败时，仅对“未找到”场景回退空对象。
	if err != nil {
		// 订单未生成物流记录时，返回空对象即可。
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &admin.OrderLogistics{}, nil
		}
		// 其他查询错误直接返回，避免吞掉真实异常。
		return nil, err
	}

	return c.mapper.ToDTO(item), nil
}

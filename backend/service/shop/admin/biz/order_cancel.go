package biz

import (
	"context"
	"errors"

	shopadminv1 "shop/api/gen/go/shop/admin/v1"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
	"gorm.io/gorm"
)

// OrderCancelCase 订单取消业务实例
type OrderCancelCase struct {
	*biz.BaseCase
	*data.OrderCancelRepository
	mapper *mapper.CopierMapper[shopadminv1.OrderCancel, models.OrderCancel]
}

// NewOrderCancelCase 创建订单取消业务实例
func NewOrderCancelCase(baseCase *biz.BaseCase, orderCancelRepo *data.OrderCancelRepository) *OrderCancelCase {
	return &OrderCancelCase{
		BaseCase:              baseCase,
		OrderCancelRepository: orderCancelRepo,
		mapper:                mapper.NewCopierMapper[shopadminv1.OrderCancel, models.OrderCancel](),
	}
}

// FindFromByTradeID 按交易单查询取消信息。
func (c *OrderCancelCase) FindFromByTradeID(ctx context.Context, tradeID int64) (*shopadminv1.OrderCancel, error) {
	query := c.Query(ctx).OrderCancel
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.TradeID.Eq(tradeID)))
	item, err := c.Find(ctx, opts...)
	// 订单取消记录查询失败时，仅对“未找到”场景回退空对象。
	if err != nil {
		// 订单未生成取消记录时，返回空对象即可。
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &shopadminv1.OrderCancel{}, nil
		}
		// 其他查询错误直接返回，避免吞掉真实异常。
		return nil, err
	}
	orderCancel := c.mapper.ToDTO(item)
	return orderCancel, nil
}

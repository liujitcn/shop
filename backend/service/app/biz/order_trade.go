package biz

import (
	"context"

	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/gorm-kit/repository"
)

// OrderTradeCase 处理订单交易单业务。
type OrderTradeCase struct {
	*biz.BaseCase
	*data.OrderTradeRepository
}

// NewOrderTradeCase 创建订单交易单业务处理对象。
func NewOrderTradeCase(baseCase *biz.BaseCase, orderTradeRepo *data.OrderTradeRepository) *OrderTradeCase {
	return &OrderTradeCase{
		BaseCase:             baseCase,
		OrderTradeRepository: orderTradeRepo,
	}
}

// findByUserIDAndID 按用户和交易单 ID 查询交易单。
func (c *OrderTradeCase) findByUserIDAndID(ctx context.Context, userID, tradeID int64) (*models.OrderTrade, error) {
	query := c.Query(ctx).OrderTrade
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.ID.Eq(tradeID)))
	opts = append(opts, repository.Where(query.UserID.Eq(userID)))
	return c.Find(ctx, opts...)
}

// findByTradeNo 按交易单编号查询交易单。
func (c *OrderTradeCase) findByTradeNo(ctx context.Context, tradeNo string) (*models.OrderTrade, error) {
	query := c.Query(ctx).OrderTrade
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.TradeNo.Eq(tradeNo)))
	return c.Find(ctx, opts...)
}

package contract

import (
	"context"
	"time"
)

// OrderGoods 表示订单上下文中的商品数据。
type OrderGoods struct {
	OrderId    int64
	GoodsId    int64
	CategoryId int64
	GoodsNum   int64
	PaidAt     time.Time
}

// OrderSource 定义推荐所需的订单数据来源。
type OrderSource interface {
	ListOrderGoods(context.Context, int64) ([]*OrderGoods, error)
	ListRecentPaidGoods(context.Context, int64, time.Time, int32) ([]*OrderGoods, error)
}

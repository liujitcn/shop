package contract

import (
	"context"
	"time"
)

// OrderGoods 表示订单上下文中的商品数据。
type OrderGoods struct {
	// OrderId 表示订单编号。
	OrderId int64
	// GoodsId 表示订单商品编号。
	GoodsId int64
	// CategoryId 表示订单商品所属类目编号。
	CategoryId int64
	// GoodsNum 表示订单中的购买数量。
	GoodsNum int64
	// PaidAt 表示订单支付时间。
	PaidAt time.Time
}

// OrderSource 定义推荐所需的订单数据来源。
type OrderSource interface {
	ListOrderGoods(context.Context, int64) ([]*OrderGoods, error)
	ListRecentPaidGoods(context.Context, int64, time.Time, int32) ([]*OrderGoods, error)
}

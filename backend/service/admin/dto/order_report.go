package dto

import "time"

// OrderPaidFact 表示一笔交易已经形成的支付事实。
type OrderPaidFact struct {
	TradeID  int64
	OrderID  int64
	UserID   int64
	PayMoney int64
	PaidAt   time.Time
}

// OrderReportPaidMetrics 表示订单报表按支付事实汇总的订单和用户指标。
type OrderReportPaidMetrics struct {
	PeriodOrderCounts map[string]int64
	PeriodUserCounts  map[string]int64
	TotalOrderCount   int64
	TotalUserCount    int64
}

// OrderMonthReportRow 订单月报聚合行
type OrderMonthReportRow struct {
	// Month 月份，格式：YYYY-MM
	Month string `gorm:"column:month"`
	// PaidOrderCount 支付成功订单数
	PaidOrderCount int64 `gorm:"column:paid_order_count"`
	// PaidOrderAmount 支付成功金额，单位分
	PaidOrderAmount int64 `gorm:"column:paid_order_amount"`
	// RefundOrderCount 退款成功订单数
	RefundOrderCount int64 `gorm:"column:refund_order_count"`
	// RefundOrderAmount 退款成功金额，单位分
	RefundOrderAmount int64 `gorm:"column:refund_order_amount"`
	// PaidUserCount 支付用户数
	PaidUserCount int64 `gorm:"column:paid_user_count"`
	// GoodsCount 商品件数
	GoodsCount int64 `gorm:"column:goods_count"`
}

// OrderDayReportRow 订单日报聚合行
type OrderDayReportRow struct {
	// Day 日期，格式：YYYY-MM-DD
	Day string `gorm:"column:day"`
	// PaidOrderCount 支付成功订单数
	PaidOrderCount int64 `gorm:"column:paid_order_count"`
	// PaidOrderAmount 支付成功金额，单位分
	PaidOrderAmount int64 `gorm:"column:paid_order_amount"`
	// RefundOrderCount 退款成功订单数
	RefundOrderCount int64 `gorm:"column:refund_order_count"`
	// RefundOrderAmount 退款成功金额，单位分
	RefundOrderAmount int64 `gorm:"column:refund_order_amount"`
	// PaidUserCount 支付用户数
	PaidUserCount int64 `gorm:"column:paid_user_count"`
	// GoodsCount 商品件数
	GoodsCount int64 `gorm:"column:goods_count"`
}

package dto

// OrderStatusCountRow 保存订单状态聚合数量行。
type OrderStatusCountRow struct {
	// Status 订单状态。
	Status int32 `gorm:"column:status"`
	// Total 当前状态的订单数量。
	Total int64 `gorm:"column:total"`
}

package dto

import (
	"time"

	appv1 "shop/api/gen/go/app/v1"
)

// OrderStatusCountRow 保存订单状态聚合数量行。
type OrderStatusCountRow struct {
	// Status 订单状态。
	Status int32 `gorm:"column:status"`
	// Total 当前状态的订单数量。
	Total int64 `gorm:"column:total"`
}

// OrderPageItem 保存跨交易和门店订单合并分页的记录。
type OrderPageItem struct {
	// OrderInfo 是接口最终返回的订单记录。
	OrderInfo *appv1.OrderInfo
	// CreatedAt 用于交易和门店订单统一排序。
	CreatedAt time.Time
}

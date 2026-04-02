package dto

// CountResult 统计结果
// 用于聚合新增数量和总数量，避免在多个方法中重复定义
type CountResult struct {
	// NewNum 指定时间范围内的新增数量
	NewNum int64 `gorm:"column:new_num"`
	// TotalNum 总数量
	TotalNum int64 `gorm:"column:total_num"`
}

// OrderSummary 月份订单汇总
type OrderSummary struct {
	Key        int64 `json:"key"`
	OrderCount int64 `json:"order_count"`
	SaleAmount int64 `json:"sale_amount"`
}

// OrderGoodsStatusSummary 订单商品状态汇总
type OrderGoodsStatusSummary struct {
	Status     int32 `json:"status"`
	GoodsCount int64 `json:"goods_count"`
	CategoryId int64 `json:"category_id"`
}

// OrderGoodsSummary 商品销量汇总
type OrderGoodsSummary struct {
	GoodsCount int64 `json:"goods_count"`
	GoodsId    int64 `json:"goods_id"`
}

// GoodsCategorySummary 商品分类汇总
type GoodsCategorySummary struct {
	GoodsCount int64 `json:"goods_count"`
	CategoryId int64 `json:"category_id"`
}

// OrderStatusSummary 订单状态汇总
type OrderStatusSummary struct {
	OrderCount int64 `json:"order_count"`
	Status     int32 `json:"status"`
}

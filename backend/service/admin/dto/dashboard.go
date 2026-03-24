package dto

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

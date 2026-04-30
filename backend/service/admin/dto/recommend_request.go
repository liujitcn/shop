package dto

// RecommendRequestEventCountRow 表示推荐商品事件聚合结果。
type RecommendRequestEventCountRow struct {
	// GoodsID 商品ID。
	GoodsID int64 `gorm:"column:goods_id"`
	// Position 结果位置。
	Position int32 `gorm:"column:position"`
	// EventCount 事件数量。
	EventCount int64 `gorm:"column:event_count"`
}

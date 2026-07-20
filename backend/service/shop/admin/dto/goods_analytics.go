package dto

// GoodsAnalyticsSummaryRow 商品分析行为汇总行。
type GoodsAnalyticsSummaryRow struct {
	// 浏览次数
	ViewCount int64 `gorm:"column:view_count"`
	// 收藏次数
	CollectCount int64 `gorm:"column:collect_count"`
	// 加购件数
	CartCount int64 `gorm:"column:cart_count"`
	// 下单次数
	OrderCount int64 `gorm:"column:order_count"`
	// 支付次数
	PayCount int64 `gorm:"column:pay_count"`
	// 支付件数
	PayGoodsNum int64 `gorm:"column:pay_goods_num"`
	// 支付金额，单位分
	PayAmount int64 `gorm:"column:pay_amount"`
}

// GoodsAnalyticsTrendRow 商品分析趋势聚合行。
type GoodsAnalyticsTrendRow struct {
	// 趋势桶位键值
	Key int64 `gorm:"column:key"`
	// 浏览次数
	ViewCount int64 `gorm:"column:view_count"`
	// 加购件数
	CartCount int64 `gorm:"column:cart_count"`
	// 支付件数
	PayGoodsNum int64 `gorm:"column:pay_goods_num"`
	// 支付金额，单位分
	PayAmount int64 `gorm:"column:pay_amount"`
}

// GoodsAnalyticsTrendBucket 商品分析趋势桶位数据。
type GoodsAnalyticsTrendBucket struct {
	// 浏览次数
	ViewCount int64
	// 加购件数
	CartCount int64
	// 支付件数
	PayGoodsNum int64
	// 支付金额，单位分
	PayAmount int64
}

// GoodsAnalyticsCategorySummaryRow 商品分类分布聚合行。
type GoodsAnalyticsCategorySummaryRow struct {
	// 商品编号
	GoodsID int64 `gorm:"column:goods_id"`
	// 成交件数
	GoodsCount int64 `gorm:"column:goods_count"`
	// 分类编号
	CategoryID int64 `gorm:"column:category_id"`
}

// GoodsCategoryIDsRow 商品分类编号列表行。
type GoodsCategoryIDsRow struct {
	// 商品编号
	GoodsID int64 `gorm:"column:id"`
	// 分类编号列表
	CategoryID string `gorm:"column:category_id"`
}

// GoodsAnalyticsRankRow 商品支付排行聚合行。
type GoodsAnalyticsRankRow struct {
	// 商品编号
	GoodsID int64 `gorm:"column:goods_id"`
	// 支付金额，单位分
	PayAmount int64 `gorm:"column:pay_amount"`
}

// GoodsNameRow 商品名称行。
type GoodsNameRow struct {
	// 商品编号
	GoodsID int64 `gorm:"column:id"`
	// 商品名称
	Name string `gorm:"column:name"`
}

package dto

// GoodsAnalyticsSummaryRow 商品分析行为汇总行。
type GoodsAnalyticsSummaryRow struct {
	// ViewCount 浏览次数
	ViewCount int64 `gorm:"column:view_count"`
	// CollectCount 收藏次数
	CollectCount int64 `gorm:"column:collect_count"`
	// CartCount 加购件数
	CartCount int64 `gorm:"column:cart_count"`
	// OrderCount 下单次数
	OrderCount int64 `gorm:"column:order_count"`
	// PayCount 支付次数
	PayCount int64 `gorm:"column:pay_count"`
	// PayGoodsNum 支付件数
	PayGoodsNum int64 `gorm:"column:pay_goods_num"`
	// PayAmount 支付金额，单位分
	PayAmount int64 `gorm:"column:pay_amount"`
}

// GoodsAnalyticsTrendRow 商品分析趋势聚合行。
type GoodsAnalyticsTrendRow struct {
	// Key 趋势桶位键值
	Key int64 `gorm:"column:key"`
	// ViewCount 浏览次数
	ViewCount int64 `gorm:"column:view_count"`
	// CartCount 加购件数
	CartCount int64 `gorm:"column:cart_count"`
	// PayGoodsNum 支付件数
	PayGoodsNum int64 `gorm:"column:pay_goods_num"`
	// PayAmount 支付金额，单位分
	PayAmount int64 `gorm:"column:pay_amount"`
}

// GoodsAnalyticsTrendBucket 商品分析趋势桶位数据。
type GoodsAnalyticsTrendBucket struct {
	// ViewCount 浏览次数
	ViewCount int64
	// CartCount 加购件数
	CartCount int64
	// PayGoodsNum 支付件数
	PayGoodsNum int64
	// PayAmount 支付金额，单位分
	PayAmount int64
}

// GoodsAnalyticsCategorySummaryRow 商品分类分布聚合行。
type GoodsAnalyticsCategorySummaryRow struct {
	// GoodsCount 成交件数
	GoodsCount int64 `gorm:"column:goods_count"`
	// CategoryId 分类编号
	CategoryId int64 `gorm:"column:category_id"`
}

// GoodsAnalyticsRankRow 商品支付排行聚合行。
type GoodsAnalyticsRankRow struct {
	// GoodsId 商品编号
	GoodsId int64 `gorm:"column:goods_id"`
	// PayAmount 支付金额，单位分
	PayAmount int64 `gorm:"column:pay_amount"`
}

// GoodsNameRow 商品名称行。
type GoodsNameRow struct {
	// GoodsId 商品编号
	GoodsId int64 `gorm:"column:id"`
	// Name 商品名称
	Name string `gorm:"column:name"`
}

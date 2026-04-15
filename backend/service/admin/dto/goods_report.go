package dto

// GoodsMonthReportRow 商品月报聚合行。
type GoodsMonthReportRow struct {
	// Month 月份，格式：YYYY-MM
	Month string `gorm:"column:month"`
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
	// Score 热度分
	Score float64 `gorm:"column:score"`
}

// GoodsDayReportRow 商品日报聚合行。
type GoodsDayReportRow struct {
	// Day 日期，格式：YYYY-MM-DD
	Day string `gorm:"column:day"`
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
	// Score 热度分
	Score float64 `gorm:"column:score"`
}

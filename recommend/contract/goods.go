package contract

import (
	"context"
	"time"
)

// Goods 表示推荐所需的商品属性。
type Goods struct {
	// Id 表示商品编号。
	Id int64
	// CategoryId 表示商品所属类目编号。
	CategoryId int64
	// BrandId 表示商品所属品牌编号。
	BrandId int64
	// ShopId 表示商品所属店铺编号。
	ShopId int64
	// Price 表示商品价格，单位由业务层决定。
	Price int64
	// OnSale 表示商品当前是否处于上架状态。
	OnSale bool
	// InStock 表示商品当前是否仍有可售库存。
	InStock bool
	// CreatedAt 表示商品创建时间。
	CreatedAt time.Time
	// UpdatedAt 表示商品最近更新时间。
	UpdatedAt time.Time
}

// GoodsSource 定义推荐所需的商品数据来源。
type GoodsSource interface {
	GetGoods(context.Context, int64) (*Goods, error)
	ListGoods(context.Context, []int64) ([]*Goods, error)
	ListGoodsByCategoryIds(context.Context, []int64, int32) ([]*Goods, error)
	ListLatestGoods(context.Context, int32) ([]*Goods, error)
}

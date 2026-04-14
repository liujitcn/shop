package contract

import (
	"context"
	"time"
)

// Goods 表示推荐所需的商品属性。
type Goods struct {
	Id         int64
	CategoryId int64
	BrandId    int64
	ShopId     int64
	Price      int64
	OnSale     bool
	InStock    bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// GoodsSource 定义推荐所需的商品数据来源。
type GoodsSource interface {
	GetGoods(context.Context, int64) (*Goods, error)
	ListGoods(context.Context, []int64) ([]*Goods, error)
	ListGoodsByCategoryIds(context.Context, []int64, int32) ([]*Goods, error)
	ListLatestGoods(context.Context, int32) ([]*Goods, error)
}

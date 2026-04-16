package domain

import "shop/api/gen/go/common"

// GoodsRequest 表示推荐商品请求的领域对象。
type GoodsRequest struct {
	Scene    common.RecommendScene // 推荐场景
	OrderId  int64                 // 订单编号
	GoodsId  int64                 // 商品编号
	PageNum  int64                 // 当前页码
	PageSize int64                 // 每页数量
}

// NormalizePage 统一兜底分页参数，避免领域层继续感知接口默认值。
func (r *GoodsRequest) NormalizePage(defaultPageNum, defaultPageSize int64) {
	// 请求为空时，无法继续归一化分页参数。
	if r == nil {
		return
	}
	// 当前页码非法时，回退到调用方提供的默认页码。
	if r.PageNum <= 0 {
		r.PageNum = defaultPageNum
	}
	// 每页数量非法时，回退到调用方提供的默认分页大小。
	if r.PageSize <= 0 {
		r.PageSize = defaultPageSize
	}
}

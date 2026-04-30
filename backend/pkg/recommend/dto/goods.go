package dto

import (
	commonv1 "shop/api/gen/go/common/v1"
)

// GoodsTrace 表示推荐链路执行轨迹。
type GoodsTrace struct {
	ProviderName string // 推荐提供方
	ResultCount  int    // 当前提供方返回的商品数量
	Hit          bool   // 当前提供方是否命中结果
	ErrorMsg     string // 当前提供方错误信息
}

// GoodsRequest 表示统一推荐商品查询参数。
type GoodsRequest struct {
	Scene           commonv1.RecommendScene // 推荐场景
	Actor           *RecommendActor         // 推荐主体
	GoodsID         int64                   // 锚点商品编号
	OrderID         int64                   // 订单编号
	RequestID       int64                   // 推荐请求编号
	ContextGoodsIDs []int64                 // 上下文商品编号列表
	PageNum         int64                   // 页码
	PageSize        int64                   // 每页数量
}

// GoodsResult 表示统一推荐商品查询结果。
type GoodsResult struct {
	GoodsIDs     []int64                    // 推荐商品编号列表
	Total        int64                      // 推荐结果总数
	Strategy     commonv1.RecommendStrategy // 当前命中的推荐策略
	ProviderName string                     // 命中的推荐提供方
	Trace        []*GoodsTrace              // 推荐执行轨迹
}

// RecommendContext 表示推荐请求主表未单独存储的附加上下文。
type RecommendContext struct {
	GoodsID         int64                      `json:"goods_id"`                    // 当前推荐请求的锚点商品编号
	OrderID         int64                      `json:"order_id"`                    // 当前推荐请求关联的订单编号
	ContextGoodsIDs []int64                    `json:"context_goods_ids,omitempty"` // 当前推荐计算使用的上下文商品编号列表
	Strategy        commonv1.RecommendStrategy `json:"strategy,omitempty"`          // 当前命中的推荐策略
	ProviderName    string                     `json:"provider_name,omitempty"`     // 当前命中的推荐提供方
	Trace           []*GoodsTrace              `json:"trace,omitempty"`             // 当前推荐链路执行轨迹
}

// NewRecommendRequestContext 创建推荐请求附加上下文。
func NewRecommendRequestContext(goodsID, orderID int64, contextGoodsIDs []int64, result *GoodsResult) *RecommendContext {
	contextRecord := &RecommendContext{
		GoodsID:         goodsID,
		OrderID:         orderID,
		ContextGoodsIDs: append([]int64(nil), contextGoodsIDs...),
		Trace:           make([]*GoodsTrace, 0),
	}
	// 推荐结果为空时，仅保留当前请求侧上下文信息。
	if result == nil {
		return contextRecord
	}

	contextRecord.Strategy = result.Strategy
	contextRecord.ProviderName = result.ProviderName
	contextRecord.Trace = result.Trace
	return contextRecord
}

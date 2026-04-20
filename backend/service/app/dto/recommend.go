package dto

// RecommendRequestContextRecord 推荐请求上下文记录
type RecommendRequestContextRecord struct {
	// GoodsId 当前推荐请求的锚点商品编号
	GoodsId int64 `json:"goods_id,omitempty"`
	// OrderId 当前推荐请求关联的订单编号
	OrderId int64 `json:"order_id,omitempty"`
	// ContextGoodsIds 当前推荐计算使用的上下文商品编号列表
	ContextGoodsIds []int64 `json:"context_goods_ids,omitempty"`
	// ContextSource 当前推荐上下文来源
	ContextSource string `json:"context_source,omitempty"`
	// StrategyType 当前命中的本地兜底策略
	StrategyType string `json:"strategy_type,omitempty"`
	// Source 当前结果来源
	Source string `json:"source,omitempty"`
	// Status 当前请求执行状态
	Status string `json:"status,omitempty"`
}

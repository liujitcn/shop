package planner

import (
	"shop/api/gen/go/common"
	recommendcore "shop/pkg/recommend/core"
)

// SceneInput 表示场景桥接层传入 planner 的原始上下文。
type SceneInput struct {
	CartGoodsIds     []int64
	OrderGoodsIds    []int64
	SourceGoodsIds   []int64
	PriorityGoodsIds []int64
	CategoryIds      []int64
	CacheHitSources  []string
}

// BuildCartSceneInput 构建购物车场景的桥接输入。
func BuildCartSceneInput(cartGoodsIds []int64, priorityGoodsIds []int64, categoryIds []int64) SceneInput {
	return SceneInput{
		CartGoodsIds:     cartGoodsIds,
		SourceGoodsIds:   cartGoodsIds,
		PriorityGoodsIds: priorityGoodsIds,
		CategoryIds:      categoryIds,
	}
}

// BuildOrderSceneInput 构建订单场景的桥接输入。
func BuildOrderSceneInput(orderGoodsIds []int64, priorityGoodsIds []int64, categoryIds []int64) SceneInput {
	return SceneInput{
		OrderGoodsIds:    orderGoodsIds,
		SourceGoodsIds:   orderGoodsIds,
		PriorityGoodsIds: priorityGoodsIds,
		CategoryIds:      categoryIds,
	}
}

// BuildGoodsDetailSceneInput 构建商品详情场景的桥接输入。
func BuildGoodsDetailSceneInput(sourceGoodsIds []int64, priorityGoodsIds []int64, categoryIds []int64, cacheHitSources []string) SceneInput {
	return SceneInput{
		SourceGoodsIds:   sourceGoodsIds,
		PriorityGoodsIds: priorityGoodsIds,
		CategoryIds:      categoryIds,
		CacheHitSources:  cacheHitSources,
	}
}

// ApplySceneInput 将场景桥接层提供的原始输入写入请求计划。
func (p *RequestPlan) ApplySceneInput(input SceneInput) {
	// 计划对象为空时，无法继续写入场景输入。
	if p == nil {
		return
	}
	for _, source := range input.CacheHitSources {
		// 只有明确命中的缓存来源才写入计划对象。
		p.AddCacheHitSource(source)
	}
	switch p.Request.Scene {
	// 购物车场景使用购物车关联召回结果和类目补足。
	case common.RecommendScene_CART:
		p.ApplyCartScene(input.PriorityGoodsIds, input.CategoryIds)
	// 订单详情和支付完成场景共用订单关联召回规划。
	case common.RecommendScene_ORDER_DETAIL, common.RecommendScene_ORDER_PAID:
		p.ApplyOrderScene(input.PriorityGoodsIds, input.CategoryIds)
	// 商品详情场景允许继续合并灰度召回入池结果。
	case common.RecommendScene_GOODS_DETAIL:
		p.ApplyGoodsDetailScene(input.PriorityGoodsIds, input.CategoryIds)
	}
}

// BuildSceneSourceContext 构建场景桥接层的基础来源上下文。
func (p *RequestPlan) BuildSceneSourceContext(input SceneInput) map[string]any {
	base := make(map[string]any, 5)
	// 存在订单编号时，保留当前请求绑定的订单上下文。
	if p != nil && p.Request.OrderId > 0 {
		base["orderId"] = p.Request.OrderId
	}
	// 存在商品编号时，保留当前请求绑定的商品上下文。
	if p != nil && p.Request.GoodsId > 0 {
		base["goodsId"] = p.Request.GoodsId
	}
	// 购物车场景读到原始购物车商品时，写入来源上下文便于排障。
	if len(input.CartGoodsIds) > 0 {
		base["cartGoodsIds"] = recommendcore.DedupeInt64s(input.CartGoodsIds)
	}
	// 订单场景读到订单商品时，写入来源上下文便于排障。
	if len(input.OrderGoodsIds) > 0 {
		base["orderGoodsIds"] = recommendcore.DedupeInt64s(input.OrderGoodsIds)
	}
	// 商品详情或订单桥接命中了源商品时，统一写入来源上下文。
	if len(input.SourceGoodsIds) > 0 {
		base["sourceGoodsIds"] = recommendcore.DedupeInt64s(input.SourceGoodsIds)
	}
	return base
}

package biz

import "shop/api/gen/go/common"

const (
	recommendGoodsActionTypeUnknown = common.RecommendGoodsActionType_UNKNOWN_RGAT
	recommendGoodsActionTypeView    = common.RecommendGoodsActionType_RECOMMEND_GOODS_ACTION_VIEW
	recommendGoodsActionTypeCollect = common.RecommendGoodsActionType_RECOMMEND_GOODS_ACTION_COLLECT
	recommendGoodsActionTypeCart    = common.RecommendGoodsActionType_RECOMMEND_GOODS_ACTION_CART
	recommendGoodsActionTypeOrder   = common.RecommendGoodsActionType_RECOMMEND_GOODS_ACTION_ORDER_CREATE
	recommendGoodsActionTypePay     = common.RecommendGoodsActionType_RECOMMEND_GOODS_ACTION_ORDER_PAY
	recommendGoodsActionTypeClick   = common.RecommendGoodsActionType_RECOMMEND_GOODS_ACTION_CLICK
)

// convertRecommendEventTypeToGoodsActionType 将推荐事件类型转换为行为事实表枚举值。
func convertRecommendEventTypeToGoodsActionType(eventType string) common.RecommendGoodsActionType {
	// 浏览事件映射为浏览行为枚举。
	if eventType == recommendEventTypeView {
		return recommendGoodsActionTypeView
	}
	// 收藏事件映射为收藏行为枚举。
	if eventType == recommendEventTypeCollect {
		return recommendGoodsActionTypeCollect
	}
	// 加购事件映射为加购行为枚举。
	if eventType == recommendEventTypeCart {
		return recommendGoodsActionTypeCart
	}
	// 下单事件映射为下单行为枚举。
	if eventType == recommendEventTypeOrder {
		return recommendGoodsActionTypeOrder
	}
	// 支付事件映射为支付行为枚举。
	if eventType == recommendEventTypePay {
		return recommendGoodsActionTypePay
	}
	// 推荐点击事件映射为点击行为枚举。
	if eventType == recommendEventTypeClick {
		return recommendGoodsActionTypeClick
	}
	return recommendGoodsActionTypeUnknown
}

// formatRecommendGoodsActionType 将行为事实表枚举值转换回推荐事件字符串。
func formatRecommendGoodsActionType(eventType common.RecommendGoodsActionType) string {
	// 点击枚举回转为点击事件名。
	if eventType == recommendGoodsActionTypeClick {
		return recommendEventTypeClick
	}
	// 浏览枚举回转为浏览事件名。
	if eventType == recommendGoodsActionTypeView {
		return recommendEventTypeView
	}
	// 收藏枚举回转为收藏事件名。
	if eventType == recommendGoodsActionTypeCollect {
		return recommendEventTypeCollect
	}
	// 加购枚举回转为加购事件名。
	if eventType == recommendGoodsActionTypeCart {
		return recommendEventTypeCart
	}
	// 下单枚举回转为下单事件名。
	if eventType == recommendGoodsActionTypeOrder {
		return recommendEventTypeOrder
	}
	// 支付枚举回转为支付事件名。
	if eventType == recommendGoodsActionTypePay {
		return recommendEventTypePay
	}
	return ""
}

package event

import (
	"encoding/json"

	"shop/api/gen/go/common"
)

const (
	PreferenceTypeCategory = "category"

	ActorTypeAnonymous = int32(0)
	ActorTypeUser      = int32(1)

	AggregateWindowDays = 30
)

// NormalizeGoodsNum 统一商品数量的权重下限。
func NormalizeGoodsNum(goodsNum int64) float64 {
	// 非法数量统一回退到最小权重，避免把 0 或负数带入后续计算。
	if goodsNum <= 0 {
		return 1
	}
	return float64(goodsNum)
}

// NormalizeGoodsCount 统一商品数量的计数下限。
func NormalizeGoodsCount(goodsNum int64) int64 {
	// 非法数量统一回退到最小计数，避免后续聚合被 0 吃掉。
	if goodsNum <= 0 {
		return 1
	}
	return goodsNum
}

// IsSingleGoodsEvent 判断是否为单商品事件。
func IsSingleGoodsEvent(eventType common.RecommendGoodsActionType) bool {
	// 按事件类型判断是否属于单商品行为。
	switch eventType {
	case common.RecommendGoodsActionType_CLICK,
		common.RecommendGoodsActionType_VIEW,
		common.RecommendGoodsActionType_COLLECT,
		common.RecommendGoodsActionType_ADD_CART:
		return true
	default:
		return false
	}
}

// IsOrderGoodsEvent 判断是否为订单级多商品事件。
func IsOrderGoodsEvent(eventType common.RecommendGoodsActionType) bool {
	// 按事件类型判断是否属于订单级多商品行为。
	switch eventType {
	case common.RecommendGoodsActionType_ORDER_CREATE,
		common.RecommendGoodsActionType_ORDER_PAY:
		return true
	default:
		return false
	}
}

// EventWeight 返回用户偏好聚合所使用的事件权重。
func EventWeight(eventType common.RecommendGoodsActionType) float64 {
	// 按行为强弱映射登录态偏好聚合权重。
	switch eventType {
	case common.RecommendGoodsActionType_CLICK:
		return 3
	case common.RecommendGoodsActionType_VIEW:
		return 2
	case common.RecommendGoodsActionType_COLLECT:
		return 4
	case common.RecommendGoodsActionType_ADD_CART:
		return 6
	case common.RecommendGoodsActionType_ORDER_CREATE:
		return 8
	case common.RecommendGoodsActionType_ORDER_PAY:
		return 10
	default:
		return 0
	}
}

// IsRelationEvent 判断是否为可生成商品关联的行为事件。
func IsRelationEvent(eventType common.RecommendGoodsActionType) bool {
	// 只把可体现商品关联关系的行为纳入共现计算。
	switch eventType {
	case common.RecommendGoodsActionType_CLICK,
		common.RecommendGoodsActionType_VIEW,
		common.RecommendGoodsActionType_ORDER_CREATE,
		common.RecommendGoodsActionType_ORDER_PAY:
		return true
	default:
		return false
	}
}

// RelationWeight 返回商品关联聚合所使用的关系权重。
func RelationWeight(eventType common.RecommendGoodsActionType) float64 {
	// 按行为强弱映射商品关联聚合权重。
	switch eventType {
	case common.RecommendGoodsActionType_CLICK:
		return 3
	case common.RecommendGoodsActionType_VIEW:
		return 2
	case common.RecommendGoodsActionType_ORDER_CREATE:
		return 8
	case common.RecommendGoodsActionType_ORDER_PAY:
		return 10
	default:
		return 0
	}
}

// AddBehaviorSummaryCount 累加 JSON 汇总中的行为计数。
func AddBehaviorSummaryCount(summaryJSON string, key common.RecommendGoodsActionType, delta int64) (string, error) {
	// 未知事件或零增量无需改写汇总结果。
	if key == common.RecommendGoodsActionType_UNKNOWN_RGAT || delta == 0 {
		return summaryJSON, nil
	}

	summary := make(map[string]int64)
	// 历史汇总存在时，先反序列化后再做增量合并。
	if summaryJSON != "" {
		// 历史 JSON 非法时，直接返回错误避免写入损坏数据。
		if err := json.Unmarshal([]byte(summaryJSON), &summary); err != nil {
			return "", err
		}
	}
	summary[key.String()] += delta
	rawBody, err := json.Marshal(summary)
	if err != nil {
		return "", err
	}
	return string(rawBody), nil
}

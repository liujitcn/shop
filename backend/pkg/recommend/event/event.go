package event

import (
	"encoding/json"

	"shop/api/gen/go/common"
	"shop/api/gen/go/conf"
)

const (
	PreferenceTypeCategory = "category"

	ActorTypeAnonymous = int32(0)
	ActorTypeUser      = int32(1)
)

var (
	// AggregateWindowDays 推荐聚合统一窗口天数。
	AggregateWindowDays int32 = 30
	eventWeightConfig         = &conf.RecommendEventWeightConfig{
		ClickWeight:       float64Ptr(3),
		ViewWeight:        float64Ptr(2),
		CollectWeight:     float64Ptr(4),
		AddCartWeight:     float64Ptr(6),
		OrderCreateWeight: float64Ptr(8),
		OrderPayWeight:    float64Ptr(10),
	}
	relationWeightConfig = &conf.RecommendRelationWeightConfig{
		ClickWeight:       float64Ptr(3),
		ViewWeight:        float64Ptr(2),
		OrderCreateWeight: float64Ptr(8),
		OrderPayWeight:    float64Ptr(10),
	}
)

// ApplyRecommendConfig 应用推荐行为相关运行时配置。
func ApplyRecommendConfig(cfg *conf.GoodsRecommendConfig) {
	// 配置缺失时，保留当前默认行为配置。
	if cfg == nil {
		return
	}
	if cfg.AggregateWindowDays != nil {
		AggregateWindowDays = cfg.GetAggregateWindowDays()
	}
	if cfg.GetEventWeight() != nil {
		eventWeightConfig = cfg.GetEventWeight()
	}
	if cfg.GetRelationWeight() != nil {
		relationWeightConfig = cfg.GetRelationWeight()
	}
}

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
		return eventWeightConfig.GetClickWeight()
	case common.RecommendGoodsActionType_VIEW:
		return eventWeightConfig.GetViewWeight()
	case common.RecommendGoodsActionType_COLLECT:
		return eventWeightConfig.GetCollectWeight()
	case common.RecommendGoodsActionType_ADD_CART:
		return eventWeightConfig.GetAddCartWeight()
	case common.RecommendGoodsActionType_ORDER_CREATE:
		return eventWeightConfig.GetOrderCreateWeight()
	case common.RecommendGoodsActionType_ORDER_PAY:
		return eventWeightConfig.GetOrderPayWeight()
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
		return relationWeightConfig.GetClickWeight()
	case common.RecommendGoodsActionType_VIEW:
		return relationWeightConfig.GetViewWeight()
	case common.RecommendGoodsActionType_ORDER_CREATE:
		return relationWeightConfig.GetOrderCreateWeight()
	case common.RecommendGoodsActionType_ORDER_PAY:
		return relationWeightConfig.GetOrderPayWeight()
	default:
		return 0
	}
}

// float64Ptr 返回 float64 指针，便于初始化默认 optional 字段。
func float64Ptr(value float64) *float64 {
	return &value
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

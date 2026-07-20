package dto

import (
	"encoding/json"
	"strconv"
	"strings"

	shopcommonv1 "shop/api/gen/go/shop/common/v1"

	_const "shop/service/shop/consts"
)

// RecommendActor 表示推荐链路内部使用的推荐主体。
type RecommendActor struct {
	ActorType shopcommonv1.RecommendActorType `json:"actor_type"` // 推荐主体类型
	ActorID   int64                           `json:"actor_id"`   // 推荐主体编号
}

// IsValid 判断当前推荐主体是否有效。
func (r *RecommendActor) IsValid() bool {
	return r != nil && r.ActorID > 0
}

// IsUser 判断当前推荐主体是否为登录用户。
func (r *RecommendActor) IsUser() bool {
	return r != nil && r.ActorType == shopcommonv1.RecommendActorType(_const.RECOMMEND_ACTOR_TYPE_USER) && r.ActorID > 0
}

// FormatRecommendStrategyCode 将推荐策略枚举转换为稳定的策略编码。
func FormatRecommendStrategyCode(strategy shopcommonv1.RecommendStrategy) string {
	switch strategy {
	case shopcommonv1.RecommendStrategy(_const.RECOMMEND_STRATEGY_REMOTE):
		return "remote"
	case shopcommonv1.RecommendStrategy(_const.RECOMMEND_STRATEGY_LOCAL):
		return "local"
	default:
		return ""
	}
}

// ParseRecommendStrategyCode 根据稳定的策略编码解析推荐策略枚举。
func ParseRecommendStrategyCode(code string) shopcommonv1.RecommendStrategy {
	normalizedCode := strings.ToLower(code)
	switch normalizedCode {
	case "remote", "remote_strategy":
		return shopcommonv1.RecommendStrategy(_const.RECOMMEND_STRATEGY_REMOTE)
	case "local", "local_strategy":
		return shopcommonv1.RecommendStrategy(_const.RECOMMEND_STRATEGY_LOCAL)
	}

	// 兼容透传 proto 枚举名称的场景，避免大小写差异导致策略丢失。
	switch code {
	case shopcommonv1.RecommendStrategy(_const.RECOMMEND_STRATEGY_REMOTE).String():
		return shopcommonv1.RecommendStrategy(_const.RECOMMEND_STRATEGY_REMOTE)
	case shopcommonv1.RecommendStrategy(_const.RECOMMEND_STRATEGY_LOCAL).String():
		return shopcommonv1.RecommendStrategy(_const.RECOMMEND_STRATEGY_LOCAL)
	}

	value, err := strconv.ParseInt(code, 10, 32)
	if err != nil {
		return shopcommonv1.RecommendStrategy(_const.RECOMMEND_STRATEGY_UNKNOWN)
	}
	return NormalizeRecommendStrategy(shopcommonv1.RecommendStrategy(value))
}

// NormalizeRecommendStrategy 过滤非法推荐策略枚举值，统一回退到未知状态。
func NormalizeRecommendStrategy(strategy shopcommonv1.RecommendStrategy) shopcommonv1.RecommendStrategy {
	switch strategy {
	case shopcommonv1.RecommendStrategy(_const.RECOMMEND_STRATEGY_UNKNOWN),
		shopcommonv1.RecommendStrategy(_const.RECOMMEND_STRATEGY_REMOTE),
		shopcommonv1.RecommendStrategy(_const.RECOMMEND_STRATEGY_LOCAL):
		return strategy
	default:
		return shopcommonv1.RecommendStrategy(_const.RECOMMEND_STRATEGY_UNKNOWN)
	}
}

// ParseRecommendStrategyRaw 兼容历史字符串与当前枚举值两种格式解析推荐策略。
func ParseRecommendStrategyRaw(raw json.RawMessage) shopcommonv1.RecommendStrategy {
	normalizedRaw := string(raw)
	if normalizedRaw == "" || normalizedRaw == "null" {
		return shopcommonv1.RecommendStrategy(_const.RECOMMEND_STRATEGY_UNKNOWN)
	}

	var err error
	// 历史上下文使用字符串编码记录策略，这里优先兼容旧格式数据。
	if strings.HasPrefix(normalizedRaw, "\"") {
		code := ""
		err = json.Unmarshal(raw, &code)
		if err != nil {
			return shopcommonv1.RecommendStrategy(_const.RECOMMEND_STRATEGY_UNKNOWN)
		}
		return ParseRecommendStrategyCode(code)
	}

	value := int32(0)
	err = json.Unmarshal(raw, &value)
	if err != nil {
		return shopcommonv1.RecommendStrategy(_const.RECOMMEND_STRATEGY_UNKNOWN)
	}
	return NormalizeRecommendStrategy(shopcommonv1.RecommendStrategy(value))
}

// MarshalJSON 将推荐上下文序列化为兼容历史数据的 JSON 结构。
func (r *RecommendContext) MarshalJSON() ([]byte, error) {
	type recommendContextJSON struct {
		GoodsID         int64         `json:"goods_id"`
		OrderID         int64         `json:"order_id"`
		TradeID         int64         `json:"trade_id"`
		ContextGoodsIDs []int64       `json:"context_goods_ids,omitempty"`
		Strategy        string        `json:"strategy,omitempty"`
		ProviderName    string        `json:"provider_name,omitempty"`
		Trace           []*GoodsTrace `json:"trace,omitempty"`
	}

	if r == nil {
		return json.Marshal(&recommendContextJSON{})
	}

	return json.Marshal(&recommendContextJSON{
		GoodsID:         r.GoodsID,
		OrderID:         r.OrderID,
		TradeID:         r.TradeID,
		ContextGoodsIDs: append([]int64(nil), r.ContextGoodsIDs...),
		Strategy:        FormatRecommendStrategyCode(r.Strategy),
		ProviderName:    r.ProviderName,
		Trace:           r.Trace,
	})
}

// UnmarshalJSON 兼容历史字符串策略与当前枚举策略两种上下文格式。
func (r *RecommendContext) UnmarshalJSON(data []byte) error {
	type recommendContextJSON struct {
		GoodsID         int64           `json:"goods_id"`
		OrderID         int64           `json:"order_id"`
		TradeID         int64           `json:"trade_id"`
		ContextGoodsIDs []int64         `json:"context_goods_ids,omitempty"`
		Strategy        json.RawMessage `json:"strategy,omitempty"`
		ProviderName    string          `json:"provider_name,omitempty"`
		Trace           []*GoodsTrace   `json:"trace,omitempty"`
	}

	payload := &recommendContextJSON{}
	err := json.Unmarshal(data, payload)
	if err != nil {
		return err
	}

	r.GoodsID = payload.GoodsID
	r.OrderID = payload.OrderID
	r.TradeID = payload.TradeID
	r.ContextGoodsIDs = append([]int64(nil), payload.ContextGoodsIDs...)
	r.Strategy = ParseRecommendStrategyRaw(payload.Strategy)
	r.ProviderName = payload.ProviderName
	r.Trace = payload.Trace
	// 统一把空轨迹回退为非 nil 切片，避免管理端详情收到 null。
	if r.Trace == nil {
		r.Trace = make([]*GoodsTrace, 0)
	}
	return nil
}

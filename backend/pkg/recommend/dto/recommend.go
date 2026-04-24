package dto

import (
	"encoding/json"
	"strconv"
	"strings"

	"shop/api/gen/go/common"
)

// RecommendActor 表示推荐链路内部使用的推荐主体。
type RecommendActor struct {
	ActorType common.RecommendActorType `json:"actor_type"` // 推荐主体类型
	ActorId   int64                     `json:"actor_id"`   // 推荐主体编号
}

// IsValid 判断当前推荐主体是否有效。
func (r *RecommendActor) IsValid() bool {
	return r != nil && r.ActorId > 0
}

// IsUser 判断当前推荐主体是否为登录用户。
func (r *RecommendActor) IsUser() bool {
	return r != nil && r.ActorType == common.RecommendActorType_USER_ACTOR && r.ActorId > 0
}

// FormatRecommendStrategyCode 将推荐策略枚举转换为稳定的策略编码。
func FormatRecommendStrategyCode(strategy common.RecommendStrategy) string {
	switch strategy {
	case common.RecommendStrategy_REMOTE_STRATEGY:
		return "remote"
	case common.RecommendStrategy_LOCAL_STRATEGY:
		return "local"
	default:
		return ""
	}
}

// ParseRecommendStrategyCode 根据稳定的策略编码解析推荐策略枚举。
func ParseRecommendStrategyCode(code string) common.RecommendStrategy {
	normalizedCode := strings.TrimSpace(strings.ToLower(code))
	switch normalizedCode {
	case "remote", "remote_strategy":
		return common.RecommendStrategy_REMOTE_STRATEGY
	case "local", "local_strategy":
		return common.RecommendStrategy_LOCAL_STRATEGY
	}

	// 兼容透传 proto 枚举名称的场景，避免大小写差异导致策略丢失。
	switch strings.TrimSpace(code) {
	case common.RecommendStrategy_REMOTE_STRATEGY.String():
		return common.RecommendStrategy_REMOTE_STRATEGY
	case common.RecommendStrategy_LOCAL_STRATEGY.String():
		return common.RecommendStrategy_LOCAL_STRATEGY
	}

	value, err := strconv.ParseInt(strings.TrimSpace(code), 10, 32)
	if err != nil {
		return common.RecommendStrategy_UNKNOWN_RST
	}
	return NormalizeRecommendStrategy(common.RecommendStrategy(value))
}

// NormalizeRecommendStrategy 过滤非法推荐策略枚举值，统一回退到未知状态。
func NormalizeRecommendStrategy(strategy common.RecommendStrategy) common.RecommendStrategy {
	switch strategy {
	case common.RecommendStrategy_UNKNOWN_RST,
		common.RecommendStrategy_REMOTE_STRATEGY,
		common.RecommendStrategy_LOCAL_STRATEGY:
		return strategy
	default:
		return common.RecommendStrategy_UNKNOWN_RST
	}
}

// ParseRecommendStrategyRaw 兼容历史字符串与当前枚举值两种格式解析推荐策略。
func ParseRecommendStrategyRaw(raw json.RawMessage) common.RecommendStrategy {
	normalizedRaw := strings.TrimSpace(string(raw))
	if normalizedRaw == "" || normalizedRaw == "null" {
		return common.RecommendStrategy_UNKNOWN_RST
	}

	// 历史上下文使用字符串编码记录策略，这里优先兼容旧格式数据。
	if strings.HasPrefix(normalizedRaw, "\"") {
		code := ""
		err := json.Unmarshal(raw, &code)
		if err != nil {
			return common.RecommendStrategy_UNKNOWN_RST
		}
		return ParseRecommendStrategyCode(code)
	}

	value := int32(0)
	err := json.Unmarshal(raw, &value)
	if err != nil {
		return common.RecommendStrategy_UNKNOWN_RST
	}
	return NormalizeRecommendStrategy(common.RecommendStrategy(value))
}

// MarshalJSON 将推荐上下文序列化为兼容历史数据的 JSON 结构。
func (r *RecommendContext) MarshalJSON() ([]byte, error) {
	type recommendContextJSON struct {
		GoodsId         int64         `json:"goods_id"`
		OrderId         int64         `json:"order_id"`
		ContextGoodsIds []int64       `json:"context_goods_ids,omitempty"`
		Strategy        string        `json:"strategy,omitempty"`
		ProviderName    string        `json:"provider_name,omitempty"`
		Trace           []*GoodsTrace `json:"trace,omitempty"`
	}

	if r == nil {
		return json.Marshal(&recommendContextJSON{})
	}

	return json.Marshal(&recommendContextJSON{
		GoodsId:         r.GoodsId,
		OrderId:         r.OrderId,
		ContextGoodsIds: append([]int64(nil), r.ContextGoodsIds...),
		Strategy:        FormatRecommendStrategyCode(r.Strategy),
		ProviderName:    r.ProviderName,
		Trace:           r.Trace,
	})
}

// UnmarshalJSON 兼容历史字符串策略与当前枚举策略两种上下文格式。
func (r *RecommendContext) UnmarshalJSON(data []byte) error {
	type recommendContextJSON struct {
		GoodsId         int64           `json:"goods_id"`
		OrderId         int64           `json:"order_id"`
		ContextGoodsIds []int64         `json:"context_goods_ids,omitempty"`
		Strategy        json.RawMessage `json:"strategy,omitempty"`
		ProviderName    string          `json:"provider_name,omitempty"`
		Trace           []*GoodsTrace   `json:"trace,omitempty"`
	}

	payload := &recommendContextJSON{}
	err := json.Unmarshal(data, payload)
	if err != nil {
		return err
	}

	r.GoodsId = payload.GoodsId
	r.OrderId = payload.OrderId
	r.ContextGoodsIds = append([]int64(nil), payload.ContextGoodsIds...)
	r.Strategy = ParseRecommendStrategyRaw(payload.Strategy)
	r.ProviderName = payload.ProviderName
	r.Trace = payload.Trace
	// 统一把空轨迹回退为非 nil 切片，避免管理端详情收到 null。
	if r.Trace == nil {
		r.Trace = make([]*GoodsTrace, 0)
	}
	return nil
}

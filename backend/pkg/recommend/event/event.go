package event

import (
	"encoding/json"
	"strings"
	"time"

	"shop/api/gen/go/app"
	"shop/api/gen/go/common"
	"shop/pkg/gen/models"
	recommendcontext "shop/pkg/recommend/context"
	appdto "shop/service/app/dto"
)

const (
	EventTypeExposure = "recommend_exposure"
	EventTypeClick    = "recommend_click"
	EventTypeView     = "goods_view"
	EventTypeCollect  = "goods_collect"
	EventTypeCart     = "goods_cart"
	EventTypeOrder    = "order_create"
	EventTypePay      = "order_pay"

	PreferenceTypeCategory = "category"
	RelationTypeCoClick    = "co_click"
	RelationTypeCoView     = "co_view"
	RelationTypeCoOrder    = "co_order"
	RelationTypeCoPay      = "co_pay"

	ActorTypeAnonymous = int32(0)
	ActorTypeUser      = int32(1)

	AggregateWindowDays = 30
)

// BuildGoodsItemsFromOrderGoods 将订单商品转换为推荐事件商品项。
func BuildGoodsItemsFromOrderGoods(orderGoodsList []*models.OrderGoods) []*appdto.RecommendEventGoodsItem {
	goodsItems := make([]*appdto.RecommendEventGoodsItem, 0, len(orderGoodsList))
	for _, orderGoods := range orderGoodsList {
		if orderGoods == nil || orderGoods.GoodsID <= 0 {
			continue
		}
		goodsItems = append(goodsItems, &appdto.RecommendEventGoodsItem{
			GoodsID:   orderGoods.GoodsID,
			GoodsNum:  orderGoods.Num,
			Scene:     orderGoods.Scene,
			RequestID: orderGoods.RequestID,
			Position:  orderGoods.Position,
		})
	}
	return goodsItems
}

// BuildGoodsItemsFromActionItems 将 proto 商品项转换为内部事件商品项。
func BuildGoodsItemsFromActionItems(goodsItems []*app.RecommendGoodsActionItem) []*appdto.RecommendEventGoodsItem {
	list := make([]*appdto.RecommendEventGoodsItem, 0, len(goodsItems))
	for _, goodsItem := range goodsItems {
		if goodsItem == nil || goodsItem.GetGoodsId() <= 0 {
			continue
		}
		recommendCtx := goodsItem.GetRecommendContext()
		list = append(list, &appdto.RecommendEventGoodsItem{
			GoodsID:   goodsItem.GetGoodsId(),
			GoodsNum:  goodsItem.GetGoodsNum(),
			Scene:     recommendcontext.NormalizeSceneEnum(recommendCtx.GetScene()),
			RequestID: strings.TrimSpace(recommendCtx.GetRequestId()),
			Position:  recommendCtx.GetPosition(),
		})
	}
	return list
}

// NormalizeGoodsItems 过滤非法商品项并兜底数量。
func NormalizeGoodsItems(goodsItems []*appdto.RecommendEventGoodsItem) []*appdto.RecommendEventGoodsItem {
	list := make([]*appdto.RecommendEventGoodsItem, 0, len(goodsItems))
	for _, goodsItem := range goodsItems {
		if goodsItem == nil || goodsItem.GoodsID <= 0 {
			continue
		}
		if goodsItem.GoodsNum <= 0 {
			goodsItem.GoodsNum = 1
		}
		list = append(list, goodsItem)
	}
	return list
}

// NormalizeGoodsNum 统一商品数量的权重下限。
func NormalizeGoodsNum(goodsNum int64) float64 {
	if goodsNum <= 0 {
		return 1
	}
	return float64(goodsNum)
}

// NormalizeGoodsCount 统一商品数量的计数下限。
func NormalizeGoodsCount(goodsNum int64) int64 {
	if goodsNum <= 0 {
		return 1
	}
	return goodsNum
}

// EventTime 获取事件发生时间。
func EventTime(event *appdto.RecommendEvent) time.Time {
	if event == nil || event.OccurredAt <= 0 {
		return time.Now()
	}
	return time.Unix(event.OccurredAt, 0)
}

// IsSingleGoodsEvent 判断是否为单商品事件。
func IsSingleGoodsEvent(eventType string) bool {
	switch eventType {
	case EventTypeClick, EventTypeView, EventTypeCollect, EventTypeCart:
		return true
	default:
		return false
	}
}

// IsOrderGoodsEvent 判断是否为订单级多商品事件。
func IsOrderGoodsEvent(eventType string) bool {
	switch eventType {
	case EventTypeOrder, EventTypePay:
		return true
	default:
		return false
	}
}

// EventWeight 返回用户偏好聚合所使用的事件权重。
func EventWeight(eventType string) float64 {
	switch eventType {
	case EventTypeClick:
		return 3
	case EventTypeView:
		return 2
	case EventTypeCollect:
		return 4
	case EventTypeCart:
		return 6
	case EventTypeOrder:
		return 8
	case EventTypePay:
		return 10
	default:
		return 0
	}
}

// RelationWeight 返回商品关联聚合所使用的关系权重。
func RelationWeight(relationType string) float64 {
	switch relationType {
	case RelationTypeCoClick:
		return 3
	case RelationTypeCoView:
		return 2
	case RelationTypeCoOrder:
		return 8
	case RelationTypeCoPay:
		return 10
	default:
		return 0
	}
}

// RelationType 根据事件类型映射商品关系类型。
func RelationType(eventType string) string {
	switch eventType {
	case EventTypeClick:
		return RelationTypeCoClick
	case EventTypeView:
		return RelationTypeCoView
	case EventTypeOrder:
		return RelationTypeCoOrder
	case EventTypePay:
		return RelationTypeCoPay
	default:
		return ""
	}
}

// EventSummaryKey 返回行为汇总 JSON 中的字段名。
func EventSummaryKey(eventType string) string {
	switch eventType {
	case EventTypeClick:
		return "click_count"
	case EventTypeView:
		return "view_count"
	case EventTypeCollect:
		return "collect_count"
	case EventTypeCart:
		return "cart_count"
	case EventTypeOrder:
		return "order_count"
	case EventTypePay:
		return "pay_count"
	default:
		return ""
	}
}

// ConvertEventTypeToGoodsActionType 将推荐事件类型转换为行为事实表枚举值。
func ConvertEventTypeToGoodsActionType(eventType string) common.RecommendGoodsActionType {
	switch eventType {
	case EventTypeView:
		return common.RecommendGoodsActionType_RECOMMEND_GOODS_ACTION_VIEW
	case EventTypeCollect:
		return common.RecommendGoodsActionType_RECOMMEND_GOODS_ACTION_COLLECT
	case EventTypeCart:
		return common.RecommendGoodsActionType_RECOMMEND_GOODS_ACTION_CART
	case EventTypeOrder:
		return common.RecommendGoodsActionType_RECOMMEND_GOODS_ACTION_ORDER_CREATE
	case EventTypePay:
		return common.RecommendGoodsActionType_RECOMMEND_GOODS_ACTION_ORDER_PAY
	case EventTypeClick:
		return common.RecommendGoodsActionType_RECOMMEND_GOODS_ACTION_CLICK
	default:
		return common.RecommendGoodsActionType_UNKNOWN_RGAT
	}
}

// FormatGoodsActionType 将行为事实表枚举值转换回推荐事件字符串。
func FormatGoodsActionType(eventType common.RecommendGoodsActionType) string {
	switch eventType {
	case common.RecommendGoodsActionType_RECOMMEND_GOODS_ACTION_CLICK:
		return EventTypeClick
	case common.RecommendGoodsActionType_RECOMMEND_GOODS_ACTION_VIEW:
		return EventTypeView
	case common.RecommendGoodsActionType_RECOMMEND_GOODS_ACTION_COLLECT:
		return EventTypeCollect
	case common.RecommendGoodsActionType_RECOMMEND_GOODS_ACTION_CART:
		return EventTypeCart
	case common.RecommendGoodsActionType_RECOMMEND_GOODS_ACTION_ORDER_CREATE:
		return EventTypeOrder
	case common.RecommendGoodsActionType_RECOMMEND_GOODS_ACTION_ORDER_PAY:
		return EventTypePay
	default:
		return ""
	}
}

// AddBehaviorSummaryCount 累加 JSON 汇总中的行为计数。
func AddBehaviorSummaryCount(summaryJSON, key string, delta int64) (string, error) {
	if key == "" || delta == 0 {
		return summaryJSON, nil
	}

	summary := make(map[string]int64)
	if summaryJSON != "" {
		if err := json.Unmarshal([]byte(summaryJSON), &summary); err != nil {
			return "", err
		}
	}
	summary[key] += delta
	rawBody, err := json.Marshal(summary)
	if err != nil {
		return "", err
	}
	return string(rawBody), nil
}

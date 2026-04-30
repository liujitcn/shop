package queue

import (
	"time"

	appv1 "shop/api/gen/go/app/v1"
	commonv1 "shop/api/gen/go/common/v1"
	_const "shop/pkg/const"
	"shop/pkg/recommend/dto"
)

// RecommendEventReportEvent 表示推荐事件队列消息。
type RecommendEventReportEvent struct {
	RecommendActor *dto.RecommendActor         // 推荐主体信息
	EventType      commonv1.RecommendEventType // 推荐事件类型
	Scene          int32                       // 推荐场景
	RequestID      int64                       // 推荐请求 ID
	EventTime      time.Time                   // 事件发生时间
	Items          []*RecommendEventItem       // 推荐事件商品项
}

// RecommendEventItem 表示推荐事件里的单商品事实。
type RecommendEventItem struct {
	GoodsID  int64 // 商品编号
	GoodsNum int64 // 商品数量
	Position int32 // 推荐位次
}

// DispatchRecommendEvent 将推荐事件转换为队列消息并投递到本地推荐事件链路。
func DispatchRecommendEvent(actor *dto.RecommendActor, req *appv1.RecommendEventReportRequest, eventTime time.Time) {
	// 请求体为空时，无法继续构建事件消息。
	if req == nil {
		return
	}
	// 主体缺失或主体 ID 非法时，不投递无法归因的行为事件。
	if actor == nil || !actor.IsValid() {
		return
	}

	eventType := req.GetEventType()
	// 未知行为类型不投递，避免污染后续聚合口径。
	if eventType == commonv1.RecommendEventType(_const.RECOMMEND_EVENT_TYPE_UNKNOWN) {
		return
	}

	// 调用方未显式传入事件时间时，统一回退到当前时间。
	if eventTime.IsZero() {
		eventTime = time.Now()
	}

	recommendContext := req.GetRecommendContext()
	scene := int32(0)
	requestID := int64(0)
	// 事件请求携带推荐归因上下文时，再补齐场景和请求编号。
	if recommendContext != nil {
		scene = int32(recommendContext.GetScene())
		requestID = recommendContext.GetRequestId()
	}

	items := req.GetItems()
	recommendEventItems := make([]*RecommendEventItem, 0, len(items))
	for _, item := range items {
		// 商品项为空或商品 ID 非法时，直接跳过当前事件项。
		if item == nil || item.GetGoodsId() <= 0 {
			continue
		}
		recommendEventItems = append(recommendEventItems, &RecommendEventItem{
			GoodsID:  item.GetGoodsId(),
			GoodsNum: item.GetGoodsNum(),
			Position: item.GetPosition(),
		})
	}
	// 当前请求没有有效商品项时，不再继续投递队列消息。
	if len(recommendEventItems) == 0 {
		return
	}

	AddQueue(_const.RECOMMEND_EVENT_REPORT, &RecommendEventReportEvent{
		RecommendActor: actor,
		EventType:      eventType,
		Scene:          scene,
		RequestID:      requestID,
		EventTime:      eventTime,
		Items:          recommendEventItems,
	})
}

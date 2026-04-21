package queue

import (
	"time"

	"shop/api/gen/go/app"
	"shop/api/gen/go/common"
	_const "shop/pkg/const"
)

// RecommendEventReportEvent 表示推荐事件队列消息。
type RecommendEventReportEvent struct {
	RecommendActor *app.RecommendActor       // 推荐主体信息
	EventType      common.RecommendEventType // 推荐事件类型
	Scene          int32                     // 推荐场景
	RequestId      int64                     // 推荐请求 ID
	EventTime      time.Time                 // 事件发生时间
	Items          []*RecommendEventItem     // 推荐事件商品项
}

// RecommendEventItem 表示推荐事件里的单商品事实。
type RecommendEventItem struct {
	GoodsId  int64 // 商品编号
	GoodsNum int64 // 商品数量
	Position int32 // 推荐位次
}

// DispatchRecommendEvent 将推荐事件转换为队列消息并投递到本地推荐事件链路。
func DispatchRecommendEvent(actor *app.RecommendActor, req *app.RecommendEventReportRequest, eventTime time.Time) {
	event := buildRecommendEventReportEvent(actor, req, eventTime)
	// 当前请求无法构造成有效推荐事件时，不再继续投递。
	if event == nil {
		return
	}
	AddQueue(_const.RecommendEventReport, event)
}

// buildRecommendEventReportEvent 构建推荐事件队列载荷。
func buildRecommendEventReportEvent(actor *app.RecommendActor, req *app.RecommendEventReportRequest, eventTime time.Time) *RecommendEventReportEvent {
	// 请求体为空时，无法继续构建事件消息。
	if req == nil {
		return nil
	}
	// 主体缺失或主体 ID 非法时，不投递无法归因的行为事件。
	if actor == nil || actor.GetActorId() <= 0 {
		return nil
	}

	eventType := req.GetEventType()
	// 未知行为类型不投递，避免污染后续聚合口径。
	if eventType == common.RecommendEventType_UNKNOWN_RET {
		return nil
	}

	// 调用方未显式传入事件时间时，统一回退到当前时间。
	if eventTime.IsZero() {
		eventTime = time.Now()
	}

	recommendContext := req.GetRecommendContext()
	scene := int32(0)
	requestId := int64(0)
	// 事件请求携带推荐归因上下文时，再补齐场景和请求编号。
	if recommendContext != nil {
		scene = int32(recommendContext.GetScene())
		requestId = recommendContext.GetRequestId()
	}

	items := req.GetItems()
	recommendEventItems := make([]*RecommendEventItem, 0, len(items))
	for _, item := range items {
		// 商品项为空或商品 ID 非法时，直接跳过当前事件项。
		if item == nil || item.GetGoodsId() <= 0 {
			continue
		}
		recommendEventItems = append(recommendEventItems, &RecommendEventItem{
			GoodsId:  item.GetGoodsId(),
			GoodsNum: item.GetGoodsNum(),
			Position: item.GetPosition(),
		})
	}
	// 当前请求没有有效商品项时，不再继续投递队列消息。
	if len(recommendEventItems) == 0 {
		return nil
	}

	return &RecommendEventReportEvent{
		RecommendActor: actor,
		EventType:      eventType,
		Scene:          scene,
		RequestId:      requestId,
		EventTime:      eventTime,
		Items:          recommendEventItems,
	}
}

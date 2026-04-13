package utils

import (
	"time"

	"shop/api/gen/go/app"
	"shop/api/gen/go/common"
	_const "shop/pkg/const"
	"shop/pkg/gen/models"
	recommendEvent "shop/pkg/recommend/event"
	appDto "shop/service/app/dto"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/liujitcn/kratos-kit/sdk"
)

// RecommendGoodsActionEvent 表示推荐商品行为事件。
type RecommendGoodsActionEvent struct {
	RecommendActor *appDto.RecommendActor          // 推荐主体信息
	EventType      common.RecommendGoodsActionType // 商品行为事件类型
	EventTime      time.Time                       // 事件发生时间
	GoodsItems     []*models.RecommendGoodsAction  // 商品行为列表
}

// AddQueue 向运行时队列追加异步消息。
func AddQueue(queue _const.Queue, data any) {
	queueId := string(queue)
	// 运行时队列未初始化时，直接跳过异步投递。
	q := sdk.Runtime.GetQueue()
	// 运行时队列未初始化时，直接跳过异步投递。
	if q == nil {
		return
	}

	messageData := map[string]any{
		"data": data,
	}
	message, err := sdk.Runtime.GetStreamMessage(queueId, messageData)
	if err != nil {
		log.Errorf("GetStreamMessage error, %s", err.Error())
		return
	}

	err = q.Append(queueId, message)
	// 队列追加失败时，只记录日志，不影响主流程。
	if err != nil {
		log.Errorf("Append message error, %s", err.Error())
	}
}

// DispatchRecommendGoodsActionEvent 将后端事实转换为推荐行为事件并投递到队列。
func DispatchRecommendGoodsActionEvent(actor *appDto.RecommendActor, req *app.RecommendGoodsActionReportRequest, eventTime time.Time) {
	// 请求体为空时，无法继续构建行为事件。
	if req == nil {
		return
	}
	// 主体缺失或主体 ID 非法时，不投递无法归因的行为事件。
	if actor == nil || actor.ActorId <= 0 {
		return
	}

	eventType := req.GetEventType()
	// 未知行为类型不投递，避免污染后续聚合口径。
	if eventType == common.RecommendGoodsActionType_UNKNOWN_RGAT {
		return
	}

	// 调用方未显式传入事件时间时，统一回退到当前时间。
	if eventTime.IsZero() {
		eventTime = time.Now()
	}

	goodsItems := req.GetGoodsItems()
	recommendGoodsActions := make([]*models.RecommendGoodsAction, 0, len(goodsItems))
	for _, item := range goodsItems {
		// 商品项为空或商品 ID 非法时，直接跳过当前行为项。
		if item == nil || item.GetGoodsId() <= 0 {
			continue
		}
		recommendContext := item.GetRecommendContext()
		recommendGoodsActions = append(recommendGoodsActions, &models.RecommendGoodsAction{
			ActorType: actor.ActorType,
			ActorID:   actor.ActorId,
			EventType: int32(eventType),
			GoodsID:   item.GetGoodsId(),
			GoodsNum:  recommendEvent.NormalizeGoodsCount(item.GetGoodsNum()),
			Scene:     int32(recommendContext.GetScene()),
			RequestID: recommendContext.GetRequestId(),
			Position:  recommendContext.GetPosition(),
			CreatedAt: eventTime,
		})
	}
	// 没有有效商品明细时，不返回空行为事件。
	// 当前请求没有有效商品行为时，不再继续投递队列消息。
	if len(recommendGoodsActions) == 0 {
		return
	}

	AddQueue(_const.RecommendGoodsActionEvent, &RecommendGoodsActionEvent{
		RecommendActor: actor,
		EventType:      eventType,
		EventTime:      eventTime,
		GoodsItems:     recommendGoodsActions,
	})
}

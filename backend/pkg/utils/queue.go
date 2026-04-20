package utils

import (
	"bytes"
	"encoding/json"
	"time"

	"shop/api/gen/go/app"
	"shop/api/gen/go/common"
	_const "shop/pkg/const"

	"github.com/go-kratos/kratos/v2/log"
	queueData "github.com/liujitcn/kratos-kit/queue/data"
	"github.com/liujitcn/kratos-kit/sdk"
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

// AddQueue 向运行时队列追加异步消息。
func AddQueue(queue _const.Queue, data any) {
	queueId := string(queue)
	// 运行时队列未初始化时，直接跳过异步投递。
	q := sdk.Runtime.GetQueue()
	// 运行时队列未初始化时，直接跳过异步投递。
	if q == nil {
		return
	}

	messageData, err := buildQueueMessageData(data)
	// 队列消息体无法序列化时，只记录日志，不影响主流程。
	if err != nil {
		log.Errorf("build queue message data error, %s", err.Error())
		return
	}
	// 消息编号交由底层队列适配器决定，业务层不直接感知 Redis 的 `*` 约定。
	var message queueData.Message
	message, err = sdk.Runtime.GetStreamMessage("", messageData)
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

// buildQueueMessageData 将任意消息体编码成 Redis 队列可接收的键值结构。
func buildQueueMessageData(data any) (map[string]any, error) {
	rawBody, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"data": string(rawBody),
	}, nil
}

// DecodeQueueData 解析队列消息中的 data 字段，并兼容内存队列与 Redis 队列两种载荷形态。
func DecodeQueueData[T any](message queueData.Message) (*T, error) {
	rawData, ok := message.Values["data"]
	// 队列消息里没有 data 字段时，说明当前消息无需继续处理。
	if !ok || rawData == nil {
		return nil, nil
	}

	switch value := rawData.(type) {
	// Redis 队列会把复杂对象保存成 JSON 字符串，这里直接按目标类型解析。
	case string:
		return decodeQueueDataBytes[T]([]byte(value))
	// 少数字节载荷场景复用同一套 JSON 解析逻辑。
	case []byte:
		return decodeQueueDataBytes[T](value)
	default:
		// 内存队列仍可能直接传递结构体对象，这里先转成 JSON 再统一解析。
		rawBody, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		return decodeQueueDataBytes[T](rawBody)
	}
}

// decodeQueueDataBytes 将 JSON 字节载荷还原成业务消息对象。
func decodeQueueDataBytes[T any](rawBody []byte) (*T, error) {
	trimmedBody := bytes.TrimSpace(rawBody)
	// 空载荷或 null 载荷都视为当前消息没有有效业务数据。
	if len(trimmedBody) == 0 || bytes.Equal(trimmedBody, []byte("null")) {
		return nil, nil
	}

	var data T
	err := json.Unmarshal(trimmedBody, &data)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

// DispatchRecommendEvent 将推荐事件转换为队列消息并投递。
func DispatchRecommendEvent(actor *app.RecommendActor, req *app.RecommendEventReportRequest, eventTime time.Time) {
	// 请求体为空时，无法继续构建事件消息。
	if req == nil {
		return
	}
	// 主体缺失或主体 ID 非法时，不投递无法归因的行为事件。
	if actor == nil || actor.ActorId <= 0 {
		return
	}

	eventType := req.GetEventType()
	// 未知行为类型不投递，避免污染后续聚合口径。
	if eventType == common.RecommendEventType_UNKNOWN_RET {
		return
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
		return
	}

	AddQueue(_const.RecommendEventReport, &RecommendEventReportEvent{
		RecommendActor: actor,
		EventType:      eventType,
		Scene:          scene,
		RequestId:      requestId,
		EventTime:      eventTime,
		Items:          recommendEventItems,
	})
}

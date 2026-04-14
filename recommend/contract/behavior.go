package contract

import (
	"context"
	"time"
)

// SessionEvent 表示会话召回所需的最近行为事件。
type SessionEvent struct {
	// GoodsId 表示会话事件关联的商品编号。
	GoodsId int64
	// EventType 表示会话事件类型。
	EventType string
	// CreatedAt 表示会话事件发生时间。
	CreatedAt time.Time
}

// BehaviorEvent 表示离线构建使用的历史行为事件。
type BehaviorEvent struct {
	// ActorType 表示事件主体类型。
	ActorType int32
	// ActorId 表示事件主体编号。
	ActorId int64
	// Scene 表示事件所属推荐场景。
	Scene string
	// RequestId 表示事件关联的推荐请求编号。
	RequestId string
	// GoodsId 表示事件关联的商品编号。
	GoodsId int64
	// EventType 表示行为事件类型。
	EventType string
	// GoodsNum 表示事件关联的商品数量。
	GoodsNum int64
	// CreatedAt 表示事件发生时间。
	CreatedAt time.Time
}

// BehaviorSource 定义推荐所需的行为数据来源。
type BehaviorSource interface {
	ListSessionEvents(context.Context, int32, int64, int32) ([]*SessionEvent, error)
	ListBehaviorEvents(context.Context, int32, int64, time.Time, int32) ([]*BehaviorEvent, error)
}

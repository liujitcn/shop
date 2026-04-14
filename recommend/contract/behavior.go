package contract

import (
	"context"
	"time"
)

// SessionEvent 表示会话召回所需的最近行为事件。
type SessionEvent struct {
	GoodsId   int64
	EventType string
	CreatedAt time.Time
}

// BehaviorEvent 表示离线构建使用的历史行为事件。
type BehaviorEvent struct {
	ActorType int32
	ActorId   int64
	Scene     string
	RequestId string
	GoodsId   int64
	EventType string
	GoodsNum  int64
	CreatedAt time.Time
}

// BehaviorSource 定义推荐所需的行为数据来源。
type BehaviorSource interface {
	ListSessionEvents(context.Context, int32, int64, int32) ([]*SessionEvent, error)
	ListBehaviorEvents(context.Context, int32, int64, time.Time, int32) ([]*BehaviorEvent, error)
}

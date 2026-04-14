package contract

import (
	"context"
	"time"
)

// User 表示推荐所需的最小用户画像。
type User struct {
	Id           int64
	RegisteredAt time.Time
	Tags         []string
}

// UserSource 定义推荐所需的用户数据来源。
type UserSource interface {
	GetUser(context.Context, int64) (*User, error)
}

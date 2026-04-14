package contract

import (
	"context"
	"time"
)

// User 表示推荐所需的最小用户画像。
type User struct {
	// Id 表示用户编号。
	Id int64
	// RegisteredAt 表示用户注册时间。
	RegisteredAt time.Time
	// Tags 表示用户画像标签集合。
	Tags []string
}

// UserSource 定义推荐所需的用户数据来源。
type UserSource interface {
	GetUser(context.Context, int64) (*User, error)
}

package dto

import client "github.com/gorse-io/gorse-go"

// FeedbackIterator 表示 Gorse 推荐反馈游标分页结果。
type FeedbackIterator struct {
	Cursor   string            `json:"Cursor"`   // 下一页游标
	Feedback []client.Feedback `json:"Feedback"` // 反馈列表
}

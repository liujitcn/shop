package dto

// CommentDiscussionCountRow 评论讨论数量聚合行。
type CommentDiscussionCountRow struct {
	// CommentID 评论编号
	CommentID int64 `gorm:"column:comment_id"`
	// DiscussionCount 讨论总数
	DiscussionCount int64 `gorm:"column:discussion_count"`
	// PendingDiscussionCount 待审核讨论数
	PendingDiscussionCount int64 `gorm:"column:pending_discussion_count"`
}

// CommentAiContentItem 保存评价 AI 摘要标签和内容。
type CommentAiContentItem struct {
	// Label 摘要标签
	Label string `json:"label"`
	// Content 摘要内容
	Content string `json:"content"`
}

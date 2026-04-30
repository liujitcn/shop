package dto

// CommentSummary 保存商品评价摘要统计。
type CommentSummary struct {
	TotalCount     int32
	RecentGoodRate int32
}

// CommentFilterStats 保存评价筛选统计。
type CommentFilterStats struct {
	MediaCount  int32
	GoodCount   int32
	MiddleCount int32
	BadCount    int32
}

// CommentAiContentItem 保存评价 AI 摘要标签和内容。
type CommentAiContentItem struct {
	Label   string `json:"label"`
	Content string `json:"content"`
}

// CommentIDCountRow 保存评价编号聚合数量行。
type CommentIDCountRow struct {
	// CommentID 评价编号。
	CommentID int64 `gorm:"column:comment_id"`
	// Total 聚合数量。
	Total int64 `gorm:"column:total"`
}

// CommentTargetCountRow 保存互动目标聚合数量行。
type CommentTargetCountRow struct {
	// TargetID 互动目标编号。
	TargetID int64 `gorm:"column:target_id"`
	// Total 聚合数量。
	Total int64 `gorm:"column:total"`
}

// CommentTargetReactionRow 保存当前用户互动状态行。
type CommentTargetReactionRow struct {
	// TargetID 互动目标编号。
	TargetID int64 `gorm:"column:target_id"`
	// ReactionType 互动类型。
	ReactionType int32 `gorm:"column:reaction_type"`
}

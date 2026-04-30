package queue

import _const "shop/pkg/const"

// CommentAuditEvent 表示评价或讨论审核队列消息。
type CommentAuditEvent struct {
	// TargetType 审核目标类型：1 评价，2 讨论。
	TargetType int32 `json:"targetType"`
	// TargetID 审核目标编号。
	TargetID int64 `json:"targetId"`
}

// DispatchCommentAudit 投递评价或讨论审核消息。
func DispatchCommentAudit(targetType int32, targetID int64) {
	// 审核目标非法时，不投递无效队列消息。
	if targetType <= 0 || targetID <= 0 {
		return
	}
	AddQueue(_const.COMMENT_AUDIT, &CommentAuditEvent{
		TargetType: targetType,
		TargetID:   targetID,
	})
}

// DispatchCommentAiRefresh 投递商品评价 AI 摘要刷新消息。
func DispatchCommentAiRefresh(goodsID int64) {
	// 商品编号非法时，不投递无效队列消息。
	if goodsID <= 0 {
		return
	}
	AddQueue(_const.COMMENT_AI_REFRESH, goodsID)
}

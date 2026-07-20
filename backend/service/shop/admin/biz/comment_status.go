package biz

import (
	_const "shop/pkg/const"
	"shop/pkg/errorsx"
)

// validateCommentStatus 校验人工评论审核目标状态。
func validateCommentStatus(status int32) error {
	// 人工审核操作只允许设置通过或不通过，待审核状态由提交流程创建。
	if status == _const.COMMENT_STATUS_APPROVED || status == _const.COMMENT_STATUS_REJECTED {
		return nil
	}
	return errorsx.InvalidArgument("审核状态仅支持通过或不通过")
}

// commentReviewStatusByCommentStatus 将主表审核状态转换为审核记录状态。
func commentReviewStatusByCommentStatus(status int32) int32 {
	switch status {
	case _const.COMMENT_STATUS_APPROVED:
		return _const.COMMENT_REVIEW_STATUS_APPROVED
	case _const.COMMENT_STATUS_REJECTED:
		return _const.COMMENT_REVIEW_STATUS_REJECTED
	default:
		return _const.COMMENT_REVIEW_STATUS_EXCEPTION
	}
}

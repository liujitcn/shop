package core

import _time "github.com/liujitcn/go-utils/time"

// ShouldCandidateRankAhead 判断左侧候选是否应该排在右侧候选前面。
func ShouldCandidateRankAhead(left *Candidate, right *Candidate) bool {
	// 右侧候选为空时，左侧候选默认优先。
	if right == nil || right.Goods == nil {
		return left != nil && left.Goods != nil
	}
	// 左侧候选为空时，不允许抢占更靠前位置。
	if left == nil || left.Goods == nil {
		return false
	}
	if left.FinalScore != right.FinalScore {
		return left.FinalScore > right.FinalScore
	}
	if left.ScenePopularityScore != right.ScenePopularityScore {
		return left.ScenePopularityScore > right.ScenePopularityScore
	}

	leftUpdatedAt := _time.StringTimeToTime(left.Goods.UpdatedAt)
	rightUpdatedAt := _time.StringTimeToTime(right.Goods.UpdatedAt)
	// 双方都有有效更新时间且时间不同，优先返回更新的商品。
	if leftUpdatedAt != nil && !leftUpdatedAt.IsZero() && rightUpdatedAt != nil && !rightUpdatedAt.IsZero() && !leftUpdatedAt.Equal(*rightUpdatedAt) {
		return leftUpdatedAt.After(*rightUpdatedAt)
	}
	// 仅左侧存在有效更新时间时，左侧优先。
	if leftUpdatedAt != nil && !leftUpdatedAt.IsZero() && (rightUpdatedAt == nil || rightUpdatedAt.IsZero()) {
		return true
	}
	// 仅右侧存在有效更新时间时，右侧优先。
	if rightUpdatedAt != nil && !rightUpdatedAt.IsZero() && (leftUpdatedAt == nil || leftUpdatedAt.IsZero()) {
		return false
	}
	// 所有排序信号都相同时，再按商品编号打平，避免 map 遍历顺序导致分页漂移。
	return left.Goods.Id > right.Goods.Id
}

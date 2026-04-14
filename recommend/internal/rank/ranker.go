package rank

import (
	"recommend/internal/model"
	"sort"
	"time"
)

// RankCandidates 对候选商品执行统一排序和基础打散。
func RankCandidates(candidates []*model.Candidate, options RankOptions) []*model.Candidate {
	list := filterCandidates(candidates)
	sortCandidates(list)

	maxPerCategory := options.MaxPerCategory
	// 未显式指定打散上限时，统一使用默认类目上限。
	if maxPerCategory <= 0 {
		maxPerCategory = defaultMaxPerCategory
	}
	return diversifyCandidates(list, maxPerCategory)
}

// filterCandidates 过滤空候选和缺失商品实体的候选。
func filterCandidates(candidates []*model.Candidate) []*model.Candidate {
	list := make([]*model.Candidate, 0, len(candidates))
	for _, item := range candidates {
		// 空候选或缺失商品实体时，当前候选无法参与排序。
		if item == nil || item.Goods == nil {
			continue
		}
		list = append(list, item)
	}
	return list
}

// sortCandidates 对候选商品执行稳定排序。
func sortCandidates(candidates []*model.Candidate) {
	sort.SliceStable(candidates, func(i, j int) bool {
		left := candidates[i]
		right := candidates[j]

		// 最终分不同时时，优先按最终分倒序排序。
		if left.Score.FinalScore != right.Score.FinalScore {
			return left.Score.FinalScore > right.Score.FinalScore
		}
		// 最终分相同时，优先按商品关联得分排序。
		if left.Score.RelationScore != right.Score.RelationScore {
			return left.Score.RelationScore > right.Score.RelationScore
		}
		// 商品关联仍相同时，优先按场景热度得分排序。
		if left.Score.SceneHotScore != right.Score.SceneHotScore {
			return left.Score.SceneHotScore > right.Score.SceneHotScore
		}
		// 场景热度仍相同时，优先按新鲜度排序。
		if left.Score.FreshnessScore != right.Score.FreshnessScore {
			return left.Score.FreshnessScore > right.Score.FreshnessScore
		}

		leftTime := resolveSortTime(left)
		rightTime := resolveSortTime(right)
		// 时间都存在且不同时时，优先返回更新更近的商品。
		if !leftTime.Equal(rightTime) {
			return leftTime.After(rightTime)
		}
		return left.Goods.Id < right.Goods.Id
	})
}

// resolveSortTime 解析排序使用的时间字段。
func resolveSortTime(candidate *model.Candidate) time.Time {
	if candidate == nil || candidate.Goods == nil {
		return time.Time{}
	}
	result := candidate.Goods.UpdatedAt
	// 更新时间缺失时，回退到创建时间，保证排序可稳定比较。
	if result.IsZero() {
		result = candidate.Goods.CreatedAt
	}
	return result
}

// diversifyCandidates 按类目做基础打散。
func diversifyCandidates(candidates []*model.Candidate, maxPerCategory int) []*model.Candidate {
	categoryCount := make(map[int64]int, len(candidates))
	result := make([]*model.Candidate, 0, len(candidates))
	overflow := make([]*model.Candidate, 0)

	for _, item := range candidates {
		categoryId := item.CategoryId()
		// 当前类目未超上限时，优先进入主结果集。
		if categoryId <= 0 || categoryCount[categoryId] < maxPerCategory {
			categoryCount[categoryId]++
			result = append(result, item)
			continue
		}
		overflow = append(overflow, item)
	}
	return append(result, overflow...)
}

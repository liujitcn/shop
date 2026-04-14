package replace

import "recommend/internal/model"

const defaultMaxPerCategory = 2

// DiversifyByCategory 对结果按类目进行基础打散。
func DiversifyByCategory(candidates []*model.Candidate, maxPerCategory int) []*model.Candidate {
	// 未显式指定类目上限时，统一使用默认值。
	if maxPerCategory <= 0 {
		maxPerCategory = defaultMaxPerCategory
	}

	categoryCount := make(map[int64]int, len(candidates))
	result := make([]*model.Candidate, 0, len(candidates))
	overflow := make([]*model.Candidate, 0)

	for _, item := range candidates {
		// 空候选或缺失商品实体时，当前候选不能进入最终结果。
		if item == nil || item.Goods == nil {
			continue
		}

		categoryId := item.Goods.CategoryId
		// 当前类目未达到上限时，优先进入主结果集。
		if categoryId <= 0 || categoryCount[categoryId] < maxPerCategory {
			categoryCount[categoryId]++
			result = append(result, item)
			continue
		}
		overflow = append(overflow, item)
	}
	return append(result, overflow...)
}

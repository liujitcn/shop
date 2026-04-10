package filter

import (
	"shop/api/gen/go/app"
	recommendCore "shop/pkg/recommend/core"
)

const DefaultMaxPerCategory = 2

// DiversifyCandidates 对排序后的候选列表执行类目打散。
func DiversifyCandidates(candidates []*recommendCore.Candidate, maxPerCategory int) []*app.GoodsInfo {
	if len(candidates) == 0 {
		// 空候选集直接返回空商品列表。
		return []*app.GoodsInfo{}
	}
	if maxPerCategory <= 0 {
		// 非法上限回退到默认类目限制。
		maxPerCategory = DefaultMaxPerCategory
	}
	result := make([]*app.GoodsInfo, 0, len(candidates))
	categoryCount := make(map[int64]int, len(candidates))
	overflow := make([]*app.GoodsInfo, 0)
	for _, item := range candidates {
		if item == nil || item.Goods == nil {
			// 跳过缺失商品实体的候选。
			continue
		}
		categoryId := item.Goods.CategoryId
		if categoryId > 0 && categoryCount[categoryId] >= maxPerCategory {
			// 单个类目达到上限后先放入溢出区，保持结果多样性。
			overflow = append(overflow, item.Goods)
			continue
		}
		categoryCount[categoryId]++
		result = append(result, item.Goods)
	}
	return append(result, overflow...)
}

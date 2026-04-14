package replace

import "recommend/internal/model"

// MergeFallback 将兜底候选按顺序补入主候选集。
func MergeFallback(primary []*model.Candidate, fallback []*model.Candidate, limit int) []*model.Candidate {
	result := make([]*model.Candidate, 0, len(primary)+len(fallback))
	goodsIdMap := make(map[int64]struct{}, len(primary)+len(fallback))

	appendCandidate := func(item *model.Candidate) bool {
		// 空候选或缺失商品实体时，当前候选不能进入最终结果。
		if item == nil || item.Goods == nil || item.Goods.Id <= 0 {
			return false
		}
		_, ok := goodsIdMap[item.Goods.Id]
		if ok {
			return false
		}
		goodsIdMap[item.Goods.Id] = struct{}{}
		result = append(result, item)
		return true
	}

	for _, item := range primary {
		appendCandidate(item)
		// 达到结果上限后，不再继续补足更多候选。
		if limit > 0 && len(result) >= limit {
			return result
		}
	}
	for _, item := range fallback {
		appendCandidate(item)
		// 达到结果上限后，不再继续补足更多候选。
		if limit > 0 && len(result) >= limit {
			return result
		}
	}
	return result
}

package recall

import (
	"context"
	"recommend/internal/model"
)

// RecallLatest 召回最新商品。
func RecallLatest(ctx context.Context, request Request) ([]*model.Candidate, error) {
	limit := ResolveLimit(request.Limit)
	list, err := request.Dependencies.Goods.ListLatestGoods(ctx, limit)
	if err != nil {
		return nil, err
	}

	candidates := make([]*model.Candidate, 0, len(list))
	for _, item := range list {
		// 缺失商品实体时，不参与最新商品召回结果。
		if item == nil || item.Id <= 0 {
			continue
		}
		candidate := model.BuildCandidate(item)
		candidate.AddRecallSource(RecallSourceLatest)
		candidates = append(candidates, candidate)
	}
	return candidates, nil
}

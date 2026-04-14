package recall

import (
	"context"
	"recommend/internal/model"
)

// RecallGlobalHot 召回全站热销商品。
func RecallGlobalHot(ctx context.Context, request Request) ([]*model.Candidate, error) {
	rows, err := request.Dependencies.Recommend.ListGlobalHotGoods(ctx, request.ReferenceTime, ResolveLimit(request.Limit))
	if err != nil {
		return nil, err
	}
	return buildWeightedGoodsCandidates(ctx, request.Dependencies.Goods, rows, RecallSourceGlobalHot, func(candidate *model.Candidate, score float64) {
		candidate.Score.GlobalHotScore = score
	})
}

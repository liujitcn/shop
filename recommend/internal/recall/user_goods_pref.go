package recall

import (
	"context"
	"recommend/internal/model"
)

// RecallUserGoodsPreference 召回用户商品偏好结果。
func RecallUserGoodsPreference(ctx context.Context, request Request) ([]*model.Candidate, error) {
	// 登录用户缺失时，用户商品偏好召回无法执行。
	if !request.Actor.IsUser() {
		return []*model.Candidate{}, nil
	}

	rows, err := request.Dependencies.Recommend.ListUserGoodsPreference(ctx, request.Actor.Id, ResolveLimit(request.Limit))
	if err != nil {
		return nil, err
	}
	return buildWeightedGoodsCandidates(ctx, request.Dependencies.Goods, rows, RecallSourceUserGoods, func(candidate *model.Candidate, score float64) {
		candidate.Score.UserGoodsScore = score
	})
}

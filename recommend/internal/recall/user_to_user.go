package recall

import (
	"context"
	"recommend/internal/model"
)

// RecallUserToUser 召回 user-to-user 增强结果。
func RecallUserToUser(ctx context.Context, request Request) ([]*model.Candidate, error) {
	// 登录用户缺失时，user-to-user 召回无法执行。
	if !request.Actor.IsUser() {
		return []*model.Candidate{}, nil
	}

	rows, err := request.Dependencies.Recommend.ListUserToUserGoods(ctx, request.Actor.Id, ResolveLimit(request.Limit))
	if err != nil {
		return nil, err
	}
	return buildWeightedGoodsCandidates(ctx, request.Dependencies.Goods, rows, RecallSourceUserToUser, func(candidate *model.Candidate, score float64) {
		candidate.Score.UserNeighborScore = score
	})
}

package recall

import (
	"context"
	"recommend/internal/model"
)

// RecallCollaborative 召回协同过滤增强结果。
func RecallCollaborative(ctx context.Context, request Request) ([]*model.Candidate, error) {
	// 登录用户缺失时，协同过滤召回无法执行。
	if !request.Actor.IsUser() {
		return []*model.Candidate{}, nil
	}

	rows, err := request.Dependencies.Recommend.ListCollaborativeGoods(ctx, request.Actor.Id, ResolveLimit(request.Limit))
	if err != nil {
		return nil, err
	}
	return buildWeightedGoodsCandidates(ctx, request.Dependencies.Goods, rows, RecallSourceCollaborative, func(candidate *model.Candidate, score float64) {
		candidate.Score.CollaborativeScore = score
	})
}

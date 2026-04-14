package recall

import (
	"context"
	"recommend/internal/model"
)

// RecallSessionContext 基于当前主体最近会话行为召回上下文商品。
func RecallSessionContext(ctx context.Context, request Request) ([]*model.Candidate, error) {
	rows, err := buildSessionWeightedGoods(ctx, request.Dependencies.Behavior, request.Dependencies.Recommend, request.Actor, ResolveLimit(request.Limit))
	if err != nil {
		return nil, err
	}
	return buildWeightedGoodsCandidates(ctx, request.Dependencies.Goods, rows, RecallSourceSession, func(candidate *model.Candidate, score float64) {
		candidate.Score.SessionScore = score
	})
}

package recall

import (
	"context"
	"recommend/internal/model"
)

// RecallExternal 召回活动池、营销池等外部结果。
func RecallExternal(ctx context.Context, request Request) ([]*model.Candidate, error) {
	// 外部策略未指定时，不执行外部召回。
	if request.Context.ExternalStrategy == "" {
		return []*model.Candidate{}, nil
	}

	rows, err := request.Dependencies.Recommend.ListExternalGoods(
		ctx,
		request.Scene.String(),
		request.Context.ExternalStrategy,
		int32(request.Actor.Type),
		request.Actor.Id,
		ResolveLimit(request.Limit),
	)
	if err != nil {
		return nil, err
	}
	return buildWeightedGoodsCandidates(ctx, request.Dependencies.Goods, rows, RecallSourceExternal, func(candidate *model.Candidate, score float64) {
		candidate.Score.ExternalScore = score
	})
}

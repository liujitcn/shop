package recall

import (
	"context"
	"errors"
	"recommend/internal/model"

	goleveldb "github.com/syndtr/goleveldb/leveldb"
)

// RecallExternal 召回活动池、营销池等外部结果。
func RecallExternal(ctx context.Context, request Request) ([]*model.Candidate, error) {
	// 外部策略未指定时，不执行外部召回。
	if request.Context.ExternalStrategy == "" {
		return []*model.Candidate{}, nil
	}

	if request.PoolStore != nil {
		pool, err := request.PoolStore.GetExternalPool(
			request.Scene.String(),
			request.Context.ExternalStrategy,
			int32(request.Actor.Type),
			request.Actor.Id,
		)
		if err == nil {
			return buildPoolGoodsCandidates(ctx, request.Dependencies.Goods, pool.GetItems(), RecallSourceExternal, func(candidate *model.Candidate, score float64) {
				candidate.Score.ExternalScore = score
			})
		}
		// 外部池不存在时回退到事实源，避免策略刚发布时出现空召回。
		if !errors.Is(err, goleveldb.ErrNotFound) {
			return nil, err
		}
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

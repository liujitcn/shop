package recall

import (
	"context"
	"errors"
	"recommend/internal/model"

	goleveldb "github.com/syndtr/goleveldb/leveldb"
)

// RecallCollaborative 召回协同过滤增强结果。
func RecallCollaborative(ctx context.Context, request Request) ([]*model.Candidate, error) {
	// 登录用户缺失时，协同过滤召回无法执行。
	if !request.Actor.IsUser() {
		return []*model.Candidate{}, nil
	}

	if request.PoolStore != nil {
		pool, err := request.PoolStore.GetCollaborativePool(request.Scene.String(), request.Actor.Id)
		if err == nil {
			return buildPoolGoodsCandidates(ctx, request.Dependencies.Goods, pool.GetItems(), RecallSourceCollaborative, func(candidate *model.Candidate, score float64) {
				candidate.Score.CollaborativeScore = score
			})
		}
		// 协同过滤池不存在时回退到事实源，避免离线池未构建时丢失增强召回。
		if !errors.Is(err, goleveldb.ErrNotFound) {
			return nil, err
		}
	}

	rows, err := request.Dependencies.Recommend.ListCollaborativeGoods(ctx, request.Actor.Id, ResolveLimit(request.Limit))
	if err != nil {
		return nil, err
	}
	return buildWeightedGoodsCandidates(ctx, request.Dependencies.Goods, rows, RecallSourceCollaborative, func(candidate *model.Candidate, score float64) {
		candidate.Score.CollaborativeScore = score
	})
}

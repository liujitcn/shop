package recall

import (
	"context"
	"errors"
	"recommend/internal/model"

	goleveldb "github.com/syndtr/goleveldb/leveldb"
)

// RecallUserToUser 召回 user-to-user 增强结果。
func RecallUserToUser(ctx context.Context, request Request) ([]*model.Candidate, error) {
	// 登录用户缺失时，user-to-user 召回无法执行。
	if !request.Actor.IsUser() {
		return []*model.Candidate{}, nil
	}

	if request.PoolStore != nil {
		pool, err := request.PoolStore.GetUserNeighborPool(request.Actor.Id)
		if err == nil {
			list, buildErr := buildPoolGoodsCandidates(ctx, request.Dependencies.Goods, pool.GetItems(), RecallSourceUserToUser, func(candidate *model.Candidate, score float64) {
				candidate.Score.UserNeighborScore = score
			})
			if buildErr != nil {
				return nil, buildErr
			}
			// 邻居池包含 user-to-user 商品项时，优先使用离线池结果。
			if len(list) > 0 {
				return list, nil
			}
		}
		// 邻居池不存在时，回退到推荐事实源。
		if err != nil && !errors.Is(err, goleveldb.ErrNotFound) {
			return nil, err
		}
	}

	rows, err := request.Dependencies.Recommend.ListUserToUserGoods(ctx, request.Actor.Id, ResolveLimit(request.Limit))
	if err != nil {
		return nil, err
	}
	return buildWeightedGoodsCandidates(ctx, request.Dependencies.Goods, rows, RecallSourceUserToUser, func(candidate *model.Candidate, score float64) {
		candidate.Score.UserNeighborScore = score
	})
}

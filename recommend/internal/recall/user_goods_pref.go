package recall

import (
	"context"
	"errors"
	"recommend/internal/model"

	goleveldb "github.com/syndtr/goleveldb/leveldb"
)

// RecallUserGoodsPreference 召回用户商品偏好结果。
func RecallUserGoodsPreference(ctx context.Context, request Request) ([]*model.Candidate, error) {
	// 登录用户缺失时，用户商品偏好召回无法执行。
	if !request.Actor.IsUser() {
		return []*model.Candidate{}, nil
	}

	if request.PoolStore != nil {
		pool, poolErr := request.PoolStore.GetUserCandidatePool(request.Scene.String(), request.Actor.Id)
		if poolErr == nil {
			list, buildErr := buildPoolGoodsCandidates(ctx, request.Dependencies.Goods, pool.GetItems(), RecallSourceUserGoods, func(candidate *model.Candidate, score float64) {
				candidate.Score.UserGoodsScore = score
			})
			if buildErr != nil {
				return nil, buildErr
			}
			// 用户候选池存在且能恢复商品偏好原始分值时，优先使用离线池结果。
			if len(list) > 0 {
				return list, nil
			}
		}
		// 用户候选池不存在时，回退到推荐事实源。
		if poolErr != nil && !errors.Is(poolErr, goleveldb.ErrNotFound) {
			return nil, poolErr
		}
	}

	rows, err := request.Dependencies.Recommend.ListUserGoodsPreference(ctx, request.Actor.Id, ResolveLimit(request.Limit))
	if err != nil {
		return nil, err
	}
	return buildWeightedGoodsCandidates(ctx, request.Dependencies.Goods, rows, RecallSourceUserGoods, func(candidate *model.Candidate, score float64) {
		candidate.Score.UserGoodsScore = score
	})
}

package recall

import (
	"context"
	"errors"
	"recommend/internal/model"

	goleveldb "github.com/syndtr/goleveldb/leveldb"
)

// RecallUserCategoryPreference 召回用户类目偏好结果。
func RecallUserCategoryPreference(ctx context.Context, request Request) ([]*model.Candidate, error) {
	// 登录用户缺失时，用户类目偏好召回无法执行。
	if !request.Actor.IsUser() {
		return []*model.Candidate{}, nil
	}

	if request.PoolStore != nil {
		pool, poolErr := request.PoolStore.GetUserCandidatePool(request.Scene.String(), request.Actor.Id)
		if poolErr == nil {
			list, buildErr := buildPoolGoodsCandidates(ctx, request.Dependencies.Goods, pool.GetItems(), RecallSourceUserCategory, func(candidate *model.Candidate, score float64) {
				candidate.Score.CategoryScore = score
			})
			if buildErr != nil {
				return nil, buildErr
			}
			// 用户候选池存在且能恢复类目偏好原始分值时，优先使用离线池结果。
			if len(list) > 0 {
				return list, nil
			}
		}
		// 用户候选池不存在时，回退到推荐事实源。
		if poolErr != nil && !errors.Is(poolErr, goleveldb.ErrNotFound) {
			return nil, poolErr
		}
	}

	rows, err := request.Dependencies.Recommend.ListUserCategoryPreference(ctx, request.Actor.Id, ResolveLimit(request.Limit))
	if err != nil {
		return nil, err
	}
	return buildCategoryCandidates(ctx, request.Dependencies.Goods, rows, ResolveLimit(request.Limit))
}

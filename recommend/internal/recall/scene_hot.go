package recall

import (
	"context"
	"errors"
	"recommend/internal/core"
	"recommend/internal/model"

	goleveldb "github.com/syndtr/goleveldb/leveldb"
)

// RecallSceneHot 召回场景热销商品。
func RecallSceneHot(ctx context.Context, request Request) ([]*model.Candidate, error) {
	if request.PoolStore != nil {
		pool, poolErr := request.PoolStore.GetCandidatePool(request.Scene.String(), int32(core.ActorTypeAnonymous), 0)
		if poolErr == nil {
			list, buildErr := buildPoolGoodsCandidates(ctx, request.Dependencies.Goods, pool.GetItems(), RecallSourceSceneHot, func(candidate *model.Candidate, score float64) {
				candidate.Score.SceneHotScore = score
			})
			if buildErr != nil {
				return nil, buildErr
			}
			// 非个性化池存在且能恢复场景热度原始分值时，优先使用离线池结果。
			if len(list) > 0 {
				return list, nil
			}
		}
		// 匿名通用池不存在时，回退到推荐事实源。
		if poolErr != nil && !errors.Is(poolErr, goleveldb.ErrNotFound) {
			return nil, poolErr
		}
	}

	rows, err := request.Dependencies.Recommend.ListSceneHotGoods(ctx, request.Scene.String(), request.ReferenceTime, ResolveLimit(request.Limit))
	if err != nil {
		return nil, err
	}
	return buildWeightedGoodsCandidates(ctx, request.Dependencies.Goods, rows, RecallSourceSceneHot, func(candidate *model.Candidate, score float64) {
		candidate.Score.SceneHotScore = score
	})
}

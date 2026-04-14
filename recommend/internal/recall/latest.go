package recall

import (
	"context"
	"errors"
	"recommend/internal/core"
	"recommend/internal/model"

	goleveldb "github.com/syndtr/goleveldb/leveldb"
)

// RecallLatest 召回最新商品。
func RecallLatest(ctx context.Context, request Request) ([]*model.Candidate, error) {
	if request.PoolStore != nil && request.Scene != "" {
		pool, poolErr := request.PoolStore.GetCandidatePool(request.Scene.String(), int32(core.ActorTypeAnonymous), 0)
		if poolErr == nil {
			list, buildErr := buildPoolGoodsCandidates(ctx, request.Dependencies.Goods, pool.GetItems(), RecallSourceLatest, nil)
			if buildErr != nil {
				return nil, buildErr
			}
			// 非个性化池存在且能还原最新商品来源时，优先使用离线池结果。
			if len(list) > 0 {
				return list, nil
			}
		}
		// 匿名通用池不存在时，回退到商品事实源。
		if poolErr != nil && !errors.Is(poolErr, goleveldb.ErrNotFound) {
			return nil, poolErr
		}
	}

	limit := ResolveLimit(request.Limit)
	list, err := request.Dependencies.Goods.ListLatestGoods(ctx, limit)
	if err != nil {
		return nil, err
	}

	candidates := make([]*model.Candidate, 0, len(list))
	for _, item := range list {
		// 缺失商品实体时，不参与最新商品召回结果。
		if item == nil || item.Id <= 0 {
			continue
		}
		candidate := model.BuildCandidate(item)
		candidate.AddRecallSource(RecallSourceLatest)
		candidates = append(candidates, candidate)
	}
	return candidates, nil
}

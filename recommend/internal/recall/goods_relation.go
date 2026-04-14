package recall

import (
	"context"
	"errors"
	"recommend/internal/model"

	goleveldb "github.com/syndtr/goleveldb/leveldb"
)

// RecallGoodsRelation 召回与当前商品强相关的商品。
func RecallGoodsRelation(ctx context.Context, request Request) ([]*model.Candidate, error) {
	// 当前上下文缺少商品编号时，商品关联召回无法执行。
	if request.Context.GoodsId <= 0 {
		return []*model.Candidate{}, nil
	}

	if request.PoolStore != nil {
		pool, err := request.PoolStore.GetRelatedGoodsPool(request.Scene.String(), request.Context.GoodsId)
		if err == nil {
			return buildPoolGoodsCandidates(ctx, request.Dependencies.Goods, pool.GetItems(), RecallSourceGoodsRelation, func(candidate *model.Candidate, score float64) {
				candidate.Score.RelationScore = score
			})
		}
		// 关联池不存在时回退到事实源，避免离线构建未完成时直接丢召回。
		if !errors.Is(err, goleveldb.ErrNotFound) {
			return nil, err
		}
	}

	rows, err := request.Dependencies.Recommend.ListRelatedGoods(ctx, request.Context.GoodsId, ResolveLimit(request.Limit))
	if err != nil {
		return nil, err
	}
	return buildWeightedGoodsCandidates(ctx, request.Dependencies.Goods, rows, RecallSourceGoodsRelation, func(candidate *model.Candidate, score float64) {
		candidate.Score.RelationScore = score
	})
}

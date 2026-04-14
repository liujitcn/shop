package recall

import (
	"context"
	"recommend/internal/model"
)

// RecallGoodsRelation 召回与当前商品强相关的商品。
func RecallGoodsRelation(ctx context.Context, request Request) ([]*model.Candidate, error) {
	// 当前上下文缺少商品编号时，商品关联召回无法执行。
	if request.Context.GoodsId <= 0 {
		return []*model.Candidate{}, nil
	}

	rows, err := request.Dependencies.Recommend.ListRelatedGoods(ctx, request.Context.GoodsId, ResolveLimit(request.Limit))
	if err != nil {
		return nil, err
	}
	return buildWeightedGoodsCandidates(ctx, request.Dependencies.Goods, rows, RecallSourceGoodsRelation, func(candidate *model.Candidate, score float64) {
		candidate.Score.RelationScore = score
	})
}

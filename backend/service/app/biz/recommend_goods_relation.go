package biz

import (
	"context"

	"shop/pkg/biz"
	"shop/pkg/gen/data"
	recommendcore "shop/pkg/recommend/core"
	recommendevent "shop/pkg/recommend/event"

	"github.com/liujitcn/gorm-kit/repo"
)

// RecommendGoodsRelationCase 推荐商品关联业务处理对象。
type RecommendGoodsRelationCase struct {
	*biz.BaseCase
	*data.RecommendGoodsRelationRepo
}

// NewRecommendGoodsRelationCase 创建推荐商品关联业务处理对象。
func NewRecommendGoodsRelationCase(baseCase *biz.BaseCase, recommendGoodsRelationRepo *data.RecommendGoodsRelationRepo) *RecommendGoodsRelationCase {
	return &RecommendGoodsRelationCase{
		BaseCase:                   baseCase,
		RecommendGoodsRelationRepo: recommendGoodsRelationRepo,
	}
}

// listRelatedGoodsIds 查询关联商品 ID 列表。
func (c *RecommendGoodsRelationCase) listRelatedGoodsIds(ctx context.Context, goodsIds []int64, limit int) ([]int64, error) {
	if len(goodsIds) == 0 || limit <= 0 {
		return []int64{}, nil
	}
	relationQuery := c.RecommendGoodsRelationRepo.Query(ctx).RecommendGoodsRelation
	list, _, err := c.RecommendGoodsRelationRepo.Page(
		ctx,
		1,
		int64(limit),
		repo.Where(relationQuery.GoodsID.In(goodsIds...)),
		repo.Order(relationQuery.Score.Desc()),
		repo.Order(relationQuery.UpdatedAt.Desc()),
	)
	if err != nil {
		return nil, err
	}

	relatedGoodsIds := make([]int64, 0, len(list))
	for _, item := range list {
		relatedGoodsIds = append(relatedGoodsIds, item.RelatedGoodsID)
	}
	return recommendcore.DedupeInt64s(relatedGoodsIds), nil
}

// loadRelationScores 加载候选商品的关联商品分数。
func (c *RecommendGoodsRelationCase) loadRelationScores(ctx context.Context, sourceGoodsIds []int64) (map[int64]float64, error) {
	if len(sourceGoodsIds) == 0 {
		return map[int64]float64{}, nil
	}
	relationQuery := c.RecommendGoodsRelationRepo.Query(ctx).RecommendGoodsRelation
	list, err := c.RecommendGoodsRelationRepo.List(ctx,
		repo.Where(relationQuery.GoodsID.In(sourceGoodsIds...)),
		repo.Where(relationQuery.WindowDays.Eq(recommendevent.AggregateWindowDays)),
	)
	if err != nil {
		return nil, err
	}

	scores := make(map[int64]float64, len(list))
	for _, item := range list {
		scores[item.RelatedGoodsID] += item.Score
	}
	return scores, nil
}

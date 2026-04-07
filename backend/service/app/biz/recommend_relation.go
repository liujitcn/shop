package biz

import (
	"context"

	"shop/pkg/biz"
	"shop/pkg/gen/data"

	"github.com/liujitcn/gorm-kit/repo"
)

// RecommendRelationCase 推荐关联业务处理对象。
type RecommendRelationCase struct {
	*biz.BaseCase
	*data.RecommendGoodsRelationRepo
}

// NewRecommendRelationCase 创建推荐关联业务处理对象。
func NewRecommendRelationCase(baseCase *biz.BaseCase, recommendGoodsRelationRepo *data.RecommendGoodsRelationRepo) *RecommendRelationCase {
	return &RecommendRelationCase{
		BaseCase:                   baseCase,
		RecommendGoodsRelationRepo: recommendGoodsRelationRepo,
	}
}

// ListRelatedGoodsIds 查询关联商品ID列表。
func (c *RecommendRelationCase) ListRelatedGoodsIds(ctx context.Context, goodsIds []int64, limit int) ([]int64, error) {
	if len(goodsIds) == 0 || limit <= 0 {
		return []int64{}, nil
	}
	relationQuery := c.RecommendGoodsRelationRepo.Query(ctx).RecommendGoodsRelation
	opts := make([]repo.QueryOption, 0, 4)
	opts = append(opts, repo.Where(relationQuery.GoodsID.In(goodsIds...)))
	opts = append(opts, repo.Order(relationQuery.Score.Desc()))
	opts = append(opts, repo.Order(relationQuery.UpdatedAt.Desc()))
	list, _, err := c.Page(ctx, 1, int64(limit), opts...)
	if err != nil {
		return nil, err
	}
	relatedGoodsIds := make([]int64, 0, len(list))
	seen := make(map[int64]struct{}, len(list))
	for _, item := range list {
		if _, ok := seen[item.RelatedGoodsID]; ok {
			continue
		}
		seen[item.RelatedGoodsID] = struct{}{}
		relatedGoodsIds = append(relatedGoodsIds, item.RelatedGoodsID)
	}
	return relatedGoodsIds, nil
}

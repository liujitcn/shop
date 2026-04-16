package biz

import (
	"context"

	"shop/pkg/biz"
	"shop/pkg/gen/data"
	recommendcore "shop/pkg/recommend/core"
	recommendEvent "shop/pkg/recommend/event"
	recommendAggregate "shop/pkg/recommend/offline/aggregate"

	"github.com/liujitcn/gorm-kit/repo"
)

// RecommendGoodsRelationCase 推荐商品关联业务处理对象。
type RecommendGoodsRelationCase struct {
	*biz.BaseCase
	*data.RecommendGoodsRelationRepo
	recommendGoodsActionRepo *data.RecommendGoodsActionRepo
	recommendRequestRepo     *data.RecommendRequestRepo
	recommendRequestItemRepo *data.RecommendRequestItemRepo
}

// NewRecommendGoodsRelationCase 创建推荐商品关联业务处理对象。
func NewRecommendGoodsRelationCase(
	baseCase *biz.BaseCase,
	recommendGoodsRelationRepo *data.RecommendGoodsRelationRepo,
	recommendGoodsActionRepo *data.RecommendGoodsActionRepo,
	recommendRequestRepo *data.RecommendRequestRepo,
	recommendRequestItemRepo *data.RecommendRequestItemRepo,
) *RecommendGoodsRelationCase {
	return &RecommendGoodsRelationCase{
		BaseCase:                   baseCase,
		RecommendGoodsRelationRepo: recommendGoodsRelationRepo,
		recommendGoodsActionRepo:   recommendGoodsActionRepo,
		recommendRequestRepo:       recommendRequestRepo,
		recommendRequestItemRepo:   recommendRequestItemRepo,
	}
}

// RebuildRecommendGoodsRelation 重建商品关联聚合。
func (c *RecommendGoodsRelationCase) RebuildRecommendGoodsRelation(ctx context.Context, windowDays int32) error {
	actionList, err := recommendAggregate.ListRelationActionFacts(ctx, c.recommendGoodsActionRepo, windowDays)
	if err != nil {
		return err
	}

	relationQuery := c.Query(ctx).RecommendGoodsRelation
	relationOpts := make([]repo.QueryOption, 0, 1)
	relationOpts = append(relationOpts, repo.Where(relationQuery.WindowDays.Eq(windowDays)))
	err = c.Delete(ctx, relationOpts...)
	if err != nil {
		return err
	}

	list, err := recommendAggregate.RebuildGoodsRelations(ctx, c.recommendRequestRepo, c.recommendRequestItemRepo, actionList, windowDays)
	if err != nil {
		return err
	}
	// 当前窗口没有沉淀出有效关联结果时，直接结束重建。
	if len(list) == 0 {
		return nil
	}
	return c.BatchCreate(ctx, list)
}

// listRelatedGoodsIds 查询关联商品 ID 列表。
func (c *RecommendGoodsRelationCase) listRelatedGoodsIds(ctx context.Context, goodsIds []int64, limit int64) ([]int64, error) {
	// 商品集合为空或限制数量非法时，直接返回空结果。
	if len(goodsIds) == 0 || limit <= 0 {
		return []int64{}, nil
	}

	query := c.Query(ctx).RecommendGoodsRelation
	opts := make([]repo.QueryOption, 0, 3)
	opts = append(opts, repo.Where(query.GoodsID.In(goodsIds...)))
	opts = append(opts, repo.Order(query.Score.Desc()))
	opts = append(opts, repo.Order(query.UpdatedAt.Desc()))

	list, _, err := c.Page(ctx, 1, limit, opts...)
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
	// 源商品为空时，不需要继续查询关联分数。
	if len(sourceGoodsIds) == 0 {
		return map[int64]float64{}, nil
	}

	query := c.Query(ctx).RecommendGoodsRelation
	opts := make([]repo.QueryOption, 0, 2)
	opts = append(opts, repo.Where(query.GoodsID.In(sourceGoodsIds...)))
	opts = append(opts, repo.Where(query.WindowDays.Eq(recommendEvent.AggregateWindowDays)))

	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	scores := make(map[int64]float64, len(list))
	for _, item := range list {
		scores[item.RelatedGoodsID] += item.Score
	}
	return scores, nil
}

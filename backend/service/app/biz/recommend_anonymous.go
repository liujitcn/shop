package biz

import (
	"context"
	"time"

	"shop/api/gen/go/common"
	"shop/pkg/gen/models"

	"github.com/liujitcn/gorm-kit/repo"
)

// listGoodsByAnonymousStats 按匿名公共热度查询商品。
func (c *RecommendCase) listGoodsByAnonymousStats(ctx context.Context, scene common.RecommendScene, pageNum, pageSize int64) ([]*models.GoodsInfo, int64, error) {
	if pageNum <= 0 || pageSize <= 0 {
		return []*models.GoodsInfo{}, 0, nil
	}

	endDate := time.Now()
	startDate := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 0, 0, 0, 0, endDate.Location()).AddDate(0, 0, -recommendAnonymousRecallDays)
	recommendGoodsIds, err := c.listSceneHotGoodsIds(ctx, scene, startDate, int(pageNum*pageSize*2))
	if err != nil {
		return nil, 0, err
	}
	goodsIds, err := c.mergeAnonymousGoodsIds(ctx, recommendGoodsIds, startDate, int(pageNum*pageSize*4))
	if err != nil {
		return nil, 0, err
	}
	total := int64(len(goodsIds))
	offset := int((pageNum - 1) * pageSize)
	if int64(offset) >= total {
		return []*models.GoodsInfo{}, total, nil
	}
	end := offset + int(pageSize)
	if int64(end) > total {
		end = int(total)
	}
	pageGoodsIds := goodsIds[offset:end]
	list, err := c.listGoodsByIds(ctx, pageGoodsIds)
	if err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// listSceneHotGoodsIds 查询场景热度商品。
func (c *RecommendCase) listSceneHotGoodsIds(ctx context.Context, scene common.RecommendScene, startDate time.Time, limit int) ([]int64, error) {
	if limit <= 0 {
		return []int64{}, nil
	}

	recommendStatQuery := c.recommendGoodsStatDayRepo.Query(ctx).RecommendGoodsStatDay
	list, _, err := c.recommendGoodsStatDayRepo.Page(
		ctx,
		1,
		int64(limit),
		repo.Where(recommendStatQuery.Scene.Eq(scene.String())),
		repo.Where(recommendStatQuery.StatDate.Gte(startDate)),
		repo.Order(recommendStatQuery.Score.Desc()),
		repo.Order(recommendStatQuery.StatDate.Desc()),
	)
	if err != nil {
		return nil, err
	}
	goodsIds := make([]int64, 0, len(list))
	for _, item := range list {
		goodsIds = append(goodsIds, item.GoodsID)
	}
	return dedupeInt64s(goodsIds), nil
}

// mergeAnonymousGoodsIds 合并场景热度与公共热度商品。
func (c *RecommendCase) mergeAnonymousGoodsIds(ctx context.Context, sceneGoodsIds []int64, startDate time.Time, limit int) ([]int64, error) {
	result := dedupeInt64s(sceneGoodsIds)
	if len(result) >= limit {
		return result[:limit], nil
	}

	goodsStatQuery := c.goodsStatDayRepo.Query(ctx).GoodsStatDay
	list, _, err := c.goodsStatDayRepo.Page(
		ctx,
		1,
		int64(limit),
		repo.Where(goodsStatQuery.StatDate.Gte(startDate)),
		repo.Order(goodsStatQuery.Score.Desc()),
		repo.Order(goodsStatQuery.StatDate.Desc()),
	)
	if err != nil {
		return nil, err
	}
	for _, item := range list {
		result = append(result, item.GoodsID)
	}
	return dedupeInt64s(result), nil
}

package biz

import (
	"context"
	"time"

	"shop/pkg/biz"
	"shop/pkg/gen/data"
	recommendcandidate "shop/pkg/recommend/candidate"
	recommendcore "shop/pkg/recommend/core"
	recommendrank "shop/pkg/recommend/rank"

	"github.com/liujitcn/gorm-kit/repo"
)

// GoodsStatDayCase 商品统计日表业务处理对象。
type GoodsStatDayCase struct {
	*biz.BaseCase
	*data.GoodsStatDayRepo
}

// NewGoodsStatDayCase 创建商品统计日表业务处理对象。
func NewGoodsStatDayCase(baseCase *biz.BaseCase, goodsStatDayRepo *data.GoodsStatDayRepo) *GoodsStatDayCase {
	return &GoodsStatDayCase{
		BaseCase:         baseCase,
		GoodsStatDayRepo: goodsStatDayRepo,
	}
}

// mergeAnonymousGoodsIds 合并场景热度与公共热度商品。
func (c *GoodsStatDayCase) mergeAnonymousGoodsIds(ctx context.Context, sceneGoodsIds []int64, startDate time.Time, limit int) ([]int64, error) {
	result := recommendcore.DedupeInt64s(sceneGoodsIds)
	if len(result) >= limit {
		return result[:limit], nil
	}

	goodsStatQuery := c.GoodsStatDayRepo.Query(ctx).GoodsStatDay
	list, _, err := c.GoodsStatDayRepo.Page(
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
	return recommendcore.DedupeInt64s(result), nil
}

// loadGlobalPopularityScores 加载全站热度分数。
func (c *GoodsStatDayCase) loadGlobalPopularityScores(ctx context.Context, goodsIds []int64) (map[int64]float64, error) {
	if len(goodsIds) == 0 {
		return map[int64]float64{}, nil
	}
	statQuery := c.GoodsStatDayRepo.Query(ctx).GoodsStatDay
	startDate := time.Now().AddDate(0, 0, -recommendcandidate.StatLookbackDays)
	list, err := c.GoodsStatDayRepo.List(ctx,
		repo.Where(statQuery.GoodsID.In(goodsIds...)),
		repo.Where(statQuery.StatDate.Gte(startDate)),
	)
	if err != nil {
		return nil, err
	}

	scores := make(map[int64]float64, len(list))
	for _, item := range list {
		scores[item.GoodsID] += item.Score * recommendrank.CalculateDayDecay(item.StatDate)
	}
	return scores, nil
}

package biz

import (
	"context"
	"time"

	"shop/pkg/biz"
	"shop/pkg/gen/data"
	recommendcandidate "shop/pkg/recommend/candidate"
	recommendCore "shop/pkg/recommend/core"
	recommendRank "shop/pkg/recommend/rank"

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
	result := recommendCore.DedupeInt64s(sceneGoodsIds)
	if len(result) >= limit {
		return result[:limit], nil
	}

	query := c.GoodsStatDayRepo.Query(ctx).GoodsStatDay
	opts := make([]repo.QueryOption, 0, 3)
	opts = append(opts, repo.Where(query.StatDate.Gte(startDate)))
	opts = append(opts, repo.Order(query.Score.Desc()))
	opts = append(opts, repo.Order(query.StatDate.Desc()))
	list, _, err := c.Page(ctx, 1, int64(limit), opts...)
	if err != nil {
		return nil, err
	}

	for _, item := range list {
		result = append(result, item.GoodsID)
	}
	return recommendCore.DedupeInt64s(result), nil
}

// loadGlobalPopularityScores 加载全站热度分数。
func (c *GoodsStatDayCase) loadGlobalPopularityScores(ctx context.Context, goodsIds []int64) (map[int64]float64, error) {
	if len(goodsIds) == 0 {
		return map[int64]float64{}, nil
	}
	statQuery := c.GoodsStatDayRepo.Query(ctx).GoodsStatDay
	startDate := time.Now().AddDate(0, 0, -recommendcandidate.StatLookbackDays)
	opts := make([]repo.QueryOption, 0, 3)
	opts = append(opts, repo.Where(statQuery.GoodsID.In(goodsIds...)))
	opts = append(opts, repo.Where(statQuery.StatDate.Gte(startDate)))
	list, err := c.GoodsStatDayRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	scores := make(map[int64]float64, len(list))
	for _, item := range list {
		scores[item.GoodsID] += item.Score * recommendRank.CalculateDayDecay(item.StatDate)
	}
	return scores, nil
}

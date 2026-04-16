package biz

import (
	"context"
	"time"

	"shop/pkg/biz"
	"shop/pkg/gen/data"
	recommendCandidate "shop/pkg/recommend/candidate"
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

// listGlobalHotGoodsIds 查询全站热度商品编号列表。
func (c *GoodsStatDayCase) listGlobalHotGoodsIds(ctx context.Context, startDate time.Time, limit int64) ([]int64, error) {
	query := c.GoodsStatDayRepo.Query(ctx).GoodsStatDay
	opts := make([]repo.QueryOption, 0, 3)
	opts = append(opts, repo.Where(query.StatDate.Gte(startDate)))
	opts = append(opts, repo.Order(query.Score.Desc()))
	opts = append(opts, repo.Order(query.StatDate.Desc()))
	list, _, err := c.Page(ctx, 1, limit, opts...)
	if err != nil {
		return nil, err
	}

	goodsIds := make([]int64, 0, len(list))
	for _, item := range list {
		goodsIds = append(goodsIds, item.GoodsID)
	}
	return recommendCore.DedupeInt64s(goodsIds), nil
}

// loadGlobalPopularityScores 加载全站热度分数。
func (c *GoodsStatDayCase) loadGlobalPopularityScores(ctx context.Context, goodsIds []int64) (map[int64]float64, error) {
	// 候选商品为空时，不需要继续加载全站热度。
	if len(goodsIds) == 0 {
		return map[int64]float64{}, nil
	}
	query := c.GoodsStatDayRepo.Query(ctx).GoodsStatDay
	startDate := time.Now().AddDate(0, 0, -recommendCandidate.StatLookbackDays)
	opts := make([]repo.QueryOption, 0, 3)
	opts = append(opts, repo.Where(query.GoodsID.In(goodsIds...)))
	opts = append(opts, repo.Where(query.StatDate.Gte(startDate)))
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

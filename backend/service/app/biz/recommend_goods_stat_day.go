package biz

import (
	"context"
	"time"

	"shop/api/gen/go/common"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	recommendCandidate "shop/pkg/recommend/candidate"
	recommendCore "shop/pkg/recommend/core"
	recommendRank "shop/pkg/recommend/rank"

	"github.com/liujitcn/gorm-kit/repo"
)

// RecommendGoodsStatDayCase 推荐商品统计日表业务处理对象。
type RecommendGoodsStatDayCase struct {
	*biz.BaseCase
	*data.RecommendGoodsStatDayRepo
}

// NewRecommendGoodsStatDayCase 创建推荐商品统计日表业务处理对象。
func NewRecommendGoodsStatDayCase(baseCase *biz.BaseCase, recommendGoodsStatDayRepo *data.RecommendGoodsStatDayRepo) *RecommendGoodsStatDayCase {
	return &RecommendGoodsStatDayCase{
		BaseCase:                  baseCase,
		RecommendGoodsStatDayRepo: recommendGoodsStatDayRepo,
	}
}

// listSceneHotGoodsIds 查询场景热度商品。
func (c *RecommendGoodsStatDayCase) listSceneHotGoodsIds(ctx context.Context, scene common.RecommendScene, startDate time.Time, limit int64) ([]int64, error) {
	query := c.RecommendGoodsStatDayRepo.Query(ctx).RecommendGoodsStatDay
	opts := make([]repo.QueryOption, 0, 4)
	opts = append(opts, repo.Where(query.Scene.Eq(int32(scene))))
	opts = append(opts, repo.Where(query.StatDate.Gte(startDate)))
	opts = append(opts, repo.Order(query.Score.Desc()))
	opts = append(opts, repo.Order(query.StatDate.Desc()))
	list, _, err := c.RecommendGoodsStatDayRepo.Page(ctx, 1, limit, opts...)
	if err != nil {
		return nil, err
	}

	goodsIds := make([]int64, 0, len(list))
	for _, item := range list {
		goodsIds = append(goodsIds, item.GoodsID)
	}
	return recommendCore.DedupeInt64s(goodsIds), nil
}

// loadScenePopularitySignals 加载场景热度和曝光惩罚信号。
func (c *RecommendGoodsStatDayCase) loadScenePopularitySignals(ctx context.Context, scene int32, goodsIds []int64) (map[int64]float64, map[int64]float64, error) {
	// 场景或商品集合为空时，不需要查询任何热度信号。
	if scene == 0 || len(goodsIds) == 0 {
		return map[int64]float64{}, map[int64]float64{}, nil
	}
	query := c.RecommendGoodsStatDayRepo.Query(ctx).RecommendGoodsStatDay
	startDate := time.Now().AddDate(0, 0, -recommendCandidate.StatLookbackDays)
	opts := make([]repo.QueryOption, 0, 3)
	opts = append(opts, repo.Where(query.Scene.Eq(scene)))
	opts = append(opts, repo.Where(query.GoodsID.In(goodsIds...)))
	opts = append(opts, repo.Where(query.StatDate.Gte(startDate)))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, nil, err
	}

	scores := make(map[int64]float64, len(list))
	penalties := make(map[int64]float64, len(list))
	for _, item := range list {
		dayDecay := recommendRank.CalculateDayDecay(item.StatDate)
		scores[item.GoodsID] += item.Score * dayDecay
		penalties[item.GoodsID] += recommendRank.CalculateExposurePenalty(item.ExposureCount, item.ClickCount) * dayDecay
	}
	return scores, penalties, nil
}

package biz

import (
	"context"
	"time"

	"shop/api/gen/go/common"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	recommendcandidate "shop/pkg/recommend/candidate"
	recommendcore "shop/pkg/recommend/core"
	recommendrank "shop/pkg/recommend/rank"

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
func (c *RecommendGoodsStatDayCase) listSceneHotGoodsIds(ctx context.Context, scene common.RecommendScene, startDate time.Time, limit int) ([]int64, error) {
	if limit <= 0 {
		return []int64{}, nil
	}
	recommendStatQuery := c.RecommendGoodsStatDayRepo.Query(ctx).RecommendGoodsStatDay
	list, _, err := c.RecommendGoodsStatDayRepo.Page(
		ctx,
		1,
		int64(limit),
		repo.Where(recommendStatQuery.Scene.Eq(int32(scene))),
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
	return recommendcore.DedupeInt64s(goodsIds), nil
}

// loadScenePopularitySignals 加载场景热度和曝光惩罚信号。
func (c *RecommendGoodsStatDayCase) loadScenePopularitySignals(ctx context.Context, scene int32, goodsIds []int64) (map[int64]float64, map[int64]float64, error) {
	if scene == 0 || len(goodsIds) == 0 {
		return map[int64]float64{}, map[int64]float64{}, nil
	}
	statQuery := c.RecommendGoodsStatDayRepo.Query(ctx).RecommendGoodsStatDay
	startDate := time.Now().AddDate(0, 0, -recommendcandidate.StatLookbackDays)
	list, err := c.RecommendGoodsStatDayRepo.List(ctx,
		repo.Where(statQuery.Scene.Eq(scene)),
		repo.Where(statQuery.GoodsID.In(goodsIds...)),
		repo.Where(statQuery.StatDate.Gte(startDate)),
	)
	if err != nil {
		return nil, nil, err
	}

	scores := make(map[int64]float64, len(list))
	penalties := make(map[int64]float64, len(list))
	for _, item := range list {
		dayDecay := recommendrank.CalculateDayDecay(item.StatDate)
		scores[item.GoodsID] += item.Score * dayDecay
		penalties[item.GoodsID] += recommendrank.CalculateExposurePenalty(item.ExposureCount, item.ClickCount) * dayDecay
	}
	return scores, penalties, nil
}

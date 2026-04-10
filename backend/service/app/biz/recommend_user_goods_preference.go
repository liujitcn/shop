package biz

import (
	"context"
	"time"

	"shop/pkg/biz"
	"shop/pkg/gen/data"
	recommendcandidate "shop/pkg/recommend/candidate"
	recommendevent "shop/pkg/recommend/event"

	"github.com/liujitcn/gorm-kit/repo"
)

// RecommendUserGoodsPreferenceCase 推荐用户商品偏好业务处理对象。
type RecommendUserGoodsPreferenceCase struct {
	*biz.BaseCase
	*data.RecommendUserGoodsPreferenceRepo
}

// NewRecommendUserGoodsPreferenceCase 创建推荐用户商品偏好业务处理对象。
func NewRecommendUserGoodsPreferenceCase(baseCase *biz.BaseCase, recommendUserGoodsPreferenceRepo *data.RecommendUserGoodsPreferenceRepo) *RecommendUserGoodsPreferenceCase {
	return &RecommendUserGoodsPreferenceCase{
		BaseCase:                         baseCase,
		RecommendUserGoodsPreferenceRepo: recommendUserGoodsPreferenceRepo,
	}
}

// loadUserGoodsSignals 加载用户对候选商品的偏好分和近期支付集合。
func (c *RecommendUserGoodsPreferenceCase) loadUserGoodsSignals(ctx context.Context, userID int64, goodsIds []int64) (map[int64]float64, map[int64]struct{}, error) {
	if userID == 0 || len(goodsIds) == 0 {
		return map[int64]float64{}, map[int64]struct{}{}, nil
	}
	preferenceQuery := c.RecommendUserGoodsPreferenceRepo.Query(ctx).RecommendUserGoodsPreference
	list, err := c.RecommendUserGoodsPreferenceRepo.List(ctx,
		repo.Where(preferenceQuery.UserID.Eq(userID)),
		repo.Where(preferenceQuery.GoodsID.In(goodsIds...)),
		repo.Where(preferenceQuery.WindowDays.Eq(recommendevent.AggregateWindowDays)),
	)
	if err != nil {
		return nil, nil, err
	}

	scores := make(map[int64]float64, len(list))
	recentPaidGoodsMap := make(map[int64]struct{})
	cutoff := time.Now().AddDate(0, 0, -recommendcandidate.RecentPayPenaltyDays)
	for _, item := range list {
		scores[item.GoodsID] = item.Score
		if item.LastBehaviorType == recommendevent.EventTypePay && item.LastBehaviorAt.After(cutoff) {
			recentPaidGoodsMap[item.GoodsID] = struct{}{}
		}
	}
	return scores, recentPaidGoodsMap, nil
}

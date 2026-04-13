package biz

import (
	"context"
	"errors"
	"shop/api/gen/go/common"
	"time"

	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	recommendCandidate "shop/pkg/recommend/candidate"
	recommendEvent "shop/pkg/recommend/event"

	"github.com/liujitcn/gorm-kit/repo"
	"gorm.io/gorm"
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
func (c *RecommendUserGoodsPreferenceCase) loadUserGoodsSignals(ctx context.Context, userId int64, goodsIds []int64) (map[int64]float64, map[int64]struct{}, error) {
	// 用户 ID 或候选商品为空时，不需要继续查询偏好信号。
	if userId == 0 || len(goodsIds) == 0 {
		return map[int64]float64{}, map[int64]struct{}{}, nil
	}

	query := c.Query(ctx).RecommendUserGoodsPreference
	opts := make([]repo.QueryOption, 0, 3)
	opts = append(opts, repo.Where(query.UserID.Eq(userId)))
	opts = append(opts, repo.Where(query.GoodsID.In(goodsIds...)))
	opts = append(opts, repo.Where(query.WindowDays.Eq(recommendEvent.AggregateWindowDays)))

	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, nil, err
	}

	scores := make(map[int64]float64, len(list))
	recentPaidGoodsMap := make(map[int64]struct{})
	cutoff := time.Now().AddDate(0, 0, -recommendCandidate.RecentPayPenaltyDays)
	for _, item := range list {
		scores[item.GoodsID] = item.Score
		// 最近支付过的商品需要单独记录，用于推荐时做惩罚或过滤。
		if item.LastBehaviorType == common.RecommendGoodsActionType_ORDER_PAY.String() && item.LastBehaviorAt.After(cutoff) {
			recentPaidGoodsMap[item.GoodsID] = struct{}{}
		}
	}
	return scores, recentPaidGoodsMap, nil
}

// upsertUserGoodsPreference 累计用户对具体商品的偏好得分。
func (c *RecommendUserGoodsPreferenceCase) upsertUserGoodsPreference(ctx context.Context, userId, goodsId int64, eventType common.RecommendGoodsActionType, eventTime time.Time, goodsNum int64) error {
	query := c.Query(ctx).RecommendUserGoodsPreference
	opts := make([]repo.QueryOption, 0, 3)
	opts = append(opts, repo.Where(query.UserID.Eq(userId)))
	opts = append(opts, repo.Where(query.GoodsID.Eq(goodsId)))
	opts = append(opts, repo.Where(query.WindowDays.Eq(recommendEvent.AggregateWindowDays)))

	entity, err := c.Find(ctx, opts...)
	// 除记录不存在外的查询异常都应中断聚合，避免覆盖脏数据。
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	summaryJson := ""
	score := recommendEvent.EventWeight(eventType) * recommendEvent.NormalizeGoodsNum(goodsNum)
	// 已有聚合记录时，在原有分数和行为汇总上继续累加。
	if entity != nil {
		score += entity.Score
		summaryJson = entity.BehaviorSummary
	}
	summaryJson, err = recommendEvent.AddBehaviorSummaryCount(summaryJson, eventType, recommendEvent.NormalizeGoodsCount(goodsNum))
	if err != nil {
		return err
	}

	// 不存在历史记录时，创建新的商品偏好聚合数据。
	if entity == nil || entity.ID == 0 {
		return c.Create(ctx, &models.RecommendUserGoodsPreference{
			UserID:           userId,
			GoodsID:          goodsId,
			Score:            score,
			LastBehaviorType: eventType.String(),
			LastBehaviorAt:   eventTime,
			BehaviorSummary:  summaryJson,
			WindowDays:       recommendEvent.AggregateWindowDays,
			CreatedAt:        eventTime,
			UpdatedAt:        eventTime,
		})
	}

	// 命中历史记录时，更新累计分数和最近行为信息。
	entity.Score = score
	entity.LastBehaviorType = eventType.String()
	entity.LastBehaviorAt = eventTime
	entity.BehaviorSummary = summaryJson
	entity.UpdatedAt = eventTime
	return c.UpdateById(ctx, entity)
}

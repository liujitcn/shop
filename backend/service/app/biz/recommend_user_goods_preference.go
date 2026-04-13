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
	recommendGoodsActionRepo *data.RecommendGoodsActionRepo
}

// NewRecommendUserGoodsPreferenceCase 创建推荐用户商品偏好业务处理对象。
func NewRecommendUserGoodsPreferenceCase(
	baseCase *biz.BaseCase,
	recommendUserGoodsPreferenceRepo *data.RecommendUserGoodsPreferenceRepo,
	recommendGoodsActionRepo *data.RecommendGoodsActionRepo,
) *RecommendUserGoodsPreferenceCase {
	return &RecommendUserGoodsPreferenceCase{
		BaseCase:                         baseCase,
		RecommendUserGoodsPreferenceRepo: recommendUserGoodsPreferenceRepo,
		recommendGoodsActionRepo:         recommendGoodsActionRepo,
	}
}

type recommendUserGoodsPreferenceKey struct {
	userId  int64
	goodsId int64
}

// RebuildRecommendUserGoodsPreference 重建用户商品偏好聚合。
func (c *RecommendUserGoodsPreferenceCase) RebuildRecommendUserGoodsPreference(ctx context.Context, userIds []int64, windowDays int32) error {
	// 没有命中重建用户时，无需继续重建商品偏好。
	if len(userIds) == 0 {
		return nil
	}

	endAt := time.Now()
	startAt := endAt.AddDate(0, 0, -int(windowDays))

	preferenceQuery := c.Query(ctx).RecommendUserGoodsPreference
	preferenceOpts := make([]repo.QueryOption, 0, 2)
	preferenceOpts = append(preferenceOpts, repo.Where(preferenceQuery.UserID.In(userIds...)))
	preferenceOpts = append(preferenceOpts, repo.Where(preferenceQuery.WindowDays.Eq(windowDays)))
	err := c.Delete(ctx, preferenceOpts...)
	if err != nil {
		return err
	}

	actionQuery := c.recommendGoodsActionRepo.Query(ctx).RecommendGoodsAction
	actionOpts := make([]repo.QueryOption, 0, 5)
	actionOpts = append(actionOpts, repo.Where(actionQuery.ActorType.Eq(recommendEvent.ActorTypeUser)))
	actionOpts = append(actionOpts, repo.Where(actionQuery.ActorID.In(userIds...)))
	actionOpts = append(actionOpts, repo.Where(actionQuery.CreatedAt.Gte(startAt)))
	actionOpts = append(actionOpts, repo.Where(actionQuery.CreatedAt.Lte(endAt)))
	actionOpts = append(actionOpts, repo.Order(actionQuery.CreatedAt.Asc()))
	actionList, err := c.recommendGoodsActionRepo.List(ctx, actionOpts...)
	if err != nil {
		return err
	}
	// 当前窗口没有行为数据时，只保留清理动作。
	if len(actionList) == 0 {
		return nil
	}

	preferenceMap := make(map[recommendUserGoodsPreferenceKey]*models.RecommendUserGoodsPreference)
	for _, item := range actionList {
		// 非法行为明细不参与商品偏好重建。
		if item == nil || item.ActorID <= 0 || item.GoodsID <= 0 {
			continue
		}

		eventType := common.RecommendGoodsActionType(item.EventType)
		// 无法识别的行为类型不参与商品偏好重建。
		if eventType == common.RecommendGoodsActionType_UNKNOWN_RGAT {
			continue
		}

		key := recommendUserGoodsPreferenceKey{
			userId:  item.ActorID,
			goodsId: item.GoodsID,
		}
		entity, ok := preferenceMap[key]
		// 当前用户商品组合首次出现时，先初始化重建实体。
		if !ok {
			entity = &models.RecommendUserGoodsPreference{
				UserID:         item.ActorID,
				GoodsID:        item.GoodsID,
				LastBehaviorAt: item.CreatedAt,
				WindowDays:     windowDays,
				CreatedAt:      item.CreatedAt,
				UpdatedAt:      item.CreatedAt,
			}
			preferenceMap[key] = entity
		}

		entity.Score += recommendEvent.EventWeight(eventType) * recommendEvent.NormalizeGoodsNum(item.GoodsNum)
		entity.BehaviorSummary, err = recommendEvent.AddBehaviorSummaryCount(entity.BehaviorSummary, eventType, recommendEvent.NormalizeGoodsCount(item.GoodsNum))
		if err != nil {
			return err
		}
		// 当前行为时间更晚时，刷新最近行为信息。
		if !item.CreatedAt.Before(entity.LastBehaviorAt) {
			entity.LastBehaviorType = eventType.String()
			entity.LastBehaviorAt = item.CreatedAt
		}
		if item.CreatedAt.After(entity.UpdatedAt) {
			entity.UpdatedAt = item.CreatedAt
		}
	}

	list := make([]*models.RecommendUserGoodsPreference, 0, len(preferenceMap))
	for _, item := range preferenceMap {
		list = append(list, item)
	}
	if len(list) == 0 {
		return nil
	}
	return c.BatchCreate(ctx, list)
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

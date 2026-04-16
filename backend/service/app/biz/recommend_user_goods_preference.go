package biz

import (
	"context"
	"sort"
	"time"

	"shop/api/gen/go/common"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	recommendCandidate "shop/pkg/recommend/candidate"
	recommendEvent "shop/pkg/recommend/event"
	recommendAggregate "shop/pkg/recommend/offline/aggregate"

	"github.com/liujitcn/gorm-kit/repo"
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

// RebuildRecommendUserGoodsPreference 重建用户商品偏好聚合。
func (c *RecommendUserGoodsPreferenceCase) RebuildRecommendUserGoodsPreference(ctx context.Context, userIds []int64, windowDays int32) error {
	// 没有命中重建用户时，无需继续重建商品偏好。
	if len(userIds) == 0 {
		return nil
	}

	actionList, err := recommendAggregate.ListUserActionFacts(ctx, c.recommendGoodsActionRepo, userIds, windowDays)
	if err != nil {
		return err
	}

	preferenceQuery := c.Query(ctx).RecommendUserGoodsPreference
	preferenceOpts := make([]repo.QueryOption, 0, 2)
	preferenceOpts = append(preferenceOpts, repo.Where(preferenceQuery.UserID.In(userIds...)))
	preferenceOpts = append(preferenceOpts, repo.Where(preferenceQuery.WindowDays.Eq(windowDays)))
	err = c.Delete(ctx, preferenceOpts...)
	if err != nil {
		return err
	}

	list, err := recommendAggregate.RebuildUserGoodsPreferences(actionList, windowDays)
	if err != nil {
		return err
	}
	// 重建后没有沉淀出有效偏好数据时，直接结束。
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

// listObservedGoodsIdsByUserIds 查询一组用户偏好商品的观测结果。
func (c *RecommendUserGoodsPreferenceCase) listObservedGoodsIdsByUserIds(ctx context.Context, userIds []int64, limit int64, excludeGoodsIds []int64) ([]int64, error) {
	// 观测用户为空或限制数量非法时，不需要继续查询。
	if len(userIds) == 0 || limit <= 0 {
		return []int64{}, nil
	}

	query := c.Query(ctx).RecommendUserGoodsPreference
	opts := make([]repo.QueryOption, 0, 3)
	opts = append(opts, repo.Where(query.UserID.In(userIds...)))
	opts = append(opts, repo.Where(query.WindowDays.Eq(recommendEvent.AggregateWindowDays)))
	// 需要排除的商品不参与相似用户观测结果。
	if len(excludeGoodsIds) > 0 {
		opts = append(opts, repo.Where(query.GoodsID.NotIn(excludeGoodsIds...)))
	}

	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	// 没有命中任何偏好商品时，直接返回空结果。
	if len(list) == 0 {
		return []int64{}, nil
	}

	type observedGoodsScore struct {
		goodsId         int64
		totalScore      float64
		observedUserCnt int
		lastBehaviorAt  time.Time
	}

	scoreMap := make(map[int64]*observedGoodsScore)
	for _, item := range list {
		// 非法商品不参与相似用户观测统计。
		if item == nil || item.GoodsID <= 0 {
			continue
		}
		scoreItem, ok := scoreMap[item.GoodsID]
		if !ok {
			scoreItem = &observedGoodsScore{
				goodsId: item.GoodsID,
			}
			scoreMap[item.GoodsID] = scoreItem
		}
		scoreItem.totalScore += item.Score
		scoreItem.observedUserCnt++
		// 使用最近一次行为时间作为同分情况下的排序补充信号。
		if item.LastBehaviorAt.After(scoreItem.lastBehaviorAt) {
			scoreItem.lastBehaviorAt = item.LastBehaviorAt
		}
	}

	scoreList := make([]*observedGoodsScore, 0, len(scoreMap))
	for _, item := range scoreMap {
		scoreList = append(scoreList, item)
	}
	sort.Slice(scoreList, func(i, j int) bool {
		// 先按聚合偏好分倒序，保证更强偏好优先返回。
		if scoreList[i].totalScore != scoreList[j].totalScore {
			return scoreList[i].totalScore > scoreList[j].totalScore
		}
		// 聚合分相同的情况下，优先返回被更多相似用户共同偏好的商品。
		if scoreList[i].observedUserCnt != scoreList[j].observedUserCnt {
			return scoreList[i].observedUserCnt > scoreList[j].observedUserCnt
		}
		// 仍然相同时，优先返回最近有行为的商品。
		if !scoreList[i].lastBehaviorAt.Equal(scoreList[j].lastBehaviorAt) {
			return scoreList[i].lastBehaviorAt.After(scoreList[j].lastBehaviorAt)
		}
		return scoreList[i].goodsId < scoreList[j].goodsId
	})

	observedGoodsIds := make([]int64, 0, len(scoreList))
	for _, item := range scoreList {
		observedGoodsIds = append(observedGoodsIds, item.goodsId)
		// 达到观测数量上限后，直接结束。
		if int64(len(observedGoodsIds)) >= limit {
			break
		}
	}
	return observedGoodsIds, nil
}

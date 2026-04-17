package biz

import (
	"context"
	"errors"
	"sort"
	"time"

	"shop/api/gen/go/common"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	recommendCandidate "shop/pkg/recommend/candidate"
	recommendEvent "shop/pkg/recommend/event"
	recommendAggregate "shop/pkg/recommend/offline/aggregate"

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

// RebuildRecommendUserGoodsPreference 重建用户商品偏好聚合。
func (c *RecommendUserGoodsPreferenceCase) RebuildRecommendUserGoodsPreference(ctx context.Context, userIds []int64, windowDays int32) error {
	// 没有命中重建用户时，无需继续重建商品偏好。
	if len(userIds) == 0 {
		return nil
	}

	actionList, err := c.listUserActionFacts(ctx, userIds, windowDays)
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

// projectGoodsAction 将单条商品行为投影到用户商品偏好表。
func (c *RecommendUserGoodsPreferenceCase) projectGoodsAction(ctx context.Context, userId int64, eventType common.RecommendGoodsActionType, item *models.RecommendGoodsAction) error {
	// 空行为或非法商品编号不参与后续投影。
	if item == nil || item.GoodsID <= 0 {
		return nil
	}

	query := c.Query(ctx).RecommendUserGoodsPreference
	opts := make([]repo.QueryOption, 0, 3)
	opts = append(opts, repo.Where(query.UserID.Eq(userId)))
	opts = append(opts, repo.Where(query.GoodsID.Eq(item.GoodsID)))
	opts = append(opts, repo.Where(query.WindowDays.Eq(recommendEvent.AggregateWindowDays)))

	entity, err := c.Find(ctx, opts...)
	// 除记录不存在外的查询异常都应中断聚合，避免覆盖脏数据。
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	summaryJson := ""
	score := recommendEvent.EventWeight(eventType) * recommendEvent.NormalizeGoodsNum(item.GoodsNum)
	// 已有聚合记录时，在原有分数和行为汇总上继续累加。
	if entity != nil {
		score += entity.Score
		summaryJson = entity.BehaviorSummary
	}
	summaryJson, err = recommendEvent.AddBehaviorSummaryCount(summaryJson, eventType, recommendEvent.NormalizeGoodsCount(item.GoodsNum))
	if err != nil {
		return err
	}

	// 不存在历史记录时，创建新的商品偏好聚合数据。
	if entity == nil || entity.ID == 0 {
		return c.Create(ctx, &models.RecommendUserGoodsPreference{
			UserID:           userId,
			GoodsID:          item.GoodsID,
			Score:            score,
			LastBehaviorType: eventType.String(),
			LastBehaviorAt:   item.CreatedAt,
			BehaviorSummary:  summaryJson,
			WindowDays:       recommendEvent.AggregateWindowDays,
			CreatedAt:        item.CreatedAt,
			UpdatedAt:        item.CreatedAt,
		})
	}

	// 命中历史记录时，更新累计分数和最近行为信息。
	entity.Score = score
	entity.LastBehaviorType = eventType.String()
	entity.LastBehaviorAt = item.CreatedAt
	entity.BehaviorSummary = summaryJson
	entity.UpdatedAt = item.CreatedAt
	return c.UpdateById(ctx, entity)
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

// listObservedGoodsIdsByUserIds 查询一组用户偏好商品的观测结果及其归一化分数。
func (c *RecommendUserGoodsPreferenceCase) listObservedGoodsIdsByUserIds(ctx context.Context, userIds []int64, limit int64, excludeGoodsIds []int64) ([]int64, map[int64]float64, error) {
	// 观测用户为空或限制数量非法时，不需要继续查询。
	if len(userIds) == 0 || limit <= 0 {
		return []int64{}, map[int64]float64{}, nil
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
		return nil, nil, err
	}
	// 没有命中任何偏好商品时，直接返回空结果。
	if len(list) == 0 {
		return []int64{}, map[int64]float64{}, nil
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
	observedGoodsScoreMap := make(map[int64]float64, len(scoreList))
	maxObservedScore := float64(0)
	for _, item := range scoreList {
		// 最大观测分仅用于后续归一化，不影响当前 TopN 的排序顺序。
		if item.totalScore > maxObservedScore {
			maxObservedScore = item.totalScore
		}
	}
	for _, item := range scoreList {
		observedGoodsIds = append(observedGoodsIds, item.goodsId)
		observedGoodsScoreMap[item.goodsId] = item.totalScore
		// 达到观测数量上限后，直接结束。
		if int64(len(observedGoodsIds)) >= limit {
			break
		}
	}
	// 为了让相似用户观测分能和在线其它规则分共同参与排序，这里统一压缩到 0 到 1 范围。
	if maxObservedScore > 0 {
		for goodsId, score := range observedGoodsScoreMap {
			observedGoodsScoreMap[goodsId] = score / maxObservedScore
		}
	}
	return observedGoodsIds, observedGoodsScoreMap, nil
}

// listUserActionFacts 读取用户偏好重建所需的行为事实。
func (c *RecommendUserGoodsPreferenceCase) listUserActionFacts(ctx context.Context, userIds []int64, windowDays int32) ([]*models.RecommendGoodsAction, error) {
	// 没有命中用户集合时，不需要继续查询行为事实。
	if len(userIds) == 0 {
		return []*models.RecommendGoodsAction{}, nil
	}

	endAt := time.Now()
	startAt := endAt.AddDate(0, 0, -int(windowDays))
	query := c.recommendGoodsActionRepo.Query(ctx).RecommendGoodsAction
	opts := make([]repo.QueryOption, 0, 5)
	opts = append(opts, repo.Where(query.ActorType.Eq(recommendEvent.ActorTypeUser)))
	opts = append(opts, repo.Where(query.ActorID.In(userIds...)))
	opts = append(opts, repo.Where(query.CreatedAt.Gte(startAt)))
	opts = append(opts, repo.Where(query.CreatedAt.Lte(endAt)))
	opts = append(opts, repo.Order(query.CreatedAt.Asc()))
	return c.recommendGoodsActionRepo.List(ctx, opts...)
}

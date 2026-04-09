package biz

import (
	"context"
	"encoding/json"
	"time"

	recommendrank "shop/pkg/recommend/rank"

	"github.com/liujitcn/gorm-kit/repo"
)

const (
	recommendStatLookbackDays          = 30
	recommendRecentPayPenaltyDays      = 15
	recommendActorExposureLookbackDays = 7
)

// loadRelationScores 加载候选商品的关联商品分数。
func (c *RecommendCase) loadRelationScores(ctx context.Context, sourceGoodsIds []int64) (map[int64]float64, error) {
	// 没有源商品时无法计算关联分，直接返回空分数字典。
	if len(sourceGoodsIds) == 0 {
		return map[int64]float64{}, nil
	}
	relationQuery := c.recommendRelation.Query(ctx).RecommendGoodsRelation
	list, err := c.recommendRelation.RecommendGoodsRelationRepo.List(ctx,
		repo.Where(relationQuery.GoodsID.In(sourceGoodsIds...)),
		repo.Where(relationQuery.WindowDays.Eq(recommendAggregateWindowDays)),
	)
	if err != nil {
		return nil, err
	}
	scores := make(map[int64]float64, len(list))
	for _, item := range list {
		scores[item.RelatedGoodsID] += item.Score
	}
	return scores, nil
}

// loadUserGoodsSignals 加载用户对候选商品的偏好分和近期支付集合。
func (c *RecommendCase) loadUserGoodsSignals(ctx context.Context, userID int64, goodsIds []int64) (map[int64]float64, map[int64]struct{}, error) {
	// 未登录用户或空候选集都不需要加载用户商品偏好。
	if userID == 0 || len(goodsIds) == 0 {
		return map[int64]float64{}, map[int64]struct{}{}, nil
	}
	preferenceQuery := c.RecommendUserGoodsPreferenceRepo.Query(ctx).RecommendUserGoodsPreference
	list, err := c.RecommendUserGoodsPreferenceRepo.List(ctx,
		repo.Where(preferenceQuery.UserID.Eq(userID)),
		repo.Where(preferenceQuery.GoodsID.In(goodsIds...)),
		repo.Where(preferenceQuery.WindowDays.Eq(recommendAggregateWindowDays)),
	)
	if err != nil {
		return nil, nil, err
	}
	scores := make(map[int64]float64, len(list))
	recentPaidGoodsMap := make(map[int64]struct{})
	cutoff := time.Now().AddDate(0, 0, -recommendRecentPayPenaltyDays)
	for _, item := range list {
		scores[item.GoodsID] = item.Score
		// 近期支付过的商品需要进入重复推荐惩罚集合。
		if item.LastBehaviorType == recommendEventTypePay && item.LastBehaviorAt.After(cutoff) {
			recentPaidGoodsMap[item.GoodsID] = struct{}{}
		}
	}
	return scores, recentPaidGoodsMap, nil
}

// loadProfileScores 加载用户类目画像分数。
func (c *RecommendCase) loadProfileScores(ctx context.Context, userID int64, categoryIds []int64) (map[int64]float64, error) {
	// 未登录用户或没有类目上下文时，不查询画像分。
	if userID == 0 || len(categoryIds) == 0 {
		return map[int64]float64{}, nil
	}
	preferenceQuery := c.recommendProfile.Query(ctx).RecommendUserPreference
	list, err := c.recommendProfile.RecommendUserPreferenceRepo.List(ctx,
		repo.Where(preferenceQuery.UserID.Eq(userID)),
		repo.Where(preferenceQuery.PreferenceType.Eq(recommendPreferenceTypeCategory)),
		repo.Where(preferenceQuery.TargetID.In(dedupeInt64s(categoryIds)...)),
		repo.Where(preferenceQuery.WindowDays.Eq(recommendAggregateWindowDays)),
	)
	if err != nil {
		return nil, err
	}
	scores := make(map[int64]float64, len(list))
	for _, item := range list {
		scores[item.TargetID] = item.Score
	}
	return scores, nil
}

// loadScenePopularitySignals 加载场景热度和曝光惩罚信号。
func (c *RecommendCase) loadScenePopularitySignals(ctx context.Context, scene int32, goodsIds []int64) (map[int64]float64, map[int64]float64, error) {
	// 场景未知或候选集为空时，不查询场景热度。
	if scene == 0 || len(goodsIds) == 0 {
		return map[int64]float64{}, map[int64]float64{}, nil
	}
	statQuery := c.recommendGoodsStatDayRepo.Query(ctx).RecommendGoodsStatDay
	startDate := time.Now().AddDate(0, 0, -recommendStatLookbackDays)
	list, err := c.recommendGoodsStatDayRepo.List(ctx,
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

// loadGlobalPopularityScores 加载全站热度分数。
func (c *RecommendCase) loadGlobalPopularityScores(ctx context.Context, goodsIds []int64) (map[int64]float64, error) {
	// 空候选集不需要查询全站热度。
	if len(goodsIds) == 0 {
		return map[int64]float64{}, nil
	}
	statQuery := c.goodsStatDayRepo.Query(ctx).GoodsStatDay
	startDate := time.Now().AddDate(0, 0, -recommendStatLookbackDays)
	list, err := c.goodsStatDayRepo.List(ctx,
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

// loadActorExposurePenalties 加载当前主体的曝光惩罚分。
func (c *RecommendCase) loadActorExposurePenalties(ctx context.Context, actor *RecommendActor, scene int32, goodsIds []int64) (map[int64]float64, error) {
	// 匿名主体缺少稳定 ID、场景未知或候选集为空时，不做个体惩罚。
	if actor == nil || actor.ActorId <= 0 || scene == 0 || len(goodsIds) == 0 {
		return map[int64]float64{}, nil
	}
	exposureQuery := c.RecommendExposureRepo.Query(ctx).RecommendExposure
	actionQuery := c.RecommendGoodsActionRepo.Query(ctx).RecommendGoodsAction
	cutoff := time.Now().AddDate(0, 0, -recommendActorExposureLookbackDays)
	exposureList, err := c.RecommendExposureRepo.List(ctx,
		repo.Where(exposureQuery.ActorType.Eq(actor.ActorType)),
		repo.Where(exposureQuery.ActorID.Eq(actor.ActorId)),
		repo.Where(exposureQuery.Scene.Eq(scene)),
		repo.Where(exposureQuery.CreatedAt.Gte(cutoff)),
	)
	if err != nil {
		return nil, err
	}
	clickList, err := c.RecommendGoodsActionRepo.List(ctx,
		repo.Where(actionQuery.ActorType.Eq(actor.ActorType)),
		repo.Where(actionQuery.ActorID.Eq(actor.ActorId)),
		repo.Where(actionQuery.Scene.Eq(scene)),
		repo.Where(actionQuery.EventType.Eq(int32(recommendGoodsActionTypeClick))),
		repo.Where(actionQuery.CreatedAt.Gte(cutoff)),
		repo.Where(actionQuery.GoodsID.In(goodsIds...)),
	)
	if err != nil {
		return nil, err
	}
	exposureCountMap := make(map[int64]int64, len(goodsIds))
	for _, item := range exposureList {
		ids := make([]int64, 0)
		err = json.Unmarshal([]byte(item.GoodsIds), &ids)
		// 曝光列表解析失败时跳过当前记录，避免脏数据阻塞推荐。
		if err != nil {
			continue
		}
		for _, goodsId := range ids {
			exposureCountMap[goodsId]++
		}
	}
	clickCountMap := make(map[int64]int64, len(clickList))
	for _, item := range clickList {
		clickCountMap[item.GoodsID]++
	}
	penalties := make(map[int64]float64, len(goodsIds))
	for _, goodsId := range goodsIds {
		exposureCount := exposureCountMap[goodsId]
		clickCount := clickCountMap[goodsId]
		// 连续多次曝光但零点击时，直接打最高个体惩罚。
		if exposureCount >= 3 && clickCount == 0 {
			penalties[goodsId] = 0.6
			continue
		}
		// 曝光过高且点击率极低时，施加中等个体惩罚。
		if exposureCount >= 5 && clickCount*20 < exposureCount {
			penalties[goodsId] = 0.3
		}
	}
	return penalties, nil
}

// calculateRepeatPenalty 计算近期已购商品的重复曝光惩罚。
func (c *RecommendCase) calculateRepeatPenalty(goodsID int64, recentPaidGoodsMap map[int64]struct{}) float64 {
	// 近期已支付的商品需要显式降权，避免短期重复推荐。
	if _, ok := recentPaidGoodsMap[goodsID]; ok {
		return 1.5
	}
	return 0
}

package biz

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"shop/api/gen/go/app"
	"shop/pkg/gen/models"
	"shop/service/app/util"

	"github.com/liujitcn/gorm-kit/repo"
)

const (
	recommendCandidatePoolMultiplier   = 8
	recommendCandidatePoolMin          = 80
	recommendCandidatePoolMax          = 240
	recommendMaxPerCategory            = 2
	recommendFreshnessWindowDays       = 30.0
	recommendStatLookbackDays          = 30
	recommendRecentPayPenaltyDays      = 15
	recommendHighExposureThreshold     = 20
	recommendActorExposureLookbackDays = 7
)

// RecommendCandidate 推荐候选商品。
type RecommendCandidate struct {
	Goods                 *models.GoodsInfo
	RelationScore         float64
	UserGoodsScore        float64
	ProfileScore          float64
	ScenePopularityScore  float64
	GlobalPopularityScore float64
	FreshnessScore        float64
	ExposurePenalty       float64
	ActorExposurePenalty  float64
	RepeatPenalty         float64
	FinalScore            float64
	RecallSources         map[string]struct{}
}

// RecommendScoreDetail 推荐评分明细。
type RecommendScoreDetail struct {
	GoodsID               int64    `json:"goodsId"`
	FinalScore            float64  `json:"finalScore"`
	RelationScore         float64  `json:"relationScore,omitempty"`
	UserGoodsScore        float64  `json:"userGoodsScore,omitempty"`
	ProfileScore          float64  `json:"profileScore,omitempty"`
	ScenePopularityScore  float64  `json:"scenePopularityScore,omitempty"`
	GlobalPopularityScore float64  `json:"globalPopularityScore,omitempty"`
	FreshnessScore        float64  `json:"freshnessScore,omitempty"`
	ExposurePenalty       float64  `json:"exposurePenalty,omitempty"`
	ActorExposurePenalty  float64  `json:"actorExposurePenalty,omitempty"`
	RepeatPenalty         float64  `json:"repeatPenalty,omitempty"`
	RecallSources         []string `json:"recallSources,omitempty"`
}

// listRecommendGoods 查询推荐商品列表并执行统一排序。
func (c *RecommendCase) listRecommendGoods(
	ctx context.Context,
	actor *RecommendActor,
	req *app.RecommendGoodsRequest,
	userID int64,
	priorityGoodsIds []int64,
	categoryIds []int64,
	pageNum, pageSize int64,
) ([]*app.GoodsInfo, int64, []string, map[string]any, error) {
	if pageSize <= 0 {
		return []*app.GoodsInfo{}, 0, []string{}, map[string]any{}, nil
	}

	candidateLimit := c.resolveCandidateLimit(pageNum, pageSize)
	excludeGoodsIds := dedupeInt64s(priorityGoodsIds)
	categoryCandidateIds, err := c.listCategoryCandidateGoodsIds(ctx, categoryIds, excludeGoodsIds, candidateLimit)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	excludeGoodsIds = dedupeInt64s(append(excludeGoodsIds, categoryCandidateIds...))
	latestCandidateIds, err := c.listLatestCandidateGoodsIds(ctx, excludeGoodsIds, candidateLimit)
	if err != nil {
		return nil, 0, nil, nil, err
	}

	allCandidateIds := dedupeInt64s(append(append(priorityGoodsIds, categoryCandidateIds...), latestCandidateIds...))
	if len(allCandidateIds) == 0 {
		return []*app.GoodsInfo{}, 0, []string{}, map[string]any{}, nil
	}

	goodsList, err := c.listGoodsByIds(ctx, allCandidateIds)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	candidates, err := c.buildRecommendCandidates(ctx, actor, req, userID, priorityGoodsIds, goodsList)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	rankedGoods := c.rankRecommendCandidates(candidates)
	total := int64(len(rankedGoods))

	offset := int((pageNum - 1) * pageSize)
	if offset >= len(rankedGoods) {
		return []*app.GoodsInfo{}, total, []string{}, map[string]any{}, nil
	}
	end := offset + int(pageSize)
	if end > len(rankedGoods) {
		end = len(rankedGoods)
	}

	member := util.IsMember(ctx)
	list := make([]*app.GoodsInfo, 0, end-offset)
	pageRecallSources := make(map[string]struct{}, 8)
	scoreDetails := make([]RecommendScoreDetail, 0, end-offset)
	for _, item := range rankedGoods[offset:end] {
		list = append(list, c.convertGoodsToProto(item, member))
		if candidate, ok := candidates[item.ID]; ok {
			for source := range candidate.RecallSources {
				pageRecallSources[source] = struct{}{}
			}
			scoreDetails = append(scoreDetails, buildRecommendScoreDetail(candidate))
		}
	}
	return list, total, mapKeys(pageRecallSources), map[string]any{
		"candidateLimit":       candidateLimit,
		"priorityGoodsIds":     priorityGoodsIds,
		"categoryIds":          categoryIds,
		"returnedScoreDetails": scoreDetails,
	}, nil
}

func (c *RecommendCase) resolveCandidateLimit(pageNum, pageSize int64) int {
	limit := int(pageNum * pageSize * recommendCandidatePoolMultiplier)
	if limit < recommendCandidatePoolMin {
		limit = recommendCandidatePoolMin
	}
	if limit > recommendCandidatePoolMax {
		limit = recommendCandidatePoolMax
	}
	return limit
}

func (c *RecommendCase) listCategoryCandidateGoodsIds(ctx context.Context, categoryIds []int64, excludeGoodsIds []int64, limit int) ([]int64, error) {
	if len(categoryIds) == 0 || limit <= 0 {
		return []int64{}, nil
	}
	list, _, err := c.pageGoods(ctx, categoryIds, excludeGoodsIds, 1, int64(limit))
	if err != nil {
		return nil, err
	}
	goodsIds := make([]int64, 0, len(list))
	for _, item := range list {
		goodsIds = append(goodsIds, item.ID)
	}
	return dedupeInt64s(goodsIds), nil
}

func (c *RecommendCase) listLatestCandidateGoodsIds(ctx context.Context, excludeGoodsIds []int64, limit int) ([]int64, error) {
	if limit <= 0 {
		return []int64{}, nil
	}
	list, _, err := c.pageGoods(ctx, nil, excludeGoodsIds, 1, int64(limit))
	if err != nil {
		return nil, err
	}
	goodsIds := make([]int64, 0, len(list))
	for _, item := range list {
		goodsIds = append(goodsIds, item.ID)
	}
	return dedupeInt64s(goodsIds), nil
}

func (c *RecommendCase) buildRecommendCandidates(
	ctx context.Context,
	actor *RecommendActor,
	req *app.RecommendGoodsRequest,
	userID int64,
	priorityGoodsIds []int64,
	goodsList []*models.GoodsInfo,
) (map[int64]*RecommendCandidate, error) {
	candidates := make(map[int64]*RecommendCandidate, len(goodsList))
	candidateGoodsIds := make([]int64, 0, len(goodsList))
	categoryIds := make([]int64, 0, len(goodsList))
	for _, item := range goodsList {
		if item == nil || item.ID <= 0 {
			continue
		}
		candidateGoodsIds = append(candidateGoodsIds, item.ID)
		categoryIds = append(categoryIds, item.CategoryID)
		candidates[item.ID] = &RecommendCandidate{
			Goods:         item,
			RecallSources: make(map[string]struct{}, 4),
		}
	}

	relationScores, err := c.loadRelationScores(ctx, priorityGoodsIds)
	if err != nil {
		return nil, err
	}
	userGoodsScores, recentPaidGoodsMap, err := c.loadUserGoodsSignals(ctx, userID, candidateGoodsIds)
	if err != nil {
		return nil, err
	}
	actorExposurePenalties, err := c.loadActorExposurePenalties(ctx, actor, int32(req.GetScene()), candidateGoodsIds)
	if err != nil {
		return nil, err
	}
	profileScores, err := c.loadProfileScores(ctx, userID, categoryIds)
	if err != nil {
		return nil, err
	}
	scenePopularityScores, sceneExposurePenalties, err := c.loadScenePopularitySignals(ctx, int32(req.GetScene()), candidateGoodsIds)
	if err != nil {
		return nil, err
	}
	globalPopularityScores, err := c.loadGlobalPopularityScores(ctx, candidateGoodsIds)
	if err != nil {
		return nil, err
	}

	for goodsID, candidate := range candidates {
		candidate.RelationScore = relationScores[goodsID]
		candidate.UserGoodsScore = userGoodsScores[goodsID]
		candidate.ProfileScore = profileScores[candidate.Goods.CategoryID]
		candidate.ScenePopularityScore = scenePopularityScores[goodsID]
		candidate.GlobalPopularityScore = globalPopularityScores[goodsID]
		candidate.FreshnessScore = c.calculateFreshnessScore(candidate.Goods.CreatedAt)
		candidate.ExposurePenalty = sceneExposurePenalties[goodsID]
		candidate.ActorExposurePenalty = actorExposurePenalties[goodsID]
		candidate.RepeatPenalty = c.calculateRepeatPenalty(goodsID, recentPaidGoodsMap)
		candidate.FinalScore = c.calculateRecommendFinalScore(candidate)
		if candidate.RelationScore > 0 {
			candidate.RecallSources["relation"] = struct{}{}
		}
		if candidate.UserGoodsScore > 0 {
			candidate.RecallSources["user_goods"] = struct{}{}
		}
		if candidate.ProfileScore > 0 {
			candidate.RecallSources["profile"] = struct{}{}
		}
		if candidate.ScenePopularityScore > 0 {
			candidate.RecallSources["scene_hot"] = struct{}{}
		}
		if candidate.GlobalPopularityScore > 0 {
			candidate.RecallSources["global_hot"] = struct{}{}
		}
		if len(candidate.RecallSources) == 0 {
			candidate.RecallSources["latest"] = struct{}{}
		}
		if candidate.ActorExposurePenalty > 0 {
			candidate.RecallSources["actor_penalty"] = struct{}{}
		}
	}
	return candidates, nil
}

// listAnonymousRecommendGoods 查询匿名推荐商品列表并执行统一排序。
func (c *RecommendCase) listAnonymousRecommendGoods(ctx context.Context, actor *RecommendActor, req *app.RecommendGoodsRequest, pageNum, pageSize int64) ([]*app.GoodsInfo, int64, []string, map[string]any, error) {
	if pageSize <= 0 {
		return []*app.GoodsInfo{}, 0, []string{"anonymous_hot"}, map[string]any{}, nil
	}
	candidateLimit := c.resolveCandidateLimit(pageNum, pageSize)
	sceneGoodsIds, err := c.listSceneHotGoodsIds(ctx, req.GetScene(), time.Now().AddDate(0, 0, -recommendAnonymousRecallDays), candidateLimit)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	candidateGoodsIds, err := c.mergeAnonymousGoodsIds(ctx, sceneGoodsIds, time.Now().AddDate(0, 0, -recommendAnonymousRecallDays), candidateLimit)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	if len(candidateGoodsIds) == 0 {
		fallbackList, fallbackTotal, fallbackErr := c.pageGoods(ctx, nil, nil, pageNum, pageSize)
		if fallbackErr != nil {
			return nil, 0, nil, nil, fallbackErr
		}
		return c.convertGoodsListToProto(ctx, fallbackList), fallbackTotal, []string{"latest"}, map[string]any{
			"candidateLimit":   candidateLimit,
			"sceneHotGoodsIds": sceneGoodsIds,
		}, nil
	}
	goodsList, err := c.listGoodsByIds(ctx, candidateGoodsIds)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	candidates, err := c.buildAnonymousRecommendCandidates(ctx, actor, req, goodsList)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	rankedGoods := c.rankRecommendCandidates(candidates)
	total := int64(len(rankedGoods))
	offset := int((pageNum - 1) * pageSize)
	if offset >= len(rankedGoods) {
		return []*app.GoodsInfo{}, total, []string{"anonymous_hot"}, map[string]any{}, nil
	}
	end := offset + int(pageSize)
	if end > len(rankedGoods) {
		end = len(rankedGoods)
	}
	pageRecallSources := make(map[string]struct{}, 6)
	scoreDetails := make([]RecommendScoreDetail, 0, end-offset)
	for _, item := range rankedGoods[offset:end] {
		if candidate, ok := candidates[item.ID]; ok {
			for source := range candidate.RecallSources {
				pageRecallSources[source] = struct{}{}
			}
			scoreDetails = append(scoreDetails, buildRecommendScoreDetail(candidate))
		}
	}
	return c.convertGoodsListToProto(ctx, rankedGoods[offset:end]), total, mapKeys(pageRecallSources), map[string]any{
		"candidateLimit":             candidateLimit,
		"sceneHotGoodsIds":           sceneGoodsIds,
		"anonymousCandidateGoodsIds": candidateGoodsIds,
		"returnedScoreDetails":       scoreDetails,
	}, nil
}

func (c *RecommendCase) buildAnonymousRecommendCandidates(
	ctx context.Context,
	actor *RecommendActor,
	req *app.RecommendGoodsRequest,
	goodsList []*models.GoodsInfo,
) (map[int64]*RecommendCandidate, error) {
	candidates := make(map[int64]*RecommendCandidate, len(goodsList))
	candidateGoodsIds := make([]int64, 0, len(goodsList))
	for _, item := range goodsList {
		if item == nil || item.ID <= 0 {
			continue
		}
		candidateGoodsIds = append(candidateGoodsIds, item.ID)
		candidates[item.ID] = &RecommendCandidate{
			Goods:         item,
			RecallSources: make(map[string]struct{}, 3),
		}
	}
	scenePopularityScores, sceneExposurePenalties, err := c.loadScenePopularitySignals(ctx, int32(req.GetScene()), candidateGoodsIds)
	if err != nil {
		return nil, err
	}
	actorExposurePenalties, err := c.loadActorExposurePenalties(ctx, actor, int32(req.GetScene()), candidateGoodsIds)
	if err != nil {
		return nil, err
	}
	globalPopularityScores, err := c.loadGlobalPopularityScores(ctx, candidateGoodsIds)
	if err != nil {
		return nil, err
	}
	for goodsID, candidate := range candidates {
		candidate.ScenePopularityScore = scenePopularityScores[goodsID]
		candidate.GlobalPopularityScore = globalPopularityScores[goodsID]
		candidate.FreshnessScore = c.calculateFreshnessScore(candidate.Goods.CreatedAt)
		candidate.ExposurePenalty = sceneExposurePenalties[goodsID]
		candidate.ActorExposurePenalty = actorExposurePenalties[goodsID]
		candidate.FinalScore = candidate.ScenePopularityScore*0.55 +
			candidate.GlobalPopularityScore*0.30 +
			candidate.FreshnessScore*0.15 -
			candidate.ExposurePenalty -
			candidate.ActorExposurePenalty
		if candidate.ScenePopularityScore > 0 {
			candidate.RecallSources["scene_hot"] = struct{}{}
		}
		if candidate.GlobalPopularityScore > 0 {
			candidate.RecallSources["global_hot"] = struct{}{}
		}
		if len(candidate.RecallSources) == 0 {
			candidate.RecallSources["latest"] = struct{}{}
		}
		if candidate.ActorExposurePenalty > 0 {
			candidate.RecallSources["actor_penalty"] = struct{}{}
		}
	}
	return candidates, nil
}

func (c *RecommendCase) loadRelationScores(ctx context.Context, sourceGoodsIds []int64) (map[int64]float64, error) {
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

func (c *RecommendCase) loadUserGoodsSignals(ctx context.Context, userID int64, goodsIds []int64) (map[int64]float64, map[int64]struct{}, error) {
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
		if item.LastBehaviorType == recommendEventTypePay && item.LastBehaviorAt.After(cutoff) {
			recentPaidGoodsMap[item.GoodsID] = struct{}{}
		}
	}
	return scores, recentPaidGoodsMap, nil
}

func (c *RecommendCase) loadProfileScores(ctx context.Context, userID int64, categoryIds []int64) (map[int64]float64, error) {
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

func (c *RecommendCase) loadScenePopularitySignals(ctx context.Context, scene int32, goodsIds []int64) (map[int64]float64, map[int64]float64, error) {
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
		scores[item.GoodsID] += item.Score * c.calculateDayDecay(item.StatDate)
		penalties[item.GoodsID] += c.calculateExposurePenalty(item.ExposureCount, item.ClickCount) * c.calculateDayDecay(item.StatDate)
	}
	return scores, penalties, nil
}

func (c *RecommendCase) loadGlobalPopularityScores(ctx context.Context, goodsIds []int64) (map[int64]float64, error) {
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
		scores[item.GoodsID] += item.Score * c.calculateDayDecay(item.StatDate)
	}
	return scores, nil
}

func (c *RecommendCase) loadActorExposurePenalties(ctx context.Context, actor *RecommendActor, scene int32, goodsIds []int64) (map[int64]float64, error) {
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
		repo.Where(actionQuery.EventType.Eq(recommendEventTypeClick)),
		repo.Where(actionQuery.CreatedAt.Gte(cutoff)),
		repo.Where(actionQuery.GoodsID.In(goodsIds...)),
	)
	if err != nil {
		return nil, err
	}
	exposureCountMap := make(map[int64]int64, len(goodsIds))
	for _, item := range exposureList {
		ids := make([]int64, 0)
		if err = json.Unmarshal([]byte(item.GoodsIds), &ids); err != nil {
			continue
		}
		for _, goodsID := range ids {
			exposureCountMap[goodsID]++
		}
	}
	clickCountMap := make(map[int64]int64, len(clickList))
	for _, item := range clickList {
		clickCountMap[item.GoodsID]++
	}
	penalties := make(map[int64]float64, len(goodsIds))
	for _, goodsID := range goodsIds {
		exposureCount := exposureCountMap[goodsID]
		clickCount := clickCountMap[goodsID]
		if exposureCount >= 3 && clickCount == 0 {
			penalties[goodsID] = 0.6
			continue
		}
		if exposureCount >= 5 && clickCount*20 < exposureCount {
			penalties[goodsID] = 0.3
		}
	}
	return penalties, nil
}

func (c *RecommendCase) calculateDayDecay(statDate time.Time) float64 {
	daysAgo := time.Since(statDate).Hours() / 24
	if daysAgo <= 0 {
		return 1
	}
	return 1 / (1 + daysAgo*0.08)
}

func (c *RecommendCase) calculateExposurePenalty(exposureCount, clickCount int64) float64 {
	if exposureCount < recommendHighExposureThreshold {
		return 0
	}
	if clickCount <= 0 {
		return 1.2
	}
	ctr := float64(clickCount) / float64(exposureCount)
	switch {
	case ctr < 0.005:
		return 0.8
	case ctr < 0.01:
		return 0.4
	default:
		return 0
	}
}

func (c *RecommendCase) calculateFreshnessScore(createdAt time.Time) float64 {
	if createdAt.IsZero() {
		return 0
	}
	daysAgo := time.Since(createdAt).Hours() / 24
	if daysAgo <= 0 {
		return 1
	}
	score := 1 - (daysAgo / recommendFreshnessWindowDays)
	if score < 0 {
		return 0
	}
	return score
}

func (c *RecommendCase) calculateRecommendFinalScore(candidate *RecommendCandidate) float64 {
	if candidate == nil {
		return 0
	}
	return candidate.RelationScore*0.30 +
		candidate.UserGoodsScore*0.25 +
		candidate.ProfileScore*0.15 +
		candidate.ScenePopularityScore*0.20 +
		candidate.GlobalPopularityScore*0.10 +
		candidate.FreshnessScore*0.10 -
		candidate.ExposurePenalty -
		candidate.ActorExposurePenalty -
		candidate.RepeatPenalty
}

func (c *RecommendCase) calculateRepeatPenalty(goodsID int64, recentPaidGoodsMap map[int64]struct{}) float64 {
	if _, ok := recentPaidGoodsMap[goodsID]; ok {
		return 1.5
	}
	return 0
}

func (c *RecommendCase) rankRecommendCandidates(candidates map[int64]*RecommendCandidate) []*models.GoodsInfo {
	if len(candidates) == 0 {
		return []*models.GoodsInfo{}
	}
	list := make([]*RecommendCandidate, 0, len(candidates))
	for _, item := range candidates {
		if item == nil || item.Goods == nil {
			continue
		}
		list = append(list, item)
	}
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].FinalScore == list[j].FinalScore {
			if list[i].ScenePopularityScore == list[j].ScenePopularityScore {
				return list[i].Goods.CreatedAt.After(list[j].Goods.CreatedAt)
			}
			return list[i].ScenePopularityScore > list[j].ScenePopularityScore
		}
		return list[i].FinalScore > list[j].FinalScore
	})
	return c.diversifyRecommendCandidates(list)
}

func (c *RecommendCase) diversifyRecommendCandidates(candidates []*RecommendCandidate) []*models.GoodsInfo {
	result := make([]*models.GoodsInfo, 0, len(candidates))
	categoryCount := make(map[int64]int, len(candidates))
	overflow := make([]*models.GoodsInfo, 0)
	for _, item := range candidates {
		if item == nil || item.Goods == nil {
			continue
		}
		categoryID := item.Goods.CategoryID
		if categoryID > 0 && categoryCount[categoryID] >= recommendMaxPerCategory {
			overflow = append(overflow, item.Goods)
			continue
		}
		categoryCount[categoryID]++
		result = append(result, item.Goods)
	}
	return append(result, overflow...)
}

// convertGoodsListToProto 批量转换商品列表。
func (c *RecommendCase) convertGoodsListToProto(ctx context.Context, list []*models.GoodsInfo) []*app.GoodsInfo {
	member := util.IsMember(ctx)
	result := make([]*app.GoodsInfo, 0, len(list))
	for _, item := range list {
		result = append(result, c.convertGoodsToProto(item, member))
	}
	return result
}

func mapKeys(input map[string]struct{}) []string {
	if len(input) == 0 {
		return []string{}
	}
	result := make([]string, 0, len(input))
	for key := range input {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}

func buildRecommendScoreDetail(candidate *RecommendCandidate) RecommendScoreDetail {
	if candidate == nil || candidate.Goods == nil {
		return RecommendScoreDetail{}
	}
	return RecommendScoreDetail{
		GoodsID:               candidate.Goods.ID,
		FinalScore:            candidate.FinalScore,
		RelationScore:         candidate.RelationScore,
		UserGoodsScore:        candidate.UserGoodsScore,
		ProfileScore:          candidate.ProfileScore,
		ScenePopularityScore:  candidate.ScenePopularityScore,
		GlobalPopularityScore: candidate.GlobalPopularityScore,
		FreshnessScore:        candidate.FreshnessScore,
		ExposurePenalty:       candidate.ExposurePenalty,
		ActorExposurePenalty:  candidate.ActorExposurePenalty,
		RepeatPenalty:         candidate.RepeatPenalty,
		RecallSources:         mapKeys(candidate.RecallSources),
	}
}

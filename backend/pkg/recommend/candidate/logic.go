package candidate

import (
	"shop/api/gen/go/app"
	recommendCore "shop/pkg/recommend/core"
	recommendFilter "shop/pkg/recommend/filter"
	recommendRank "shop/pkg/recommend/rank"
)

const (
	PoolMultiplier = 8
	PoolMin        = 80
	PoolMax        = 240

	AnonymousRecallDays       = 30
	StatLookbackDays          = 30
	RecentPayPenaltyDays      = 15
	ActorExposureLookbackDays = 7

	RecallSourceRelation     = "relation"
	RecallSourceUserGoods    = "user_goods"
	RecallSourceProfile      = "profile"
	RecallSourceSceneHot     = "scene_hot"
	RecallSourceGlobalHot    = "global_hot"
	RecallSourceLatest       = "latest"
	RecallSourceActorPenalty = "actor_penalty"
)

// PersonalizedSignals 表示登录态候选所需的评分信号。
type PersonalizedSignals struct {
	RelationScores         map[int64]float64
	UserGoodsScores        map[int64]float64
	ProfileScores          map[int64]float64
	ScenePopularityScores  map[int64]float64
	GlobalPopularityScores map[int64]float64
	SceneExposurePenalties map[int64]float64
	ActorExposurePenalties map[int64]float64
	RecentPaidGoods        map[int64]struct{}
}

// AnonymousSignals 表示匿名候选所需的评分信号。
type AnonymousSignals struct {
	ScenePopularityScores  map[int64]float64
	GlobalPopularityScores map[int64]float64
	SceneExposurePenalties map[int64]float64
	ActorExposurePenalties map[int64]float64
}

// ResolveCandidateLimit 计算当前分页请求的候选池大小。
func ResolveCandidateLimit(pageNum, pageSize int64) int {
	limit := int(pageNum * pageSize * PoolMultiplier)
	if limit < PoolMin {
		limit = PoolMin
	}
	if limit > PoolMax {
		limit = PoolMax
	}
	return limit
}

// BuildPersonalized 根据召回信号构建登录态候选集。
func BuildPersonalized(goodsList []*app.GoodsInfo, signals PersonalizedSignals) map[int64]*recommendCore.Candidate {
	candidates := make(map[int64]*recommendCore.Candidate, len(goodsList))
	for _, item := range goodsList {
		if item == nil || item.Id <= 0 {
			continue
		}
		candidate := &recommendCore.Candidate{
			Goods:         item,
			RecallSources: make(map[string]struct{}, 6),
		}
		candidate.RelationScore = signals.RelationScores[item.Id]
		candidate.UserGoodsScore = signals.UserGoodsScores[item.Id]
		candidate.ProfileScore = signals.ProfileScores[item.CategoryId]
		candidate.ScenePopularityScore = signals.ScenePopularityScores[item.Id]
		candidate.GlobalPopularityScore = signals.GlobalPopularityScores[item.Id]
		candidate.FreshnessScore = recommendRank.CalculateFreshnessScore(item.UpdatedAt)
		candidate.ExposurePenalty = signals.SceneExposurePenalties[item.Id]
		candidate.ActorExposurePenalty = signals.ActorExposurePenalties[item.Id]
		candidate.RepeatPenalty = CalculateRepeatPenalty(item.Id, signals.RecentPaidGoods)
		candidate.FinalScore = recommendRank.CalculateFinalScore(candidate)

		if candidate.RelationScore > 0 {
			candidate.RecallSources[RecallSourceRelation] = struct{}{}
		}
		if candidate.UserGoodsScore > 0 {
			candidate.RecallSources[RecallSourceUserGoods] = struct{}{}
		}
		if candidate.ProfileScore > 0 {
			candidate.RecallSources[RecallSourceProfile] = struct{}{}
		}
		if candidate.ScenePopularityScore > 0 {
			candidate.RecallSources[RecallSourceSceneHot] = struct{}{}
		}
		if candidate.GlobalPopularityScore > 0 {
			candidate.RecallSources[RecallSourceGlobalHot] = struct{}{}
		}
		if len(candidate.RecallSources) == 0 {
			candidate.RecallSources[RecallSourceLatest] = struct{}{}
		}
		if candidate.ActorExposurePenalty > 0 {
			candidate.RecallSources[RecallSourceActorPenalty] = struct{}{}
		}

		candidates[item.Id] = candidate
	}
	return candidates
}

// BuildAnonymous 根据公共信号构建匿名候选集。
func BuildAnonymous(goodsList []*app.GoodsInfo, signals AnonymousSignals) map[int64]*recommendCore.Candidate {
	candidates := make(map[int64]*recommendCore.Candidate, len(goodsList))
	for _, item := range goodsList {
		if item == nil || item.Id <= 0 {
			continue
		}
		candidate := &recommendCore.Candidate{
			Goods:         item,
			RecallSources: make(map[string]struct{}, 4),
		}
		candidate.ScenePopularityScore = signals.ScenePopularityScores[item.Id]
		candidate.GlobalPopularityScore = signals.GlobalPopularityScores[item.Id]
		candidate.FreshnessScore = recommendRank.CalculateFreshnessScore(item.UpdatedAt)
		candidate.ExposurePenalty = signals.SceneExposurePenalties[item.Id]
		candidate.ActorExposurePenalty = signals.ActorExposurePenalties[item.Id]
		candidate.FinalScore = recommendRank.CalculateAnonymousFinalScore(candidate)

		if candidate.ScenePopularityScore > 0 {
			candidate.RecallSources[RecallSourceSceneHot] = struct{}{}
		}
		if candidate.GlobalPopularityScore > 0 {
			candidate.RecallSources[RecallSourceGlobalHot] = struct{}{}
		}
		if len(candidate.RecallSources) == 0 {
			candidate.RecallSources[RecallSourceLatest] = struct{}{}
		}
		if candidate.ActorExposurePenalty > 0 {
			candidate.RecallSources[RecallSourceActorPenalty] = struct{}{}
		}

		candidates[item.Id] = candidate
	}
	return candidates
}

// CalculateRepeatPenalty 计算近期已购商品的重复推荐惩罚。
func CalculateRepeatPenalty(goodsID int64, recentPaidGoods map[int64]struct{}) float64 {
	if _, ok := recentPaidGoods[goodsID]; ok {
		return 1.5
	}
	return 0
}

// RankGoods 对候选商品执行统一排序和类目打散。
func RankGoods(candidates map[int64]*recommendCore.Candidate) []*app.GoodsInfo {
	return recommendFilter.DiversifyCandidates(
		recommendRank.RankCandidates(candidates),
		recommendFilter.DefaultMaxPerCategory,
	)
}

package rank

import (
	"recommend/contract"
	"recommend/internal/model"
	"time"
)

const freshnessWindowDays = 30.0

// ScoreCandidate 计算单个候选商品的最终得分。
func ScoreCandidate(candidate *model.Candidate, weights ScoreWeights, scoreTime time.Time) float64 {
	if candidate == nil {
		return 0
	}

	// 候选商品存在实体信息时，统一按商品时间计算新鲜度，避免不同召回器重复实现。
	if candidate.Goods != nil {
		candidate.Score.FreshnessScore = CalculateFreshnessScore(candidate.Goods, scoreTime)
	}

	finalScore := candidate.Score.RelationScore*weights.RelationWeight +
		candidate.Score.UserGoodsScore*weights.UserGoodsWeight +
		candidate.Score.CategoryScore*weights.CategoryWeight +
		candidate.Score.SceneHotScore*weights.SceneHotWeight +
		candidate.Score.GlobalHotScore*weights.GlobalHotWeight +
		candidate.Score.FreshnessScore*weights.FreshnessWeight +
		candidate.Score.SessionScore*weights.SessionWeight +
		candidate.Score.ExternalScore*weights.ExternalWeight +
		candidate.Score.CollaborativeScore*weights.CollaborativeWeight +
		candidate.Score.UserNeighborScore*weights.UserNeighborWeight -
		candidate.Score.ExposurePenalty*weights.ExposurePenalty -
		candidate.Score.RepeatPenalty*weights.RepeatPenalty

	candidate.Score.FinalScore = finalScore
	return finalScore
}

// ScoreCandidates 批量计算候选商品得分。
func ScoreCandidates(candidates []*model.Candidate, weights ScoreWeights, scoreTime time.Time) {
	for _, item := range candidates {
		ScoreCandidate(item, weights, scoreTime)
	}
}

// CalculateFreshnessScore 计算商品新鲜度得分。
func CalculateFreshnessScore(goods *contract.Goods, scoreTime time.Time) float64 {
	// 商品实体缺失时，无法基于时间信息计算新鲜度。
	if goods == nil {
		return 0
	}
	// 调用方未显式传入打分时间时，默认使用当前时间。
	if scoreTime.IsZero() {
		scoreTime = time.Now()
	}

	referenceTime := goods.UpdatedAt
	// 更新时间缺失时，回退到创建时间，保证新鲜度仍有统一口径。
	if referenceTime.IsZero() {
		referenceTime = goods.CreatedAt
	}
	// 时间字段都缺失时，无法计算新鲜度。
	if referenceTime.IsZero() {
		return 0
	}

	daysAgo := scoreTime.Sub(referenceTime).Hours() / 24
	// 商品时间晚于当前打分时间时，统一按满分处理。
	if daysAgo <= 0 {
		return 1
	}

	score := 1 - daysAgo/freshnessWindowDays
	// 超出时间窗口后，不再给新鲜度加分。
	if score < 0 {
		return 0
	}
	return score
}

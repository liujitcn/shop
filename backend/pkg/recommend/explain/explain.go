package explain

import (
	"sort"

	recommendcore "shop/pkg/recommend/core"
)

// ListRecallSources 返回稳定排序后的召回来源列表。
func ListRecallSources(input map[string]struct{}) []string {
	if len(input) == 0 {
		// 没有召回来源时返回空数组，便于上层直接序列化。
		return []string{}
	}
	result := make([]string, 0, len(input))
	for key := range input {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}

// BuildScoreDetail 构建单个候选商品的评分明细。
func BuildScoreDetail(candidate *recommendcore.Candidate) recommendcore.ScoreDetail {
	if candidate == nil || candidate.Goods == nil {
		// 缺失候选实体时返回空明细，避免上层崩溃。
		return recommendcore.ScoreDetail{}
	}
	return recommendcore.ScoreDetail{
		GoodsId:               candidate.Goods.ID,
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
		RecallSources:         ListRecallSources(candidate.RecallSources),
	}
}

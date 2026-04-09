package rank

import (
	"sort"
	"time"

	recommendcore "shop/pkg/recommend/core"
)

const (
	defaultFreshnessWindowDays   = 30.0
	defaultHighExposureThreshold = 20
)

// CalculateDayDecay 计算统计指标按天衰减后的权重。
func CalculateDayDecay(statDate time.Time) float64 {
	if statDate.IsZero() {
		// 空统计时间说明当前样本不可用。
		return 0
	}
	daysAgo := time.Since(statDate).Hours() / 24
	if daysAgo <= 0 {
		// 当天数据保持满权重。
		return 1
	}
	return 1 / (1 + daysAgo*0.08)
}

// CalculateExposurePenalty 计算高曝光低点击场景下的惩罚分。
func CalculateExposurePenalty(exposureCount, clickCount int64) float64 {
	if exposureCount < defaultHighExposureThreshold {
		// 低曝光商品不做额外惩罚。
		return 0
	}
	if clickCount <= 0 {
		// 已经被大量曝光但完全没有点击时给最高惩罚。
		return 1.2
	}
	ctr := float64(clickCount) / float64(exposureCount)
	switch {
	case ctr < 0.005:
		// 点击率极低时施加强惩罚。
		return 0.8
	case ctr < 0.01:
		// 点击率偏低时施加中等惩罚。
		return 0.4
	default:
		// 点击率正常时不额外扣分。
		return 0
	}
}

// CalculateFreshnessScore 计算商品的新鲜度分数。
func CalculateFreshnessScore(createdAt time.Time) float64 {
	if createdAt.IsZero() {
		// 缺失创建时间时无法计算新鲜度。
		return 0
	}
	daysAgo := time.Since(createdAt).Hours() / 24
	if daysAgo <= 0 {
		// 当天创建的商品给予满分。
		return 1
	}
	score := 1 - (daysAgo / defaultFreshnessWindowDays)
	if score < 0 {
		// 超过窗口期后新鲜度归零。
		return 0
	}
	return score
}

// CalculateFinalScore 计算个性化推荐的最终得分。
func CalculateFinalScore(candidate *recommendcore.Candidate) float64 {
	if candidate == nil {
		// 空候选直接返回零分。
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

// CalculateAnonymousFinalScore 计算匿名推荐的最终得分。
func CalculateAnonymousFinalScore(candidate *recommendcore.Candidate) float64 {
	if candidate == nil {
		// 空候选直接返回零分。
		return 0
	}
	return candidate.ScenePopularityScore*0.55 +
		candidate.GlobalPopularityScore*0.30 +
		candidate.FreshnessScore*0.15 -
		candidate.ExposurePenalty -
		candidate.ActorExposurePenalty
}

// RankCandidates 按得分和兜底规则排序候选列表。
func RankCandidates(candidates map[int64]*recommendcore.Candidate) []*recommendcore.Candidate {
	if len(candidates) == 0 {
		// 空候选集直接返回空结果。
		return []*recommendcore.Candidate{}
	}
	list := make([]*recommendcore.Candidate, 0, len(candidates))
	for _, item := range candidates {
		if item == nil || item.Goods == nil {
			// 跳过缺失商品实体的脏数据。
			continue
		}
		list = append(list, item)
	}
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].FinalScore == list[j].FinalScore {
			// 最终分相同时优先比较场景热度。
			if list[i].ScenePopularityScore == list[j].ScenePopularityScore {
				// 场景热度也相同时优先返回更新的商品。
				return list[i].Goods.CreatedAt.After(list[j].Goods.CreatedAt)
			}
			return list[i].ScenePopularityScore > list[j].ScenePopularityScore
		}
		return list[i].FinalScore > list[j].FinalScore
	})
	return list
}

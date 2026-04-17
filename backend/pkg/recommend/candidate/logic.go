package candidate

import (
	"sort"

	"shop/api/gen/go/app"
	"shop/api/gen/go/conf"
	recommendCore "shop/pkg/recommend/core"
	recommendDomain "shop/pkg/recommend/domain"
	recommendRank "shop/pkg/recommend/rank"

	_time "github.com/liujitcn/go-utils/time"
)

const (
	RecallSourceRelation     = "relation"
	RecallSourceUserGoods    = "user_goods"
	RecallSourceSimilarUser  = "similar_user"
	RecallSourceProfile      = "profile"
	RecallSourceContentBased = "content_based"
	RecallSourceCF           = "collaborative_filtering"
	RecallSourceSceneHot     = "scene_hot"
	RecallSourceGlobalHot    = "global_hot"
	RecallSourceLatest       = "latest"
	RecallSourceActorPenalty = "actor_penalty"
)

var (
	PoolMultiplier int64 = 8
	PoolMin        int64 = 80
	PoolMax        int64 = 240

	DefaultMaxPerCategory int = 2

	AnonymousRecallDays       = 30
	StatLookbackDays          = 30
	RecentPayPenaltyDays      = 15
	ActorExposureLookbackDays = 7

	ActorNoClickExposureThreshold int64   = 3
	ActorNoClickPenalty           float64 = 0.6
	ActorLowCtrExposureThreshold  int64   = 5
	ActorLowCtrPenalty            float64 = 0.3
	ActorLowCtrThreshold          float64 = 0.05
)

// ApplyRecommendConfig 应用推荐召回与主体曝光惩罚配置。
func ApplyRecommendConfig(cfg *conf.GoodsRecommendConfig) {
	// 配置缺失时，保留当前默认召回参数。
	if cfg == nil {
		return
	}
	if cfg.GetRecall() != nil {
		recallConfig := cfg.GetRecall()
		PoolMultiplier = int64(recallConfig.GetPoolMultiplier())
		PoolMin = int64(recallConfig.GetPoolMin())
		PoolMax = int64(recallConfig.GetPoolMax())
		DefaultMaxPerCategory = int(recallConfig.GetMaxPerCategory())
		AnonymousRecallDays = int(recallConfig.GetAnonymousRecallDays())
		StatLookbackDays = int(recallConfig.GetStatLookbackDays())
		RecentPayPenaltyDays = int(recallConfig.GetRecentPayPenaltyDays())
		ActorExposureLookbackDays = int(recallConfig.GetActorExposureLookbackDays())
	}
	if cfg.GetActorExposurePenalty() != nil {
		penaltyConfig := cfg.GetActorExposurePenalty()
		ActorNoClickExposureThreshold = int64(penaltyConfig.GetNoClickExposureThreshold())
		ActorNoClickPenalty = penaltyConfig.GetNoClickPenalty()
		ActorLowCtrExposureThreshold = int64(penaltyConfig.GetLowCtrExposureThreshold())
		ActorLowCtrPenalty = penaltyConfig.GetLowCtrPenalty()
		ActorLowCtrThreshold = penaltyConfig.GetLowCtrThreshold()
	}
}

// PersonalizedSignals 兼容旧引用，实际类型已下沉到领域层维护。
// 阶段 4 召回层统一切换到领域对象后，可统一搜索 “阶段 4 后删除：兼容旧引用” 清理本别名。
// 阶段 4 后删除：兼容旧引用。
type PersonalizedSignals = recommendDomain.PersonalizedSignals

// AnonymousSignals 兼容旧引用，实际类型已下沉到领域层维护。
// 阶段 4 召回层统一切换到领域对象后，可统一搜索 “阶段 4 后删除：兼容旧引用” 清理本别名。
// 阶段 4 后删除：兼容旧引用。
type AnonymousSignals = recommendDomain.AnonymousSignals

// ResolveCandidateLimit 计算当前分页请求的候选池大小。
func ResolveCandidateLimit(pageNum, pageSize int64) int64 {
	limit := pageNum * pageSize * PoolMultiplier
	// 候选池过小时，回退到系统允许的最小容量。
	if limit < PoolMin {
		limit = PoolMin
	}
	// 候选池过大时，截断到系统允许的最大容量。
	if limit > PoolMax {
		limit = PoolMax
	}
	return limit
}

// BuildPersonalized 根据召回信号构建登录态候选集。
func BuildPersonalized(
	goodsList []*app.GoodsInfo,
	signals PersonalizedSignals,
	rankWeightConfig *conf.GoodsRecommendPersonalizedRankWeightConfig,
) map[int64]*recommendCore.Candidate {
	candidates := make(map[int64]*recommendCore.Candidate, len(goodsList))
	for _, item := range goodsList {
		// 过滤空商品和非法商品，避免脏数据进入候选池。
		if item == nil || item.Id <= 0 {
			continue
		}
		candidate := &recommendCore.Candidate{
			Goods:         item,
			RecallSources: make(map[string]struct{}, 7),
		}
		candidate.RelationScore = signals.RelationScores[item.Id]
		candidate.UserGoodsScore = signals.UserGoodsScores[item.Id]
		candidate.SimilarUserScore = signals.SimilarUserScores[item.Id]
		candidate.ProfileScore = signals.ProfileScores[item.CategoryId]
		candidate.ScenePopularityScore = signals.ScenePopularityScores[item.Id]
		candidate.GlobalPopularityScore = signals.GlobalPopularityScores[item.Id]
		candidate.FreshnessScore = recommendRank.CalculateFreshnessScore(item.UpdatedAt)
		candidate.ExposurePenalty = signals.SceneExposurePenalties[item.Id]
		candidate.ActorExposurePenalty = signals.ActorExposurePenalties[item.Id]
		// 近期已购商品需要附加重复推荐惩罚，避免短时间内反复推荐。
		if _, ok := signals.RecentPaidGoods[item.Id]; ok {
			candidate.RepeatPenalty = 1.5
		}
		candidate.RuleScore = candidate.RelationScore*rankWeightConfig.GetRelationWeight() +
			candidate.UserGoodsScore*rankWeightConfig.GetUserGoodsWeight() +
			candidate.SimilarUserScore*rankWeightConfig.GetSimilarUserWeight() +
			candidate.ProfileScore*rankWeightConfig.GetProfileWeight() +
			candidate.ScenePopularityScore*rankWeightConfig.GetScenePopularityWeight() +
			candidate.GlobalPopularityScore*rankWeightConfig.GetGlobalPopularityWeight() +
			candidate.FreshnessScore*rankWeightConfig.GetFreshnessWeight() -
			candidate.ExposurePenalty*rankWeightConfig.GetExposurePenaltyWeight() -
			candidate.ActorExposurePenalty*rankWeightConfig.GetActorExposurePenaltyWeight() -
			candidate.RepeatPenalty*rankWeightConfig.GetRepeatPenaltyWeight()
		candidate.FinalScore = candidate.RuleScore

		// 命中了商品关联召回时记录来源，便于 explain 返回。
		if candidate.RelationScore > 0 {
			candidate.RecallSources[RecallSourceRelation] = struct{}{}
		}
		// 命中了用户商品偏好召回时记录来源。
		if candidate.UserGoodsScore > 0 {
			candidate.RecallSources[RecallSourceUserGoods] = struct{}{}
		}
		// 命中了相似用户偏好召回时记录来源。
		if candidate.SimilarUserScore > 0 {
			candidate.RecallSources[RecallSourceSimilarUser] = struct{}{}
		}
		// 命中了类目画像召回时记录来源。
		if candidate.ProfileScore > 0 {
			candidate.RecallSources[RecallSourceProfile] = struct{}{}
		}
		// 命中了场景热度召回时记录来源。
		if candidate.ScenePopularityScore > 0 {
			candidate.RecallSources[RecallSourceSceneHot] = struct{}{}
		}
		// 命中了全站热度召回时记录来源。
		if candidate.GlobalPopularityScore > 0 {
			candidate.RecallSources[RecallSourceGlobalHot] = struct{}{}
		}
		// 没有任何显式召回来源时，说明当前候选来自 latest 兜底。
		if len(candidate.RecallSources) == 0 {
			candidate.RecallSources[RecallSourceLatest] = struct{}{}
		}
		// 记录用户级曝光惩罚命中情况，便于排查降权原因。
		if candidate.ActorExposurePenalty > 0 {
			candidate.RecallSources[RecallSourceActorPenalty] = struct{}{}
		}

		candidates[item.Id] = candidate
	}
	return candidates
}

// BuildAnonymous 根据公共信号构建匿名候选集。
func BuildAnonymous(
	goodsList []*app.GoodsInfo,
	signals AnonymousSignals,
	rankWeightConfig *conf.GoodsRecommendAnonymousRankWeightConfig,
) map[int64]*recommendCore.Candidate {
	candidates := make(map[int64]*recommendCore.Candidate, len(goodsList))
	for _, item := range goodsList {
		// 过滤空商品和非法商品，避免匿名候选池混入脏数据。
		if item == nil || item.Id <= 0 {
			continue
		}
		candidate := &recommendCore.Candidate{
			Goods:         item,
			RecallSources: make(map[string]struct{}, 5),
		}
		candidate.RelationScore = signals.RelationScores[item.Id]
		candidate.ScenePopularityScore = signals.ScenePopularityScores[item.Id]
		candidate.GlobalPopularityScore = signals.GlobalPopularityScores[item.Id]
		candidate.FreshnessScore = recommendRank.CalculateFreshnessScore(item.UpdatedAt)
		candidate.ExposurePenalty = signals.SceneExposurePenalties[item.Id]
		candidate.ActorExposurePenalty = signals.ActorExposurePenalties[item.Id]
		candidate.RuleScore = candidate.RelationScore*rankWeightConfig.GetRelationWeight() +
			candidate.ScenePopularityScore*rankWeightConfig.GetScenePopularityWeight() +
			candidate.GlobalPopularityScore*rankWeightConfig.GetGlobalPopularityWeight() +
			candidate.FreshnessScore*rankWeightConfig.GetFreshnessWeight() -
			candidate.ExposurePenalty*rankWeightConfig.GetExposurePenaltyWeight() -
			candidate.ActorExposurePenalty*rankWeightConfig.GetActorExposurePenaltyWeight()
		candidate.FinalScore = candidate.RuleScore

		// 命中了商品关联召回时记录来源，便于详情页匿名推荐解释。
		if candidate.RelationScore > 0 {
			candidate.RecallSources[RecallSourceRelation] = struct{}{}
		}
		// 命中了场景热度召回时记录来源。
		if candidate.ScenePopularityScore > 0 {
			candidate.RecallSources[RecallSourceSceneHot] = struct{}{}
		}
		// 命中了全站热度召回时记录来源。
		if candidate.GlobalPopularityScore > 0 {
			candidate.RecallSources[RecallSourceGlobalHot] = struct{}{}
		}
		// 没有公共召回来源时，说明当前候选来自 latest 兜底。
		if len(candidate.RecallSources) == 0 {
			candidate.RecallSources[RecallSourceLatest] = struct{}{}
		}
		// 记录匿名主体曝光惩罚命中情况，便于 explain 返回。
		if candidate.ActorExposurePenalty > 0 {
			candidate.RecallSources[RecallSourceActorPenalty] = struct{}{}
		}

		candidates[item.Id] = candidate
	}
	return candidates
}

// RankGoods 对候选商品执行统一排序和类目打散。
func RankGoods(candidates map[int64]*recommendCore.Candidate) []*app.GoodsInfo {
	// 没有候选商品时，直接返回空结果避免继续排序。
	if len(candidates) == 0 {
		// 空候选集直接返回空商品列表。
		return []*app.GoodsInfo{}
	}

	rankedCandidates := make([]*recommendCore.Candidate, 0, len(candidates))
	for _, item := range candidates {
		// 缺失商品实体的候选无法参与排序，直接跳过。
		if item == nil || item.Goods == nil {
			continue
		}
		rankedCandidates = append(rankedCandidates, item)
	}
	sort.SliceStable(rankedCandidates, func(i, j int) bool {
		// 最终分相同时，继续按次级指标打破并列顺序。
		if rankedCandidates[i].FinalScore == rankedCandidates[j].FinalScore {
			// 最终分相同时优先比较场景热度。
			if rankedCandidates[i].ScenePopularityScore == rankedCandidates[j].ScenePopularityScore {
				iUpdatedAt := _time.StringTimeToTime(rankedCandidates[i].Goods.UpdatedAt)
				jUpdatedAt := _time.StringTimeToTime(rankedCandidates[j].Goods.UpdatedAt)
				// 左侧时间为空时不抢占前位，避免空时间排到前面。
				if iUpdatedAt == nil || iUpdatedAt.IsZero() {
					return false
				}
				// 右侧时间为空时左侧优先，保证有更新时间的商品排序更稳定。
				if jUpdatedAt == nil || jUpdatedAt.IsZero() {
					return true
				}
				// 场景热度也相同时优先返回更新的商品。
				return iUpdatedAt.After(*jUpdatedAt)
			}
			return rankedCandidates[i].ScenePopularityScore > rankedCandidates[j].ScenePopularityScore
		}
		return rankedCandidates[i].FinalScore > rankedCandidates[j].FinalScore
	})

	result := make([]*app.GoodsInfo, 0, len(rankedCandidates))
	categoryCount := make(map[int64]int, len(rankedCandidates))
	overflow := make([]*app.GoodsInfo, 0)
	for _, item := range rankedCandidates {
		categoryId := item.Goods.CategoryId
		// 单个类目达到上限后先放入溢出区，保持结果多样性。
		if categoryId > 0 && categoryCount[categoryId] >= DefaultMaxPerCategory {
			overflow = append(overflow, item.Goods)
			continue
		}
		categoryCount[categoryId]++
		result = append(result, item.Goods)
	}
	return append(result, overflow...)
}

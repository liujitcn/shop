package rank

import (
	"sort"
	"strings"

	recommendcore "shop/pkg/recommend/core"
	recommendDomain "shop/pkg/recommend/domain"

	_time "github.com/liujitcn/go-utils/time"
)

const (
	// rankingStageRule 表示规则粗排阶段。
	rankingStageRule = "rule"
	// rankingStageRanker 表示模型精排阶段。
	rankingStageRanker = "ranker"
	// rankingStageLlmRerank 表示 LLM 二次重排阶段。
	rankingStageLlmRerank = "llm_rerank"
)

// StageScoreSet 表示排序阶段需要消费的外部得分集合。
type StageScoreSet struct {
	RankerScores map[int64]float64 // 模型精排得分集合。
	LlmScores    map[int64]float64 // LLM 二次重排得分集合。
}

// ApplyRankingStrategy 将版本策略中的模型精排与 LLM 重排应用到候选集。
func ApplyRankingStrategy(
	candidates map[int64]*recommendcore.Candidate,
	strategy *recommendDomain.StrategyVersionConfig,
	stageScores StageScoreSet,
) map[string]any {
	stageContext := map[string]any{
		"appliedStages": []string{rankingStageRule},
		"rule": map[string]any{
			"candidateCount": len(candidates),
		},
	}
	// 当前没有候选时，只保留规则粗排阶段上下文。
	if len(candidates) == 0 {
		return stageContext
	}

	appliedStages := []string{rankingStageRule}
	if strategy != nil && strategy.Ranker != nil {
		rankerContext := applyRankerStage(candidates, strategy.Ranker, stageScores.RankerScores)
		stageContext[rankingStageRanker] = rankerContext
		applied, _ := rankerContext["applied"].(bool)
		// 当前模型精排阶段实际写回了得分时，再记录到已执行阶段。
		if applied {
			appliedStages = append(appliedStages, rankingStageRanker)
		}
	}
	if strategy != nil && strategy.LlmRerank != nil {
		llmContext := applyLlmStage(candidates, strategy.LlmRerank, stageScores.LlmScores)
		stageContext[rankingStageLlmRerank] = llmContext
		applied, _ := llmContext["applied"].(bool)
		// 当前 LLM 重排阶段实际写回了得分时，再记录到已执行阶段。
		if applied {
			appliedStages = append(appliedStages, rankingStageLlmRerank)
		}
	}
	stageContext["appliedStages"] = appliedStages
	return stageContext
}

// applyRankerStage 应用模型精排阶段得分。
func applyRankerStage(
	candidates map[int64]*recommendcore.Candidate,
	strategy *recommendDomain.RankerStrategy,
	stageScores map[int64]float64,
) map[string]any {
	stageContext := map[string]any{
		"enabled": strategy != nil && strategy.Enabled,
		"type":    recommendDomain.RankerTypeNone,
		"applied": false,
	}
	// 当前模型精排配置为空时，不进入阶段应用。
	if strategy == nil {
		stageContext["skippedReason"] = "not_configured"
		return stageContext
	}

	stageContext["type"] = strategy.NormalizeType()
	stageContext["topN"] = strategy.ResolveTopN(int64(len(candidates)))
	stageContext["weight"] = strategy.ResolveWeight(1)
	if !strategy.IsEnabled() {
		stageContext["skippedReason"] = "disabled"
		return stageContext
	}
	if len(stageScores) == 0 {
		stageContext["skippedReason"] = "no_scores"
		return stageContext
	}

	appliedCount := applyStageScores(
		candidates,
		stageScores,
		strategy.ResolveTopN(int64(len(candidates))),
		func(candidate *recommendcore.Candidate, score float64) {
			candidate.ModelScore = score
			candidate.FinalScore += score * strategy.ResolveWeight(1)
		},
	)
	stageContext["requestedScoreCount"] = len(stageScores)
	stageContext["appliedCount"] = appliedCount
	stageContext["applied"] = appliedCount > 0
	// 当前阶段存在外部得分但都未命中候选时，明确补充跳过原因。
	if appliedCount == 0 {
		stageContext["skippedReason"] = "no_candidate_match"
	}
	return stageContext
}

// applyLlmStage 应用 LLM 二次重排阶段得分。
func applyLlmStage(
	candidates map[int64]*recommendcore.Candidate,
	strategy *recommendDomain.LlmRerankStrategy,
	stageScores map[int64]float64,
) map[string]any {
	stageContext := map[string]any{
		"enabled": strategy != nil && strategy.Enabled,
		"model":   "",
		"applied": false,
	}
	// 当前 LLM 重排配置为空时，不进入阶段应用。
	if strategy == nil {
		stageContext["skippedReason"] = "not_configured"
		return stageContext
	}

	stageContext["model"] = strategy.Model
	stageContext["topN"] = strategy.ResolveTopN(int64(len(candidates)))
	stageContext["weight"] = strategy.ResolveWeight(1)
	stageContext["cacheTTLSeconds"] = strategy.CacheTTLSeconds
	stageContext["timeoutSeconds"] = strategy.TimeoutSeconds
	stageContext["maxCompletionTokens"] = strategy.MaxCompletionTokens
	stageContext["hasPromptTemplate"] = strings.TrimSpace(strategy.PromptTemplate) != ""
	stageContext["hasCandidateFilterExpr"] = strings.TrimSpace(strategy.CandidateFilterExpr) != ""
	stageContext["hasScoreExpr"] = strings.TrimSpace(strategy.ScoreExpr) != ""
	stageContext["hasScoreScript"] = strings.TrimSpace(strategy.ScoreScript) != ""
	// 温度显式配置时，再把运行参数写入阶段上下文，便于线上排障确认是否走了确定性输出。
	if strategy.Temperature != nil {
		stageContext["temperature"] = *strategy.Temperature
	}
	if !strategy.IsEnabled() {
		stageContext["skippedReason"] = "disabled"
		return stageContext
	}
	if len(stageScores) == 0 {
		stageContext["skippedReason"] = "no_scores"
		return stageContext
	}

	appliedCount := applyStageScores(
		candidates,
		stageScores,
		strategy.ResolveTopN(int64(len(candidates))),
		func(candidate *recommendcore.Candidate, score float64) {
			candidate.LlmScore = score
			candidate.FinalScore += score * strategy.ResolveWeight(1)
		},
	)
	stageContext["requestedScoreCount"] = len(stageScores)
	stageContext["appliedCount"] = appliedCount
	stageContext["applied"] = appliedCount > 0
	// 当前阶段存在外部得分但都未命中候选时，明确补充跳过原因。
	if appliedCount == 0 {
		stageContext["skippedReason"] = "no_candidate_match"
	}
	return stageContext
}

// applyStageScores 将外部阶段得分写回到粗排 TopN 候选。
func applyStageScores(
	candidates map[int64]*recommendcore.Candidate,
	stageScores map[int64]float64,
	topN int64,
	applier func(candidate *recommendcore.Candidate, score float64),
) int {
	// 当前没有候选、没有得分或没有写回函数时，不执行阶段应用。
	if len(candidates) == 0 || len(stageScores) == 0 || applier == nil {
		return 0
	}

	rankedCandidates := sortCandidatesByScore(candidates)
	// TopN 非法时，统一回退为处理全部候选。
	if topN <= 0 || topN > int64(len(rankedCandidates)) {
		topN = int64(len(rankedCandidates))
	}
	appliedCount := 0
	for _, candidate := range rankedCandidates[:topN] {
		// 候选为空、商品为空或商品编号非法时，不执行得分写回。
		if candidate == nil || candidate.Goods == nil || candidate.Goods.Id <= 0 {
			continue
		}
		score, ok := stageScores[candidate.Goods.Id]
		if !ok {
			continue
		}
		applier(candidate, score)
		appliedCount++
	}
	return appliedCount
}

// sortCandidatesByScore 按当前最终分和次级指标稳定排序候选。
func sortCandidatesByScore(candidates map[int64]*recommendcore.Candidate) []*recommendcore.Candidate {
	rankedCandidates := make([]*recommendcore.Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		// 缺失商品实体的候选无法参与阶段排序。
		if candidate == nil || candidate.Goods == nil {
			continue
		}
		rankedCandidates = append(rankedCandidates, candidate)
	}
	sort.SliceStable(rankedCandidates, func(i, j int) bool {
		// 最终分相同时，继续按场景热度和更新时间打破并列顺序。
		if rankedCandidates[i].FinalScore == rankedCandidates[j].FinalScore {
			if rankedCandidates[i].ScenePopularityScore == rankedCandidates[j].ScenePopularityScore {
				iUpdatedAt := _time.StringTimeToTime(rankedCandidates[i].Goods.UpdatedAt)
				jUpdatedAt := _time.StringTimeToTime(rankedCandidates[j].Goods.UpdatedAt)
				// 左侧时间为空时，不抢占更靠前的位置。
				if iUpdatedAt == nil || iUpdatedAt.IsZero() {
					return false
				}
				// 右侧时间为空时，优先保留左侧候选。
				if jUpdatedAt == nil || jUpdatedAt.IsZero() {
					return true
				}
				return iUpdatedAt.After(*jUpdatedAt)
			}
			return rankedCandidates[i].ScenePopularityScore > rankedCandidates[j].ScenePopularityScore
		}
		return rankedCandidates[i].FinalScore > rankedCandidates[j].FinalScore
	})
	return rankedCandidates
}

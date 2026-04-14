package rank

import (
	"context"
	"errors"
	"fmt"
	recommendv1 "recommend/api/gen/go/recommend/v1"
	"recommend/contract"
	cachex "recommend/internal/cache"
	"recommend/internal/core"
	"recommend/internal/model"
	"sort"

	goleveldb "github.com/syndtr/goleveldb/leveldb"
)

// ApplyRankingMode 根据实例配置应用二阶段排序模式。
func ApplyRankingMode(
	ctx context.Context,
	request model.Request,
	dependencies core.Dependencies,
	config core.ServiceConfig,
	runtimeStore *cachex.RuntimeStore,
	candidates []*model.Candidate,
) error {
	switch config.Ranking.Mode {
	case "", core.RankingModeRule, core.RankingModeCustom:
		return nil
	case core.RankingModeFm:
		return applyFmRanking(request, runtimeStore, config, candidates)
	case core.RankingModeLlm:
		return applyLlmRanking(ctx, request, dependencies, config, candidates)
	default:
		return fmt.Errorf("recommend: 不支持的排序模式 %q", config.Ranking.Mode)
	}
}

// PredictFmScore 根据已训练模型状态计算单个候选的轻量 FM 分值。
func PredictFmScore(candidate *model.Candidate, state *recommendv1.RecommendRankingModelState) float64 {
	if candidate == nil || state == nil {
		return 0
	}
	return predictFmByFeatures(BuildFeatureMap(candidate), state)
}

// predictFmByFeatures 根据特征向量和模型状态计算分值。
func predictFmByFeatures(features map[string]float64, state *recommendv1.RecommendRankingModelState) float64 {
	if len(features) == 0 || state == nil {
		return 0
	}

	score := state.GetBias()
	linearWeights := buildLinearWeightMap(state)
	for featureName, value := range features {
		score += linearWeights[featureName] * value
	}

	interactionWeights := buildInteractionWeightMap(state)
	featureNames := RankingFeatureNames()
	for leftIndex := 0; leftIndex < len(featureNames); leftIndex++ {
		leftFeature := featureNames[leftIndex]
		leftValue := features[leftFeature]
		if leftValue == 0 {
			continue
		}
		for rightIndex := leftIndex + 1; rightIndex < len(featureNames); rightIndex++ {
			rightFeature := featureNames[rightIndex]
			rightValue := features[rightFeature]
			if rightValue == 0 {
				continue
			}
			score += interactionWeights[PairFeatureKey(leftFeature, rightFeature)] * leftValue * rightValue
		}
	}
	return score
}

// buildLinearWeightMap 将 proto 模型状态转换为一阶权重 map。
func buildLinearWeightMap(state *recommendv1.RecommendRankingModelState) map[string]float64 {
	result := make(map[string]float64, len(state.GetLinearWeights()))
	for _, item := range state.GetLinearWeights() {
		if item == nil || item.GetFeature() == "" {
			continue
		}
		result[item.GetFeature()] = item.GetWeight()
	}
	return result
}

// buildInteractionWeightMap 将 proto 模型状态转换为二阶权重 map。
func buildInteractionWeightMap(state *recommendv1.RecommendRankingModelState) map[string]float64 {
	result := make(map[string]float64, len(state.GetInteractionWeights()))
	for _, item := range state.GetInteractionWeights() {
		if item == nil || item.GetLeftFeature() == "" || item.GetRightFeature() == "" {
			continue
		}
		result[PairFeatureKey(item.GetLeftFeature(), item.GetRightFeature())] = item.GetWeight()
	}
	return result
}

// ResolveRankingModelName 解析当前排序模型状态使用的名称。
func ResolveRankingModelName(config core.ServiceConfig) string {
	if config.Training.CtrMode != "" {
		return config.Training.CtrMode
	}
	return string(core.RankingModeFm)
}

// applyFmRanking 应用学习排序模型分值。
func applyFmRanking(
	request model.Request,
	runtimeStore *cachex.RuntimeStore,
	config core.ServiceConfig,
	candidates []*model.Candidate,
) error {
	// 运行态缓存缺失时，无法加载已训练模型，直接保留规则分兜底。
	if runtimeStore == nil {
		return nil
	}
	state, err := runtimeStore.GetRankingModelState(request.Scene.String(), ResolveRankingModelName(config))
	if err != nil {
		if errors.Is(err, goleveldb.ErrNotFound) {
			return nil
		}
		return err
	}
	for _, item := range candidates {
		if item == nil {
			continue
		}
		item.Score.FmScore = PredictFmScore(item, state)
		item.Score.FinalScore = item.Score.FmScore
	}
	return nil
}

// applyLlmRanking 应用 LLM 重排结果。
func applyLlmRanking(
	ctx context.Context,
	request model.Request,
	dependencies core.Dependencies,
	config core.ServiceConfig,
	candidates []*model.Candidate,
) error {
	if dependencies.Reranker == nil {
		return errors.New("recommend: llm 排序模式未配置重排器")
	}
	if len(candidates) == 0 {
		return nil
	}

	blendWeight := config.Ranking.LlmBlendWeight
	rerankLimit := int(config.Ranking.LlmCandidateLimit)
	if rerankLimit <= 0 || rerankLimit > len(candidates) {
		rerankLimit = len(candidates)
	}

	sortedCandidates := append([]*model.Candidate(nil), candidates...)
	sortCandidates(sortedCandidates)
	sortedCandidates = sortedCandidates[:rerankLimit]

	results, err := dependencies.Reranker.Rerank(ctx, buildLlmRerankRequest(request, sortedCandidates))
	if err != nil {
		return err
	}

	resultMap := make(map[int64]*contract.LlmRerankResult, len(results))
	for _, item := range results {
		if item == nil || item.GoodsId <= 0 {
			continue
		}
		resultMap[item.GoodsId] = item
	}

	for _, item := range candidates {
		if item == nil {
			continue
		}
		result, ok := resultMap[item.GoodsId()]
		if !ok {
			continue
		}
		item.Score.LlmScore = result.Score
		item.Score.FinalScore = item.Score.RuleScore*(1-blendWeight) + item.Score.LlmScore*blendWeight
		// LLM 返回了解释文本时，写入 trace 原因，便于 explain 时继续追溯来源。
		if result.Reason != "" {
			item.AddTraceReason(result.Reason)
		}
	}
	return nil
}

// buildLlmRerankRequest 构建传给 LLM 重排器的请求结构。
func buildLlmRerankRequest(request model.Request, candidates []*model.Candidate) contract.LlmRerankRequest {
	result := contract.LlmRerankRequest{
		Scene:            request.Scene.String(),
		ActorType:        int32(request.Actor.Type),
		ActorId:          request.Actor.Id,
		SessionId:        request.Actor.SessionId,
		GoodsId:          request.Context.GoodsId,
		OrderId:          request.Context.OrderId,
		ExternalStrategy: request.Context.ExternalStrategy,
		Attributes:       cloneStringMap(request.Context.Attributes),
		Candidates:       make([]*contract.LlmRerankCandidate, 0, len(candidates)),
	}
	for _, item := range candidates {
		if item == nil || item.Goods == nil {
			continue
		}
		result.Candidates = append(result.Candidates, &contract.LlmRerankCandidate{
			GoodsId:            item.GoodsId(),
			CategoryId:         item.CategoryId(),
			BaseScore:          item.Score.RuleScore,
			RelationScore:      item.Score.RelationScore,
			UserGoodsScore:     item.Score.UserGoodsScore,
			CategoryScore:      item.Score.CategoryScore,
			SceneHotScore:      item.Score.SceneHotScore,
			GlobalHotScore:     item.Score.GlobalHotScore,
			FreshnessScore:     item.Score.FreshnessScore,
			SessionScore:       item.Score.SessionScore,
			ExternalScore:      item.Score.ExternalScore,
			CollaborativeScore: item.Score.CollaborativeScore,
			UserNeighborScore:  item.Score.UserNeighborScore,
			VectorScore:        item.Score.VectorScore,
			RecallSources:      item.RecallSourceList(),
		})
	}
	return result
}

// cloneStringMap 复制字符串 map，避免重排器修改原始上下文数据。
func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	result := make(map[string]string, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}

// SortCandidatesByRuleScore 返回按规则分稳定排序后的候选副本。
func SortCandidatesByRuleScore(candidates []*model.Candidate) []*model.Candidate {
	result := append([]*model.Candidate(nil), candidates...)
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Score.RuleScore != result[j].Score.RuleScore {
			return result[i].Score.RuleScore > result[j].Score.RuleScore
		}
		return result[i].GoodsId() < result[j].GoodsId()
	})
	return result
}

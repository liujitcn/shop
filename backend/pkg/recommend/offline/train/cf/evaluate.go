package cf

import (
	"math"

	"shop/pkg/recommend/offline/train/util"
)

// Score 表示协同过滤排序评估结果。
type Score struct {
	NDCG      float32
	Precision float32
	Recall    float32
}

// EvaluateConfig 表示协同过滤评估配置。
type EvaluateConfig struct {
	TopK       int
	Candidates int
	Seed       int64
}

// fillDefault 补齐评估默认参数。
func (c EvaluateConfig) fillDefault() EvaluateConfig {
	// 未显式指定 topK 时，回退到常见的离线评估窗口。
	if c.TopK <= 0 {
		c.TopK = 10
	}
	// 未显式指定候选数量时，回退到轻量负采样规模。
	if c.Candidates <= 0 {
		c.Candidates = 100
	}
	return c
}

// Evaluate 评估 BPR 模型在验证集上的排序质量。
func Evaluate(model *Model, trainSet *Dataset, testSet *Dataset, config EvaluateConfig) Score {
	// 模型或数据集缺失时，当前评估无法继续执行。
	if model == nil || trainSet == nil || testSet == nil || testSet.Count() == 0 {
		return Score{}
	}
	config = config.fillDefault()
	rng := util.NewRandomGenerator(config.Seed)
	itemPool := model.ItemIds()
	totalScore := Score{}
	evaluatedUserCount := 0

	for _, userId := range testSet.UserIds() {
		targetItemIdSet := filterPredictableTargets(model, testSet.UserItems(userId))
		// 当前用户在验证集中没有可评分的目标商品时，不计入评估。
		if len(targetItemIdSet) == 0 {
			continue
		}

		candidateItemIds := buildEvaluationCandidates(
			targetItemIdSet,
			trainSet.UserItemSet(userId),
			itemPool,
			config.Candidates,
			rng,
		)
		rankedList := model.Rank(userId, candidateItemIds, config.TopK)
		// 用户不存在于训练模型或候选构造失败时，当前样本跳过。
		if len(rankedList) == 0 {
			continue
		}

		rankedItemIds := make([]string, len(rankedList))
		for index, item := range rankedList {
			rankedItemIds[index] = item.ItemId
		}
		totalScore.NDCG += NDCG(targetItemIdSet, rankedItemIds)
		totalScore.Precision += Precision(targetItemIdSet, rankedItemIds)
		totalScore.Recall += Recall(targetItemIdSet, rankedItemIds)
		evaluatedUserCount++
	}
	// 没有任何用户进入评估时，统一返回零分。
	if evaluatedUserCount == 0 {
		return Score{}
	}
	scale := float32(1) / float32(evaluatedUserCount)
	totalScore.NDCG *= scale
	totalScore.Precision *= scale
	totalScore.Recall *= scale
	return totalScore
}

// NDCG 计算归一化折损累计增益。
func NDCG(targetItemIdSet map[string]struct{}, rankedItemIds []string) float32 {
	// 目标集或排序列表为空时，NDCG 统一回退为零。
	if len(targetItemIdSet) == 0 || len(rankedItemIds) == 0 {
		return 0
	}
	idcg := float64(0)
	upperBound := min(len(targetItemIdSet), len(rankedItemIds))
	for index := 0; index < upperBound; index++ {
		idcg += 1 / math.Log2(float64(index)+2)
	}
	dcg := float64(0)
	for index, itemId := range rankedItemIds {
		if _, ok := targetItemIdSet[itemId]; ok {
			dcg += 1 / math.Log2(float64(index)+2)
		}
	}
	// 理想增益为零时，当前样本无法产生有效排序收益。
	if idcg == 0 {
		return 0
	}
	return float32(dcg / idcg)
}

// Precision 计算精确率。
func Precision(targetItemIdSet map[string]struct{}, rankedItemIds []string) float32 {
	// 目标集或排序列表为空时，Precision 统一回退为零。
	if len(targetItemIdSet) == 0 || len(rankedItemIds) == 0 {
		return 0
	}
	hitCount := 0
	for _, itemId := range rankedItemIds {
		if _, ok := targetItemIdSet[itemId]; ok {
			hitCount++
		}
	}
	return float32(hitCount) / float32(len(rankedItemIds))
}

// Recall 计算召回率。
func Recall(targetItemIdSet map[string]struct{}, rankedItemIds []string) float32 {
	// 目标集或排序列表为空时，Recall 统一回退为零。
	if len(targetItemIdSet) == 0 || len(rankedItemIds) == 0 {
		return 0
	}
	hitCount := 0
	for _, itemId := range rankedItemIds {
		if _, ok := targetItemIdSet[itemId]; ok {
			hitCount++
		}
	}
	return float32(hitCount) / float32(len(targetItemIdSet))
}

// filterPredictableTargets 过滤掉模型当前无法打分的目标商品。
func filterPredictableTargets(model *Model, itemIds []string) map[string]struct{} {
	result := make(map[string]struct{})
	// 模型为空时，不存在任何可评分目标商品。
	if model == nil {
		return result
	}
	for _, itemId := range itemIds {
		// 模型未学习到该商品向量时，当前目标无法进入评估。
		if _, ok := model.itemIndex[itemId]; !ok {
			continue
		}
		result[itemId] = struct{}{}
	}
	return result
}

// buildEvaluationCandidates 组装当前用户的验证候选集。
func buildEvaluationCandidates(
	targetItemIdSet map[string]struct{},
	excludedItemIdSet map[string]struct{},
	itemPool []string,
	candidateCount int,
	rng util.RandomGenerator,
) []string {
	result := make([]string, 0, len(targetItemIdSet)+candidateCount)
	for itemId := range targetItemIdSet {
		result = append(result, itemId)
	}
	eligibleNegativeItemIds := make([]string, 0, len(itemPool))
	for _, itemId := range itemPool {
		// 目标商品必须保留在候选里，不能在过滤阶段被误删。
		if _, ok := targetItemIdSet[itemId]; ok {
			continue
		}
		// 训练集中已经出现过的商品不能作为验证负样本。
		if _, ok := excludedItemIdSet[itemId]; ok {
			continue
		}
		eligibleNegativeItemIds = append(eligibleNegativeItemIds, itemId)
	}
	// 可选负样本不足时，直接全部拼入候选集。
	if candidateCount >= len(eligibleNegativeItemIds) {
		return append(result, eligibleNegativeItemIds...)
	}
	shuffleStrings(rng, eligibleNegativeItemIds)
	return append(result, eligibleNegativeItemIds[:candidateCount]...)
}

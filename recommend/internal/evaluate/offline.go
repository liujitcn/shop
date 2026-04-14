package evaluate

import (
	"context"
	"errors"
	"recommend/contract"
	"recommend/internal/core"
	"sort"
	"time"
)

// EvaluateOffline 基于请求、曝光和行为事实执行离线评估。
func EvaluateOffline(ctx context.Context, dependencies core.Dependencies, request core.EvaluateRequest) (*core.EvaluateResult, error) {
	// 离线评估直接基于推荐事实表执行，没有推荐数据源时无法开展评估。
	if dependencies.Recommend == nil {
		return nil, errors.New("recommend: 推荐数据源未配置")
	}

	scenes := normalizeEvaluateScenes(request.Scenes)
	topK := normalizeTopK(request.TopK)
	startAt, endAt := resolveEvaluateWindow(request.StatDate)
	result := &core.EvaluateResult{
		GeneratedAt: time.Now(),
		Scenes:      make([]core.SceneMetric, 0, len(scenes)),
	}

	for _, scene := range scenes {
		metric, err := evaluateScene(ctx, dependencies, scene, startAt, endAt, topK)
		if err != nil {
			return nil, err
		}
		result.Scenes = append(result.Scenes, metric)
	}
	return result, nil
}

// evaluateScene 评估单个场景的离线指标。
func evaluateScene(
	ctx context.Context,
	dependencies core.Dependencies,
	scene core.Scene,
	startAt time.Time,
	endAt time.Time,
	topK int,
) (core.SceneMetric, error) {
	requestFacts, err := dependencies.Recommend.ListRequestFacts(ctx, string(scene), startAt, endAt)
	if err != nil {
		return core.SceneMetric{}, err
	}
	exposureFacts, err := dependencies.Recommend.ListExposureFacts(ctx, string(scene), startAt, endAt)
	if err != nil {
		return core.SceneMetric{}, err
	}
	actionFacts, err := dependencies.Recommend.ListActionFacts(ctx, string(scene), startAt, endAt)
	if err != nil {
		return core.SceneMetric{}, err
	}

	metric := core.SceneMetric{
		Scene:         scene,
		RequestCount:  int64(countValidRequests(requestFacts)),
		ExposureCount: countExposureItems(exposureFacts),
	}

	requestGoodsByRequestId := buildRequestGoodsMap(requestFacts, topK)
	positiveGoodsByRequestId := make(map[string]map[int64]int32)
	precisionSum := 0.0
	recallSum := 0.0
	ndcgSum := 0.0
	evalRequestCount := int64(0)

	for _, item := range actionFacts {
		// 无效请求编号不会参与离线评估回放。
		if item == nil || item.RequestId == "" {
			continue
		}

		switch item.EventType {
		// 点击是 CTR 的分子。
		case string(core.BehaviorClick):
			metric.ClickCount++
		// 下单是下单率的分子。
		case string(core.BehaviorOrderCreate):
			metric.OrderCount++
		// 支付是支付率的分子。
		case string(core.BehaviorOrderPay):
			metric.PayCount++
		}

		relevance, ok := actionRelevance(item.EventType)
		// 只有正反馈事件才进入排序评估样本。
		if !ok || item.GoodsId <= 0 {
			continue
		}
		requestGoods, exists := requestGoodsByRequestId[item.RequestId]
		// 找不到推荐请求事实时，无法计算 ranking 指标。
		if !exists || len(requestGoods) == 0 {
			continue
		}
		positiveGoodsMap, exists := positiveGoodsByRequestId[item.RequestId]
		if !exists {
			positiveGoodsMap = make(map[int64]int32)
			positiveGoodsByRequestId[item.RequestId] = positiveGoodsMap
		}
		// 同一请求商品多次命中正反馈时，只保留更强相关性等级。
		if relevance > positiveGoodsMap[item.GoodsId] {
			positiveGoodsMap[item.GoodsId] = relevance
		}
	}

	requestIds := make([]string, 0, len(positiveGoodsByRequestId))
	for requestId := range positiveGoodsByRequestId {
		requestIds = append(requestIds, requestId)
	}
	sort.Strings(requestIds)

	for _, requestId := range requestIds {
		rankedGoods := requestGoodsByRequestId[requestId]
		positiveGoodsMap := positiveGoodsByRequestId[requestId]
		// 当前请求没有候选列表或正反馈时，不进入排序评估均值。
		if len(rankedGoods) == 0 || len(positiveGoodsMap) == 0 {
			continue
		}

		hitCount, ndcg := calculateRankingMetrics(rankedGoods, positiveGoodsMap)
		evalRequestCount++
		precisionSum += float64(hitCount) / float64(len(rankedGoods))
		recallSum += float64(hitCount) / float64(len(positiveGoodsMap))
		ndcgSum += ndcg
	}

	if evalRequestCount > 0 {
		metric.Precision = precisionSum / float64(evalRequestCount)
		metric.Recall = recallSum / float64(evalRequestCount)
		metric.Ndcg = ndcgSum / float64(evalRequestCount)
	}
	if metric.ExposureCount > 0 {
		metric.Ctr = float64(metric.ClickCount) / float64(metric.ExposureCount)
	}
	if metric.ClickCount > 0 {
		// 下单率与支付率都沿用点击后转化口径，便于和现有后端报表对齐。
		metric.OrderRate = float64(metric.OrderCount) / float64(metric.ClickCount)
		metric.PayRate = float64(metric.PayCount) / float64(metric.ClickCount)
	}
	return metric, nil
}

// normalizeEvaluateScenes 归一化评估场景列表。
func normalizeEvaluateScenes(input []core.Scene) []core.Scene {
	list := input
	if len(list) == 0 {
		list = []core.Scene{
			core.SceneHome,
			core.SceneGoodsDetail,
			core.SceneCart,
			core.SceneProfile,
			core.SceneOrderDetail,
			core.SceneOrderPaid,
		}
	}

	result := make([]core.Scene, 0, len(list))
	seen := make(map[core.Scene]struct{}, len(list))
	for _, scene := range list {
		// 空场景不参与离线评估，避免向下游查询发送无效条件。
		if scene == "" {
			continue
		}
		if _, ok := seen[scene]; ok {
			continue
		}
		seen[scene] = struct{}{}
		result = append(result, scene)
	}
	return result
}

// normalizeTopK 归一化评估使用的 topK。
func normalizeTopK(topK int32) int {
	if topK <= 0 {
		return 10
	}
	return int(topK)
}

// resolveEvaluateWindow 解析评估统计窗口。
func resolveEvaluateWindow(statDate time.Time) (time.Time, time.Time) {
	if statDate.IsZero() {
		statDate = time.Now()
	}
	startAt := time.Date(statDate.Year(), statDate.Month(), statDate.Day(), 0, 0, 0, 0, statDate.Location())
	return startAt, startAt.AddDate(0, 0, 1)
}

// countValidRequests 统计有效请求数量。
func countValidRequests(list []*contract.RequestFact) int {
	count := 0
	for _, item := range list {
		// 请求编号缺失的事实无法作为独立评估样本。
		if item == nil || item.RequestId == "" {
			continue
		}
		count++
	}
	return count
}

// countExposureItems 统计逐商品曝光量。
func countExposureItems(list []*contract.ExposureFact) int64 {
	total := int64(0)
	for _, item := range list {
		if item == nil {
			continue
		}
		for _, goodsId := range item.GoodsIds {
			// 非法商品编号不计入曝光分母。
			if goodsId <= 0 {
				continue
			}
			total++
		}
	}
	return total
}

// buildRequestGoodsMap 构建请求编号到去重排序商品列表的映射。
func buildRequestGoodsMap(list []*contract.RequestFact, topK int) map[string][]int64 {
	result := make(map[string][]int64, len(list))
	for _, item := range list {
		// 请求编号缺失的事实无法参与后续行为回放。
		if item == nil || item.RequestId == "" {
			continue
		}
		result[item.RequestId] = dedupeGoodsIds(item.GoodsIds, topK)
	}
	return result
}

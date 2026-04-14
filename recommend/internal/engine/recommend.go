package engine

import (
	"context"
	"recommend/internal/core"
	"recommend/internal/model"
	"recommend/internal/scene"
	"sort"
)

// Recommend 执行推荐主链路并返回对外结果。
func Recommend(ctx context.Context, dependencies core.Dependencies, config core.ServiceConfig, request core.RecommendRequest) (*core.RecommendResult, error) {
	internalRequest := model.ResolveRequest(request)
	traceId := internalRequest.Context.RequestId
	// 调用方显式要求 explain 时，没有追踪编号会导致后续无法回查，因此在这里补一个兜底编号。
	if internalRequest.NeedExplain && traceId == "" {
		traceId = buildGeneratedTraceId()
		internalRequest.Context.RequestId = traceId
	}

	candidates, err := scene.Run(ctx, internalRequest, dependencies, config)
	if err != nil {
		return nil, err
	}

	total := int64(len(candidates))
	offset := internalRequest.Offset()
	limit := internalRequest.Limit()
	// 当前页偏移超出候选数量时，直接返回空页结果。
	if offset >= len(candidates) {
		result := &core.RecommendResult{
			TraceId: traceId,
			Total:   total,
		}
		err = saveRecommendTrace(ctx, dependencies, config, internalRequest, traceId, candidates, nil)
		// explain 被显式请求且实例开启严格 trace 持久化时，保存失败应直接暴露给调用方。
		if err != nil && internalRequest.NeedExplain && config.Explain.StrictTracePersistence {
			return nil, err
		}
		return result, nil
	}

	end := offset + limit
	// 当前页结束位置越界时，按候选集末尾截断。
	if end > len(candidates) {
		end = len(candidates)
	}

	currentPage := candidates[offset:end]
	items := make([]core.RecommendItem, 0, len(currentPage))
	goodsIds := make([]int64, 0, len(currentPage))
	recallSourceMap := make(map[string]struct{})

	for _, item := range currentPage {
		recallSources := item.RecallSourceList()
		for _, source := range recallSources {
			recallSourceMap[source] = struct{}{}
		}
		items = append(items, core.RecommendItem{
			GoodsId:       item.GoodsId(),
			Score:         item.Score.FinalScore,
			RecallSources: recallSources,
		})
		goodsIds = append(goodsIds, item.GoodsId())
	}

	recallSources := make([]string, 0, len(recallSourceMap))
	for source := range recallSourceMap {
		recallSources = append(recallSources, source)
	}
	sort.Strings(recallSources)

	result := &core.RecommendResult{
		TraceId:       traceId,
		Total:         total,
		Items:         items,
		GoodsIds:      goodsIds,
		RecallSources: recallSources,
	}

	err = saveRecommendTrace(ctx, dependencies, config, internalRequest, traceId, candidates, currentPage)
	// explain 被显式请求且实例开启严格 trace 持久化时，保存失败应直接暴露给调用方。
	if err != nil && internalRequest.NeedExplain && config.Explain.StrictTracePersistence {
		return nil, err
	}
	return result, nil
}

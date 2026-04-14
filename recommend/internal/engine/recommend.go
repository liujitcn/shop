package engine

import (
	"context"
	"recommend"
	"recommend/internal/model"
	"recommend/internal/scene"
	"sort"
)

// Recommend 执行推荐主链路并返回对外结果。
func Recommend(ctx context.Context, dependencies recommend.Dependencies, request recommend.RecommendRequest) (*recommend.RecommendResult, error) {
	internalRequest := model.ResolveRequest(request)
	candidates, err := scene.Run(ctx, internalRequest, dependencies)
	if err != nil {
		return nil, err
	}

	total := int64(len(candidates))
	offset := internalRequest.Offset()
	limit := internalRequest.Limit()
	// 当前页偏移超出候选数量时，直接返回空页结果。
	if offset >= len(candidates) {
		return &recommend.RecommendResult{
			TraceId: internalRequest.Context.RequestId,
			Total:   total,
		}, nil
	}

	end := offset + limit
	// 当前页结束位置越界时，按候选集末尾截断。
	if end > len(candidates) {
		end = len(candidates)
	}

	currentPage := candidates[offset:end]
	items := make([]recommend.RecommendItem, 0, len(currentPage))
	goodsIds := make([]int64, 0, len(currentPage))
	recallSourceMap := make(map[string]struct{})

	for _, item := range currentPage {
		recallSources := item.RecallSourceList()
		for _, source := range recallSources {
			recallSourceMap[source] = struct{}{}
		}
		items = append(items, recommend.RecommendItem{
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

	return &recommend.RecommendResult{
		TraceId:       internalRequest.Context.RequestId,
		Total:         total,
		Items:         items,
		GoodsIds:      goodsIds,
		RecallSources: recallSources,
	}, nil
}

package scene

import (
	"context"
	"recommend"
	"recommend/internal/model"
	"recommend/internal/recall"
)

// runCartPipeline 执行购物车推荐流水线。
func runCartPipeline(ctx context.Context, request model.Request, dependencies recommend.Dependencies) ([]*model.Candidate, error) {
	relationRequest := request
	// 购物车存在商品时，优先使用第一个商品作为商品关联召回锚点。
	if len(request.Context.CartGoodsIds) > 0 {
		relationRequest = withGoodsId(request, request.Context.CartGoodsIds[0])
	}
	recallRequest := buildRecallRequest(relationRequest, dependencies)

	relationList, err := recall.RecallGoodsRelation(ctx, recallRequest)
	if err != nil {
		return nil, err
	}
	sessionList, err := recall.RecallSessionContext(ctx, recallRequest)
	if err != nil {
		return nil, err
	}
	sceneHotList, err := recall.RecallSceneHot(ctx, recallRequest)
	if err != nil {
		return nil, err
	}
	externalList, err := recall.RecallExternal(ctx, recallRequest)
	if err != nil {
		return nil, err
	}
	collaborativeList, err := recall.RecallCollaborative(ctx, recallRequest)
	if err != nil {
		return nil, err
	}
	latestList, err := recall.RecallLatest(ctx, recallRequest)
	if err != nil {
		return nil, err
	}

	primary := mergeCandidates(relationList, sessionList, sceneHotList, externalList, collaborativeList)
	return finalizeCandidates(ctx, request, dependencies, primary, latestList)
}

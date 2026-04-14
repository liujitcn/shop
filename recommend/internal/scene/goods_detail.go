package scene

import (
	"context"
	"recommend"
	"recommend/internal/model"
	"recommend/internal/recall"
)

// runGoodsDetailPipeline 执行商品详情推荐流水线。
func runGoodsDetailPipeline(ctx context.Context, request model.Request, dependencies recommend.Dependencies) ([]*model.Candidate, error) {
	recallRequest := buildRecallRequest(request, dependencies)
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

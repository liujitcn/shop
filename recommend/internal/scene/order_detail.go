package scene

import (
	"context"
	cachex "recommend/internal/cache"
	"recommend/internal/core"
	"recommend/internal/model"
	"recommend/internal/recall"
)

// runOrderDetailPipeline 执行订单详情推荐流水线。
func runOrderDetailPipeline(
	ctx context.Context,
	request model.Request,
	dependencies core.Dependencies,
	config core.ServiceConfig,
	poolStore *cachex.PoolStore,
	runtimeStore *cachex.RuntimeStore,
) ([]*model.Candidate, error) {
	anchorGoodsId, err := loadOrderAnchorGoodsId(ctx, dependencies, request.Context.OrderId)
	if err != nil {
		return nil, err
	}
	if anchorGoodsId > 0 {
		request = withGoodsId(request, anchorGoodsId)
	}
	recallRequest := buildRecallRequest(request, dependencies, poolStore, runtimeStore)

	var relationList []*model.Candidate
	relationList, err = recall.RecallGoodsRelation(ctx, recallRequest)
	if err != nil {
		return nil, err
	}
	var sceneHotList []*model.Candidate
	sceneHotList, err = recall.RecallSceneHot(ctx, recallRequest)
	if err != nil {
		return nil, err
	}

	var externalList []*model.Candidate
	externalList, err = recall.RecallExternal(ctx, recallRequest)
	if err != nil {
		return nil, err
	}

	var collaborativeList []*model.Candidate
	collaborativeList, err = recall.RecallCollaborative(ctx, recallRequest)
	if err != nil {
		return nil, err
	}

	var vectorList []*model.Candidate
	vectorList, err = recall.RecallVector(ctx, recallRequest, config.Vector)
	if err != nil {
		return nil, err
	}

	var latestList []*model.Candidate
	latestList, err = recall.RecallLatest(ctx, recallRequest)
	if err != nil {
		return nil, err
	}

	primary := mergeCandidates(relationList, sceneHotList, externalList, collaborativeList, vectorList)
	return finalizeCandidates(ctx, request, dependencies, config, runtimeStore, primary, latestList)
}

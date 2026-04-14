package scene

import (
	"context"
	cachex "recommend/internal/cache"
	"recommend/internal/core"
	"recommend/internal/model"
	"recommend/internal/recall"
)

// runGoodsDetailPipeline 执行商品详情推荐流水线。
func runGoodsDetailPipeline(
	ctx context.Context,
	request model.Request,
	dependencies core.Dependencies,
	config core.ServiceConfig,
	poolStore *cachex.PoolStore,
	runtimeStore *cachex.RuntimeStore,
) ([]*model.Candidate, error) {
	recallRequest := buildRecallRequest(request, dependencies, poolStore, runtimeStore)
	relationList, err := recall.RecallGoodsRelation(ctx, recallRequest)
	if err != nil {
		return nil, err
	}

	var sessionList []*model.Candidate
	sessionList, err = recall.RecallSessionContext(ctx, recallRequest)
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

	primary := mergeCandidates(relationList, sessionList, sceneHotList, externalList, collaborativeList, vectorList)
	return finalizeCandidates(ctx, request, dependencies, config, runtimeStore, primary, latestList)
}

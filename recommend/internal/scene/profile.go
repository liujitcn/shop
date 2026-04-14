package scene

import (
	"context"
	cachex "recommend/internal/cache"
	"recommend/internal/core"
	"recommend/internal/model"
	"recommend/internal/recall"
)

// runProfilePipeline 执行个人中心推荐流水线。
func runProfilePipeline(
	ctx context.Context,
	request model.Request,
	dependencies core.Dependencies,
	config core.ServiceConfig,
	poolStore *cachex.PoolStore,
	runtimeStore *cachex.RuntimeStore,
) ([]*model.Candidate, error) {
	recallRequest := buildRecallRequest(request, dependencies, poolStore, runtimeStore)
	userGoodsList, err := recall.RecallUserGoodsPreference(ctx, recallRequest)
	if err != nil {
		return nil, err
	}

	var userCategoryList []*model.Candidate
	userCategoryList, err = recall.RecallUserCategoryPreference(ctx, recallRequest)
	if err != nil {
		return nil, err
	}

	var globalHotList []*model.Candidate
	globalHotList, err = recall.RecallGlobalHot(ctx, recallRequest)
	if err != nil {
		return nil, err
	}

	var externalList []*model.Candidate
	externalList, err = recall.RecallExternal(ctx, recallRequest)
	if err != nil {
		return nil, err
	}

	var userToUserList []*model.Candidate
	userToUserList, err = recall.RecallUserToUser(ctx, recallRequest)
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

	primary := mergeCandidates(userGoodsList, userCategoryList, globalHotList, externalList, userToUserList, collaborativeList, vectorList)
	return finalizeCandidates(ctx, request, dependencies, config, runtimeStore, primary, latestList)
}

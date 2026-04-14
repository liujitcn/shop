package scene

import (
	"context"
	"recommend"
	"recommend/internal/model"
	"recommend/internal/recall"
)

// runHomePipeline 执行首页推荐流水线。
func runHomePipeline(ctx context.Context, request model.Request, dependencies recommend.Dependencies) ([]*model.Candidate, error) {
	recallRequest := buildRecallRequest(request, dependencies)
	userGoodsList, err := recall.RecallUserGoodsPreference(ctx, recallRequest)
	if err != nil {
		return nil, err
	}
	userCategoryList, err := recall.RecallUserCategoryPreference(ctx, recallRequest)
	if err != nil {
		return nil, err
	}
	sceneHotList, err := recall.RecallSceneHot(ctx, recallRequest)
	if err != nil {
		return nil, err
	}
	globalHotList, err := recall.RecallGlobalHot(ctx, recallRequest)
	if err != nil {
		return nil, err
	}
	externalList, err := recall.RecallExternal(ctx, recallRequest)
	if err != nil {
		return nil, err
	}
	userToUserList, err := recall.RecallUserToUser(ctx, recallRequest)
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

	primary := mergeCandidates(
		userGoodsList,
		userCategoryList,
		sceneHotList,
		globalHotList,
		externalList,
		userToUserList,
		collaborativeList,
	)
	return finalizeCandidates(ctx, request, dependencies, primary, latestList)
}

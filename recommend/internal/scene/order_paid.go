package scene

import (
	"context"
	"recommend"
	"recommend/internal/model"
	"recommend/internal/recall"
)

// runOrderPaidPipeline 执行支付完成推荐流水线。
func runOrderPaidPipeline(ctx context.Context, request model.Request, dependencies recommend.Dependencies) ([]*model.Candidate, error) {
	anchorGoodsId, err := loadOrderAnchorGoodsId(ctx, dependencies, request.Context.OrderId)
	if err != nil {
		return nil, err
	}
	if anchorGoodsId > 0 {
		request = withGoodsId(request, anchorGoodsId)
	}
	recallRequest := buildRecallRequest(request, dependencies)

	relationList, err := recall.RecallGoodsRelation(ctx, recallRequest)
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

	primary := mergeCandidates(relationList, sceneHotList, externalList, userToUserList, collaborativeList)
	return finalizeCandidates(ctx, request, dependencies, primary, latestList)
}

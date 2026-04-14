package recall

import (
	"context"
	"recommend/internal/model"
)

// RecallSceneHot 召回场景热销商品。
func RecallSceneHot(ctx context.Context, request Request) ([]*model.Candidate, error) {
	rows, err := request.Dependencies.Recommend.ListSceneHotGoods(ctx, request.Scene.String(), request.ReferenceTime, ResolveLimit(request.Limit))
	if err != nil {
		return nil, err
	}
	return buildWeightedGoodsCandidates(ctx, request.Dependencies.Goods, rows, RecallSourceSceneHot, func(candidate *model.Candidate, score float64) {
		candidate.Score.SceneHotScore = score
	})
}

package local

import (
	"context"

	"shop/api/gen/go/common"
)

// ExploreReceiver 表示本地探索推荐接收器。
type ExploreReceiver struct {
	recommend *Recommend
}

// NewExploreReceiver 创建本地探索推荐接收器。
func NewExploreReceiver(recommend *Recommend) *ExploreReceiver {
	return &ExploreReceiver{recommend: recommend}
}

// GetGoodsPage 查询全量商品探索推荐分页结果。
func (r *ExploreReceiver) GetGoodsPage(
	ctx context.Context,
	scene common.RecommendScene,
	requestId int64,
	excludedGoodsIds []int64,
	pageNum, pageSize int64,
) ([]int64, int64, error) {
	// 接收器未启用时，直接返回空结果。
	if r == nil || r.recommend == nil || !r.recommend.Enabled() {
		return []int64{}, 0, nil
	}
	seed := r.recommend.buildRotationSeed(scene, requestId)
	return r.recommend.ListExploreGoodsPage(ctx, excludedGoodsIds, seed, pageNum, pageSize)
}

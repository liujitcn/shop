package local

import (
	"context"

	commonv1 "shop/api/gen/go/common/v1"
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
	scene commonv1.RecommendScene,
	requestID int64,
	excludedGoodsIDs []int64,
	pageNum, pageSize int64,
) ([]int64, int64, error) {
	// 接收器未启用时，直接返回空结果。
	if r == nil || r.recommend == nil || !r.recommend.Enabled() {
		return []int64{}, 0, nil
	}
	seed := requestID*131 + int64(scene)*17
	// 轮转种子需要保持非负，避免取模后出现负序。
	if seed < 0 {
		seed = -seed
	}
	seed = seed % 1000003
	return r.recommend.ListExploreGoodsPage(ctx, excludedGoodsIDs, seed, pageNum, pageSize)
}

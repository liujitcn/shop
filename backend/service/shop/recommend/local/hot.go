package local

import "context"

// HotReceiver 表示本地热度推荐接收器。
type HotReceiver struct {
	recommend *Recommend
}

// NewHotReceiver 创建本地热度推荐接收器。
func NewHotReceiver(recommend *Recommend) *HotReceiver {
	return &HotReceiver{recommend: recommend}
}

// GetGoodsPage 查询全站热度推荐商品分页结果。
func (r *HotReceiver) GetGoodsPage(
	ctx context.Context,
	excludedGoodsIDs []int64,
	statDays int,
	scoreWeight localScoreWeight,
	pageNum, pageSize int64,
) ([]int64, int64, error) {
	// 接收器未启用时，直接返回空结果。
	if r == nil || r.recommend == nil || !r.recommend.Enabled() {
		return []int64{}, 0, nil
	}
	return r.recommend.ListRankedGoodsPage(ctx, nil, excludedGoodsIDs, statDays, scoreWeight, pageNum, pageSize)
}

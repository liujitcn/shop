package local

import "context"

// LocalHotReceiver 表示本地热度推荐接收器。
type LocalHotReceiver struct {
	recommend *Recommend
}

// NewLocalHotReceiver 创建本地热度推荐接收器。
func NewLocalHotReceiver(recommend *Recommend) *LocalHotReceiver {
	return &LocalHotReceiver{recommend: recommend}
}

// GetGoodsPage 查询全站热度推荐商品分页结果。
func (r *LocalHotReceiver) GetGoodsPage(
	ctx context.Context,
	excludedGoodsIds []int64,
	statDays int,
	scoreWeight localScoreWeight,
	pageNum, pageSize int64,
) ([]int64, int64, error) {
	// 接收器未启用时，直接返回空结果。
	if r == nil || r.recommend == nil || !r.recommend.Enabled() {
		return []int64{}, 0, nil
	}
	return r.recommend.ListRankedGoodsPage(ctx, nil, excludedGoodsIds, statDays, scoreWeight, pageNum, pageSize)
}

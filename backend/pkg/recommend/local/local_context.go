package local

import "context"

// LocalContextReceiver 表示本地上下文推荐接收器。
type LocalContextReceiver struct {
	recommend *Recommend
}

// NewLocalContextReceiver 创建本地上下文推荐接收器。
func NewLocalContextReceiver(recommend *Recommend) *LocalContextReceiver {
	return &LocalContextReceiver{recommend: recommend}
}

// GetGoodsPage 查询上下文类目推荐商品分页结果。
func (r *LocalContextReceiver) GetGoodsPage(
	ctx context.Context,
	contextGoodsIds []int64,
	statDays int,
	scoreWeight localScoreWeight,
	pageNum, pageSize int64,
) ([]int64, int64, error) {
	// 接收器未启用时，直接返回空结果。
	if r == nil || r.recommend == nil || !r.recommend.Enabled() {
		return []int64{}, 0, nil
	}
	// 没有上下文商品时，当前接收器无法构建相关候选池。
	if len(contextGoodsIds) == 0 {
		return []int64{}, 0, nil
	}

	categoryIds, err := r.recommend.ListCategoryIdsByGoodsIds(ctx, contextGoodsIds)
	if err != nil {
		return nil, 0, err
	}
	// 上下文商品未能解析出分类时，当前接收器没有可用候选集。
	if len(categoryIds) == 0 {
		return []int64{}, 0, nil
	}
	return r.recommend.ListRankedGoodsPage(ctx, categoryIds, contextGoodsIds, statDays, scoreWeight, pageNum, pageSize)
}

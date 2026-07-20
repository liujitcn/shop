package local

import "context"

// ContextReceiver 表示本地上下文推荐接收器。
type ContextReceiver struct {
	recommend *Recommend
}

// NewContextReceiver 创建本地上下文推荐接收器。
func NewContextReceiver(recommend *Recommend) *ContextReceiver {
	return &ContextReceiver{recommend: recommend}
}

// GetGoodsPage 查询上下文类目推荐商品分页结果。
func (r *ContextReceiver) GetGoodsPage(
	ctx context.Context,
	contextGoodsIDs []int64,
	statDays int,
	scoreWeight localScoreWeight,
	pageNum, pageSize int64,
) ([]int64, int64, error) {
	// 接收器未启用时，直接返回空结果。
	if r == nil || r.recommend == nil || !r.recommend.Enabled() {
		return []int64{}, 0, nil
	}
	// 没有上下文商品时，当前接收器无法构建相关候选池。
	if len(contextGoodsIDs) == 0 {
		return []int64{}, 0, nil
	}

	categoryIDs, err := r.recommend.ListCategoryIDsByGoodsIDs(ctx, contextGoodsIDs)
	if err != nil {
		return nil, 0, err
	}
	// 上下文商品未能解析出分类时，当前接收器没有可用候选集。
	if len(categoryIDs) == 0 {
		return []int64{}, 0, nil
	}
	return r.recommend.ListRankedGoodsPage(ctx, categoryIDs, contextGoodsIDs, statDays, scoreWeight, pageNum, pageSize)
}

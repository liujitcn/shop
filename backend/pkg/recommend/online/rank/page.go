package rank

import (
	app "shop/api/gen/go/app"
	recommendcore "shop/pkg/recommend/core"
	recommendDomain "shop/pkg/recommend/domain"
)

// RankedPageSnapshot 表示排序结果分页窗口的快照。
type RankedPageSnapshot struct {
	Total       int64
	Offset      int
	End         int
	PageGoods   []*app.GoodsInfo
	IsEmptyPage bool
}

// PageExplainSnapshot 表示当前页 explain 组装后的结果快照。
type PageExplainSnapshot struct {
	RecallSources    []string
	ScoreDetails     []recommendcore.ScoreDetail
	ReturnedGoodsIds []int64
}

// BuildRankedPageSnapshot 构建排序结果分页窗口快照。
func BuildRankedPageSnapshot(request *recommendDomain.GoodsRequest, rankedGoods []*app.GoodsInfo) RankedPageSnapshot {
	snapshot := RankedPageSnapshot{
		Total:       int64(len(rankedGoods)),
		PageGoods:   []*app.GoodsInfo{},
		IsEmptyPage: true,
	}
	pageNum := int64(0)
	pageSize := int64(0)
	// 当前存在请求对象时，继续复用请求中的分页参数。
	if request != nil {
		pageNum = request.PageNum
		pageSize = request.PageSize
	}
	// 分页参数非法时，直接返回空页快照。
	if pageNum <= 0 || pageSize <= 0 {
		return snapshot
	}
	snapshot.Offset = int((pageNum - 1) * pageSize)
	// 分页偏移超出候选集范围时，直接返回空页快照。
	if snapshot.Offset >= len(rankedGoods) {
		return snapshot
	}
	snapshot.End = snapshot.Offset + int(pageSize)
	// 分页结束位置超过候选集时，按末尾截断。
	if snapshot.End > len(rankedGoods) {
		snapshot.End = len(rankedGoods)
	}
	snapshot.PageGoods = rankedGoods[snapshot.Offset:snapshot.End]
	snapshot.IsEmptyPage = len(snapshot.PageGoods) == 0
	return snapshot
}

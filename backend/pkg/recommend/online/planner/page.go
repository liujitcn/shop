package planner

import app "shop/api/gen/go/app"

// RankedPageSnapshot 表示排序结果分页窗口的快照。
type RankedPageSnapshot struct {
	Total       int64
	Offset      int
	End         int
	PageGoods   []*app.GoodsInfo
	IsEmptyPage bool
}

// GoodsPoolPageSnapshot 表示候选池桥接分页结果的快照。
type GoodsPoolPageSnapshot struct {
	Total     int64
	GoodsList []*app.GoodsInfo
	GoodsIds  []int64
}

// BuildEmptyGoodsPoolPageResponse 构建候选池桥接查询的空分页结果。
func BuildEmptyGoodsPoolPageResponse() *app.PageGoodsInfoResponse {
	return &app.PageGoodsInfoResponse{
		List:  []*app.GoodsInfo{},
		Total: 0,
	}
}

// BuildGoodsPoolPageSnapshot 构建候选池桥接分页结果快照。
func BuildGoodsPoolPageSnapshot(pageResp *app.PageGoodsInfoResponse) GoodsPoolPageSnapshot {
	snapshot := GoodsPoolPageSnapshot{
		GoodsList: []*app.GoodsInfo{},
		GoodsIds:  []int64{},
	}
	// 分页响应为空时，直接返回空快照。
	if pageResp == nil {
		return snapshot
	}
	snapshot.Total = int64(pageResp.Total)
	snapshot.GoodsList = pageResp.List
	snapshot.GoodsIds = ListGoodsIds(pageResp.List)
	return snapshot
}

// BuildRankedPageSnapshot 构建排序结果分页窗口快照。
func (p *RequestPlan) BuildRankedPageSnapshot(rankedGoods []*app.GoodsInfo) RankedPageSnapshot {
	snapshot := RankedPageSnapshot{
		Total:       int64(len(rankedGoods)),
		PageGoods:   []*app.GoodsInfo{},
		IsEmptyPage: true,
	}
	pageNum := int64(0)
	pageSize := int64(0)
	// 当前存在请求计划时，继续复用请求中的分页参数。
	if p != nil {
		pageNum = p.Request.PageNum
		pageSize = p.Request.PageSize
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

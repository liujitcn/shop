package planner

import app "shop/api/gen/go/app"

// SignalSnapshot 表示排序信号加载前的候选结果快照。
type SignalSnapshot struct {
	GoodsList   []*app.GoodsInfo
	GoodsIds    []int64
	CategoryIds []int64
}

// BuildAnonymousSignalSnapshot 构建匿名态排序信号加载前的候选结果快照。
func (p *RequestPlan) BuildAnonymousSignalSnapshot(goodsList []*app.GoodsInfo) SignalSnapshot {
	snapshot := SignalSnapshot{
		GoodsList:   make([]*app.GoodsInfo, 0, len(goodsList)),
		GoodsIds:    make([]int64, 0, len(goodsList)),
		CategoryIds: []int64{},
	}
	for _, item := range goodsList {
		// 非法商品不参与匿名候选排序。
		if item == nil || item.Id <= 0 {
			continue
		}
		// 商品详情场景不返回当前详情商品本身。
		if p != nil && p.IsGoodsDetail() && p.Request.GoodsId > 0 && item.Id == p.Request.GoodsId {
			continue
		}
		snapshot.GoodsList = append(snapshot.GoodsList, item)
		snapshot.GoodsIds = append(snapshot.GoodsIds, item.Id)
	}
	return snapshot
}

// BuildPersonalizedSignalSnapshot 构建登录态排序信号加载前的候选结果快照。
func (p *RequestPlan) BuildPersonalizedSignalSnapshot(goodsList []*app.GoodsInfo) SignalSnapshot {
	snapshot := SignalSnapshot{
		GoodsList:   make([]*app.GoodsInfo, 0, len(goodsList)),
		GoodsIds:    make([]int64, 0, len(goodsList)),
		CategoryIds: make([]int64, 0, len(goodsList)),
	}
	for _, item := range goodsList {
		// 非法商品不参与候选信号计算。
		if item == nil || item.Id <= 0 {
			continue
		}
		snapshot.GoodsList = append(snapshot.GoodsList, item)
		snapshot.GoodsIds = append(snapshot.GoodsIds, item.Id)
		snapshot.CategoryIds = append(snapshot.CategoryIds, item.CategoryId)
	}
	return snapshot
}

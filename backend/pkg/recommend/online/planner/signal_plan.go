package planner

import "shop/api/gen/go/common"

// SignalLoadPlan 表示排序信号加载前的纯参数计划。
type SignalLoadPlan struct {
	Scene                  int32
	CandidateGoodsIds      []int64
	CandidateCategoryIds   []int64
	RelationSourceGoodsIds []int64
}

// BuildAnonymousSignalLoadPlan 构建匿名态排序信号加载计划。
func (p *RequestPlan) BuildAnonymousSignalLoadPlan(snapshot SignalSnapshot) SignalLoadPlan {
	plan := SignalLoadPlan{
		CandidateGoodsIds:    snapshot.GoodsIds,
		CandidateCategoryIds: []int64{},
	}
	// 当前存在请求计划时，继续复用请求中的场景信息。
	if p != nil {
		plan.Scene = int32(p.Request.Scene)
	}
	// 商品详情场景存在源商品时，补充匿名关联分数加载源。
	if p != nil && p.Request.Scene == common.RecommendScene_GOODS_DETAIL && p.Request.GoodsId > 0 {
		plan.RelationSourceGoodsIds = []int64{p.Request.GoodsId}
	}
	return plan
}

// BuildPersonalizedSignalLoadPlan 构建登录态排序信号加载计划。
func (p *RequestPlan) BuildPersonalizedSignalLoadPlan(snapshot SignalSnapshot) SignalLoadPlan {
	plan := SignalLoadPlan{
		CandidateGoodsIds:    snapshot.GoodsIds,
		CandidateCategoryIds: snapshot.CategoryIds,
	}
	// 当前存在请求计划时，继续复用请求中的场景和关系分来源。
	if p != nil {
		plan.Scene = int32(p.Request.Scene)
		plan.RelationSourceGoodsIds = p.PriorityGoodsIds
	}
	return plan
}

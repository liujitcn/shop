package planner

import recommendCandidate "shop/pkg/recommend/candidate"

// ApplyCartScene 将购物车场景的前置状态写入请求计划。
func (p *RequestPlan) ApplyCartScene(priorityGoodsIds []int64, categoryIds []int64) {
	// 计划对象为空时，无法继续写入购物车场景状态。
	if p == nil {
		return
	}
	p.AddPriorityGoodsIds(priorityGoodsIds)
	p.AddCategoryIds(categoryIds)
	// 购物车命中了业务上下文时，记录购物车召回来源。
	if len(priorityGoodsIds) > 0 {
		p.AddRecallSources("cart")
	}
}

// ApplyOrderScene 将订单场景的前置状态写入请求计划。
func (p *RequestPlan) ApplyOrderScene(priorityGoodsIds []int64, categoryIds []int64) {
	// 计划对象为空时，无法继续写入订单场景状态。
	if p == nil {
		return
	}
	p.AddPriorityGoodsIds(priorityGoodsIds)
	p.AddCategoryIds(categoryIds)
	// 订单命中了业务上下文时，记录订单召回来源。
	if len(priorityGoodsIds) > 0 {
		p.AddRecallSources("order")
	}
}

// ApplyGoodsDetailScene 将商品详情场景的前置状态写入请求计划。
func (p *RequestPlan) ApplyGoodsDetailScene(priorityGoodsIds []int64, categoryIds []int64) {
	// 计划对象为空时，无法继续写入商品详情场景状态。
	if p == nil {
		return
	}
	p.AddPriorityGoodsIds(priorityGoodsIds)
	p.AddCategoryIds(categoryIds)
	// 商品详情页命中了业务上下文时，记录详情场景召回来源。
	if len(priorityGoodsIds) > 0 || len(categoryIds) > 0 {
		p.AddRecallSources("goods_detail")
	}
}

// ApplyJoinRecall 将允许入池的灰度召回结果统一并入优先候选集合。
func (p *RequestPlan) ApplyJoinRecall() {
	// 计划对象为空时，无法继续合并灰度召回结果。
	if p == nil {
		return
	}
	// 相似用户观测结果允许入池时，补到优先候选集合。
	if len(p.SimilarUserJoinGoodsIds) > 0 {
		p.AddPriorityGoodsIds(p.SimilarUserJoinGoodsIds)
		p.AddRecallSources(recommendCandidate.RecallSourceSimilarUser)
	}
	// 内容相似召回允许入池时，补到优先候选集合。
	if len(p.ContentBasedJoinGoodsIds) > 0 {
		p.AddPriorityGoodsIds(p.ContentBasedJoinGoodsIds)
		p.AddRecallSources(recommendCandidate.RecallSourceContentBased)
	}
	// 协同过滤召回允许入池时，补到优先候选集合。
	if len(p.CollaborativeFilteringGoodsIds) > 0 {
		p.AddPriorityGoodsIds(p.CollaborativeFilteringGoodsIds)
		p.AddRecallSources(recommendCandidate.RecallSourceCF)
	}
}

// ApplyProfileScene 将用户画像类目补足状态写入请求计划。
func (p *RequestPlan) ApplyProfileScene(categoryIds []int64) {
	// 计划对象为空时，无法继续写入画像补足状态。
	if p == nil {
		return
	}
	p.AddCategoryIds(categoryIds)
	// 画像命中了补足类目时，记录画像召回来源。
	if len(categoryIds) > 0 {
		p.AddRecallSources("profile")
	}
}

// EnsureFallbackLatest 在没有任何召回来源时补最新榜兜底标记。
func (p *RequestPlan) EnsureFallbackLatest() {
	// 计划对象为空时，无法继续补充兜底来源。
	if p == nil {
		return
	}
	// 当前已经命中过召回来源时，不需要再补 latest 兜底标记。
	if len(p.RecallSources) > 0 {
		return
	}
	p.AddRecallSources("latest")
}

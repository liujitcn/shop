package planner

import recommendcore "shop/pkg/recommend/core"

// SetCategoryCandidateGoodsIds 设置类目补足候选商品编号。
func (p *RequestPlan) SetCategoryCandidateGoodsIds(goodsIds []int64) {
	// 计划对象为空时，无法继续写入类目补足候选。
	if p == nil {
		return
	}
	p.CategoryCandidateGoodsIds = recommendcore.DedupeInt64s(goodsIds)
}

// SetLatestCandidateGoodsIds 设置 latest 兜底候选商品编号。
func (p *RequestPlan) SetLatestCandidateGoodsIds(goodsIds []int64) {
	// 计划对象为空时，无法继续写入 latest 候选。
	if p == nil {
		return
	}
	p.LatestCandidateGoodsIds = recommendcore.DedupeInt64s(goodsIds)
}

// BuildAnonymousMergedSceneGoodsIds 构建匿名态场景热度与类目补足合并后的候选集合。
func (p *RequestPlan) BuildAnonymousMergedSceneGoodsIds(sceneHotGoodsIds []int64) []int64 {
	// 计划对象为空时，只返回场景热度去重结果。
	if p == nil {
		return recommendcore.DedupeInt64s(sceneHotGoodsIds)
	}
	return recommendcore.DedupeInt64s(append(sceneHotGoodsIds, p.CategoryCandidateGoodsIds...))
}

// BuildAnonymousCandidateGoodsIds 构建匿名态最终进入排序前的候选集合。
func (p *RequestPlan) BuildAnonymousCandidateGoodsIds(goodsIds []int64) []int64 {
	// 计划对象为空时，只返回调用方传入的候选集合。
	if p == nil {
		return recommendcore.DedupeInt64s(goodsIds)
	}
	return recommendcore.DedupeInt64s(append(p.PriorityGoodsIds, goodsIds...))
}

// BuildLatestExcludeGoodsIds 构建 latest 兜底阶段需要排除的商品集合。
func (p *RequestPlan) BuildLatestExcludeGoodsIds() []int64 {
	// 计划对象为空时，统一返回空切片。
	if p == nil {
		return []int64{}
	}
	return recommendcore.DedupeInt64s(append(p.ExcludeGoodsIds(), p.CategoryCandidateGoodsIds...))
}

// BuildPersonalizedCandidateGoodsIds 构建登录态最终进入排序前的候选集合。
func (p *RequestPlan) BuildPersonalizedCandidateGoodsIds() []int64 {
	// 计划对象为空时，统一返回空切片。
	if p == nil {
		return []int64{}
	}
	return recommendcore.DedupeInt64s(append(append(p.PriorityGoodsIds, p.CategoryCandidateGoodsIds...), p.LatestCandidateGoodsIds...))
}

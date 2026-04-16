package planner

import recommendcore "shop/pkg/recommend/core"

// GoodsPoolQuery 表示候选池桥接查询前的纯参数计划。
type GoodsPoolQuery struct {
	Enabled         bool
	Limit           int64
	CategoryIds     []int64
	ExcludeGoodsIds []int64
}

// IsEnabled 判断候选池查询计划当前是否可执行。
func (q GoodsPoolQuery) IsEnabled() bool {
	return q.Enabled && q.Limit > 0
}

// SupportsCategoryQuery 判断候选池查询计划是否具备类目补足查询条件。
func (q GoodsPoolQuery) SupportsCategoryQuery() bool {
	return q.IsEnabled() && len(q.CategoryIds) > 0
}

// BuildCategoryCandidateQuery 构建类目补足阶段的查询参数计划。
func (p *RequestPlan) BuildCategoryCandidateQuery() GoodsPoolQuery {
	// 计划对象为空时，统一返回禁用查询计划。
	if p == nil {
		return GoodsPoolQuery{}
	}
	return GoodsPoolQuery{
		Enabled:         len(p.CategoryIds) > 0 && p.CandidateLimit > 0,
		Limit:           p.CandidateLimit,
		CategoryIds:     recommendcore.DedupeInt64s(p.CategoryIds),
		ExcludeGoodsIds: p.ExcludeGoodsIds(),
	}
}

// BuildLatestCandidateQuery 构建 latest 兜底阶段的查询参数计划。
func (p *RequestPlan) BuildLatestCandidateQuery() GoodsPoolQuery {
	// 计划对象为空时，统一返回禁用查询计划。
	if p == nil {
		return GoodsPoolQuery{}
	}
	return GoodsPoolQuery{
		Enabled:         p.CandidateLimit > 0,
		Limit:           p.CandidateLimit,
		ExcludeGoodsIds: p.BuildLatestExcludeGoodsIds(),
	}
}

// BuildAnonymousLatestFallbackQuery 构建匿名态 latest fallback 的查询参数计划。
func (p *RequestPlan) BuildAnonymousLatestFallbackQuery() GoodsPoolQuery {
	excludeGoodsIds := make([]int64, 0, 1)
	// 商品详情场景回退到 latest 时，同样排除当前详情商品。
	if p != nil && p.IsGoodsDetail() && p.Request.GoodsId > 0 {
		excludeGoodsIds = append(excludeGoodsIds, p.Request.GoodsId)
	}
	limit := int64(0)
	// 计划对象存在时，继续复用当前候选池上限。
	if p != nil {
		limit = p.CandidateLimit
	}
	return GoodsPoolQuery{
		Enabled:         limit > 0,
		Limit:           limit,
		ExcludeGoodsIds: recommendcore.DedupeInt64s(excludeGoodsIds),
	}
}

// ShouldFallbackToAnonymousLatest 判断匿名态是否需要回退到 latest。
func (p *RequestPlan) ShouldFallbackToAnonymousLatest(candidateGoodsIds []int64) bool {
	// 计划对象为空时，只要当前候选为空就允许回退。
	if p == nil {
		return len(candidateGoodsIds) == 0
	}
	return len(candidateGoodsIds) == 0 && len(p.PriorityGoodsIds) == 0
}

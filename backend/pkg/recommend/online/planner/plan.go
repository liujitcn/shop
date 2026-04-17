package planner

import (
	"shop/api/gen/go/common"
	recommendCandidate "shop/pkg/recommend/candidate"
	recommendcore "shop/pkg/recommend/core"
	recommendDomain "shop/pkg/recommend/domain"
	recommendOnlineRecall "shop/pkg/recommend/online/recall"
)

// RequestPlan 表示一次在线推荐请求在候选编排前的计划对象。
type RequestPlan struct {
	Request                        recommendDomain.GoodsRequest
	CandidateLimit                 int64
	PriorityGoodsIds               []int64
	CategoryIds                    []int64
	CategoryCandidateGoodsIds      []int64
	LatestCandidateGoodsIds        []int64
	RecallSources                  []string
	CacheHitSources                []string
	CacheReadContext               map[string]any
	JoinRecallGoodsIds             map[string][]int64
	SimilarUserIds                 []int64
	SimilarUserObservedGoodsIds    []int64
	SimilarUserJoinGoodsIds        []int64
	ContentBasedJoinGoodsIds       []int64
	CollaborativeFilteringGoodsIds []int64
}

// NewAnonymousRequestPlan 创建匿名态请求计划。
func NewAnonymousRequestPlan(request *recommendDomain.GoodsRequest, probeContext map[string]any) *RequestPlan {
	plan := newRequestPlan(request)
	plan.ContentBasedJoinGoodsIds = recommendOnlineRecall.ListContentBasedJoinCandidateGoodsIds(probeContext)
	// 匿名态当前只灰度接入内容相似召回。
	if len(plan.ContentBasedJoinGoodsIds) > 0 {
		plan.JoinRecallGoodsIds[recommendCandidate.RecallSourceContentBased] = recommendcore.DedupeInt64s(plan.ContentBasedJoinGoodsIds)
	}
	return plan
}

// NewPersonalizedRequestPlan 创建登录态请求计划。
func NewPersonalizedRequestPlan(request *recommendDomain.GoodsRequest, probeContext map[string]any) *RequestPlan {
	plan := newRequestPlan(request)
	plan.SimilarUserIds = recommendOnlineRecall.ListSimilarUserProbeUserIds(probeContext)
	plan.ContentBasedJoinGoodsIds = recommendOnlineRecall.ListContentBasedJoinCandidateGoodsIds(probeContext)
	plan.CollaborativeFilteringGoodsIds = recommendOnlineRecall.ListCollaborativeFilteringJoinCandidateGoodsIds(probeContext)
	// 登录态详情页当前灰度接入内容相似与协同过滤两类召回。
	if len(plan.ContentBasedJoinGoodsIds) > 0 {
		plan.JoinRecallGoodsIds[recommendCandidate.RecallSourceContentBased] = recommendcore.DedupeInt64s(plan.ContentBasedJoinGoodsIds)
	}
	// 登录态详情页当前灰度接入内容相似与协同过滤两类召回。
	if len(plan.CollaborativeFilteringGoodsIds) > 0 {
		plan.JoinRecallGoodsIds[recommendCandidate.RecallSourceCF] = recommendcore.DedupeInt64s(plan.CollaborativeFilteringGoodsIds)
	}
	return plan
}

// AddPriorityGoodsIds 追加强业务上下文优先召回的商品编号。
func (p *RequestPlan) AddPriorityGoodsIds(goodsIds []int64) {
	// 当前没有可追加的商品编号时，不继续扩容切片。
	if p == nil || len(goodsIds) == 0 {
		return
	}
	p.PriorityGoodsIds = append(p.PriorityGoodsIds, goodsIds...)
}

// AddCategoryIds 追加类目补足候选所需的类目编号。
func (p *RequestPlan) AddCategoryIds(categoryIds []int64) {
	// 当前没有可追加的类目编号时，不继续扩容切片。
	if p == nil || len(categoryIds) == 0 {
		return
	}
	p.CategoryIds = append(p.CategoryIds, categoryIds...)
}

// AddRecallSources 追加当前请求命中的召回来源。
func (p *RequestPlan) AddRecallSources(sources ...string) {
	// 当前没有可追加的召回来源时，不继续扩容切片。
	if p == nil || len(sources) == 0 {
		return
	}
	p.RecallSources = append(p.RecallSources, sources...)
}

// AddCacheHitSource 追加一次缓存命中来源。
func (p *RequestPlan) AddCacheHitSource(source string) {
	// 缓存命中来源为空时，不写入计划对象。
	if p == nil || source == "" {
		return
	}
	p.CacheHitSources = append(p.CacheHitSources, source)
}

// MergeCacheReadContext 合并一次缓存读取调试上下文。
func (p *RequestPlan) MergeCacheReadContext(cacheReadContext map[string]any) {
	// 当前没有缓存读取上下文时，不继续合并。
	if p == nil || len(cacheReadContext) == 0 {
		return
	}
	// 首次写入缓存读取上下文时，先初始化目标映射。
	if p.CacheReadContext == nil {
		p.CacheReadContext = make(map[string]any, len(cacheReadContext))
	}
	for key, value := range cacheReadContext {
		p.CacheReadContext[key] = value
	}
}

// ApplySimilarUserObservation 设置相似用户观测结果，并按配置决定是否并入候选池。
func (p *RequestPlan) ApplySimilarUserObservation(goodsIds []int64, joinCandidate bool) {
	// 计划对象为空时，无法继续写入相似用户观测结果。
	if p == nil {
		return
	}
	observedGoodsIds := recommendcore.DedupeInt64s(goodsIds)
	p.SimilarUserObservedGoodsIds = observedGoodsIds
	// 当前观测结果不允许入池时，仅保留观测上下文。
	if !joinCandidate || len(observedGoodsIds) == 0 {
		p.SimilarUserJoinGoodsIds = []int64{}
		delete(p.JoinRecallGoodsIds, recommendCandidate.RecallSourceSimilarUser)
		return
	}
	p.SimilarUserJoinGoodsIds = observedGoodsIds
	p.JoinRecallGoodsIds[recommendCandidate.RecallSourceSimilarUser] = observedGoodsIds
}

// NormalizeState 统一去重当前计划对象中的候选前置状态。
func (p *RequestPlan) NormalizeState() {
	// 计划对象为空时，无需继续归一化。
	if p == nil {
		return
	}
	p.PriorityGoodsIds = recommendcore.DedupeInt64s(p.PriorityGoodsIds)
	p.CategoryIds = recommendcore.DedupeInt64s(p.CategoryIds)
	p.CategoryCandidateGoodsIds = recommendcore.DedupeInt64s(p.CategoryCandidateGoodsIds)
	p.LatestCandidateGoodsIds = recommendcore.DedupeInt64s(p.LatestCandidateGoodsIds)
	p.RecallSources = recommendcore.DedupeStrings(p.RecallSources)
	p.CacheHitSources = recommendcore.DedupeStrings(p.CacheHitSources)
	p.SimilarUserIds = recommendcore.DedupeInt64s(p.SimilarUserIds)
	p.SimilarUserObservedGoodsIds = recommendcore.DedupeInt64s(p.SimilarUserObservedGoodsIds)
	p.SimilarUserJoinGoodsIds = recommendcore.DedupeInt64s(p.SimilarUserJoinGoodsIds)
	p.ContentBasedJoinGoodsIds = recommendcore.DedupeInt64s(p.ContentBasedJoinGoodsIds)
	p.CollaborativeFilteringGoodsIds = recommendcore.DedupeInt64s(p.CollaborativeFilteringGoodsIds)
	p.JoinRecallGoodsIds = normalizeJoinRecallGoodsIds(p.JoinRecallGoodsIds)
}

// ExcludeGoodsIds 返回当前请求在类目补足和 latest 兜底前需要排除的商品编号。
func (p *RequestPlan) ExcludeGoodsIds() []int64 {
	// 计划对象为空时，统一返回空切片，避免调用方继续判空。
	if p == nil {
		return []int64{}
	}
	result := recommendcore.DedupeInt64s(p.PriorityGoodsIds)
	// 商品详情场景需要排除当前详情商品，避免把自己推荐给自己。
	if p.IsGoodsDetail() && p.Request.GoodsId > 0 {
		result = recommendcore.DedupeInt64s(append(result, p.Request.GoodsId))
	}
	return result
}

// ShouldFallbackToAnonymousLatest 判断匿名态是否需要回退到 latest。
func (p *RequestPlan) ShouldFallbackToAnonymousLatest(candidateGoodsIds []int64) bool {
	// 计划对象为空时，只要当前候选为空就允许回退。
	if p == nil {
		return len(candidateGoodsIds) == 0
	}
	return len(candidateGoodsIds) == 0 && len(p.PriorityGoodsIds) == 0
}

// BuildSourceContext 构建当前请求的基础来源上下文。
func (p *RequestPlan) BuildSourceContext(base map[string]any, probeContext map[string]any) map[string]any {
	// 调用方没有提供基础上下文时，先初始化一个空映射。
	if base == nil {
		base = make(map[string]any, 2)
	}
	base["cacheHitSources"] = recommendcore.DedupeStrings(p.CacheHitSources)
	base = recommendOnlineRecall.AppendProbeContext(base, probeContext)
	// 当前存在缓存读取上下文时，再写入调试字段，避免产生空字段。
	if len(p.CacheReadContext) > 0 {
		base["cacheReadContext"] = p.CacheReadContext
	}
	return base
}

// IsGoodsDetail 判断当前请求是否为商品详情场景。
func (p *RequestPlan) IsGoodsDetail() bool {
	// 计划对象为空时，不按商品详情场景处理。
	if p == nil {
		return false
	}
	return p.Request.Scene == common.RecommendScene_GOODS_DETAIL
}

// newRequestPlan 创建基础请求计划对象。
func newRequestPlan(request *recommendDomain.GoodsRequest) *RequestPlan {
	normalizedRequest := recommendDomain.GoodsRequest{}
	// 调用方传入了请求对象时，直接复制当前请求快照。
	if request != nil {
		normalizedRequest = *request
	}
	return &RequestPlan{
		Request:            normalizedRequest,
		CandidateLimit:     recommendCandidate.ResolveCandidateLimit(normalizedRequest.PageNum, normalizedRequest.PageSize),
		CacheReadContext:   make(map[string]any, 2),
		JoinRecallGoodsIds: make(map[string][]int64, 3),
	}
}

// normalizeJoinRecallGoodsIds 对灰度召回商品编号映射做稳定去重。
func normalizeJoinRecallGoodsIds(joinRecallGoodsIds map[string][]int64) map[string][]int64 {
	// 当前没有灰度召回商品映射时，统一返回空映射。
	if len(joinRecallGoodsIds) == 0 {
		return map[string][]int64{}
	}
	result := make(map[string][]int64, len(joinRecallGoodsIds))
	for source, goodsIds := range joinRecallGoodsIds {
		dedupedGoodsIds := recommendcore.DedupeInt64s(goodsIds)
		// 去重后为空的来源不再保留，避免写入噪音字段。
		if len(dedupedGoodsIds) == 0 {
			continue
		}
		result[source] = dedupedGoodsIds
	}
	return result
}

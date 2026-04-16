package planner

import (
	recommendcore "shop/pkg/recommend/core"
	recommendOnlineRank "shop/pkg/recommend/online/rank"
)

// ResultSnapshot 表示候选编排完成后需要写入来源上下文的结果快照。
type ResultSnapshot struct {
	CandidateLimit             int64
	SceneHotGoodsIds           []int64
	CandidateGoodsIds          []int64
	AnonymousCandidateGoodsIds []int64
	ReturnedScoreDetails       any
}

// LatestFallbackPayload 表示 latest fallback 返回阶段的上下文负载。
type LatestFallbackPayload struct {
	RecallSources []string
	SourceContext map[string]any
}

// OnlineResultPayload 表示在线推荐结果阶段的返回负载。
type OnlineResultPayload struct {
	RecallSources []string
	SourceContext map[string]any
}

// BuildAnonymousLatestResultSourceContext 构建匿名态 latest fallback 的来源上下文。
func (p *RequestPlan) BuildAnonymousLatestResultSourceContext(sceneInput SceneInput, sceneHotGoodsIds []int64, probeContext map[string]any) map[string]any {
	return p.BuildAnonymousResultSourceContext(sceneInput, ResultSnapshot{
		CandidateLimit:   p.CandidateLimit,
		SceneHotGoodsIds: sceneHotGoodsIds,
	}, probeContext)
}

// BuildAnonymousLatestFallbackPayload 构建匿名态 latest fallback 的返回负载。
func (p *RequestPlan) BuildAnonymousLatestFallbackPayload(sceneInput SceneInput, sceneHotGoodsIds []int64, probeContext map[string]any) LatestFallbackPayload {
	return LatestFallbackPayload{
		RecallSources: []string{"latest"},
		SourceContext: p.BuildAnonymousLatestResultSourceContext(sceneInput, sceneHotGoodsIds, probeContext),
	}
}

// BuildAnonymousEmptyOnlinePayload 构建匿名态空页返回负载。
func (p *RequestPlan) BuildAnonymousEmptyOnlinePayload(sceneInput SceneInput, sceneHotGoodsIds []int64, snapshotCandidateGoodsIds []int64, debugCandidateGoodsIds []int64, probeContext map[string]any) OnlineResultPayload {
	return OnlineResultPayload{
		RecallSources: p.RecallSources,
		SourceContext: p.BuildAnonymousEmptyOnlineResultContext(sceneInput, sceneHotGoodsIds, snapshotCandidateGoodsIds, debugCandidateGoodsIds, probeContext),
	}
}

// BuildAnonymousPageOnlinePayload 构建匿名态正常页返回负载。
func (p *RequestPlan) BuildAnonymousPageOnlinePayload(sceneInput SceneInput, sceneHotGoodsIds []int64, anonymousCandidateGoodsIds []int64, candidateGoodsIds []int64, explainSnapshot recommendOnlineRank.PageExplainSnapshot, probeContext map[string]any) OnlineResultPayload {
	return OnlineResultPayload{
		RecallSources: explainSnapshot.RecallSources,
		SourceContext: p.BuildAnonymousPageOnlineResultContext(sceneInput, sceneHotGoodsIds, anonymousCandidateGoodsIds, candidateGoodsIds, explainSnapshot, probeContext),
	}
}

// BuildAnonymousEmptyOnlineResultContext 构建匿名态空页结果的完整来源上下文。
func (p *RequestPlan) BuildAnonymousEmptyOnlineResultContext(sceneInput SceneInput, sceneHotGoodsIds []int64, snapshotCandidateGoodsIds []int64, debugCandidateGoodsIds []int64, probeContext map[string]any) map[string]any {
	return p.BuildAnonymousOnlineResultContext(sceneInput, ResultSnapshot{
		CandidateLimit:       p.CandidateLimit,
		SceneHotGoodsIds:     sceneHotGoodsIds,
		CandidateGoodsIds:    snapshotCandidateGoodsIds,
		ReturnedScoreDetails: []recommendcore.ScoreDetail{},
	}, debugCandidateGoodsIds, []int64{}, probeContext)
}

// BuildAnonymousPageOnlineResultContext 构建匿名态分页结果的完整来源上下文。
func (p *RequestPlan) BuildAnonymousPageOnlineResultContext(sceneInput SceneInput, sceneHotGoodsIds []int64, anonymousCandidateGoodsIds []int64, candidateGoodsIds []int64, explainSnapshot recommendOnlineRank.PageExplainSnapshot, probeContext map[string]any) map[string]any {
	return p.BuildAnonymousOnlineResultContext(sceneInput, ResultSnapshot{
		CandidateLimit:             p.CandidateLimit,
		SceneHotGoodsIds:           sceneHotGoodsIds,
		AnonymousCandidateGoodsIds: anonymousCandidateGoodsIds,
		ReturnedScoreDetails:       explainSnapshot.ScoreDetails,
	}, candidateGoodsIds, explainSnapshot.ReturnedGoodsIds, probeContext)
}

// BuildPersonalizedEmptyOnlineResultContext 构建登录态空页结果的完整来源上下文。
func (p *RequestPlan) BuildPersonalizedEmptyOnlineResultContext(sceneInput SceneInput, candidateGoodsIds []int64, probeContext map[string]any) map[string]any {
	return p.BuildPersonalizedOnlineResultContext(sceneInput, ResultSnapshot{}, candidateGoodsIds, []int64{}, probeContext)
}

// BuildPersonalizedEmptyOnlinePayload 构建登录态空页返回负载。
func (p *RequestPlan) BuildPersonalizedEmptyOnlinePayload(sceneInput SceneInput, candidateGoodsIds []int64, probeContext map[string]any) OnlineResultPayload {
	return OnlineResultPayload{
		RecallSources: []string{},
		SourceContext: p.BuildPersonalizedEmptyOnlineResultContext(sceneInput, candidateGoodsIds, probeContext),
	}
}

// BuildPersonalizedPageOnlineResultContext 构建登录态分页结果的完整来源上下文。
func (p *RequestPlan) BuildPersonalizedPageOnlineResultContext(sceneInput SceneInput, candidateGoodsIds []int64, explainSnapshot recommendOnlineRank.PageExplainSnapshot, probeContext map[string]any) map[string]any {
	return p.BuildPersonalizedOnlineResultContext(sceneInput, ResultSnapshot{
		CandidateLimit:       p.CandidateLimit,
		ReturnedScoreDetails: explainSnapshot.ScoreDetails,
	}, candidateGoodsIds, explainSnapshot.ReturnedGoodsIds, probeContext)
}

// BuildPersonalizedPageOnlinePayload 构建登录态正常页返回负载。
func (p *RequestPlan) BuildPersonalizedPageOnlinePayload(sceneInput SceneInput, candidateGoodsIds []int64, explainSnapshot recommendOnlineRank.PageExplainSnapshot, probeContext map[string]any) OnlineResultPayload {
	return OnlineResultPayload{
		RecallSources: explainSnapshot.RecallSources,
		SourceContext: p.BuildPersonalizedPageOnlineResultContext(sceneInput, candidateGoodsIds, explainSnapshot, probeContext),
	}
}

// BuildAnonymousResultSourceContext 构建匿名态结果来源上下文。
func (p *RequestPlan) BuildAnonymousResultSourceContext(sceneInput SceneInput, snapshot ResultSnapshot, probeContext map[string]any) map[string]any {
	fields := make(map[string]any, 6)
	// 当前存在候选池上限时，再写入结果快照。
	if snapshot.CandidateLimit > 0 {
		fields["candidateLimit"] = snapshot.CandidateLimit
	}
	// 当前存在强召回商品时，再写入结果快照。
	if len(p.PriorityGoodsIds) > 0 {
		fields["priorityGoodsIds"] = p.PriorityGoodsIds
	}
	// 当前存在类目补足时，再写入结果快照。
	if len(p.CategoryIds) > 0 {
		fields["categoryIds"] = p.CategoryIds
	}
	// explain 明细由调用方按需提供，空值时不写入上下文。
	if snapshot.ReturnedScoreDetails != nil {
		fields["returnedScoreDetails"] = snapshot.ReturnedScoreDetails
	}
	// 匿名态场景热度结果存在时，再补到来源上下文。
	if len(snapshot.SceneHotGoodsIds) > 0 {
		fields["sceneHotGoodsIds"] = snapshot.SceneHotGoodsIds
	}
	// 匿名态需要区分排序前的候选池与匿名补足后的候选池。
	if len(snapshot.CandidateGoodsIds) > 0 {
		fields["candidateGoodsIds"] = snapshot.CandidateGoodsIds
	}
	if len(snapshot.AnonymousCandidateGoodsIds) > 0 {
		fields["anonymousCandidateGoodsIds"] = snapshot.AnonymousCandidateGoodsIds
	}
	return p.BuildSceneResultSourceContext(sceneInput, fields, probeContext)
}

// BuildPersonalizedResultSourceContext 构建登录态结果来源上下文。
func (p *RequestPlan) BuildPersonalizedResultSourceContext(sceneInput SceneInput, snapshot ResultSnapshot, probeContext map[string]any) map[string]any {
	fields := make(map[string]any, 4)
	// 当前存在候选池上限时，再写入结果快照。
	if snapshot.CandidateLimit > 0 {
		fields["candidateLimit"] = snapshot.CandidateLimit
	}
	// 当前存在强召回商品时，再写入结果快照。
	if len(p.PriorityGoodsIds) > 0 {
		fields["priorityGoodsIds"] = p.PriorityGoodsIds
	}
	// 当前存在类目补足时，再写入结果快照。
	if len(p.CategoryIds) > 0 {
		fields["categoryIds"] = p.CategoryIds
	}
	// explain 明细由调用方按需提供，空值时不写入上下文。
	if snapshot.ReturnedScoreDetails != nil {
		fields["returnedScoreDetails"] = snapshot.ReturnedScoreDetails
	}
	return p.BuildSceneResultSourceContext(sceneInput, fields, probeContext)
}

// BuildAnonymousOnlineResultContext 构建匿名态最终返回的完整来源上下文。
func (p *RequestPlan) BuildAnonymousOnlineResultContext(sceneInput SceneInput, snapshot ResultSnapshot, candidateGoodsIds []int64, returnedGoodsIds []int64, probeContext map[string]any) map[string]any {
	sourceContext := p.BuildAnonymousResultSourceContext(sceneInput, snapshot, probeContext)
	return p.AppendAnonymousOnlineDebugContext(sourceContext, candidateGoodsIds, returnedGoodsIds)
}

// BuildPersonalizedOnlineResultContext 构建登录态最终返回的完整来源上下文。
func (p *RequestPlan) BuildPersonalizedOnlineResultContext(sceneInput SceneInput, snapshot ResultSnapshot, candidateGoodsIds []int64, returnedGoodsIds []int64, probeContext map[string]any) map[string]any {
	sourceContext := p.BuildPersonalizedResultSourceContext(sceneInput, snapshot, probeContext)
	return p.AppendPersonalizedOnlineDebugContext(sourceContext, candidateGoodsIds, returnedGoodsIds)
}

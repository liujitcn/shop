package planner

import recommendOnlineRecall "shop/pkg/recommend/online/recall"

// BuildSceneResultSourceContext 构建带场景字段补充的结果来源上下文。
func (p *RequestPlan) BuildSceneResultSourceContext(sceneInput SceneInput, fields map[string]any, probeContext map[string]any) map[string]any {
	base := p.BuildSceneSourceContext(sceneInput)
	for key, value := range fields {
		base[key] = value
	}
	return p.BuildSourceContext(base, probeContext)
}

// AppendAnonymousOnlineDebugContext 为匿名态来源上下文补充在线调试字段。
func (p *RequestPlan) AppendAnonymousOnlineDebugContext(sourceContext map[string]any, candidateGoodsIds []int64, returnedGoodsIds []int64) map[string]any {
	return recommendOnlineRecall.AppendJoinContext(sourceContext, p.JoinRecallGoodsIds, candidateGoodsIds, returnedGoodsIds)
}

// AppendPersonalizedOnlineDebugContext 为登录态来源上下文补充在线调试字段。
func (p *RequestPlan) AppendPersonalizedOnlineDebugContext(sourceContext map[string]any, candidateGoodsIds []int64, returnedGoodsIds []int64) map[string]any {
	sourceContext = recommendOnlineRecall.AppendJoinContext(sourceContext, p.JoinRecallGoodsIds, candidateGoodsIds, returnedGoodsIds)
	return recommendOnlineRecall.AppendSimilarUserObservationContext(sourceContext, p.SimilarUserIds, p.SimilarUserObservedGoodsIds, p.JoinRecallGoodsIds, candidateGoodsIds, returnedGoodsIds)
}

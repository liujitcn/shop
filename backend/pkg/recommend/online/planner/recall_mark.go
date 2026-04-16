package planner

import (
	recommendCandidate "shop/pkg/recommend/candidate"
	recommendcore "shop/pkg/recommend/core"
)

// AppendAnonymousExplainRecallSources 为匿名态命中的候选商品补充 explain 召回来源。
func (p *RequestPlan) AppendAnonymousExplainRecallSources(candidates map[int64]*recommendcore.Candidate) {
	// 匿名态当前只需要补内容相似灰度召回来源。
	appendCandidateRecallSources(candidates, recommendCandidate.RecallSourceContentBased, p.ContentBasedJoinGoodsIds)
}

// AppendPersonalizedExplainRecallSources 为登录态命中的候选商品补充 explain 召回来源。
func (p *RequestPlan) AppendPersonalizedExplainRecallSources(candidates map[int64]*recommendcore.Candidate) {
	// 登录态详情页当前会补内容相似与协同过滤两类灰度召回来源。
	appendCandidateRecallSources(candidates, recommendCandidate.RecallSourceContentBased, p.ContentBasedJoinGoodsIds)
	appendCandidateRecallSources(candidates, recommendCandidate.RecallSourceCF, p.CollaborativeFilteringGoodsIds)
}

// appendCandidateRecallSources 为命中的候选商品补充召回来源标记。
func appendCandidateRecallSources(candidates map[int64]*recommendcore.Candidate, source string, goodsIds []int64) {
	if len(candidates) == 0 || source == "" || len(goodsIds) == 0 {
		return
	}
	for _, goodsId := range goodsIds {
		candidate, ok := candidates[goodsId]
		// 当前商品未进入最终候选池时，不再强行补 explain 来源。
		if !ok {
			continue
		}
		candidate.AddRecallSource(source)
	}
}

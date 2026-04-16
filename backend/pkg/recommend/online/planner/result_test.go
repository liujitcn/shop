package planner

import (
	"testing"

	recommendOnlineRank "shop/pkg/recommend/online/rank"
)

// TestBuildAnonymousOnlinePayload 验证匿名态返回负载会统一携带召回来源和来源上下文。
func TestBuildAnonymousOnlinePayload(t *testing.T) {
	plan := &RequestPlan{
		CandidateLimit:   20,
		RecallSources:    []string{"scene_hot"},
		PriorityGoodsIds: []int64{11},
	}

	emptyPayload := plan.BuildAnonymousEmptyOnlinePayload(SceneInput{}, []int64{21}, []int64{31}, []int64{41}, map[string]any{})
	// 匿名态空页返回时，应继续保留计划对象中的召回来源。
	if len(emptyPayload.RecallSources) != 1 || emptyPayload.RecallSources[0] != "scene_hot" {
		t.Fatalf("unexpected empty payload recall sources: %+v", emptyPayload.RecallSources)
	}
	if emptyPayload.SourceContext == nil {
		t.Fatalf("unexpected empty payload source context: %+v", emptyPayload.SourceContext)
	}

	pagePayload := plan.BuildAnonymousPageOnlinePayload(SceneInput{}, []int64{21}, []int64{31}, []int64{41}, recommendOnlineRank.PageExplainSnapshot{
		RecallSources:    []string{"content_based"},
		ReturnedGoodsIds: []int64{51},
	}, map[string]any{})
	// 匿名态正常页返回时，应透传 explain 快照里的召回来源。
	if len(pagePayload.RecallSources) != 1 || pagePayload.RecallSources[0] != "content_based" {
		t.Fatalf("unexpected page payload recall sources: %+v", pagePayload.RecallSources)
	}
	if pagePayload.SourceContext == nil {
		t.Fatalf("unexpected page payload source context: %+v", pagePayload.SourceContext)
	}
}

// TestBuildPersonalizedOnlinePayload 验证登录态返回负载会统一携带召回来源和来源上下文。
func TestBuildPersonalizedOnlinePayload(t *testing.T) {
	plan := &RequestPlan{
		CandidateLimit: 10,
	}

	emptyPayload := plan.BuildPersonalizedEmptyOnlinePayload(SceneInput{}, []int64{11}, map[string]any{})
	// 登录态空页返回时，不额外注入召回来源。
	if len(emptyPayload.RecallSources) != 0 {
		t.Fatalf("unexpected empty payload recall sources: %+v", emptyPayload.RecallSources)
	}
	if emptyPayload.SourceContext == nil {
		t.Fatalf("unexpected empty payload source context: %+v", emptyPayload.SourceContext)
	}

	pagePayload := plan.BuildPersonalizedPageOnlinePayload(SceneInput{}, []int64{11}, recommendOnlineRank.PageExplainSnapshot{
		RecallSources:    []string{"profile", "latest"},
		ReturnedGoodsIds: []int64{21},
	}, map[string]any{})
	// 登录态正常页返回时，应透传 explain 快照里的召回来源。
	if len(pagePayload.RecallSources) != 2 || pagePayload.RecallSources[0] != "profile" || pagePayload.RecallSources[1] != "latest" {
		t.Fatalf("unexpected page payload recall sources: %+v", pagePayload.RecallSources)
	}
	if pagePayload.SourceContext == nil {
		t.Fatalf("unexpected page payload source context: %+v", pagePayload.SourceContext)
	}
}

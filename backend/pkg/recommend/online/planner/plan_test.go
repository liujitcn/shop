package planner

import (
	"testing"

	"shop/api/gen/go/common"
	recommendCandidate "shop/pkg/recommend/candidate"
	recommendDomain "shop/pkg/recommend/domain"
)

// TestNewAnonymousRequestPlan 验证匿名态请求计划的初始化结果。
func TestNewAnonymousRequestPlan(t *testing.T) {
	request := &recommendDomain.GoodsRequest{
		Scene:    common.RecommendScene_GOODS_DETAIL,
		GoodsId:  99,
		PageNum:  2,
		PageSize: 10,
	}
	probeContext := map[string]any{
		"contentBased": map[string]any{
			"enabled":       true,
			"joinCandidate": true,
			"goodsIds":      []int64{11, 12, 11},
		},
	}

	plan := NewAnonymousRequestPlan(request, probeContext)

	// 二页十条请求仍应复用最大候选池，避免前后页因为扩池发生内容回流。
	if plan.CandidateLimit != recommendCandidate.PoolMax {
		t.Fatalf("unexpected candidate limit: %d", plan.CandidateLimit)
	}
	// 内容相似灰度召回应当在计划对象里完成去重。
	if len(plan.ContentBasedJoinGoodsIds) != 2 || plan.ContentBasedJoinGoodsIds[0] != 11 || plan.ContentBasedJoinGoodsIds[1] != 12 {
		t.Fatalf("unexpected content based goods ids: %+v", plan.ContentBasedJoinGoodsIds)
	}
	// 匿名态只应把内容相似灰度召回写入 joinRecallGoodsIds。
	if len(plan.JoinRecallGoodsIds[recommendCandidate.RecallSourceContentBased]) != 2 {
		t.Fatalf("unexpected join recall goods ids: %+v", plan.JoinRecallGoodsIds)
	}
}

// TestNewPersonalizedRequestPlan 验证登录态请求计划的初始化结果。
func TestNewPersonalizedRequestPlan(t *testing.T) {
	request := &recommendDomain.GoodsRequest{
		Scene:    common.RecommendScene_GOODS_DETAIL,
		GoodsId:  88,
		PageNum:  1,
		PageSize: 10,
	}
	probeContext := map[string]any{
		"similarUser": map[string]any{
			"enabled":       true,
			"joinCandidate": true,
			"userIds":       []int64{101, 102, 101},
		},
		"contentBased": map[string]any{
			"enabled":       true,
			"joinCandidate": true,
			"goodsIds":      []int64{21, 22, 21},
		},
		"collaborativeFiltering": map[string]any{
			"enabled":       true,
			"joinCandidate": true,
			"goodsIds":      []int64{31, 32, 31},
		},
	}

	plan := NewPersonalizedRequestPlan(request, probeContext)

	// 相似用户探针命中的用户编号应在计划对象里完成去重。
	if len(plan.SimilarUserIds) != 2 || plan.SimilarUserIds[0] != 101 || plan.SimilarUserIds[1] != 102 {
		t.Fatalf("unexpected similar user ids: %+v", plan.SimilarUserIds)
	}
	// 协同过滤灰度召回应写入 joinRecallGoodsIds。
	if len(plan.JoinRecallGoodsIds[recommendCandidate.RecallSourceCF]) != 2 {
		t.Fatalf("unexpected cf join recall goods ids: %+v", plan.JoinRecallGoodsIds)
	}
	// 内容相似灰度召回应写入 joinRecallGoodsIds。
	if len(plan.JoinRecallGoodsIds[recommendCandidate.RecallSourceContentBased]) != 2 {
		t.Fatalf("unexpected content based join recall goods ids: %+v", plan.JoinRecallGoodsIds)
	}
}

// TestApplySimilarUserObservationAndJoinRecall 验证相似用户观测结果允许入池时会并入优先候选。
func TestApplySimilarUserObservationAndJoinRecall(t *testing.T) {
	plan := &RequestPlan{
		JoinRecallGoodsIds: make(map[string][]int64, 1),
	}

	plan.ApplySimilarUserObservation([]int64{71, 72, 71}, true)
	plan.ApplyJoinRecall()
	plan.NormalizeState()

	if len(plan.SimilarUserObservedGoodsIds) != 2 || plan.SimilarUserObservedGoodsIds[0] != 71 || plan.SimilarUserObservedGoodsIds[1] != 72 {
		t.Fatalf("unexpected similar user observed goods ids: %+v", plan.SimilarUserObservedGoodsIds)
	}
	if len(plan.SimilarUserJoinGoodsIds) != 2 || plan.SimilarUserJoinGoodsIds[0] != 71 || plan.SimilarUserJoinGoodsIds[1] != 72 {
		t.Fatalf("unexpected similar user join goods ids: %+v", plan.SimilarUserJoinGoodsIds)
	}
	if len(plan.JoinRecallGoodsIds[recommendCandidate.RecallSourceSimilarUser]) != 2 {
		t.Fatalf("unexpected similar user join recall goods ids: %+v", plan.JoinRecallGoodsIds)
	}
	if len(plan.PriorityGoodsIds) != 2 || plan.PriorityGoodsIds[0] != 71 || plan.PriorityGoodsIds[1] != 72 {
		t.Fatalf("unexpected priority goods ids: %+v", plan.PriorityGoodsIds)
	}
}

// TestRequestPlanNormalizeAndSourceContext 验证请求计划的归一化和来源上下文构建。
func TestRequestPlanNormalizeAndSourceContext(t *testing.T) {
	plan := &RequestPlan{
		Request: recommendDomain.GoodsRequest{
			Scene:    common.RecommendScene_GOODS_DETAIL,
			GoodsId:  66,
			PageNum:  1,
			PageSize: 10,
		},
		PriorityGoodsIds: []int64{1, 2, 1},
		CategoryIds:      []int64{3, 4, 3},
		RecallSources:    []string{"profile", "latest", "profile"},
		CacheHitSources:  []string{"latest_cache", "latest_cache"},
		CacheReadContext: map[string]any{
			"latest_cache": map[string]any{
				"hit": true,
			},
		},
		JoinRecallGoodsIds: map[string][]int64{
			recommendCandidate.RecallSourceCF: {7, 8, 7},
		},
	}
	probeContext := map[string]any{
		"observedSources": []string{"content_based_probe"},
	}

	plan.NormalizeState()
	excludeGoodsIds := plan.ExcludeGoodsIds()
	// 商品详情场景需要额外排除当前详情商品本身。
	if len(excludeGoodsIds) != 3 || excludeGoodsIds[0] != 1 || excludeGoodsIds[1] != 2 || excludeGoodsIds[2] != 66 {
		t.Fatalf("unexpected exclude goods ids: %+v", excludeGoodsIds)
	}

	sourceContext := plan.BuildSourceContext(map[string]any{
		"candidateLimit": 80,
	}, probeContext)

	// 计划对象里的缓存命中来源应收口到来源上下文。
	cacheHitSources, ok := sourceContext["cacheHitSources"].([]string)
	if !ok || len(cacheHitSources) != 1 || cacheHitSources[0] != "latest_cache" {
		t.Fatalf("unexpected cache hit sources: %+v", sourceContext["cacheHitSources"])
	}

	// 计划对象里的缓存读取上下文应直接透传到来源上下文。
	cacheReadContext, ok := sourceContext["cacheReadContext"].(map[string]any)
	if !ok || len(cacheReadContext) != 1 {
		t.Fatalf("unexpected cache read context: %+v", sourceContext["cacheReadContext"])
	}

	// 探针观测来源应继续由 recall 上下文辅助函数回写。
	observedRecallSources, ok := sourceContext["observedRecallSources"].([]string)
	if !ok || len(observedRecallSources) != 1 || observedRecallSources[0] != "content_based_probe" {
		t.Fatalf("unexpected observed recall sources: %+v", sourceContext["observedRecallSources"])
	}
}

// TestApplySceneMethods 验证场景规划方法会把前置状态收口到计划对象。
func TestApplySceneMethods(t *testing.T) {
	plan := &RequestPlan{
		ContentBasedJoinGoodsIds:       []int64{51, 52, 51},
		CollaborativeFilteringGoodsIds: []int64{61, 62, 61},
	}

	plan.ApplyCartScene([]int64{11, 12}, []int64{21})
	plan.ApplyOrderScene([]int64{13}, []int64{22, 23})
	plan.ApplyGoodsDetailScene([]int64{14}, []int64{24})
	plan.ApplyJoinRecall()
	plan.ApplyProfileScene([]int64{31, 32})
	plan.EnsureFallbackLatest()
	plan.NormalizeState()

	// 已命中过场景召回来源时，不应再额外补 latest 兜底标记。
	for _, source := range plan.RecallSources {
		if source == "latest" {
			t.Fatalf("unexpected latest fallback source: %+v", plan.RecallSources)
		}
	}

	// 商品详情场景应把内容相似与协同过滤灰度召回并入优先候选集合。
	if len(plan.PriorityGoodsIds) != 8 {
		t.Fatalf("unexpected priority goods ids: %+v", plan.PriorityGoodsIds)
	}

	// 多个场景追加的类目集合应完成去重。
	if len(plan.CategoryIds) != 6 {
		t.Fatalf("unexpected category ids: %+v", plan.CategoryIds)
	}

	// 场景方法应补齐 cart / order / goods_detail / profile 和两类灰度召回来源。
	if len(plan.RecallSources) != 6 {
		t.Fatalf("unexpected recall sources: %+v", plan.RecallSources)
	}
}

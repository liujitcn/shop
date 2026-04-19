package domain

import (
	"testing"
)

// TestPublishStrategyResolveEffectiveVersion 验证发布配置的有效版本解析。
func TestPublishStrategyResolveEffectiveVersion(t *testing.T) {
	strategy := &PublishStrategy{
		CacheVersion:    "cache-v2",
		RollbackVersion: "rollback-v1",
	}

	effectiveVersion := strategy.ResolveEffectiveVersion("default")
	if effectiveVersion != "rollback-v1" {
		t.Fatalf("unexpected effective version: %s", effectiveVersion)
	}
}

// TestSceneStrategyContextBuildPublishContext 验证发布上下文会带发布时间元数据。
func TestSceneStrategyContextBuildPublishContext(t *testing.T) {
	contextMap := (&SceneStrategyContext{
		Scene:            1,
		Version:          "v2",
		EffectiveVersion: "v1",
		PublishResolution: &PublishVersionResolution{
			BaselineVersion:    "v1",
			GrayVersion:        "v2",
			EffectiveVersion:   "v1",
			GrayRatio:          0.5,
			GrayEnabled:        true,
			GrayHit:            false,
			GrayBucket:         1234,
			GrayBucketResolved: true,
		},
		Config: &StrategyVersionConfig{
			Publish: &PublishStrategy{
				CacheVersion:    "v2",
				RollbackVersion: "v1",
				GrayRatio:       0.5,
				PublishedBy:     "tester",
				PublishedReason: "rollback",
				PublishedAt:     "2026-04-17T12:30:00Z",
			},
		},
	}).BuildPublishContext()

	if contextMap["publishedAt"] != "2026-04-17T12:30:00Z" {
		t.Fatalf("unexpected publish context: %+v", contextMap)
	}
	if contextMap["grayBucket"] != 1234 || contextMap["grayHit"] != false {
		t.Fatalf("unexpected gray publish context: %+v", contextMap)
	}
}

// TestPublishStrategyResolveVersionResolution 验证灰度版本解析会按主体稳定分桶。
func TestPublishStrategyResolveVersionResolution(t *testing.T) {
	strategy := &PublishStrategy{
		CacheVersion:    "gray-v2",
		RollbackVersion: "baseline-v1",
		GrayRatio:       1,
	}

	firstResolution := strategy.ResolveVersionResolution(3, "online-v2", &Actor{
		ActorType: 1,
		ActorId:   18,
	})
	secondResolution := strategy.ResolveVersionResolution(3, "online-v2", &Actor{
		ActorType: 1,
		ActorId:   18,
	})

	if !firstResolution.GrayEnabled || !firstResolution.GrayHit {
		t.Fatalf("unexpected gray resolution: %+v", firstResolution)
	}
	if firstResolution.EffectiveVersion != "gray-v2" || firstResolution.BaselineVersion != "baseline-v1" {
		t.Fatalf("unexpected effective resolution: %+v", firstResolution)
	}
	if firstResolution.GrayBucket != secondResolution.GrayBucket {
		t.Fatalf("unexpected unstable bucket: first=%+v second=%+v", firstResolution, secondResolution)
	}
}

// TestPublishStrategyResolveVersionResolutionWithoutActor 验证缺少稳定主体时不会误命中灰度。
func TestPublishStrategyResolveVersionResolutionWithoutActor(t *testing.T) {
	strategy := &PublishStrategy{
		CacheVersion:    "gray-v2",
		RollbackVersion: "baseline-v1",
		GrayRatio:       0.5,
	}

	resolution := strategy.ResolveVersionResolution(2, "online-v2", &Actor{})
	if resolution.EffectiveVersion != "baseline-v1" || resolution.GrayHit || resolution.GrayBucketResolved {
		t.Fatalf("unexpected anonymous gray resolution: %+v", resolution)
	}
}

// TestRankerStrategyNormalize 验证模型精排配置的类型归一化。
func TestRankerStrategyNormalize(t *testing.T) {
	strategy := &RankerStrategy{
		Enabled: true,
		Type:    " FM ",
	}

	if !strategy.IsEnabled() {
		t.Fatalf("expected ranker strategy enabled")
	}
	if strategy.NormalizeType() != RankerTypeFM {
		t.Fatalf("unexpected ranker type: %s", strategy.NormalizeType())
	}
}

// TestLlmRerankStrategyResolveRuntimeConfig 验证 LLM 重排运行时配置解析。
func TestLlmRerankStrategyResolveRuntimeConfig(t *testing.T) {
	temperature := 0.0
	strategy := &LlmRerankStrategy{
		Model:               " gpt-4o-mini ",
		TimeoutSeconds:      6,
		MaxCompletionTokens: 256,
		Temperature:         &temperature,
	}

	if strategy.ResolveModel("fallback-model") != "gpt-4o-mini" {
		t.Fatalf("unexpected llm model: %s", strategy.ResolveModel("fallback-model"))
	}
	if strategy.ResolveTimeout(0).Seconds() != 6 {
		t.Fatalf("unexpected llm timeout: %v", strategy.ResolveTimeout(0))
	}
	if strategy.ResolveMaxCompletionTokens(128) != 256 {
		t.Fatalf("unexpected llm max completion tokens: %d", strategy.ResolveMaxCompletionTokens(128))
	}
	if strategy.ResolveTemperature(0.7) != 0 {
		t.Fatalf("unexpected llm temperature: %v", strategy.ResolveTemperature(0.7))
	}
}

// TestTuneStrategyBuildContext 验证自动调参与最近训练摘要会一起写入调试上下文。
func TestTuneStrategyBuildContext(t *testing.T) {
	strategy := &TuneStrategy{
		Enabled:      true,
		TargetMetric: " AUC ",
		TrialCount:   8,
		Latest: &TuneLatestSummary{
			Task:        "ranker",
			ModelType:   "AFM",
			Backend:     "GoMLX",
			ArtifactDir: "data/recommend/train/ranker/v1/run-1",
			TrainedAt:   "2026-04-17T12:00:00Z",
			Version:     "v1",
			BestValue:   0.91,
			Score: map[string]float64{
				"AUC": 0.91,
			},
		},
		LatestEval: &TuneLatestEvalSummary{
			ReportDate:    "2026-04-17",
			StrategyName:  "recommend:v1",
			SampleSize:    12,
			RequestCount:  20,
			ExposureCount: 80,
			ClickCount:    10,
			OrderCount:    3,
			PayCount:      2,
			Ctr:           0.125,
			Cvr:           0.2,
			Ndcg:          0.66,
			Precision:     0.4,
			Recall:        0.5,
		},
	}

	contextMap := strategy.BuildContext()
	if contextMap["targetMetric"] != "auc" {
		t.Fatalf("unexpected target metric: %+v", contextMap)
	}
	latestContext, ok := contextMap["latest"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected latest context: %+v", contextMap["latest"])
	}
	if latestContext["backend"] != "gomlx" || latestContext["modelType"] != "afm" {
		t.Fatalf("unexpected latest summary: %+v", latestContext)
	}
	scoreMap, ok := latestContext["score"].(map[string]float64)
	if !ok || scoreMap["auc"] != 0.91 {
		t.Fatalf("unexpected latest score: %+v", latestContext["score"])
	}
	latestEvalContext, ok := contextMap["latestEval"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected latest eval context: %+v", contextMap["latestEval"])
	}
	if latestEvalContext["reportDate"] != "2026-04-17" || latestEvalContext["ndcg"] != 0.66 {
		t.Fatalf("unexpected latest eval summary: %+v", latestEvalContext)
	}
}

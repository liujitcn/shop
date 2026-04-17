package task

import (
	"encoding/json"
	"testing"
	"time"

	recommendDomain "shop/pkg/recommend/domain"
)

// TestMergeRecommendTuneLatestConfigJSON 验证最近训练摘要会合并到版本配置且保留原有字段。
func TestMergeRecommendTuneLatestConfigJSON(t *testing.T) {
	configJSON, err := mergeRecommendTuneLatestConfigJSON(`{
  "publish": {
    "cache_version": "gray-v2"
  },
  "tune": {
    "enabled": true,
    "target_metric": "auc",
    "trial_count": 9
  }
}`, &recommendDomain.TuneLatestSummary{
		Task:        "ranker",
		ModelType:   "afm",
		Backend:     "gomlx",
		ArtifactDir: "data/recommend/train/ranker/v1/run-1",
		TrainedAt:   "2026-04-17T12:00:00Z",
		Version:     "v1",
		BestValue:   0.91,
		Score: map[string]float64{
			"auc": 0.91,
		},
	})
	if err != nil {
		t.Fatalf("merge config json: %v", err)
	}

	config := &recommendDomain.StrategyVersionConfig{}
	if err = json.Unmarshal([]byte(configJSON), config); err != nil {
		t.Fatalf("unmarshal merged config: %v", err)
	}
	if config.Publish == nil || config.Publish.CacheVersion != "gray-v2" {
		t.Fatalf("publish config lost after merge: %+v", config.Publish)
	}
	if config.Tune == nil || !config.Tune.Enabled || config.Tune.TargetMetric != "auc" || config.Tune.TrialCount != 9 {
		t.Fatalf("tune config lost after merge: %+v", config.Tune)
	}
	if config.Tune.Latest == nil || config.Tune.Latest.ArtifactDir != "data/recommend/train/ranker/v1/run-1" {
		t.Fatalf("latest summary missing after merge: %+v", config.Tune.Latest)
	}
}

// TestBuildRecommendTuneLatestSummary 验证训练摘要会规整版本和指标字段。
func TestBuildRecommendTuneLatestSummary(t *testing.T) {
	summary := buildRecommendTuneLatestSummary(
		"collaborative_filtering",
		"BPR",
		"GoMLX",
		"data/recommend/train/collaborative_filtering/run-1",
		time.Date(2026, 4, 17, 12, 30, 0, 0, time.UTC),
		"",
		[]string{" gray-v1 ", "gray-v1", "beta"},
		0.77,
		map[string]float64{
			" NDCG ": 0.77,
			"":       1,
		},
	)

	if summary == nil {
		t.Fatalf("summary should not be nil")
	}
	if summary.ModelType != "bpr" || summary.Backend != "gomlx" {
		t.Fatalf("unexpected summary normalize: %+v", summary)
	}
	if len(summary.Versions) != 2 || summary.Versions[0] != "beta" || summary.Versions[1] != "gray-v1" {
		t.Fatalf("unexpected versions: %+v", summary.Versions)
	}
	if summary.Score["ndcg"] != 0.77 || len(summary.Score) != 1 {
		t.Fatalf("unexpected score map: %+v", summary.Score)
	}
}

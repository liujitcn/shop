package task

import (
	"encoding/json"
	"testing"
	"time"

	"shop/pkg/gen/models"
	recommendDomain "shop/pkg/recommend/domain"
)

// TestMergeRecommendTuneLatestEvalConfigJSON 验证最近评估摘要会合并到版本配置且保留原有字段。
func TestMergeRecommendTuneLatestEvalConfigJSON(t *testing.T) {
	configJSON, err := mergeRecommendTuneLatestEvalConfigJSON(`{
  "publish": {
    "cache_version": "gray-v2"
  },
  "tune": {
    "enabled": true,
    "target_metric": "auc",
    "trial_count": 9,
    "latest": {
      "task": "ranker"
    }
  }
}`, &recommendDomain.TuneLatestEvalSummary{
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
	})
	if err != nil {
		t.Fatalf("merge eval config json: %v", err)
	}

	config := &recommendDomain.StrategyVersionConfig{}
	if err = json.Unmarshal([]byte(configJSON), config); err != nil {
		t.Fatalf("unmarshal merged eval config: %v", err)
	}
	if config.Publish == nil || config.Publish.CacheVersion != "gray-v2" {
		t.Fatalf("publish config lost after eval merge: %+v", config.Publish)
	}
	if config.Tune == nil || config.Tune.Latest == nil || config.Tune.Latest.Task != "ranker" {
		t.Fatalf("latest training summary lost after eval merge: %+v", config.Tune)
	}
	if config.Tune.LatestEval == nil || config.Tune.LatestEval.ReportDate != "2026-04-17" {
		t.Fatalf("latest eval summary missing after merge: %+v", config.Tune.LatestEval)
	}
}

// TestBuildRecommendTuneLatestEvalSummary 验证评估报告会规整为版本评估摘要。
func TestBuildRecommendTuneLatestEvalSummary(t *testing.T) {
	report := recommendEvalReportStub()
	summary := buildRecommendTuneLatestEvalSummary(&report)
	if summary == nil {
		t.Fatalf("summary should not be nil")
	}
	if summary.ReportDate != "2026-04-17" || summary.StrategyName != "recommend:v1" {
		t.Fatalf("unexpected eval summary head: %+v", summary)
	}
	if summary.Ndcg != 0.66 || summary.Precision != 0.4 || summary.Recall != 0.5 {
		t.Fatalf("unexpected eval metrics: %+v", summary)
	}
}

// recommendEvalReportStub 构造评估报告测试桩。
func recommendEvalReportStub() models.RecommendEvalReport {
	return models.RecommendEvalReport{
		ReportDate:     time.Date(2026, 4, 17, 0, 0, 0, 0, time.UTC),
		StrategyName:   "recommend:v1",
		SampleSize:     12,
		RequestCount:   20,
		ExposureCount:  80,
		ClickCount:     10,
		OrderCount:     3,
		PayCount:       2,
		Ctr:            0.125,
		Cvr:            0.2,
		Ndcg:           0.66,
		PrecisionScore: 0.4,
		RecallScore:    0.5,
	}
}

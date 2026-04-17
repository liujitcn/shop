package tune

import (
	"context"
	"testing"

	"shop/pkg/recommend/offline/train/ctr"
)

// TestTuneAFM 验证 AFM 调参可以返回有效参数与指标。
func TestTuneAFM(t *testing.T) {
	sampleList := []ctr.Sample{
		{
			UserId: "u1",
			ItemId: "i1",
			UserLabels: []ctr.Label{
				{Name: "actor_type:user"},
			},
			ItemLabels: []ctr.Label{
				{Name: "category:1"},
			},
			ContextLabels: []ctr.Label{
				{Name: "scene:home"},
				{Name: "rule_score", Value: 0.9},
			},
			Target: 1,
		},
		{
			UserId: "u1",
			ItemId: "i2",
			UserLabels: []ctr.Label{
				{Name: "actor_type:user"},
			},
			ItemLabels: []ctr.Label{
				{Name: "category:2"},
			},
			ContextLabels: []ctr.Label{
				{Name: "scene:home"},
				{Name: "rule_score", Value: 0.2},
			},
			Target: 0,
		},
		{
			UserId: "u2",
			ItemId: "i1",
			UserLabels: []ctr.Label{
				{Name: "actor_type:user"},
			},
			ItemLabels: []ctr.Label{
				{Name: "category:1"},
			},
			ContextLabels: []ctr.Label{
				{Name: "scene:detail"},
				{Name: "rule_score", Value: 0.8},
			},
			Target: 1,
		},
		{
			UserId: "u2",
			ItemId: "i3",
			UserLabels: []ctr.Label{
				{Name: "actor_type:user"},
			},
			ItemLabels: []ctr.Label{
				{Name: "category:3"},
			},
			ContextLabels: []ctr.Label{
				{Name: "scene:detail"},
				{Name: "rule_score", Value: 0.1},
			},
			Target: 0,
		},
	}

	dataset := ctr.BuildDataset(sampleList)
	trainSet, testSet := dataset.Split(0.5, 42)
	result, err := TuneAFM(context.Background(), trainSet, testSet, ctr.Config{
		Epochs:    5,
		Factors:   8,
		BatchSize: 2,
		Verbose:   1,
	}, AFMOptions{
		TrialCount:   2,
		TargetMetric: "auc",
	})
	if err != nil {
		t.Fatalf("tune afm: %v", err)
	}
	if result == nil {
		t.Fatalf("expected tune result")
	}
	if result.TrialCount != 2 {
		t.Fatalf("unexpected trial count: %d", result.TrialCount)
	}
	if result.Config.Factors <= 0 || result.Config.BatchSize <= 0 || result.Config.Epochs <= 0 {
		t.Fatalf("unexpected config: %+v", result.Config)
	}
}

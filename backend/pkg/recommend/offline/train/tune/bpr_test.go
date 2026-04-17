package tune

import (
	"context"
	"testing"

	recommendCf "shop/pkg/recommend/offline/train/cf"
)

// TestTuneBPR 验证 BPR 调参可以返回有效参数与指标。
func TestTuneBPR(t *testing.T) {
	dataset := recommendCf.BuildDataset([]recommendCf.Interaction{
		{UserId: "u1", ItemId: "i1", Weight: 2},
		{UserId: "u1", ItemId: "i2", Weight: 1},
		{UserId: "u2", ItemId: "i1", Weight: 1},
		{UserId: "u2", ItemId: "i3", Weight: 2},
		{UserId: "u3", ItemId: "i2", Weight: 2},
		{UserId: "u3", ItemId: "i3", Weight: 1},
	})
	trainSet, testSet := dataset.Split(0.5, 17)
	result, err := TuneBPR(context.Background(), trainSet, testSet, recommendCf.Config{
		Backend:   recommendCf.BackendNative,
		Epochs:    10,
		Factors:   8,
		BatchSize: 2,
		Learning:  0.05,
		Reg:       0.01,
		Seed:      17,
	}, BPROptions{
		TrialCount:   2,
		TargetMetric: "ndcg",
		TopK:         2,
		Candidates:   10,
	})
	if err != nil {
		t.Fatalf("tune bpr: %v", err)
	}
	if result == nil {
		t.Fatal("expected tune result")
	}
	if result.TrialCount != 2 {
		t.Fatalf("unexpected trial count: %d", result.TrialCount)
	}
	if result.Config.Factors <= 0 || result.Config.BatchSize <= 0 || result.Config.Epochs <= 0 {
		t.Fatalf("unexpected config: %+v", result.Config)
	}
	if result.Score.NDCG <= 0 || result.Score.NDCG > 1 {
		t.Fatalf("unexpected ndcg: %.6f", result.Score.NDCG)
	}
}

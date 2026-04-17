package cf

import (
	"context"
	"math"
	"testing"
)

// TestFitWithGoMLX 验证 gomlx 后端可以完成 BPR 训练并产出可推荐因子。
func TestFitWithGoMLX(t *testing.T) {
	interactionList := []Interaction{
		{UserId: "u1", ItemId: "i1", Weight: 2},
		{UserId: "u1", ItemId: "i2", Weight: 1},
		{UserId: "u2", ItemId: "i2", Weight: 2},
		{UserId: "u2", ItemId: "i3", Weight: 1},
		{UserId: "u3", ItemId: "i1", Weight: 1},
		{UserId: "u3", ItemId: "i3", Weight: 2},
	}

	model := Fit(context.Background(), interactionList, Config{
		Backend:   BackendGoMLX,
		BatchSize: 2,
		Factors:   4,
		Epochs:    4,
		Learning:  0.05,
		Reg:       0.01,
		Optimizer: "sgd",
		Seed:      7,
	})
	if model == nil {
		t.Fatal("Fit() returned nil model")
	}
	if len(model.userFactors) != 3 {
		t.Fatalf("len(userFactors) = %d, want 3", len(model.userFactors))
	}
	if len(model.itemFactors) != 3 {
		t.Fatalf("len(itemFactors) = %d, want 3", len(model.itemFactors))
	}
	for rowIndex, row := range model.userFactors {
		// 用户因子维度必须与配置一致，避免训练参数抽取错位。
		if len(row) != 4 {
			t.Fatalf("len(userFactors[%d]) = %d, want 4", rowIndex, len(row))
		}
	}

	recommendList := model.Recommend("u1", 3, nil)
	if len(recommendList) == 0 {
		t.Fatal("Recommend() returned empty list")
	}
	for _, item := range recommendList {
		// 已交互商品不能再次出现在推荐结果中。
		if item.ItemId == "i1" || item.ItemId == "i2" {
			t.Fatalf("Recommend() returned interacted item %q", item.ItemId)
		}
		// 输出分值必须保持有限，避免把异常值写入缓存。
		if math.IsNaN(float64(item.Score)) || math.IsInf(float64(item.Score), 0) {
			t.Fatalf("Recommend() returned invalid score %v", item.Score)
		}
	}
}

package cf

import (
	"context"
	"testing"
)

// TestDatasetSplit 验证数据集切分会保留训练可学习商品，并保住单样本用户的训练数据。
func TestDatasetSplit(t *testing.T) {
	dataset := BuildDataset([]Interaction{
		{UserId: "u1", ItemId: "i1", Weight: 1},
		{UserId: "u1", ItemId: "i2", Weight: 2},
		{UserId: "u2", ItemId: "i1", Weight: 1},
		{UserId: "u2", ItemId: "i3", Weight: 1},
		{UserId: "u3", ItemId: "i2", Weight: 1},
		{UserId: "u3", ItemId: "i3", Weight: 1},
		{UserId: "u4", ItemId: "i4", Weight: 1},
	})
	trainSet, testSet := dataset.Split(0.5, 7)
	if trainSet.Count() == 0 {
		t.Fatal("expected non-empty train set")
	}
	if _, ok := trainSet.UserItemSet("u4")["i4"]; !ok {
		t.Fatal("single-item user should stay in train set")
	}

	trainItemIdSet := make(map[string]struct{}, len(trainSet.ItemIds()))
	for _, itemId := range trainSet.ItemIds() {
		trainItemIdSet[itemId] = struct{}{}
	}
	for _, item := range testSet.Interactions() {
		// 验证集商品必须在训练集中仍然可学习，避免出现无法打分的孤儿商品。
		if _, ok := trainItemIdSet[item.ItemId]; !ok {
			t.Fatalf("test item %s is missing from train set", item.ItemId)
		}
	}
}

// TestEvaluate 验证 BPR 模型可以产出有效评估指标。
func TestEvaluate(t *testing.T) {
	dataset := BuildDataset([]Interaction{
		{UserId: "u1", ItemId: "i1", Weight: 2},
		{UserId: "u1", ItemId: "i2", Weight: 1},
		{UserId: "u2", ItemId: "i1", Weight: 1},
		{UserId: "u2", ItemId: "i3", Weight: 2},
		{UserId: "u3", ItemId: "i2", Weight: 2},
		{UserId: "u3", ItemId: "i3", Weight: 1},
	})
	trainSet, testSet := dataset.Split(0.5, 11)
	model := Fit(context.Background(), trainSet.Interactions(), Config{
		Backend:  BackendNative,
		Factors:  8,
		Epochs:   40,
		Learning: 0.05,
		Reg:      0.01,
		Seed:     11,
	})
	score := Evaluate(model, trainSet, testSet, EvaluateConfig{
		TopK:       2,
		Candidates: 10,
		Seed:       11,
	})
	if score.NDCG <= 0 || score.NDCG > 1 {
		t.Fatalf("unexpected ndcg: %.6f", score.NDCG)
	}
	if score.Precision <= 0 || score.Precision > 1 {
		t.Fatalf("unexpected precision: %.6f", score.Precision)
	}
	if score.Recall <= 0 || score.Recall > 1 {
		t.Fatalf("unexpected recall: %.6f", score.Recall)
	}
}

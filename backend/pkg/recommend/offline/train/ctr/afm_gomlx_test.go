package ctr

import (
	"context"
	"math"
	"testing"
)

// TestAFMFitWithGoMLX 验证 gomlx 后端可以完成 AFM 训练与推理。
func TestAFMFitWithGoMLX(t *testing.T) {
	sampleList := []Sample{
		{UserId: "u1", ItemId: "i1", UserLabels: []Label{{Name: "vip"}}, ItemLabels: []Label{{Name: "cate_food"}}, ContextLabels: []Label{{Name: "scene_home"}}, Embeddings: []Embedding{{Name: "goods_content", Value: []float32{1, 0, 0, 0}}}, Target: 1},
		{UserId: "u1", ItemId: "i2", UserLabels: []Label{{Name: "vip"}}, ItemLabels: []Label{{Name: "cate_book"}}, ContextLabels: []Label{{Name: "scene_home"}}, Embeddings: []Embedding{{Name: "goods_content", Value: []float32{0.8, 0.2, 0, 0}}}, Target: 0},
		{UserId: "u2", ItemId: "i2", UserLabels: []Label{{Name: "new_user"}}, ItemLabels: []Label{{Name: "cate_book"}}, ContextLabels: []Label{{Name: "scene_detail"}}, Embeddings: []Embedding{{Name: "goods_content", Value: []float32{0.8, 0.2, 0, 0}}}, Target: 1},
		{UserId: "u2", ItemId: "i3", UserLabels: []Label{{Name: "new_user"}}, ItemLabels: []Label{{Name: "cate_phone"}}, ContextLabels: []Label{{Name: "scene_detail"}}, Embeddings: []Embedding{{Name: "goods_content", Value: []float32{0, 0, 1, 0}}}, Target: 0},
		{UserId: "u3", ItemId: "i1", UserLabels: []Label{{Name: "loyal"}}, ItemLabels: []Label{{Name: "cate_food"}}, ContextLabels: []Label{{Name: "scene_cart"}}, Embeddings: []Embedding{{Name: "goods_content", Value: []float32{1, 0, 0, 0}}}, Target: 1},
		{UserId: "u3", ItemId: "i3", UserLabels: []Label{{Name: "loyal"}}, ItemLabels: []Label{{Name: "cate_phone"}}, ContextLabels: []Label{{Name: "scene_cart"}}, Embeddings: []Embedding{{Name: "goods_content", Value: []float32{0, 0, 1, 0}}}, Target: 0},
	}
	dataset := BuildDataset(sampleList)
	trainSet, testSet := dataset.Split(0.34, 7)
	// 极小样本切分后若测试集为空，直接复用全量集做基础可用性校验。
	if testSet.Count() == 0 {
		testSet = dataset
	}

	model := NewAFM(Config{
		Backend:   BackendGoMLX,
		BatchSize: 2,
		Factors:   4,
		Epochs:    3,
		Verbose:   1,
		Patience:  2,
		Learning:  0.01,
		Reg:       0.0001,
		Optimizer: "adam",
		AutoScale: true,
		Seed:      7,
	})
	score := model.Fit(context.Background(), trainSet, testSet)
	if err := model.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	predictionList := model.PredictBatch([]Sample{
		{UserId: "u1", ItemId: "i1", UserLabels: []Label{{Name: "vip"}}, ItemLabels: []Label{{Name: "cate_food"}}, ContextLabels: []Label{{Name: "scene_home"}}, Embeddings: []Embedding{{Name: "goods_content", Value: []float32{1, 0, 0, 0}}}},
		{UserId: "u2", ItemId: "i3", UserLabels: []Label{{Name: "new_user"}}, ItemLabels: []Label{{Name: "cate_phone"}}, ContextLabels: []Label{{Name: "scene_detail"}}, Embeddings: []Embedding{{Name: "goods_content", Value: []float32{0, 0, 1, 0}}}},
	})
	// 推理输出条数必须与输入样本数一致，避免批处理对齐错误。
	if len(predictionList) != 2 {
		t.Fatalf("PredictBatch() len = %d, want 2", len(predictionList))
	}
	for index, value := range predictionList {
		// 训练结果必须是有限数，避免把 NaN/Inf 写入后续排序链路。
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			t.Fatalf("prediction[%d] invalid = %v", index, value)
		}
	}
	// 评估结果同样必须保持有限，确保训练闭环没有数值爆炸。
	if math.IsNaN(float64(score.AUC)) || math.IsInf(float64(score.AUC), 0) {
		t.Fatalf("score.AUC invalid = %v", score.AUC)
	}
}

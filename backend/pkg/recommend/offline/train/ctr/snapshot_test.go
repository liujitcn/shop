package ctr

import (
	"context"
	"math"
	"testing"
)

// TestSnapshotBuildModel 验证 AFM 快照恢复后仍能输出稳定打分。
func TestSnapshotBuildModel(t *testing.T) {
	testCases := []struct {
		name    string
		backend string
	}{
		{name: "native", backend: BackendNative},
		{name: "gomlx", backend: BackendGoMLX},
	}

	sampleList := []Sample{
		{UserId: "u1", ItemId: "i1", UserLabels: []Label{{Name: "vip"}}, ItemLabels: []Label{{Name: "cate_food"}}, ContextLabels: []Label{{Name: "scene_home"}}, Embeddings: []Embedding{{Name: "goods_content", Value: []float32{1, 0, 0, 0}}}, Target: 1},
		{UserId: "u1", ItemId: "i2", UserLabels: []Label{{Name: "vip"}}, ItemLabels: []Label{{Name: "cate_book"}}, ContextLabels: []Label{{Name: "scene_home"}}, Embeddings: []Embedding{{Name: "goods_content", Value: []float32{0.8, 0.2, 0, 0}}}, Target: 0},
		{UserId: "u2", ItemId: "i2", UserLabels: []Label{{Name: "new_user"}}, ItemLabels: []Label{{Name: "cate_book"}}, ContextLabels: []Label{{Name: "scene_detail"}}, Embeddings: []Embedding{{Name: "goods_content", Value: []float32{0.8, 0.2, 0, 0}}}, Target: 1},
		{UserId: "u2", ItemId: "i3", UserLabels: []Label{{Name: "new_user"}}, ItemLabels: []Label{{Name: "cate_phone"}}, ContextLabels: []Label{{Name: "scene_detail"}}, Embeddings: []Embedding{{Name: "goods_content", Value: []float32{0, 0, 1, 0}}}, Target: 0},
		{UserId: "u3", ItemId: "i1", UserLabels: []Label{{Name: "loyal"}}, ItemLabels: []Label{{Name: "cate_food"}}, ContextLabels: []Label{{Name: "scene_cart"}}, Embeddings: []Embedding{{Name: "goods_content", Value: []float32{1, 0, 0, 0}}}, Target: 1},
		{UserId: "u3", ItemId: "i3", UserLabels: []Label{{Name: "loyal"}}, ItemLabels: []Label{{Name: "cate_phone"}}, ContextLabels: []Label{{Name: "scene_cart"}}, Embeddings: []Embedding{{Name: "goods_content", Value: []float32{0, 0, 1, 0}}}, Target: 0},
	}
	inferenceSampleList := []Sample{
		{UserId: "u1", ItemId: "i1", UserLabels: []Label{{Name: "vip"}}, ItemLabels: []Label{{Name: "cate_food"}}, ContextLabels: []Label{{Name: "scene_home"}}, Embeddings: []Embedding{{Name: "goods_content", Value: []float32{1, 0, 0, 0}}}},
		{UserId: "u2", ItemId: "i3", UserLabels: []Label{{Name: "new_user"}}, ItemLabels: []Label{{Name: "cate_phone"}}, ContextLabels: []Label{{Name: "scene_detail"}}, Embeddings: []Embedding{{Name: "goods_content", Value: []float32{0, 0, 1, 0}}}},
	}

	for _, item := range testCases {
		t.Run(item.name, func(t *testing.T) {
			dataset := BuildDataset(sampleList)
			trainSet, testSet := dataset.Split(0.34, 31)
			if testSet.Count() == 0 {
				testSet = dataset
			}

			model := NewAFM(Config{
				Backend:   item.backend,
				BatchSize: 2,
				Factors:   4,
				Epochs:    4,
				Verbose:   1,
				Patience:  2,
				Learning:  0.01,
				Reg:       0.0001,
				Optimizer: "adam",
				AutoScale: true,
				Seed:      31,
			})
			model.Fit(context.Background(), trainSet, testSet)
			snapshot, err := model.ExportSnapshot()
			if err != nil {
				t.Fatalf("export snapshot: %v", err)
			}
			restoredModel, err := snapshot.BuildModel()
			if err != nil {
				t.Fatalf("build model: %v", err)
			}

			originalPredictionList := model.PredictBatch(inferenceSampleList)
			restoredPredictionList := restoredModel.PredictBatch(inferenceSampleList)
			if len(originalPredictionList) != len(restoredPredictionList) {
				t.Fatalf("prediction length mismatch: %d != %d", len(originalPredictionList), len(restoredPredictionList))
			}
			for index := range originalPredictionList {
				// 快照恢复后的打分允许极小浮点误差，但不能发生语义偏移。
				if math.Abs(float64(originalPredictionList[index]-restoredPredictionList[index])) > 1e-4 {
					t.Fatalf("prediction mismatch at %d: %v != %v", index, originalPredictionList[index], restoredPredictionList[index])
				}
			}
		})
	}
}

package cf

import (
	"context"
	"testing"
)

// TestSnapshotBuildModel 验证 BPR 快照恢复后仍能输出一致推荐结果。
func TestSnapshotBuildModel(t *testing.T) {
	model := Fit(context.Background(), []Interaction{
		{UserId: "u1", ItemId: "i1", Weight: 2},
		{UserId: "u1", ItemId: "i2", Weight: 1},
		{UserId: "u2", ItemId: "i2", Weight: 2},
		{UserId: "u2", ItemId: "i3", Weight: 1},
		{UserId: "u3", ItemId: "i1", Weight: 1},
		{UserId: "u3", ItemId: "i3", Weight: 2},
	}, Config{
		Backend:  BackendNative,
		Factors:  8,
		Epochs:   20,
		Learning: 0.05,
		Reg:      0.01,
		Seed:     23,
	})
	snapshot, err := model.ExportSnapshot()
	if err != nil {
		t.Fatalf("export snapshot: %v", err)
	}
	restoredModel, err := snapshot.BuildModel()
	if err != nil {
		t.Fatalf("build model: %v", err)
	}

	originalList := model.Recommend("u1", 3, nil)
	restoredList := restoredModel.Recommend("u1", 3, nil)
	if len(originalList) != len(restoredList) {
		t.Fatalf("recommend length mismatch: %d != %d", len(originalList), len(restoredList))
	}
	for index := range originalList {
		if originalList[index].ItemId != restoredList[index].ItemId {
			t.Fatalf("recommend item mismatch at %d: %s != %s", index, originalList[index].ItemId, restoredList[index].ItemId)
		}
		if originalList[index].Score != restoredList[index].Score {
			t.Fatalf("recommend score mismatch at %d: %v != %v", index, originalList[index].Score, restoredList[index].Score)
		}
	}
}

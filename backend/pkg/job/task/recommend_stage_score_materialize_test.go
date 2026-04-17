package task

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	recommendCache "shop/pkg/recommend/cache"
	"shop/pkg/recommend/offline/materialize"
)

// TestLoadRecommendStageScoreEntryList 验证快照文件同时兼容对象和数组格式。
func TestLoadRecommendStageScoreEntryList(t *testing.T) {
	testCases := []struct {
		name    string
		payload any
	}{
		{
			name: "object",
			payload: &recommendStageScoreSnapshot{
				Entries: []*recommendStageScoreEntry{
					{Scene: 1, ActorType: 0, ActorId: 0, Documents: []recommendCache.Score{{Id: "1001", Score: 0.9}}},
				},
			},
		},
		{
			name: "array",
			payload: []*recommendStageScoreEntry{
				{Scene: 2, ActorType: 1, ActorId: 9, Documents: []recommendCache.Score{{Id: "1002", Score: 0.8}}},
			},
		},
	}

	for _, item := range testCases {
		filePath := writeRecommendStageScoreSnapshotFile(t, item.name, item.payload)
		entryList, err := loadRecommendStageScoreEntryList(filePath)
		if err != nil {
			t.Fatalf("load entry list(%s): %v", item.name, err)
		}
		if len(entryList) != 1 || entryList[0] == nil {
			t.Fatalf("unexpected entry list(%s): %+v", item.name, entryList)
		}
	}
}

// TestRecommendRankerMaterializeExec 验证模型精排快照可以发布并清理旧版本子集合。
func TestRecommendRankerMaterializeExec(t *testing.T) {
	store, cleanup, err := recommendCache.NewStore(nil)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer cleanup()

	materializer := materialize.NewMaterializer(store)
	err = materializer.MaterializeRanker(context.Background(), 9, 0, 0, "v1", []recommendCache.Score{
		{Id: "9999", Score: 0.1, Timestamp: time.Now()},
	})
	if err != nil {
		t.Fatalf("prepare stale subset: %v", err)
	}

	task := NewRecommendRankerMaterialize(nil, nil, nil, nil, nil, store, materializer)
	filePath := writeRecommendStageScoreSnapshotFile(t, "ranker", &recommendStageScoreSnapshot{
		Entries: []*recommendStageScoreEntry{
			{
				Scene:     1,
				ActorType: 0,
				ActorId:   0,
				Documents: []recommendCache.Score{
					{Id: "1002", Score: 0.7},
					{Id: "1001", Score: 0.9},
					{Id: "1001", Score: 0.8},
				},
			},
		},
	})
	_, err = task.Exec(map[string]string{
		"path":       filePath,
		"version":    "v1",
		"limit":      "2",
		"clearStale": "true",
	})
	if err != nil {
		t.Fatalf("exec ranker task: %v", err)
	}

	subset := recommendCache.RankerSubset(1, 0, 0, "v1")
	documents, err := store.SearchScores(context.Background(), recommendCache.CollectionKey(recommendCache.Ranker), subset, 0, 10)
	if err != nil {
		t.Fatalf("search published scores: %v", err)
	}
	if len(documents) != 2 {
		t.Fatalf("unexpected documents count: %d", len(documents))
	}
	if documents[0].Id != "1001" || documents[1].Id != "1002" {
		t.Fatalf("unexpected documents order: %+v", documents)
	}
	// 旧子集合不在当前全量快照里时，应当在 clearStale 阶段被清理掉。
	_, err = store.SearchScores(context.Background(), recommendCache.CollectionKey(recommendCache.Ranker), recommendCache.RankerSubset(9, 0, 0, "v1"), 0, 10)
	if err != recommendCache.ErrObjectNotExist {
		t.Fatalf("stale subset should be removed, err=%v", err)
	}
}

// TestRecommendLlmRerankMaterializeExec 验证 LLM 二次重排快照会按请求哈希发布。
func TestRecommendLlmRerankMaterializeExec(t *testing.T) {
	store, cleanup, err := recommendCache.NewStore(nil)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer cleanup()

	task := NewRecommendLlmRerankMaterialize(store, materialize.NewMaterializer(store))
	filePath := writeRecommendStageScoreSnapshotFile(t, "llm_rerank", []*recommendStageScoreEntry{
		{
			Scene:       3,
			ActorType:   1,
			ActorId:     18,
			RequestHash: "req-001",
			Documents: []recommendCache.Score{
				{Id: "2001", Score: 0.6},
			},
		},
	})
	_, err = task.Exec(map[string]string{
		"path":    filePath,
		"version": "gray",
	})
	if err != nil {
		t.Fatalf("exec llm rerank task: %v", err)
	}

	subset := recommendCache.LlmRerankSubset(3, 1, 18, "req-001", "gray")
	documents, err := store.SearchScores(context.Background(), recommendCache.CollectionKey(recommendCache.LlmRerank), subset, 0, 10)
	if err != nil {
		t.Fatalf("search llm rerank scores: %v", err)
	}
	if len(documents) != 1 || documents[0].Id != "2001" {
		t.Fatalf("unexpected documents: %+v", documents)
	}
}

// writeRecommendStageScoreSnapshotFile 写入阶段分数测试快照文件。
func writeRecommendStageScoreSnapshotFile(t *testing.T, name string, payload any) string {
	t.Helper()

	fileByte, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	filePath := filepath.Join(t.TempDir(), name+".json")
	err = os.WriteFile(filePath, fileByte, 0o600)
	if err != nil {
		t.Fatalf("write file: %v", err)
	}
	return filePath
}

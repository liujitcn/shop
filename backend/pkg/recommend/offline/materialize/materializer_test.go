package materialize

import (
	"context"
	"testing"
	"time"

	recommendCache "shop/pkg/recommend/cache"
)

// TestMaterializeRanker 验证模型精排缓存会写入正确集合和元信息。
func TestMaterializeRanker(t *testing.T) {
	store, cleanup, err := recommendCache.NewStore(nil)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer cleanup()

	materializer := NewMaterializer(store)
	now := time.Now()
	err = materializer.MaterializeRanker(context.Background(), 2, 1, 9, "gray", []recommendCache.Score{
		{Id: "1002", Score: 0.7, Timestamp: now},
		{Id: "1001", Score: 0.9, Timestamp: now.Add(time.Second)},
	})
	if err != nil {
		t.Fatalf("materialize ranker: %v", err)
	}

	subset := recommendCache.RankerSubset(2, 1, 9, "gray")
	documents, err := store.SearchScores(context.Background(), recommendCache.CollectionKey(recommendCache.Ranker), subset, 0, 10)
	if err != nil {
		t.Fatalf("search scores: %v", err)
	}
	if len(documents) != 2 {
		t.Fatalf("unexpected documents count: %d", len(documents))
	}
	if documents[0].Id != "1001" || documents[1].Id != "1002" {
		t.Fatalf("unexpected documents order: %+v", documents)
	}

	digest, err := store.Get(recommendCache.DigestKey(recommendCache.Ranker, subset))
	if err != nil {
		t.Fatalf("load digest: %v", err)
	}
	if digest == "" {
		t.Fatalf("digest should not be empty")
	}
	documentCount, err := store.Get(recommendCache.DocumentCountKey(recommendCache.Ranker, subset))
	if err != nil {
		t.Fatalf("load document count: %v", err)
	}
	if documentCount != "2" {
		t.Fatalf("unexpected document count: %s", documentCount)
	}
}

// TestMaterializeLlmRerank 验证 LLM 二次重排缓存会写入请求哈希子集合。
func TestMaterializeLlmRerank(t *testing.T) {
	store, cleanup, err := recommendCache.NewStore(nil)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer cleanup()

	materializer := NewMaterializer(store)
	err = materializer.MaterializeLlmRerank(context.Background(), 4, 0, 0, "request-abc", "v2", []recommendCache.Score{
		{Id: "3001", Score: 0.4, Timestamp: time.Now()},
	})
	if err != nil {
		t.Fatalf("materialize llm rerank: %v", err)
	}

	subset := recommendCache.LlmRerankSubset(4, 0, 0, "request-abc", "v2")
	documents, err := store.SearchScores(context.Background(), recommendCache.CollectionKey(recommendCache.LlmRerank), subset, 0, 10)
	if err != nil {
		t.Fatalf("search scores: %v", err)
	}
	if len(documents) != 1 || documents[0].Id != "3001" {
		t.Fatalf("unexpected documents: %+v", documents)
	}
}

// TestMaterializeRecommend 验证最终推荐结果缓存会写入用户子集合。
func TestMaterializeRecommend(t *testing.T) {
	store, cleanup, err := recommendCache.NewStore(nil)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer cleanup()

	materializer := NewMaterializer(store)
	err = materializer.MaterializeRecommend(context.Background(), 1, 1, 28, "v3", []recommendCache.Score{
		{Id: "5002", Score: 1, Timestamp: time.Time{}},
		{Id: "5001", Score: 2, Timestamp: time.Time{}},
	})
	if err != nil {
		t.Fatalf("materialize recommend: %v", err)
	}

	subset := recommendCache.RecommendSubset(1, 1, 28, "v3")
	documents, err := store.SearchScores(context.Background(), recommendCache.CollectionKey(recommendCache.Recommend), subset, 0, 10)
	if err != nil {
		t.Fatalf("search scores: %v", err)
	}
	if len(documents) != 2 {
		t.Fatalf("unexpected documents count: %d", len(documents))
	}
	if documents[0].Id != "5001" || documents[1].Id != "5002" {
		t.Fatalf("unexpected documents order: %+v", documents)
	}
}

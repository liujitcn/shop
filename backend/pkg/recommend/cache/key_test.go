package cache

import "testing"

// TestLlmRerankSubset 验证 LLM 重排子集合键的拼接结果。
func TestLlmRerankSubset(t *testing.T) {
	subset := LlmRerankSubset(3, 1, 9, " request/hash ", " v2 ")
	expected := "scene/3/actor_type/1/actor_id/9/request_hash/request_hash/version/v2"
	if subset != expected {
		t.Fatalf("unexpected llm rerank subset: %s", subset)
	}
}

// TestRankerSubset 验证模型精排子集合键的拼接结果。
func TestRankerSubset(t *testing.T) {
	subset := RankerSubset(2, 0, 18, " gray ")
	expected := "scene/2/actor_type/0/actor_id/18/version/gray"
	if subset != expected {
		t.Fatalf("unexpected ranker subset: %s", subset)
	}
}

// TestRecommendSubset 验证最终推荐结果子集合键的拼接结果。
func TestRecommendSubset(t *testing.T) {
	subset := RecommendSubset(1, 1, 28, " v3 ")
	expected := "scene/1/actor_type/1/actor_id/28/version/v3"
	if subset != expected {
		t.Fatalf("unexpected recommend subset: %s", subset)
	}
}

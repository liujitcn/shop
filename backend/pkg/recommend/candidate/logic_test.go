package candidate

import "testing"

// TestResolveCandidateLimitExpandsForNextPage 验证翻到下一页时会继续放大候选池。
func TestResolveCandidateLimitExpandsForNextPage(t *testing.T) {
	limit := ResolveCandidateLimit(2, 10)
	if limit != 160 {
		t.Fatalf("unexpected candidate limit: %d", limit)
	}
}

// TestResolveCandidateLimitRespectsMaxLimit 验证候选池大小仍会被最大值限制。
func TestResolveCandidateLimitRespectsMaxLimit(t *testing.T) {
	limit := ResolveCandidateLimit(1, 40)
	if limit != PoolMax {
		t.Fatalf("unexpected candidate limit: %d", limit)
	}
}

// TestResolveCandidateLimitSupportsDeepPagination 验证深分页超过软上限后仍可继续扩池。
func TestResolveCandidateLimitSupportsDeepPagination(t *testing.T) {
	limit := ResolveCandidateLimit(30, 10)
	if limit != 300 {
		t.Fatalf("unexpected candidate limit: %d", limit)
	}
}

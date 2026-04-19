package candidate

import (
	"testing"

	app "shop/api/gen/go/app"
	recommendCore "shop/pkg/recommend/core"
)

// TestResolveCandidateLimitExpandsForNextPage 验证浅分页会优先复用同一批候选池。
func TestResolveCandidateLimitExpandsForNextPage(t *testing.T) {
	limit := ResolveCandidateLimit(2, 10)
	if limit != PoolMax {
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

// TestRankGoodsKeepsDeterministicOrder 验证并列候选会按商品编号稳定打平。
func TestRankGoodsKeepsDeterministicOrder(t *testing.T) {
	candidates := map[int64]*recommendCore.Candidate{
		101: {Goods: &app.GoodsInfo{Id: 101}, FinalScore: 1},
		103: {Goods: &app.GoodsInfo{Id: 103}, FinalScore: 1},
		102: {Goods: &app.GoodsInfo{Id: 102}, FinalScore: 1},
	}

	rankedGoods := RankGoods(candidates)
	if len(rankedGoods) != 3 {
		t.Fatalf("unexpected ranked goods length: %d", len(rankedGoods))
	}
	if rankedGoods[0].Id != 103 || rankedGoods[1].Id != 102 || rankedGoods[2].Id != 101 {
		t.Fatalf("unexpected ranked goods order: %+v", rankedGoods)
	}
}

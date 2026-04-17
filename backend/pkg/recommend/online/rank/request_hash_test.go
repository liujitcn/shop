package rank

import (
	"testing"

	"shop/api/gen/go/common"
	recommendDomain "shop/pkg/recommend/domain"
)

// TestBuildRerankRequestHash 验证重排请求哈希会受主体和候选商品变化影响。
func TestBuildRerankRequestHash(t *testing.T) {
	request := &recommendDomain.GoodsRequest{
		Scene:    common.RecommendScene_GOODS_DETAIL,
		GoodsId:  88,
		PageNum:  1,
		PageSize: 10,
	}
	actor := &recommendDomain.Actor{
		ActorType: 1,
		ActorId:   99,
	}
	strategy := &recommendDomain.LlmRerankStrategy{
		Model:          "gpt-4o-mini",
		PromptTemplate: "goods={{ candidatesJson }}",
	}

	hashValue := BuildRerankRequestHash(request, actor, strategy, []int64{7, 8, 9}, 2)
	if hashValue == "" {
		t.Fatalf("expected non-empty rerank hash")
	}

	otherHashValue := BuildRerankRequestHash(request, actor, strategy, []int64{7, 10, 9}, 2)
	if hashValue == otherHashValue {
		t.Fatalf("expected rerank hash changed when candidate snapshot changed")
	}

	otherStrategyHashValue := BuildRerankRequestHash(request, actor, &recommendDomain.LlmRerankStrategy{
		Model:          "gpt-4o-mini",
		PromptTemplate: "goods={{ candidatesJson }}\nscene={{ request.scene }}",
	}, []int64{7, 8, 9}, 2)
	if hashValue == otherStrategyHashValue {
		t.Fatalf("expected rerank hash changed when strategy snapshot changed")
	}
}

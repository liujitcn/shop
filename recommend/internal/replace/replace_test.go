package replace

import (
	"recommend/contract"
	"recommend/internal/core"
	"recommend/internal/model"
	"testing"
)

func TestFilterUnavailableGoods(t *testing.T) {
	candidates := []*model.Candidate{
		model.BuildCandidate(&contract.Goods{Id: 1, OnSale: true, InStock: true}),
		model.BuildCandidate(&contract.Goods{Id: 2, OnSale: false, InStock: true}),
		model.BuildCandidate(&contract.Goods{Id: 3, OnSale: true, InStock: false}),
	}

	result := FilterUnavailableGoods(candidates)
	if len(result) != 1 || result[0].GoodsId() != 1 {
		t.Fatalf("过滤上下架和库存结果不符合预期: %+v", result)
	}
}

func TestFilterContextGoods(t *testing.T) {
	request := model.ResolveRequest(core.RecommendRequest{
		Context: core.RecommendContext{
			GoodsId:      1,
			CartGoodsIds: []int64{3},
		},
	})
	candidates := []*model.Candidate{
		model.BuildCandidate(&contract.Goods{Id: 1, CategoryId: 11}),
		model.BuildCandidate(&contract.Goods{Id: 2, CategoryId: 12}),
		model.BuildCandidate(&contract.Goods{Id: 3, CategoryId: 13}),
	}

	result := FilterContextGoods(request, candidates)
	if len(result) != 1 || result[0].GoodsId() != 2 {
		t.Fatalf("过滤上下文商品结果不符合预期: %+v", result)
	}
}

func TestApplyPenalty(t *testing.T) {
	first := model.BuildCandidate(&contract.Goods{Id: 1})
	second := model.BuildCandidate(&contract.Goods{Id: 2})
	candidates := []*model.Candidate{first, second}

	ApplyExposurePenalty(candidates, map[int64]float64{1: 0.4})
	ApplyRepeatPenalty(candidates, []int64{2}, 0.6)

	if first.Score.ExposurePenalty != 0.4 {
		t.Fatalf("曝光惩罚不符合预期: %+v", first.Score)
	}
	if second.Score.RepeatPenalty != 0.6 {
		t.Fatalf("重复购买惩罚不符合预期: %+v", second.Score)
	}
}

func TestMergeFallback(t *testing.T) {
	first := model.BuildCandidate(&contract.Goods{Id: 1})
	second := model.BuildCandidate(&contract.Goods{Id: 2})
	third := model.BuildCandidate(&contract.Goods{Id: 3})

	result := MergeFallback([]*model.Candidate{first}, []*model.Candidate{first, second, third}, 2)
	if len(result) != 2 {
		t.Fatalf("兜底补足结果数量不符合预期: %d", len(result))
	}
	if result[0].GoodsId() != 1 || result[1].GoodsId() != 2 {
		t.Fatalf("兜底补足结果顺序不符合预期: %+v", result)
	}
}

func TestDiversifyByCategory(t *testing.T) {
	first := model.BuildCandidate(&contract.Goods{Id: 1, CategoryId: 11})
	second := model.BuildCandidate(&contract.Goods{Id: 2, CategoryId: 11})
	third := model.BuildCandidate(&contract.Goods{Id: 3, CategoryId: 12})

	result := DiversifyByCategory([]*model.Candidate{first, second, third}, 1)
	if len(result) != 3 {
		t.Fatalf("类目打散结果数量不符合预期: %d", len(result))
	}
	if result[0].GoodsId() != 1 || result[1].GoodsId() != 3 || result[2].GoodsId() != 2 {
		t.Fatalf("类目打散结果顺序不符合预期: %+v", result)
	}
}

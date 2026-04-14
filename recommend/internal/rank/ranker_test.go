package rank

import (
	"recommend/contract"
	"recommend/internal/model"
	"testing"
	"time"
)

func TestRankCandidates(t *testing.T) {
	first := model.BuildCandidate(&contract.Goods{
		Id:         1,
		CategoryId: 10,
		UpdatedAt:  time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC),
	})
	first.Score.FinalScore = 10

	second := model.BuildCandidate(&contract.Goods{
		Id:         2,
		CategoryId: 10,
		UpdatedAt:  time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC),
	})
	second.Score.FinalScore = 9

	third := model.BuildCandidate(&contract.Goods{
		Id:         3,
		CategoryId: 20,
		UpdatedAt:  time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC),
	})
	third.Score.FinalScore = 8

	result := RankCandidates([]*model.Candidate{second, third, first}, RankOptions{MaxPerCategory: 1})

	if len(result) != 3 {
		t.Fatalf("排序结果数量不符合预期: %d", len(result))
	}
	if result[0].GoodsId() != 1 {
		t.Fatalf("排序第一位不符合预期: %+v", result[0])
	}
	if result[1].GoodsId() != 3 {
		t.Fatalf("类目打散结果不符合预期: %+v", result)
	}
	if result[2].GoodsId() != 2 {
		t.Fatalf("溢出候选位置不符合预期: %+v", result)
	}
}

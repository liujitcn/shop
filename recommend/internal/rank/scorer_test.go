package rank

import (
	"recommend/contract"
	"recommend/internal/model"
	"testing"
	"time"
)

func TestScoreCandidate(t *testing.T) {
	candidate := model.BuildCandidate(&contract.Goods{
		Id:        1001,
		CreatedAt: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC),
	})
	candidate.Score.RelationScore = 2
	candidate.Score.UserGoodsScore = 3
	candidate.Score.CategoryScore = 1
	candidate.Score.SceneHotScore = 4
	candidate.Score.GlobalHotScore = 2
	candidate.Score.SessionScore = 1
	candidate.Score.ExternalScore = 1
	candidate.Score.CollaborativeScore = 0.5
	candidate.Score.UserNeighborScore = 0.5
	candidate.Score.ExposurePenalty = 0.3
	candidate.Score.RepeatPenalty = 0.2

	weights := ScoreWeights{
		RelationWeight:      0.4,
		UserGoodsWeight:     0.2,
		CategoryWeight:      0.1,
		SceneHotWeight:      0.1,
		GlobalHotWeight:     0.05,
		FreshnessWeight:     0.05,
		SessionWeight:       0.03,
		ExternalWeight:      0.02,
		CollaborativeWeight: 0.02,
		UserNeighborWeight:  0.01,
		ExposurePenalty:     1,
		RepeatPenalty:       1,
	}

	scoreTime := time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC)
	score := ScoreCandidate(candidate, weights, scoreTime)

	if score <= 0 {
		t.Fatalf("候选商品得分不符合预期: %v", score)
	}
	if candidate.Score.FreshnessScore <= 0 {
		t.Fatalf("新鲜度得分未正确计算: %+v", candidate.Score)
	}
	if candidate.Score.FinalScore != score {
		t.Fatalf("最终得分未正确回填: %+v", candidate.Score)
	}
}

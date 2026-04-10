package candidate

import (
	"testing"
	"time"

	"shop/pkg/gen/models"
)

func TestResolveCandidateLimit(t *testing.T) {
	if got := ResolveCandidateLimit(1, 1); got != PoolMin {
		t.Fatalf("ResolveCandidateLimit(1,1) = %d, want %d", got, PoolMin)
	}
	if got := ResolveCandidateLimit(10, 10); got != PoolMax {
		t.Fatalf("ResolveCandidateLimit(10,10) = %d, want %d", got, PoolMax)
	}
}

func TestBuildPersonalized(t *testing.T) {
	now := time.Now()
	goodsList := []*models.GoodsInfo{
		{ID: 1, CategoryID: 11, CreatedAt: now},
		{ID: 2, CategoryID: 12, CreatedAt: now.AddDate(0, 0, -40)},
	}

	candidates := BuildPersonalized(goodsList, PersonalizedSignals{
		RelationScores:         map[int64]float64{1: 2},
		UserGoodsScores:        map[int64]float64{1: 1},
		ProfileScores:          map[int64]float64{11: 3},
		ScenePopularityScores:  map[int64]float64{1: 4},
		GlobalPopularityScores: map[int64]float64{1: 5},
		RecentPaidGoods:        map[int64]struct{}{2: {}},
	})

	if _, ok := candidates[1].RecallSources[RecallSourceRelation]; !ok {
		t.Fatalf("candidate 1 missing %s", RecallSourceRelation)
	}
	if _, ok := candidates[1].RecallSources[RecallSourceProfile]; !ok {
		t.Fatalf("candidate 1 missing %s", RecallSourceProfile)
	}
	if candidates[2].RepeatPenalty == 0 {
		t.Fatalf("candidate 2 repeat penalty = 0, want > 0")
	}
	if _, ok := candidates[2].RecallSources[RecallSourceLatest]; !ok {
		t.Fatalf("candidate 2 missing %s", RecallSourceLatest)
	}
}

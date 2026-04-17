package rank

import (
	"testing"

	app "shop/api/gen/go/app"
	"shop/api/gen/go/common"
	"shop/api/gen/go/conf"
	recommendDomain "shop/pkg/recommend/domain"
)

// TestExecuteAnonymousRanking 验证匿名态排序桥接结果。
func TestExecuteAnonymousRanking(t *testing.T) {
	request := &recommendDomain.GoodsRequest{
		Scene:    common.RecommendScene_GOODS_DETAIL,
		PageNum:  1,
		PageSize: 2,
	}
	goodsList := []*app.GoodsInfo{
		{Id: 1, CategoryId: 11, UpdatedAt: "2026-04-16 10:00:00"},
		{Id: 2, CategoryId: 12, UpdatedAt: "2026-04-15 10:00:00"},
	}
	signals := recommendDomain.AnonymousSignals{
		RelationScores:         map[int64]float64{1: 2, 2: 1},
		ScenePopularityScores:  map[int64]float64{},
		GlobalPopularityScores: map[int64]float64{},
		SceneExposurePenalties: map[int64]float64{},
		ActorExposurePenalties: map[int64]float64{},
	}
	rankConfig := &conf.GoodsRecommendAnonymousRankWeightConfig{
		RelationWeight:             float64Ptr(1),
		ScenePopularityWeight:      float64Ptr(0),
		GlobalPopularityWeight:     float64Ptr(0),
		FreshnessWeight:            float64Ptr(0),
		ExposurePenaltyWeight:      float64Ptr(0),
		ActorExposurePenaltyWeight: float64Ptr(0),
	}

	result := ExecuteAnonymousRanking(request, goodsList, signals, rankConfig, nil, nil, StageScoreSet{})

	if result.PageSnapshot.Total != 2 || result.PageSnapshot.IsEmptyPage {
		t.Fatalf("unexpected page snapshot: %+v", result.PageSnapshot)
	}
	if len(result.PageSnapshot.PageGoods) != 2 || result.PageSnapshot.PageGoods[0].Id != 1 {
		t.Fatalf("unexpected page goods: %+v", result.PageSnapshot.PageGoods)
	}
	if len(result.ExplainSnapshot.ScoreDetails) != 2 {
		t.Fatalf("unexpected explain snapshot: %+v", result.ExplainSnapshot)
	}
}

// TestExecutePersonalizedRanking 验证登录态排序桥接结果。
func TestExecutePersonalizedRanking(t *testing.T) {
	request := &recommendDomain.GoodsRequest{
		Scene:    common.RecommendScene_CART,
		PageNum:  1,
		PageSize: 1,
	}
	goodsList := []*app.GoodsInfo{
		{Id: 1, CategoryId: 11, UpdatedAt: "2026-04-15 10:00:00"},
		{Id: 2, CategoryId: 12, UpdatedAt: "2026-04-16 10:00:00"},
	}
	signals := recommendDomain.PersonalizedSignals{
		RelationScores:         map[int64]float64{},
		UserGoodsScores:        map[int64]float64{2: 3},
		SimilarUserScores:      map[int64]float64{1: 4},
		ProfileScores:          map[int64]float64{},
		ScenePopularityScores:  map[int64]float64{},
		GlobalPopularityScores: map[int64]float64{},
		SceneExposurePenalties: map[int64]float64{},
		ActorExposurePenalties: map[int64]float64{},
		RecentPaidGoods:        map[int64]struct{}{},
	}
	rankConfig := &conf.GoodsRecommendPersonalizedRankWeightConfig{
		RelationWeight:             float64Ptr(0),
		UserGoodsWeight:            float64Ptr(1),
		SimilarUserWeight:          float64Ptr(1),
		ProfileWeight:              float64Ptr(0),
		ScenePopularityWeight:      float64Ptr(0),
		GlobalPopularityWeight:     float64Ptr(0),
		FreshnessWeight:            float64Ptr(0),
		ExposurePenaltyWeight:      float64Ptr(0),
		ActorExposurePenaltyWeight: float64Ptr(0),
		RepeatPenaltyWeight:        float64Ptr(0),
	}

	result := ExecutePersonalizedRanking(request, goodsList, signals, rankConfig, nil, nil, StageScoreSet{})

	if result.PageSnapshot.Total != 2 || result.PageSnapshot.IsEmptyPage {
		t.Fatalf("unexpected page snapshot: %+v", result.PageSnapshot)
	}
	if len(result.PageSnapshot.PageGoods) != 1 || result.PageSnapshot.PageGoods[0].Id != 1 {
		t.Fatalf("unexpected page goods: %+v", result.PageSnapshot.PageGoods)
	}
	if len(result.ExplainSnapshot.ReturnedGoodsIds) != 1 || result.ExplainSnapshot.ReturnedGoodsIds[0] != 1 {
		t.Fatalf("unexpected explain snapshot: %+v", result.ExplainSnapshot)
	}
	if len(result.ExplainSnapshot.ScoreDetails) != 1 || result.ExplainSnapshot.ScoreDetails[0].SimilarUserScore != 4 {
		t.Fatalf("unexpected similar user score detail: %+v", result.ExplainSnapshot.ScoreDetails)
	}
}

// TestExecutePersonalizedRankingWithStageScores 验证模型精排与 LLM 重排会改写最终排序。
func TestExecutePersonalizedRankingWithStageScores(t *testing.T) {
	request := &recommendDomain.GoodsRequest{
		Scene:    common.RecommendScene_CART,
		PageNum:  1,
		PageSize: 1,
	}
	goodsList := []*app.GoodsInfo{
		{Id: 1, CategoryId: 11, UpdatedAt: "2026-04-15 10:00:00"},
		{Id: 2, CategoryId: 12, UpdatedAt: "2026-04-16 10:00:00"},
	}
	signals := recommendDomain.PersonalizedSignals{
		RelationScores:         map[int64]float64{},
		UserGoodsScores:        map[int64]float64{1: 1, 2: 2},
		SimilarUserScores:      map[int64]float64{},
		ProfileScores:          map[int64]float64{},
		ScenePopularityScores:  map[int64]float64{},
		GlobalPopularityScores: map[int64]float64{},
		SceneExposurePenalties: map[int64]float64{},
		ActorExposurePenalties: map[int64]float64{},
		RecentPaidGoods:        map[int64]struct{}{},
	}
	rankConfig := &conf.GoodsRecommendPersonalizedRankWeightConfig{
		RelationWeight:             float64Ptr(0),
		UserGoodsWeight:            float64Ptr(1),
		SimilarUserWeight:          float64Ptr(0),
		ProfileWeight:              float64Ptr(0),
		ScenePopularityWeight:      float64Ptr(0),
		GlobalPopularityWeight:     float64Ptr(0),
		FreshnessWeight:            float64Ptr(0),
		ExposurePenaltyWeight:      float64Ptr(0),
		ActorExposurePenaltyWeight: float64Ptr(0),
		RepeatPenaltyWeight:        float64Ptr(0),
	}
	strategy := &recommendDomain.StrategyVersionConfig{
		Ranker: &recommendDomain.RankerStrategy{
			Enabled: true,
			Type:    recommendDomain.RankerTypeFM,
			TopN:    2,
			Weight:  2,
		},
		LlmRerank: &recommendDomain.LlmRerankStrategy{
			Enabled: true,
			Model:   "gpt-rerank",
			TopN:    2,
			Weight:  3,
		},
	}
	stageScores := StageScoreSet{
		RankerScores: map[int64]float64{1: 1.2},
		LlmScores:    map[int64]float64{1: 0.8},
	}

	result := ExecutePersonalizedRanking(request, goodsList, signals, rankConfig, nil, strategy, stageScores)

	if len(result.PageSnapshot.PageGoods) != 1 || result.PageSnapshot.PageGoods[0].Id != 1 {
		t.Fatalf("unexpected stage ranking goods: %+v", result.PageSnapshot.PageGoods)
	}
	if len(result.ExplainSnapshot.ScoreDetails) != 1 {
		t.Fatalf("unexpected stage explain snapshot: %+v", result.ExplainSnapshot)
	}
	scoreDetail := result.ExplainSnapshot.ScoreDetails[0]
	if scoreDetail.RuleScore <= 0 || scoreDetail.ModelScore <= 0 || scoreDetail.LlmScore <= 0 {
		t.Fatalf("unexpected stage score detail: %+v", scoreDetail)
	}
	appliedStages, ok := result.StageContext["appliedStages"].([]string)
	if !ok || len(appliedStages) != 3 {
		t.Fatalf("unexpected stage context: %+v", result.StageContext)
	}
}

// float64Ptr 返回 float64 指针，便于测试 optional 字段。
func float64Ptr(value float64) *float64 {
	return &value
}

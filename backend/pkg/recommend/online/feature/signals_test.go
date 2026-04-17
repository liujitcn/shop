package feature

import "testing"

// TestBuildAnonymousSignals 验证匿名态信号装配结果。
func TestBuildAnonymousSignals(t *testing.T) {
	relationScores := map[int64]float64{1: 1.2}
	scenePopularityScores := map[int64]float64{2: 2.3}
	globalPopularityScores := map[int64]float64{3: 3.4}
	sceneExposurePenalties := map[int64]float64{4: 0.4}
	actorExposurePenalties := map[int64]float64{5: 0.5}

	signals := BuildAnonymousSignals(
		relationScores,
		scenePopularityScores,
		globalPopularityScores,
		sceneExposurePenalties,
		actorExposurePenalties,
	)

	if signals.RelationScores[1] != 1.2 {
		t.Fatalf("unexpected relation score: %+v", signals.RelationScores)
	}
	if signals.ScenePopularityScores[2] != 2.3 {
		t.Fatalf("unexpected scene popularity scores: %+v", signals.ScenePopularityScores)
	}
	if signals.GlobalPopularityScores[3] != 3.4 {
		t.Fatalf("unexpected global popularity scores: %+v", signals.GlobalPopularityScores)
	}
	if signals.SceneExposurePenalties[4] != 0.4 {
		t.Fatalf("unexpected scene exposure penalties: %+v", signals.SceneExposurePenalties)
	}
	if signals.ActorExposurePenalties[5] != 0.5 {
		t.Fatalf("unexpected actor exposure penalties: %+v", signals.ActorExposurePenalties)
	}
}

// TestBuildPersonalizedSignals 验证登录态信号装配结果。
func TestBuildPersonalizedSignals(t *testing.T) {
	relationScores := map[int64]float64{1: 1.2}
	userGoodsScores := map[int64]float64{2: 2.3}
	similarUserScores := map[int64]float64{3: 3.4}
	profileScores := map[int64]float64{4: 4.5}
	scenePopularityScores := map[int64]float64{5: 5.6}
	globalPopularityScores := map[int64]float64{6: 6.7}
	sceneExposurePenalties := map[int64]float64{7: 0.6}
	actorExposurePenalties := map[int64]float64{8: 0.7}
	recentPaidGoods := map[int64]struct{}{9: {}}

	signals := BuildPersonalizedSignals(
		relationScores,
		userGoodsScores,
		similarUserScores,
		profileScores,
		scenePopularityScores,
		globalPopularityScores,
		sceneExposurePenalties,
		actorExposurePenalties,
		recentPaidGoods,
	)

	if signals.RelationScores[1] != 1.2 {
		t.Fatalf("unexpected relation score: %+v", signals.RelationScores)
	}
	if signals.UserGoodsScores[2] != 2.3 {
		t.Fatalf("unexpected user goods scores: %+v", signals.UserGoodsScores)
	}
	if signals.SimilarUserScores[3] != 3.4 {
		t.Fatalf("unexpected similar user scores: %+v", signals.SimilarUserScores)
	}
	if signals.ProfileScores[4] != 4.5 {
		t.Fatalf("unexpected profile scores: %+v", signals.ProfileScores)
	}
	if signals.ScenePopularityScores[5] != 5.6 {
		t.Fatalf("unexpected scene popularity scores: %+v", signals.ScenePopularityScores)
	}
	if signals.GlobalPopularityScores[6] != 6.7 {
		t.Fatalf("unexpected global popularity scores: %+v", signals.GlobalPopularityScores)
	}
	if signals.SceneExposurePenalties[7] != 0.6 {
		t.Fatalf("unexpected scene exposure penalties: %+v", signals.SceneExposurePenalties)
	}
	if signals.ActorExposurePenalties[8] != 0.7 {
		t.Fatalf("unexpected actor exposure penalties: %+v", signals.ActorExposurePenalties)
	}
	if _, ok := signals.RecentPaidGoods[9]; !ok {
		t.Fatalf("unexpected recent paid goods: %+v", signals.RecentPaidGoods)
	}
}

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
	profileScores := map[int64]float64{3: 3.4}
	scenePopularityScores := map[int64]float64{4: 4.5}
	globalPopularityScores := map[int64]float64{5: 5.6}
	sceneExposurePenalties := map[int64]float64{6: 0.6}
	actorExposurePenalties := map[int64]float64{7: 0.7}
	recentPaidGoods := map[int64]struct{}{8: {}}

	signals := BuildPersonalizedSignals(
		relationScores,
		userGoodsScores,
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
	if signals.ProfileScores[3] != 3.4 {
		t.Fatalf("unexpected profile scores: %+v", signals.ProfileScores)
	}
	if signals.ScenePopularityScores[4] != 4.5 {
		t.Fatalf("unexpected scene popularity scores: %+v", signals.ScenePopularityScores)
	}
	if signals.GlobalPopularityScores[5] != 5.6 {
		t.Fatalf("unexpected global popularity scores: %+v", signals.GlobalPopularityScores)
	}
	if signals.SceneExposurePenalties[6] != 0.6 {
		t.Fatalf("unexpected scene exposure penalties: %+v", signals.SceneExposurePenalties)
	}
	if signals.ActorExposurePenalties[7] != 0.7 {
		t.Fatalf("unexpected actor exposure penalties: %+v", signals.ActorExposurePenalties)
	}
	if _, ok := signals.RecentPaidGoods[8]; !ok {
		t.Fatalf("unexpected recent paid goods: %+v", signals.RecentPaidGoods)
	}
}

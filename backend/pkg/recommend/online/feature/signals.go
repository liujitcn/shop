package feature

import recommendDomain "shop/pkg/recommend/domain"

// BuildAnonymousSignals 构建匿名态排序所需的领域信号对象。
func BuildAnonymousSignals(
	relationScores map[int64]float64,
	scenePopularityScores map[int64]float64,
	globalPopularityScores map[int64]float64,
	sceneExposurePenalties map[int64]float64,
	actorExposurePenalties map[int64]float64,
) recommendDomain.AnonymousSignals {
	return recommendDomain.AnonymousSignals{
		RelationScores:         relationScores,
		ScenePopularityScores:  scenePopularityScores,
		GlobalPopularityScores: globalPopularityScores,
		SceneExposurePenalties: sceneExposurePenalties,
		ActorExposurePenalties: actorExposurePenalties,
	}
}

// BuildPersonalizedSignals 构建登录态排序所需的领域信号对象。
func BuildPersonalizedSignals(
	relationScores map[int64]float64,
	userGoodsScores map[int64]float64,
	profileScores map[int64]float64,
	scenePopularityScores map[int64]float64,
	globalPopularityScores map[int64]float64,
	sceneExposurePenalties map[int64]float64,
	actorExposurePenalties map[int64]float64,
	recentPaidGoods map[int64]struct{},
) recommendDomain.PersonalizedSignals {
	return recommendDomain.PersonalizedSignals{
		RelationScores:         relationScores,
		UserGoodsScores:        userGoodsScores,
		ProfileScores:          profileScores,
		ScenePopularityScores:  scenePopularityScores,
		GlobalPopularityScores: globalPopularityScores,
		SceneExposurePenalties: sceneExposurePenalties,
		ActorExposurePenalties: actorExposurePenalties,
		RecentPaidGoods:        recentPaidGoods,
	}
}

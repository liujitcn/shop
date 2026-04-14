package rank

import (
	"recommend/internal/core"
	"recommend/internal/model"
)

// DefaultWeights 返回指定场景的默认规则排序权重。
func DefaultWeights(scene model.Scene) core.ScoreWeights {
	return ResolveWeights(scene, core.RankingConfig{})
}

// ResolveWeights 返回指定场景在当前配置下的排序权重。
func ResolveWeights(scene model.Scene, config core.RankingConfig) core.ScoreWeights {
	weightsByScene := core.DefaultSceneWeights()
	for sceneName, weights := range config.SceneWeights {
		weightsByScene[sceneName] = weights
	}

	sceneWeights, ok := weightsByScene[core.Scene(scene)]
	if ok {
		return sceneWeights
	}

	return core.ScoreWeights{
		RelationWeight:      0.20,
		UserGoodsWeight:     0.15,
		CategoryWeight:      0.10,
		SceneHotWeight:      0.12,
		GlobalHotWeight:     0.08,
		FreshnessWeight:     0.08,
		SessionWeight:       0.06,
		ExternalWeight:      0.08,
		CollaborativeWeight: 0.06,
		UserNeighborWeight:  0.05,
		VectorWeight:        0.08,
		ExposurePenalty:     1.0,
		RepeatPenalty:       1.0,
	}
}

package rank

// ScoreWeights 表示推荐排序使用的权重配置。
type ScoreWeights struct {
	RelationWeight      float64
	UserGoodsWeight     float64
	CategoryWeight      float64
	SceneHotWeight      float64
	GlobalHotWeight     float64
	FreshnessWeight     float64
	SessionWeight       float64
	ExternalWeight      float64
	CollaborativeWeight float64
	UserNeighborWeight  float64
	ExposurePenalty     float64
	RepeatPenalty       float64
}

// RankOptions 表示排序阶段的可配置参数。
type RankOptions struct {
	MaxPerCategory int
}

const defaultMaxPerCategory = 2

package core

import (
	"shop/api/gen/go/app"
)

// Candidate 表示推荐引擎内部流转的候选商品。
type Candidate struct {
	Goods                 *app.GoodsInfo
	RelationScore         float64
	UserGoodsScore        float64
	ProfileScore          float64
	ScenePopularityScore  float64
	GlobalPopularityScore float64
	FreshnessScore        float64
	ExposurePenalty       float64
	ActorExposurePenalty  float64
	RepeatPenalty         float64
	FinalScore            float64
	RecallSources         map[string]struct{}
}

// ScoreDetail 表示推荐结果的评分明细。
type ScoreDetail struct {
	GoodsId               int64    `json:"goodsId"`
	FinalScore            float64  `json:"finalScore"`
	RelationScore         float64  `json:"relationScore,omitempty"`
	UserGoodsScore        float64  `json:"userGoodsScore,omitempty"`
	ProfileScore          float64  `json:"profileScore,omitempty"`
	ScenePopularityScore  float64  `json:"scenePopularityScore,omitempty"`
	GlobalPopularityScore float64  `json:"globalPopularityScore,omitempty"`
	FreshnessScore        float64  `json:"freshnessScore,omitempty"`
	ExposurePenalty       float64  `json:"exposurePenalty,omitempty"`
	ActorExposurePenalty  float64  `json:"actorExposurePenalty,omitempty"`
	RepeatPenalty         float64  `json:"repeatPenalty,omitempty"`
	RecallSources         []string `json:"recallSources,omitempty"`
}

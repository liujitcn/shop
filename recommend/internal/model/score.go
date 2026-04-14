package model

import "recommend"

// Score 表示候选商品的评分信号。
type Score struct {
	RelationScore      float64
	UserGoodsScore     float64
	CategoryScore      float64
	SceneHotScore      float64
	GlobalHotScore     float64
	FreshnessScore     float64
	SessionScore       float64
	ExternalScore      float64
	CollaborativeScore float64
	UserNeighborScore  float64
	ExposurePenalty    float64
	RepeatPenalty      float64
	FinalScore         float64
}

// ToRecommendScoreDetail 转换为对外评分明细结构。
func (s Score) ToRecommendScoreDetail(goodsId int64, recallSources []string) recommend.ScoreDetail {
	return recommend.ScoreDetail{
		GoodsId:         goodsId,
		FinalScore:      s.FinalScore,
		RelationScore:   s.RelationScore,
		UserGoodsScore:  s.UserGoodsScore,
		CategoryScore:   s.CategoryScore,
		SceneHotScore:   s.SceneHotScore,
		GlobalHotScore:  s.GlobalHotScore,
		FreshnessScore:  s.FreshnessScore,
		ExposurePenalty: s.ExposurePenalty,
		RepeatPenalty:   s.RepeatPenalty,
		RecallSources:   recallSources,
	}
}

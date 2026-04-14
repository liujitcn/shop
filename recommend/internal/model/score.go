package model

import "recommend/internal/core"

// Score 表示候选商品的评分信号。
type Score struct {
	// RelationScore 表示商品关联召回带来的得分。
	RelationScore float64
	// UserGoodsScore 表示用户商品偏好召回带来的得分。
	UserGoodsScore float64
	// CategoryScore 表示用户类目偏好召回带来的得分。
	CategoryScore float64
	// SceneHotScore 表示场景热销召回带来的得分。
	SceneHotScore float64
	// GlobalHotScore 表示全站热销召回带来的得分。
	GlobalHotScore float64
	// FreshnessScore 表示商品新鲜度得分。
	FreshnessScore float64
	// SessionScore 表示会话上下文召回带来的得分。
	SessionScore float64
	// ExternalScore 表示外部推荐池召回带来的得分。
	ExternalScore float64
	// CollaborativeScore 表示协同过滤召回带来的得分。
	CollaborativeScore float64
	// UserNeighborScore 表示相似用户召回带来的得分。
	UserNeighborScore float64
	// VectorScore 表示向量召回带来的得分。
	VectorScore float64
	// ExposurePenalty 表示曝光惩罚扣分。
	ExposurePenalty float64
	// RepeatPenalty 表示重复购买惩罚扣分。
	RepeatPenalty float64
	// RuleScore 表示规则排序阶段产出的基础分。
	RuleScore float64
	// FmScore 表示学习排序模型产出的预测分。
	FmScore float64
	// LlmScore 表示 LLM 重排阶段产出的相关性分。
	LlmScore float64
	// FinalScore 表示最终用于排序的分值。
	FinalScore float64
}

// ToRecommendScoreDetail 转换为对外评分明细结构。
func (s Score) ToRecommendScoreDetail(goodsId int64, recallSources []string) core.ScoreDetail {
	return core.ScoreDetail{
		GoodsId:            goodsId,
		FinalScore:         s.FinalScore,
		RelationScore:      s.RelationScore,
		UserGoodsScore:     s.UserGoodsScore,
		CategoryScore:      s.CategoryScore,
		SceneHotScore:      s.SceneHotScore,
		GlobalHotScore:     s.GlobalHotScore,
		FreshnessScore:     s.FreshnessScore,
		SessionScore:       s.SessionScore,
		ExternalScore:      s.ExternalScore,
		CollaborativeScore: s.CollaborativeScore,
		UserNeighborScore:  s.UserNeighborScore,
		VectorScore:        s.VectorScore,
		ExposurePenalty:    s.ExposurePenalty,
		RepeatPenalty:      s.RepeatPenalty,
		RuleScore:          s.RuleScore,
		FmScore:            s.FmScore,
		LlmScore:           s.LlmScore,
		RecallSources:      recallSources,
	}
}

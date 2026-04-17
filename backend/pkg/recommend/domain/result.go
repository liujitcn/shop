package domain

import "shop/api/gen/go/app"

// ScoreDetail 表示推荐结果的评分明细。
type ScoreDetail struct {
	GoodsId               int64    `json:"goodsId"`                         // 商品编号
	FinalScore            float64  `json:"finalScore"`                      // 最终排序分
	RelationScore         float64  `json:"relationScore,omitempty"`         // 商品关系分
	UserGoodsScore        float64  `json:"userGoodsScore,omitempty"`        // 用户商品偏好分
	SimilarUserScore      float64  `json:"similarUserScore,omitempty"`      // 相似用户偏好分
	ProfileScore          float64  `json:"profileScore,omitempty"`          // 用户类目画像分
	ScenePopularityScore  float64  `json:"scenePopularityScore,omitempty"`  // 场景热度分
	GlobalPopularityScore float64  `json:"globalPopularityScore,omitempty"` // 全站热度分
	FreshnessScore        float64  `json:"freshnessScore,omitempty"`        // 新鲜度分
	ExposurePenalty       float64  `json:"exposurePenalty,omitempty"`       // 场景曝光惩罚分
	ActorExposurePenalty  float64  `json:"actorExposurePenalty,omitempty"`  // 主体曝光惩罚分
	RepeatPenalty         float64  `json:"repeatPenalty,omitempty"`         // 重复推荐惩罚分
	RuleScore             float64  `json:"ruleScore,omitempty"`             // 规则粗排得分
	ModelScore            float64  `json:"modelScore,omitempty"`            // 模型精排得分
	LlmScore              float64  `json:"llmScore,omitempty"`              // LLM 二次重排得分
	RecallSources         []string `json:"recallSources,omitempty"`         // 当前商品命中的召回来源列表
}

// PageResult 表示推荐列表页的领域返回结果。
type PageResult struct {
	List          []*app.GoodsInfo // 当前页商品列表
	Total         int64            // 推荐总数
	RecallSources []string         // 当前页命中的召回来源列表
	ScoreDetails  []ScoreDetail    // 当前页评分明细
	SourceContext map[string]any   // 来源上下文扩展信息
}

// NormalizeSourceContext 确保结果上下文始终返回非空映射。
func (r *PageResult) NormalizeSourceContext() map[string]any {
	// 结果为空或上下文为空时，统一返回空映射，避免调用方判空。
	if r == nil || r.SourceContext == nil {
		return map[string]any{}
	}
	return r.SourceContext
}

package domain

import (
	"sort"

	"shop/api/gen/go/app"
)

// Candidate 表示推荐引擎内部流转的候选商品。
type Candidate struct {
	Goods                 *app.GoodsInfo      // 候选商品实体
	RelationScore         float64             // 商品关系分
	UserGoodsScore        float64             // 用户商品偏好分
	SimilarUserScore      float64             // 相似用户偏好分
	ProfileScore          float64             // 用户类目画像分
	ScenePopularityScore  float64             // 场景热度分
	GlobalPopularityScore float64             // 全站热度分
	FreshnessScore        float64             // 新鲜度分
	ExposurePenalty       float64             // 场景曝光惩罚分
	ActorExposurePenalty  float64             // 主体曝光惩罚分
	RepeatPenalty         float64             // 重复推荐惩罚分
	RuleScore             float64             // 规则粗排得分
	ModelScore            float64             // 模型精排得分
	LlmScore              float64             // LLM 二次重排得分
	FinalScore            float64             // 最终排序分
	RecallSources         map[string]struct{} // 命中的召回来源集合
}

// AddRecallSource 为候选商品补充一个召回来源。
func (c *Candidate) AddRecallSource(source string) {
	// 候选为空或来源名为空时，直接跳过写入。
	if c == nil || source == "" {
		return
	}
	// 首次追加召回来源时，先初始化来源集合。
	if c.RecallSources == nil {
		c.RecallSources = make(map[string]struct{}, 4)
	}
	c.RecallSources[source] = struct{}{}
}

// RecallSourceList 返回排序稳定的召回来源列表。
func (c *Candidate) RecallSourceList() []string {
	// 没有召回来源时，统一返回空切片，避免调用方判空。
	if c == nil || len(c.RecallSources) == 0 {
		return []string{}
	}
	result := make([]string, 0, len(c.RecallSources))
	for source := range c.RecallSources {
		result = append(result, source)
	}
	sort.Strings(result)
	return result
}

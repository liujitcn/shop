package domain

// RecallStrategy 表示单条召回链配置。
type RecallStrategy struct {
	Name    string            // 召回器名称
	Type    string            // 召回器类型
	Limit   int64             // 召回数量上限
	Enabled bool              // 是否启用当前召回器
	Options map[string]string // 召回器扩展配置
}

// RankStrategy 表示排序链配置。
type RankStrategy struct {
	Name             string            // 排序器名称
	Type             string            // 排序器类型
	Enabled          bool              // 是否启用当前排序器
	Recommenders     []string          // 排序阶段依赖的召回链列表
	QueryTemplate    string            // LLM 或重排使用的查询模板
	DocumentTemplate string            // LLM 或重排使用的文档模板
	Options          map[string]string // 排序器扩展配置
}

// ReplacementStrategy 表示替换与衰减策略配置。
type ReplacementStrategy struct {
	EnableReplacement        bool    // 是否启用替换策略
	PositiveReplacementDecay float64 // 正反馈衰减系数
	ReadReplacementDecay     float64 // 浏览反馈衰减系数
}

// SceneStrategy 表示一个场景下完整的推荐策略。
type SceneStrategy struct {
	Scene       int32               // 推荐场景
	RecallList  []RecallStrategy    // 当前场景下启用的召回链
	Rank        RankStrategy        // 当前场景下使用的排序链
	Replacement ReplacementStrategy // 当前场景下的替换与衰减策略
}

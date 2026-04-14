package rank

import "recommend/internal/core"

// ScoreWeights 表示推荐排序使用的权重配置。
type ScoreWeights = core.ScoreWeights

// RankOptions 表示排序阶段的可配置参数。
type RankOptions struct {
	// MaxPerCategory 表示主结果区内同一类目允许保留的最大数量。
	MaxPerCategory int
}

const (
	// defaultMaxPerCategory 表示未显式配置时的默认类目打散上限。
	defaultMaxPerCategory = 2
)

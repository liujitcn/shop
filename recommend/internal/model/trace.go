package model

import "recommend/internal/core"

// TraceStep 表示内部推荐追踪步骤。
type TraceStep struct {
	// Stage 表示当前追踪步骤名称。
	Stage string
	// Reason 表示当前追踪步骤的说明。
	Reason string
	// GoodsIds 表示当前步骤涉及的商品编号集合。
	GoodsIds []int64
}

// Trace 表示内部推荐追踪结果。
type Trace struct {
	// TraceId 表示追踪编号。
	TraceId string
	// RequestId 表示关联的推荐请求编号。
	RequestId string
	// Scene 表示当前追踪对应的推荐场景。
	Scene Scene
	// Steps 表示推荐链路步骤列表。
	Steps []TraceStep
	// ResultGoodsIds 表示最终输出的商品编号列表。
	ResultGoodsIds []int64
}

// AddStep 向追踪结果追加一个步骤。
func (t *Trace) AddStep(stage, reason string, goodsIds []int64) {
	if t == nil {
		return
	}

	step := TraceStep{
		Stage:  stage,
		Reason: reason,
	}
	if len(goodsIds) > 0 {
		step.GoodsIds = append(step.GoodsIds, goodsIds...)
	}
	t.Steps = append(t.Steps, step)
}

// ToExplainSteps 转换为对外追踪步骤。
func (t *Trace) ToExplainSteps() []core.TraceStep {
	if t == nil || len(t.Steps) == 0 {
		return nil
	}

	result := make([]core.TraceStep, 0, len(t.Steps))
	for _, item := range t.Steps {
		result = append(result, core.TraceStep{
			Stage:    item.Stage,
			Reason:   item.Reason,
			GoodsIds: append([]int64(nil), item.GoodsIds...),
		})
	}
	return result
}

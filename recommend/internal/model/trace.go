package model

import "recommend"

// TraceStep 表示内部推荐追踪步骤。
type TraceStep struct {
	Stage    string
	Reason   string
	GoodsIds []int64
}

// Trace 表示内部推荐追踪结果。
type Trace struct {
	TraceId        string
	RequestId      string
	Scene          Scene
	Steps          []TraceStep
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
func (t *Trace) ToExplainSteps() []recommend.TraceStep {
	if t == nil || len(t.Steps) == 0 {
		return nil
	}

	result := make([]recommend.TraceStep, 0, len(t.Steps))
	for _, item := range t.Steps {
		result = append(result, recommend.TraceStep{
			Stage:    item.Stage,
			Reason:   item.Reason,
			GoodsIds: append([]int64(nil), item.GoodsIds...),
		})
	}
	return result
}

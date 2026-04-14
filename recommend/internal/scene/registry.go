package scene

import (
	"context"
	"fmt"
	"recommend"
	"recommend/internal/model"
)

// Pipeline 定义单个场景推荐流水线。
type Pipeline func(context.Context, model.Request, recommend.Dependencies) ([]*model.Candidate, error)

// ResolvePipeline 返回指定场景对应的推荐流水线。
func ResolvePipeline(scene model.Scene) (Pipeline, error) {
	// 每个场景都绑定固定的流水线实现，避免在线推荐动态拼接出不可控链路。
	switch scene {
	case model.SceneHome:
		return runHomePipeline, nil
	case model.SceneGoodsDetail:
		return runGoodsDetailPipeline, nil
	case model.SceneCart:
		return runCartPipeline, nil
	case model.SceneProfile:
		return runProfilePipeline, nil
	case model.SceneOrderDetail:
		return runOrderDetailPipeline, nil
	case model.SceneOrderPaid:
		return runOrderPaidPipeline, nil
	default:
		return nil, fmt.Errorf("recommend: 不支持的推荐场景 %q", scene)
	}
}

package recommend

import (
	"context"

	_const "shop/pkg/const"

	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/errorsx"
	"shop/pkg/recommend/dto"
	"shop/pkg/recommend/gorse"
	"shop/pkg/recommend/local"
)

// GoodsReceiver 表示推荐商品统一入口。
type GoodsReceiver struct {
	gorseChain *gorse.ChainReceiver
	localChain *local.ChainReceiver
}

// NewGoodsReceiver 创建推荐商品统一入口。
func NewGoodsReceiver(gorseChain *gorse.ChainReceiver, localChain *local.ChainReceiver) *GoodsReceiver {
	return &GoodsReceiver{
		gorseChain: gorseChain,
		localChain: localChain,
	}
}

// RecommendGoods 按配置选择统一的推荐商品来源。
func (r *GoodsReceiver) RecommendGoods(ctx context.Context, req *dto.GoodsRequest) (*dto.GoodsResult, error) {
	// 场景未指定时，当前请求不具备推荐语义。
	if req.Scene == commonv1.RecommendScene(_const.RECOMMEND_SCENE_UNKNOWN) {
		return nil, errorsx.InvalidArgument("推荐场景不能为空")
	}

	// 当前配置了 Gorse 推荐链路时优先使用远端结果，未命中或调用失败时回退本地推荐。
	if r.gorseChain != nil && r.gorseChain.Enabled() {
		var result *dto.GoodsResult
		var err error
		result, err = r.gorseChain.ExecutePlan(
			ctx,
			req.Scene,
			req.Actor,
			req.GoodsID,
			req.ContextGoodsIDs,
			req.PageNum,
			req.PageSize,
		)
		if err == nil && result != nil && len(result.GoodsIDs) > 0 {
			return result, nil
		}
	}
	return r.localChain.ExecutePlan(
		ctx,
		req.Scene,
		req.Actor,
		req.GoodsID,
		req.RequestID,
		req.ContextGoodsIDs,
		req.PageNum,
		req.PageSize,
	)
}

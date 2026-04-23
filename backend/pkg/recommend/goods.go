package recommend

import (
	"context"

	"shop/api/gen/go/common"
	"shop/pkg/errorsx"
	"shop/pkg/recommend/dto"
	pkgLocal "shop/pkg/recommend/local"
	pkgRemote "shop/pkg/recommend/remote"
)

// GoodsReceiver 表示推荐商品统一入口。
type GoodsReceiver struct {
	remoteChain *pkgRemote.ChainReceiver
	localChain  *pkgLocal.ChainReceiver
}

// NewGoodsReceiver 创建推荐商品统一入口。
func NewGoodsReceiver(remoteChain *pkgRemote.ChainReceiver, localChain *pkgLocal.ChainReceiver) *GoodsReceiver {
	return &GoodsReceiver{
		remoteChain: remoteChain,
		localChain:  localChain,
	}
}

// RecommendGoods 按配置选择统一的推荐商品来源。
func (r *GoodsReceiver) RecommendGoods(ctx context.Context, req *dto.GoodsRequest) (*dto.GoodsResult, error) {
	// 推荐请求为空时，无法继续查询推荐商品。
	if req == nil {
		return nil, errorsx.InvalidArgument("推荐请求不能为空")
	}
	// 场景未指定时，当前请求不具备推荐语义。
	if req.Scene == common.RecommendScene_UNKNOWN_RS {
		return nil, errorsx.InvalidArgument("推荐场景不能为空")
	}

	// 当前配置了远端推荐链路时，统一只走远端推荐，不再混入本地来源。
	if r.remoteChain != nil && r.remoteChain.Enabled() {
		return r.remoteChain.ExecutePlan(
			ctx,
			req.Scene,
			req.Actor,
			req.GoodsId,
			req.ContextGoodsIds,
			req.PageNum,
			req.PageSize,
		)
	}
	return r.localChain.ExecutePlan(
		ctx,
		req.Scene,
		req.Actor,
		req.GoodsId,
		req.RequestId,
		req.ContextGoodsIds,
		req.PageNum,
		req.PageSize,
	)
}

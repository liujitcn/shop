package biz

import (
	"context"
	"shop/api/gen/go/app"
	"shop/pkg/biz"
	"shop/pkg/gen/data"

	"github.com/liujitcn/go-utils/id"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// RecommendCase 推荐业务处理对象。
type RecommendCase struct {
	*biz.BaseCase
	tx data.Transaction
}

// NewRecommendCase 创建推荐业务处理对象。
func NewRecommendCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
) *RecommendCase {
	return &RecommendCase{
		BaseCase: baseCase,
		tx:       tx,
	}
}

// RecommendAnonymousActor 获取匿名推荐主体。
func (c *RecommendCase) RecommendAnonymousActor(_ context.Context, _ *emptypb.Empty) (*wrapperspb.Int64Value, error) {
	actorId := id.GenSnowflakeID()
	return wrapperspb.Int64(actorId), nil
}

// BindRecommendAnonymousActor 绑定匿名推荐主体到当前登录用户。
func (c *RecommendCase) BindRecommendAnonymousActor(ctx context.Context, req *emptypb.Empty) error {
	return nil
}

// RecommendGoods 查询推荐商品列表。
func (c *RecommendCase) RecommendGoods(ctx context.Context, req *app.RecommendGoodsRequest) (*app.RecommendGoodsResponse, error) {
	return &app.RecommendGoodsResponse{}, nil
}

// RecommendExposureReport 上报推荐曝光事件。
func (c *RecommendCase) RecommendExposureReport(ctx context.Context, req *app.RecommendExposureReportRequest) error {
	return nil
}

// RecommendGoodsActionReport 上报推荐商品行为事件。
func (c *RecommendCase) RecommendGoodsActionReport(ctx context.Context, req *app.RecommendGoodsActionReportRequest) error {
	return nil
}

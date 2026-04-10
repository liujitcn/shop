package biz

import (
	"context"
	"shop/pkg/gen/data"
	recommendactor "shop/pkg/recommend/actor"

	"shop/api/gen/go/app"
	"shop/pkg/biz"
	appdto "shop/service/app/dto"

	"github.com/liujitcn/go-utils/id"
	"github.com/liujitcn/kratos-kit/auth"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type RecommendActor = appdto.RecommendActor
type RecommendEvent = appdto.RecommendEvent
type RecommendEventGoodsItem = appdto.RecommendEventGoodsItem

// RecommendCase 推荐业务处理对象。
type RecommendCase struct {
	*biz.BaseCase
	tx                       data.Transaction
	recommendRequestCase     *RecommendRequestCase
	recommendExposureCase    *RecommendExposureCase
	recommendGoodsActionCase *RecommendGoodsActionCase
}

// NewRecommendCase 创建推荐业务处理对象。
func NewRecommendCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	recommendRequestCase *RecommendRequestCase,
	recommendExposureCase *RecommendExposureCase,
	recommendGoodsActionCase *RecommendGoodsActionCase,
) *RecommendCase {
	return &RecommendCase{
		BaseCase:                 baseCase,
		tx:                       tx,
		recommendRequestCase:     recommendRequestCase,
		recommendExposureCase:    recommendExposureCase,
		recommendGoodsActionCase: recommendGoodsActionCase,
	}
}

// RecommendAnonymousActor 获取匿名推荐主体。
func (c *RecommendCase) RecommendAnonymousActor(_ context.Context, _ *emptypb.Empty) (*wrapperspb.Int64Value, error) {
	actorId := id.GenSnowflakeID()
	return wrapperspb.Int64(actorId), nil
}

// BindRecommendAnonymousActor 绑定匿名推荐主体到当前登录用户。
func (c *RecommendCase) BindRecommendAnonymousActor(ctx context.Context, req *emptypb.Empty) error {
	authInfo, err := auth.FromContext(ctx)
	// 当前上下文没有登录用户时，不需要执行匿名主体归并。
	if err != nil || authInfo == nil || authInfo.UserId <= 0 {
		return nil
	}

	// 匿名主体不存在或已经是同一个主体时，直接跳过绑定。
	anonymousId := recommendactor.ExtractAnonymousID(ctx)
	if anonymousId <= 0 || anonymousId == authInfo.UserId {
		return nil
	}

	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		err = c.recommendRequestCase.BindRecommendRequestActor(ctx, anonymousId, authInfo.UserId)
		if err != nil {
			return err
		}
		err = c.recommendExposureCase.BindRecommendExposureActor(ctx, anonymousId, authInfo.UserId)
		if err != nil {
			return err
		}
		err = c.recommendGoodsActionCase.BindRecommendGoodsActionActor(ctx, anonymousId, authInfo.UserId)
		if err != nil {
			return err
		}
		return nil
	})
}

// RecommendGoods 查询推荐商品列表。
func (c *RecommendCase) RecommendGoods(ctx context.Context, req *app.RecommendGoodsRequest) (*app.RecommendGoodsResponse, error) {
	return c.recommendRequestCase.RecommendGoods(ctx, req)
}

// RecommendExposureReport 上报推荐曝光事件。
func (c *RecommendCase) RecommendExposureReport(ctx context.Context, req *app.RecommendExposureReportRequest) error {
	return c.recommendExposureCase.RecommendExposureReport(ctx, req)
}

// RecommendGoodsActionReport 上报推荐商品行为事件。
func (c *RecommendCase) RecommendGoodsActionReport(ctx context.Context, req *app.RecommendGoodsActionReportRequest) error {
	return c.recommendGoodsActionCase.RecommendGoodsActionReport(ctx, req)
}

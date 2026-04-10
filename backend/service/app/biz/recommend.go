package biz

import (
	"context"
	"shop/api/gen/go/app"
	"shop/api/gen/go/common"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	recommendactor "shop/pkg/recommend/actor"
	recommendevent "shop/pkg/recommend/event"
	appdto "shop/service/app/dto"

	"github.com/liujitcn/go-utils/id"
	"github.com/liujitcn/kratos-kit/auth"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func recommendUserID(actor *appdto.RecommendActor) int64 {
	if actor == nil || actor.ActorType != recommendevent.ActorTypeUser {
		return 0
	}
	return actor.ActorId
}

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
		err = c.recommendRequestCase.bindRecommendRequestActor(ctx, anonymousId, authInfo.UserId)
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
	// 统一兜底分页参数，避免前端漏传导致查询异常。
	pageNum := req.GetPageNum()
	// 页码非法时回退到首页，保证分页查询始终可执行。
	if pageNum <= 0 {
		req.PageNum = 1
	}
	pageSize := req.GetPageSize()
	// 每页数量非法时使用默认值，避免查全表或空分页。
	if pageSize <= 0 {
		req.PageSize = 10
	}
	// 每次推荐请求都生成独立 requestID，用于后续曝光归因。
	requestId := id.NewGUIDv7NoHyphen()
	actor := recommendactor.Resolve(ctx)

	list := make([]*app.GoodsInfo, 0)
	total := int64(0)
	sourceContext := map[string]any{
		"orderId": req.GetOrderId(),
	}
	recallSources := make([]string, 0, 4)
	var err error
	// 匿名主体统一走公共推荐池，减少首页、购物车、我的三端内容分裂。
	if actor.ActorType == recommendevent.ActorTypeAnonymous {
		list, total, recallSources, sourceContext, err = c.recommendRequestCase.listAnonymousRecommendGoods(ctx, actor, req)
	} else {
		list, total, recallSources, sourceContext, err = c.recommendRequestCase.listRecommendGoods(ctx, actor, req, recommendUserID(actor))
	}
	if err != nil {
		return nil, err
	}
	sourceContext["actorType"] = actor.ActorType
	sourceContext["actorId"] = actor.ActorId

	err = c.recommendRequestCase.saveRecommendRequest(ctx, requestId, actor, req, sourceContext, list, recallSources)
	if err != nil {
		return nil, err
	}

	return &app.RecommendGoodsResponse{
		List:      list,
		Total:     int32(total),
		RequestId: requestId,
	}, nil
}

// RecommendExposureReport 上报推荐曝光事件。
func (c *RecommendCase) RecommendExposureReport(ctx context.Context, req *app.RecommendExposureReportRequest) error {
	// 空请求直接忽略，避免埋点接口影响主业务流程。
	if req == nil {
		return nil
	}

	actor := recommendactor.Resolve(ctx)
	c.recommendExposureCase.publishRecommendExposureEvent(
		actor,
		req.GetRequestId(),
		req.GetScene(),
		req.GetGoodsIds(),
	)
	return nil
}

// RecommendGoodsActionReport 上报推荐商品行为事件。
func (c *RecommendCase) RecommendGoodsActionReport(ctx context.Context, req *app.RecommendGoodsActionReportRequest) error {
	// 空请求直接返回，异步埋点不做额外失败放大。
	if req == nil {
		return nil
	}

	actor := recommendactor.Resolve(ctx)
	// 按商品行为类型拆分投递不同事件，保持曝光与商品行为链路独立。
	switch req.GetEventType() {
	case common.RecommendGoodsActionType_RECOMMEND_GOODS_ACTION_CLICK:
		c.recommendGoodsActionCase.publishTrackGoodsEvents(actor, req.GetGoodsItems(), publishRecommendClickEvent)
	case common.RecommendGoodsActionType_RECOMMEND_GOODS_ACTION_VIEW:
		c.recommendGoodsActionCase.publishTrackGoodsViewEvents(actor, req.GetGoodsItems())
	case common.RecommendGoodsActionType_RECOMMEND_GOODS_ACTION_COLLECT:
		c.recommendGoodsActionCase.publishTrackGoodsEvents(actor, req.GetGoodsItems(), publishGoodsCollectEvent)
	case common.RecommendGoodsActionType_RECOMMEND_GOODS_ACTION_CART:
		c.recommendGoodsActionCase.publishTrackGoodsCartEvents(actor, req.GetGoodsItems())
	case common.RecommendGoodsActionType_RECOMMEND_GOODS_ACTION_ORDER_CREATE:
		publishOrderCreateEvent(actor, recommendevent.BuildGoodsItemsFromActionItems(req.GetGoodsItems()))
	case common.RecommendGoodsActionType_RECOMMEND_GOODS_ACTION_ORDER_PAY:
		publishOrderPayEvent(actor, recommendevent.BuildGoodsItemsFromActionItems(req.GetGoodsItems()))
	}
	return nil
}

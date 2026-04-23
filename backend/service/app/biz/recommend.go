package biz

import (
	"context"
	"time"

	"shop/api/gen/go/app"
	"shop/api/gen/go/common"
	"shop/pkg/biz"
	"shop/pkg/errorsx"
	pkgRecommend "shop/pkg/recommend"
	"shop/pkg/recommend/dto"

	_slice "github.com/liujitcn/go-utils/slice"
	"github.com/liujitcn/kratos-kit/auth"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const recommendRecentHistoryLimit = 20

// RecommendCase 推荐业务处理对象。
type RecommendCase struct {
	*biz.BaseCase
	recommendAnonymousActorCase *RecommendAnonymousActorCase
	recommendRequestCase        *RecommendRequestCase
	recommendEventCase          *RecommendEventCase
	orderGoodsCase              *OrderGoodsCase
	userCartCase                *UserCartCase
	userCollectCase             *UserCollectCase
	goodsInfoCase               *GoodsInfoCase
	recommendReceiver           *pkgRecommend.GoodsReceiver
}

// NewRecommendCase 创建推荐业务处理对象。
func NewRecommendCase(
	baseCase *biz.BaseCase,
	recommendAnonymousActorCase *RecommendAnonymousActorCase,
	recommendRequestCase *RecommendRequestCase,
	recommendEventCase *RecommendEventCase,
	orderGoodsCase *OrderGoodsCase,
	userCartCase *UserCartCase,
	userCollectCase *UserCollectCase,
	goodsInfoCase *GoodsInfoCase,
	recommendReceiver *pkgRecommend.GoodsReceiver,
) *RecommendCase {
	return &RecommendCase{
		BaseCase:                    baseCase,
		recommendAnonymousActorCase: recommendAnonymousActorCase,
		recommendRequestCase:        recommendRequestCase,
		recommendEventCase:          recommendEventCase,
		orderGoodsCase:              orderGoodsCase,
		userCartCase:                userCartCase,
		userCollectCase:             userCollectCase,
		goodsInfoCase:               goodsInfoCase,
		recommendReceiver:           recommendReceiver,
	}
}

// RecommendAnonymousActor 获取匿名推荐主体。
func (c *RecommendCase) RecommendAnonymousActor(ctx context.Context, _ *emptypb.Empty) (*wrapperspb.Int64Value, error) {
	anonymousId, err := c.recommendAnonymousActorCase.getRecommendAnonymousIdFromHeader(ctx)
	if err != nil {
		return nil, err
	}
	// 请求头已携带匿名主体时，直接复用并刷新活跃时间。
	if anonymousId > 0 {
		err = c.recommendAnonymousActorCase.ensureRecommendAnonymousActor(ctx, anonymousId)
		if err != nil {
			return nil, err
		}
		return wrapperspb.Int64(anonymousId), nil
	}

	anonymousId, err = c.recommendAnonymousActorCase.createRecommendAnonymousActor(ctx)
	if err != nil {
		return nil, err
	}
	return wrapperspb.Int64(anonymousId), nil
}

// BindRecommendAnonymousActor 绑定匿名推荐主体到当前登录用户。
func (c *RecommendCase) BindRecommendAnonymousActor(ctx context.Context, _ *emptypb.Empty) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}

	anonymousId := int64(0)
	anonymousId, err = c.recommendAnonymousActorCase.getRecommendAnonymousIdFromHeader(ctx)
	if err != nil {
		return err
	}
	// 未携带匿名主体时，说明当前登录前没有匿名会话需要绑定。
	if anonymousId <= 0 {
		return nil
	}
	return c.recommendAnonymousActorCase.bindRecommendAnonymousActor(ctx, authInfo.UserId, anonymousId)
}

// RecommendGoods 查询推荐商品列表。
func (c *RecommendCase) RecommendGoods(ctx context.Context, req *app.RecommendGoodsRequest) (*app.RecommendGoodsResponse, error) {
	// 推荐请求体为空时，无法继续执行场景兜底。
	if req == nil {
		return nil, errorsx.InvalidArgument("推荐请求不能为空")
	}
	// 场景未指定时，无法确定推荐兜底口径。
	if req.GetScene() == common.RecommendScene_UNKNOWN_RS {
		return nil, errorsx.InvalidArgument("推荐场景不能为空")
	}

	actor, err := c.resolveRecommendActor(ctx, true)
	if err != nil {
		return nil, err
	}
	pageNum, pageSize := req.GetPageNum(), req.GetPageSize()
	// 页码非法时，统一回退到第 1 页。
	if pageNum <= 0 {
		pageNum = 1
	}
	// 每页条数非法时，统一回退到 10 条。
	if pageSize <= 0 {
		pageSize = 10
	}

	recommendReq := &dto.GoodsRequest{
		Scene:     req.GetScene(),
		Actor:     actor,
		GoodsId:   req.GetGoodsId(),
		OrderId:   req.GetOrderId(),
		RequestId: req.GetRequestId(),
		PageNum:   pageNum,
		PageSize:  pageSize,
	}

	contextGoodsIds := make([]int64, 0)
	contextGoodsIds, err = c.listRecommendContextGoodsIds(ctx, recommendReq)
	if err != nil {
		return nil, err
	}
	recommendReq.ContextGoodsIds = contextGoodsIds

	requestId := int64(0)
	requestId, err = c.recommendRequestCase.resolveRecommendRequestId(ctx, recommendReq)
	if err != nil {
		return nil, err
	}
	recommendReq.RequestId = requestId

	recommendResult := &dto.GoodsResult{}
	recommendResult, err = c.recommendReceiver.RecommendGoods(ctx, recommendReq)
	if err != nil {
		return nil, err
	}
	var goodsList []*app.GoodsInfo
	goodsList, err = c.goodsInfoCase.listByGoodsIds(ctx, recommendResult.GoodsIds)
	if err != nil {
		return nil, err
	}

	contextRecord := dto.NewRecommendRequestContext(
		recommendReq.GoodsId,
		recommendReq.OrderId,
		recommendReq.ContextGoodsIds,
		recommendResult,
	)
	err = c.recommendRequestCase.saveRecommendRequest(ctx, recommendReq, contextRecord, goodsList, recommendResult.Total)
	if err != nil {
		return nil, err
	}

	return &app.RecommendGoodsResponse{
		List:      goodsList,
		Total:     int32(recommendResult.Total),
		RequestId: requestId,
	}, nil
}

// RecommendEventReport 上报推荐事件。
func (c *RecommendCase) RecommendEventReport(ctx context.Context, req *app.RecommendEventReportRequest) error {
	// 空请求直接忽略，避免埋点影响主流程。
	if req == nil {
		return nil
	}

	actor, err := c.resolveRecommendActor(ctx, true)
	if err != nil {
		return err
	}
	return c.recommendEventCase.persistRecommendEventReport(ctx, actor, req, time.Now())
}

// resolveRecommendActor 解析当前请求使用的推荐主体。
func (c *RecommendCase) resolveRecommendActor(ctx context.Context, allowAnonymous bool) (*dto.RecommendActor, error) {
	authInfo, err := auth.FromContext(ctx)
	// 当前请求已登录时，优先使用登录用户作为推荐主体。
	if err == nil && authInfo != nil && authInfo.UserId > 0 {
		return &dto.RecommendActor{
			ActorType: dto.UserActorType,
			ActorId:   authInfo.UserId,
		}, nil
	}
	// 当前接口不允许匿名主体时，直接返回未登录错误。
	if !allowAnonymous {
		return nil, errorsx.Unauthenticated("用户认证失败")
	}

	anonymousId := int64(0)
	anonymousId, err = c.recommendAnonymousActorCase.getRecommendAnonymousIdFromHeader(ctx)
	if err != nil {
		return nil, err
	}
	// 未登录且未携带匿名主体时，当前推荐请求无法归因。
	if anonymousId <= 0 {
		return nil, errorsx.InvalidArgument("缺少匿名推荐主体")
	}
	err = c.recommendAnonymousActorCase.ensureRecommendAnonymousActor(ctx, anonymousId)
	if err != nil {
		return nil, err
	}
	return &dto.RecommendActor{
		ActorType: dto.AnonymousActorType,
		ActorId:   anonymousId,
	}, nil
}

// listRecommendContextGoodsIds 查询当前推荐请求的上下文商品编号列表。
func (c *RecommendCase) listRecommendContextGoodsIds(
	ctx context.Context,
	req *dto.GoodsRequest,
) ([]int64, error) {
	goodsIds := make([]int64, 0)
	actor := req.Actor
	// 不同推荐场景使用各自更稳定的上下文商品来源。
	switch req.Scene {
	case common.RecommendScene_GOODS_DETAIL:
		// 商品详情场景优先以当前商品作为锚点。
		if req.GoodsId > 0 {
			// 当前商品存在时，直接把商品详情页锚点作为上下文商品。
			goodsIds = append(goodsIds, req.GoodsId)
		}
	case common.RecommendScene_CART:
		// 登录态购物车页优先读取购物车商品做上下文。
		if actor != nil && actor.IsUser() {
			// 当前主体是登录用户时，购物车商品更能代表即时搭配意图。
			list, loadErr := c.userCartCase.listGoodsIdsByUserId(ctx, actor.ActorId)
			if loadErr != nil {
				return nil, loadErr
			}
			goodsIds = list
		}
	case common.RecommendScene_PROFILE:
		// 个人中心优先读取收藏商品做兴趣上下文。
		if actor != nil && actor.IsUser() {
			// 当前主体是登录用户时，收藏商品更能代表长期兴趣偏好。
			list, loadErr := c.userCollectCase.listGoodsIdsByUserId(ctx, actor.ActorId, recommendRecentHistoryLimit)
			if loadErr != nil {
				return nil, loadErr
			}
			goodsIds = list
		}
	case common.RecommendScene_ORDER_DETAIL, common.RecommendScene_ORDER_PAID:
		// 订单详情与支付成功页优先读取订单商品做上下文。
		if req.OrderId > 0 {
			// 当前请求带有订单编号时，优先围绕订单内商品构建上下文。
			list, loadErr := c.orderGoodsCase.listGoodsIdsByOrderId(ctx, req.OrderId)
			if loadErr != nil {
				return nil, loadErr
			}
			goodsIds = list
		}
	default:
		// 业务场景没有稳定上下文时，再回退到最近推荐行为商品。
		if actor != nil && actor.IsValid() {
			// 当前主体可识别时，回放最近推荐行为涉及的商品作为弱上下文。
			list, loadErr := c.recommendEventCase.listRecentRecommendEventGoodsIds(ctx, actor)
			if loadErr != nil {
				return nil, loadErr
			}
			goodsIds = list
		}
	}

	return _slice.Unique(goodsIds), nil
}

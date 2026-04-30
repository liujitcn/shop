package biz

import (
	"context"
	"time"

	_const "shop/pkg/const"

	appv1 "shop/api/gen/go/app/v1"
	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/recommend"
	"shop/pkg/recommend/dto"

	_slice "github.com/liujitcn/go-utils/slice"
	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-kit/auth"
)

const RECOMMEND_RECENT_HISTORY_LIMIT = 20

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
	recommendReceiver           *recommend.GoodsReceiver
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
	recommendReceiver *recommend.GoodsReceiver,
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
func (c *RecommendCase) RecommendAnonymousActor(ctx context.Context, _ *appv1.RecommendAnonymousActorRequest) (*appv1.RecommendAnonymousActorResponse, error) {
	anonymousID, err := c.recommendAnonymousActorCase.getRecommendAnonymousIDFromHeader(ctx)
	if err != nil {
		return nil, err
	}
	// 请求头已携带匿名主体时，直接复用并刷新活跃时间。
	if anonymousID > 0 {
		err = c.recommendAnonymousActorCase.ensureRecommendAnonymousActor(ctx, anonymousID)
		if err != nil {
			return nil, err
		}
		return &appv1.RecommendAnonymousActorResponse{
			AnonymousId: anonymousID,
		}, nil
	}

	anonymousID, err = c.recommendAnonymousActorCase.createRecommendAnonymousActor(ctx)
	if err != nil {
		return nil, err
	}
	return &appv1.RecommendAnonymousActorResponse{
		AnonymousId: anonymousID,
	}, nil
}

// BindRecommendAnonymousActor 绑定匿名推荐主体到当前登录用户。
func (c *RecommendCase) BindRecommendAnonymousActor(ctx context.Context, _ *appv1.BindRecommendAnonymousActorRequest) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}

	anonymousID := int64(0)
	anonymousID, err = c.recommendAnonymousActorCase.getRecommendAnonymousIDFromHeader(ctx)
	if err != nil {
		return err
	}
	// 未携带匿名主体时，说明当前登录前没有匿名会话需要绑定。
	if anonymousID <= 0 {
		return nil
	}
	return c.recommendAnonymousActorCase.bindRecommendAnonymousActor(ctx, authInfo.UserId, anonymousID)
}

// RecommendGoods 查询推荐商品列表。
func (c *RecommendCase) RecommendGoods(ctx context.Context, req *appv1.RecommendGoodsRequest) (*appv1.RecommendGoodsResponse, error) {
	// 推荐请求体为空时，无法继续执行场景兜底。
	if req == nil {
		return nil, errorsx.InvalidArgument("推荐请求不能为空")
	}
	// 场景未指定时，无法确定推荐兜底口径。
	if req.GetScene() == commonv1.RecommendScene(_const.RECOMMEND_SCENE_UNKNOWN) {
		return nil, errorsx.InvalidArgument("推荐场景不能为空")
	}

	actor, err := c.resolveRecommendActor(ctx, true)
	if err != nil {
		return nil, err
	}
	pageNum, pageSize := repository.PageDefault(req.GetPageNum(), req.GetPageSize())

	requestID := int64(0)
	// 当前请求没有稳定推荐主体时，不复用也不生成推荐会话编号，避免后续链路落库。
	if actor.IsValid() {
		requestID = req.GetRequestId()
	}
	recommendReq := &dto.GoodsRequest{
		Scene:     req.GetScene(),
		Actor:     actor,
		GoodsID:   req.GetGoodsId(),
		OrderID:   req.GetOrderId(),
		RequestID: requestID,
		PageNum:   pageNum,
		PageSize:  pageSize,
	}

	contextGoodsIDs := make([]int64, 0)
	contextGoodsIDs, err = c.listRecommendContextGoodsIDs(ctx, recommendReq)
	if err != nil {
		return nil, err
	}
	recommendReq.ContextGoodsIDs = contextGoodsIDs

	// 当前请求存在稳定推荐主体时，才维护可归因的推荐请求链路。
	if actor.IsValid() {
		requestID, err = c.recommendRequestCase.resolveRecommendRequestID(ctx, recommendReq)
		if err != nil {
			return nil, err
		}
		recommendReq.RequestID = requestID
	}

	recommendResult := &dto.GoodsResult{}
	recommendResult, err = c.recommendReceiver.RecommendGoods(ctx, recommendReq)
	if err != nil {
		return nil, err
	}
	var goodsList []*appv1.GoodsInfo
	goodsList, err = c.goodsInfoCase.listByGoodsIDs(ctx, recommendResult.GoodsIDs)
	if err != nil {
		return nil, err
	}

	// 缺少匿名头且未登录时，本次推荐仅返回结果，不保存推荐请求和结果明细。
	if actor.IsValid() {
		contextRecord := dto.NewRecommendRequestContext(
			recommendReq.GoodsID,
			recommendReq.OrderID,
			recommendReq.ContextGoodsIDs,
			recommendResult,
		)
		err = c.recommendRequestCase.saveRecommendRequest(ctx, recommendReq, contextRecord, goodsList, recommendResult.Total)
		if err != nil {
			return nil, err
		}
	}

	return &appv1.RecommendGoodsResponse{
		GoodsInfos: goodsList,
		Total:      int32(recommendResult.Total),
		RequestId:  requestID,
	}, nil
}

// RecommendEventReport 上报推荐事件。
func (c *RecommendCase) RecommendEventReport(ctx context.Context, req *appv1.RecommendEventReportRequest) error {
	// 空请求直接忽略，避免埋点影响主流程。
	if req == nil {
		return nil
	}

	actor, err := c.resolveRecommendActor(ctx, true)
	if err != nil {
		return err
	}
	// 缺少匿名头且未登录时，埋点无法归因，直接忽略避免影响主流程。
	if !actor.IsValid() {
		return nil
	}
	return c.recommendEventCase.persistRecommendEventReport(ctx, actor, req, time.Now())
}

// resolveRecommendActor 解析当前请求使用的推荐主体。
func (c *RecommendCase) resolveRecommendActor(ctx context.Context, allowAnonymous bool) (*dto.RecommendActor, error) {
	// 可匿名推荐只需要静默读取认证信息，避免游客请求被记录成认证失败日志。
	authInfo, err := auth.FromContext(ctx)
	// 当前请求已登录时，优先使用登录用户作为推荐主体。
	if err == nil && authInfo != nil && authInfo.UserId > 0 {
		return &dto.RecommendActor{
			ActorType: commonv1.RecommendActorType(_const.RECOMMEND_ACTOR_TYPE_USER),
			ActorID:   authInfo.UserId,
		}, nil
	}
	// 当前接口不允许匿名主体时，直接返回未登录错误。
	if !allowAnonymous {
		return nil, errorsx.Unauthenticated("用户认证失败")
	}

	anonymousID := int64(0)
	anonymousID, err = c.recommendAnonymousActorCase.getRecommendAnonymousIDFromHeader(ctx)
	if err != nil {
		return nil, err
	}
	// 未登录且未携带匿名主体时，允许继续走不可归因推荐，但后续不保存链路数据。
	if anonymousID <= 0 {
		return nil, nil
	}
	err = c.recommendAnonymousActorCase.ensureRecommendAnonymousActor(ctx, anonymousID)
	if err != nil {
		return nil, err
	}
	return &dto.RecommendActor{
		ActorType: commonv1.RecommendActorType(_const.RECOMMEND_ACTOR_TYPE_ANONYMOUS),
		ActorID:   anonymousID,
	}, nil
}

// listRecommendContextGoodsIDs 查询当前推荐请求的上下文商品编号列表。
func (c *RecommendCase) listRecommendContextGoodsIDs(
	ctx context.Context,
	req *dto.GoodsRequest,
) ([]int64, error) {
	goodsIDs := make([]int64, 0)
	actor := req.Actor
	// 不同推荐场景使用各自更稳定的上下文商品来源。
	switch req.Scene {
	case commonv1.RecommendScene(_const.RECOMMEND_SCENE_GOODS_DETAIL):
		// 商品详情场景优先以当前商品作为锚点。
		if req.GoodsID > 0 {
			// 当前商品存在时，直接把商品详情页锚点作为上下文商品。
			goodsIDs = append(goodsIDs, req.GoodsID)
		}
	case commonv1.RecommendScene(_const.RECOMMEND_SCENE_CART):
		// 登录态购物车页优先读取购物车商品做上下文。
		if actor != nil && actor.IsUser() {
			// 当前主体是登录用户时，购物车商品更能代表即时搭配意图。
			list, loadErr := c.userCartCase.listGoodsIDsByUserID(ctx, actor.ActorID)
			if loadErr != nil {
				return nil, loadErr
			}
			goodsIDs = list
		}
	case commonv1.RecommendScene(_const.RECOMMEND_SCENE_PROFILE):
		// 个人中心优先读取收藏商品做兴趣上下文。
		if actor != nil && actor.IsUser() {
			// 当前主体是登录用户时，收藏商品更能代表长期兴趣偏好。
			list, loadErr := c.userCollectCase.listGoodsIDsByUserID(ctx, actor.ActorID, RECOMMEND_RECENT_HISTORY_LIMIT)
			if loadErr != nil {
				return nil, loadErr
			}
			goodsIDs = list
		}
	case commonv1.RecommendScene(_const.RECOMMEND_SCENE_ORDER_DETAIL), commonv1.RecommendScene(_const.RECOMMEND_SCENE_ORDER_PAID):
		// 订单详情与支付成功页优先读取订单商品做上下文。
		if req.OrderID > 0 {
			// 当前请求带有订单编号时，优先围绕订单内商品构建上下文。
			list, loadErr := c.orderGoodsCase.listGoodsIDsByOrderID(ctx, req.OrderID)
			if loadErr != nil {
				return nil, loadErr
			}
			goodsIDs = list
		}
	default:
		// 业务场景没有稳定上下文时，再回退到最近推荐行为商品。
		if actor != nil && actor.IsValid() {
			// 当前主体可识别时，回放最近推荐行为涉及的商品作为弱上下文。
			list, loadErr := c.recommendEventCase.listRecentRecommendEventGoodsIDs(ctx, actor)
			if loadErr != nil {
				return nil, loadErr
			}
			goodsIDs = list
		}
	}

	return _slice.Unique(goodsIDs), nil
}

package biz

import (
	"context"
	"fmt"
	"strconv"
	"time"

	_const "shop/pkg/const"

	appv1 "shop/api/gen/go/app/v1"
	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"shop/pkg/queue"
	"shop/pkg/recommend"
	"shop/pkg/recommend/dto"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/go-kratos/kratos/v3/transport"
	"github.com/liujitcn/go-utils/id"
	_slice "github.com/liujitcn/go-utils/slice"
	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-kit/auth"
)

// RECOMMEND_RECENT_HISTORY_LIMIT 表示推荐最近行为历史数量上限。
const RECOMMEND_RECENT_HISTORY_LIMIT = 20

// RECOMMEND_ANONYMOUS_ACTOR_HEADER_KEY 表示匿名推荐主体请求头名称。
const RECOMMEND_ANONYMOUS_ACTOR_HEADER_KEY = "X-Recommend-Anonymous-Id"

// RecommendCase 推荐业务处理对象。
type RecommendCase struct {
	*biz.BaseCase
	tx                   data.Transaction
	recommendRequestCase *RecommendRequestCase
	recommendEventCase   *RecommendEventCase
	orderGoodsCase       *OrderGoodsCase
	orderInfoCase        *OrderInfoCase
	userCartCase         *UserCartCase
	userCollectCase      *UserCollectCase
	goodsInfoCase        *GoodsInfoCase
	recommendReceiver    *recommend.GoodsReceiver
}

// NewRecommendCase 创建推荐业务处理对象。
func NewRecommendCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	recommendRequestCase *RecommendRequestCase,
	recommendEventCase *RecommendEventCase,
	orderGoodsCase *OrderGoodsCase,
	orderInfoCase *OrderInfoCase,
	userCartCase *UserCartCase,
	userCollectCase *UserCollectCase,
	goodsInfoCase *GoodsInfoCase,
	recommendReceiver *recommend.GoodsReceiver,
) *RecommendCase {
	return &RecommendCase{
		BaseCase:             baseCase,
		tx:                   tx,
		recommendRequestCase: recommendRequestCase,
		recommendEventCase:   recommendEventCase,
		orderGoodsCase:       orderGoodsCase,
		orderInfoCase:        orderInfoCase,
		userCartCase:         userCartCase,
		userCollectCase:      userCollectCase,
		goodsInfoCase:        goodsInfoCase,
		recommendReceiver:    recommendReceiver,
	}
}

// RecommendAnonymousActor 获取匿名推荐主体。
func (c *RecommendCase) RecommendAnonymousActor(ctx context.Context, _ *appv1.RecommendAnonymousActorRequest) (*appv1.RecommendAnonymousActorResponse, error) {
	anonymousID, err := c.getRecommendAnonymousIDFromHeader(ctx)
	if err != nil {
		return nil, err
	}
	// 请求头已携带匿名主体时，直接复用当前匿名身份。
	if anonymousID > 0 {
		return &appv1.RecommendAnonymousActorResponse{
			AnonymousId: anonymousID,
		}, nil
	}

	return &appv1.RecommendAnonymousActorResponse{
		AnonymousId: id.GenSnowflakeID(),
	}, nil
}

// BindRecommendAnonymousActor 绑定匿名推荐主体到当前登录用户。
func (c *RecommendCase) BindRecommendAnonymousActor(ctx context.Context, _ *appv1.BindRecommendAnonymousActorRequest) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}

	anonymousID := int64(0)
	anonymousID, err = c.getRecommendAnonymousIDFromHeader(ctx)
	if err != nil {
		return err
	}
	// 未携带匿名主体时，说明当前登录前没有匿名会话需要绑定。
	if anonymousID <= 0 {
		return nil
	}
	return c.bindRecommendAnonymousActor(ctx, authInfo.UserId, anonymousID)
}

// RecommendGoods 查询推荐商品列表。
func (c *RecommendCase) RecommendGoods(ctx context.Context, req *appv1.RecommendGoodsRequest) (*appv1.RecommendGoodsResponse, error) {
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
		TradeID:   req.GetTradeId(),
		RequestID: requestID,
		PageNum:   pageNum,
		PageSize:  pageSize,
	}

	var contextGoodsIDs []int64
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

	var recommendResult *dto.GoodsResult
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
			recommendReq.TradeID,
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

// getRecommendAnonymousIDFromHeader 从请求头中解析匿名主体编号。
func (c *RecommendCase) getRecommendAnonymousIDFromHeader(ctx context.Context) (int64, error) {
	serverTransport, ok := transport.FromServerContext(ctx)
	// 非服务端请求上下文时，不存在可读取的请求头。
	if !ok {
		return 0, nil
	}

	headerValue := serverTransport.RequestHeader().Get(RECOMMEND_ANONYMOUS_ACTOR_HEADER_KEY)
	// 未传入匿名主体请求头时，返回 0 表示当前请求未使用匿名身份。
	if headerValue == "" {
		return 0, nil
	}

	anonymousID, err := strconv.ParseInt(headerValue, 10, 64)
	if err != nil || anonymousID <= 0 {
		return 0, errorsx.InvalidArgument("匿名推荐主体无效")
	}
	return anonymousID, nil
}

// bindRecommendAnonymousActor 绑定匿名推荐主体到当前用户。
func (c *RecommendCase) bindRecommendAnonymousActor(ctx context.Context, userID, anonymousID int64) error {
	// 当前未携带匿名主体或用户编号非法时，无需继续绑定。
	if userID <= 0 || anonymousID <= 0 {
		return nil
	}

	anonymousEventList, err := c.listRecommendEventsByActor(ctx, commonv1.RecommendActorType(_const.RECOMMEND_ACTOR_TYPE_ANONYMOUS), anonymousID)
	if err != nil {
		return err
	}

	err = c.tx.Transaction(ctx, func(ctx context.Context) error {
		err = c.rebindRecommendRequestActor(ctx, userID, anonymousID)
		if err != nil {
			return err
		}
		err = c.rebindRecommendEventActor(ctx, userID, anonymousID)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	err = c.syncRecommendActorHistoryToRecommend(userID, anonymousEventList)
	if err != nil {
		log.Error(fmt.Sprintf("syncRecommendActorHistoryToRecommend %v", err))
	}
	return nil
}

// listRecommendEventsByActor 查询指定推荐主体的历史事件列表。
func (c *RecommendCase) listRecommendEventsByActor(
	ctx context.Context,
	actorType commonv1.RecommendActorType,
	actorID int64,
) ([]*models.RecommendEvent, error) {
	// 推荐主体编号非法时，不存在可迁移的历史事件。
	if actorID <= 0 {
		return []*models.RecommendEvent{}, nil
	}

	query := c.recommendEventCase.RecommendEventRepository.Query(ctx).RecommendEvent
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Order(query.EventAt.Desc()))
	opts = append(opts, repository.Where(query.ActorType.Eq(int32(actorType))))
	opts = append(opts, repository.Where(query.ActorID.Eq(actorID)))
	list, err := c.recommendEventCase.RecommendEventRepository.List(ctx, opts...)
	if err != nil {
		return nil, errorsx.Internal("查询匿名推荐事件失败").WithCause(err)
	}
	return list, nil
}

// rebindRecommendRequestActor 将匿名主体下的推荐请求记录迁移到登录用户。
func (c *RecommendCase) rebindRecommendRequestActor(ctx context.Context, userID, anonymousID int64) error {
	// 用户编号或匿名主体编号非法时，不存在可迁移的推荐请求记录。
	if userID <= 0 || anonymousID <= 0 {
		return nil
	}

	query := c.recommendRequestCase.RecommendRequestRepository.Query(ctx).RecommendRequest
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.ActorType.Eq(_const.RECOMMEND_ACTOR_TYPE_ANONYMOUS)))
	opts = append(opts, repository.Where(query.ActorID.Eq(anonymousID)))
	err := c.recommendRequestCase.RecommendRequestRepository.Update(ctx, &models.RecommendRequest{
		ActorType: _const.RECOMMEND_ACTOR_TYPE_USER,
		ActorID:   userID,
	}, opts...)
	if err != nil {
		return errorsx.Internal("迁移匿名推荐请求失败").WithCause(err)
	}
	return nil
}

// rebindRecommendEventActor 将匿名主体下的推荐事件记录迁移到登录用户。
func (c *RecommendCase) rebindRecommendEventActor(ctx context.Context, userID, anonymousID int64) error {
	// 用户编号或匿名主体编号非法时，不存在可迁移的推荐事件记录。
	if userID <= 0 || anonymousID <= 0 {
		return nil
	}

	query := c.recommendEventCase.RecommendEventRepository.Query(ctx).RecommendEvent
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.ActorType.Eq(_const.RECOMMEND_ACTOR_TYPE_ANONYMOUS)))
	opts = append(opts, repository.Where(query.ActorID.Eq(anonymousID)))
	err := c.recommendEventCase.RecommendEventRepository.Update(ctx, &models.RecommendEvent{
		ActorType: _const.RECOMMEND_ACTOR_TYPE_USER,
		ActorID:   userID,
	}, opts...)
	if err != nil {
		return errorsx.Internal("迁移匿名推荐事件失败").WithCause(err)
	}
	return nil
}

// syncRecommendActorHistoryToRecommend 异步回放匿名阶段历史到登录用户。
func (c *RecommendCase) syncRecommendActorHistoryToRecommend(
	userID int64,
	eventList []*models.RecommendEvent,
) error {
	// 用户编号非法或历史事件为空时，无需继续回放历史。
	if userID <= 0 || len(eventList) == 0 {
		return nil
	}

	replayEventList := make([]*models.RecommendEvent, 0, len(eventList))
	for _, item := range eventList {
		// 匿名历史写入推荐系统前，先改写成登录用户主体，避免匿名身份继续向下游投递。
		if item == nil {
			continue
		}
		replayEvent := *item
		replayEvent.ActorType = _const.RECOMMEND_ACTOR_TYPE_USER
		replayEvent.ActorID = userID
		replayEventList = append(replayEventList, &replayEvent)
	}
	if len(replayEventList) == 0 {
		return nil
	}

	queue.DispatchRecommendEventList(replayEventList)
	return nil
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
	anonymousID, err = c.getRecommendAnonymousIDFromHeader(ctx)
	if err != nil {
		return nil, err
	}
	// 未登录且未携带匿名主体时，允许继续走不可归因推荐，但后续不保存链路数据。
	if anonymousID <= 0 {
		return nil, nil
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
	var err error
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
			var list []int64
			list, err = c.userCartCase.listGoodsIDsByUserID(ctx, actor.ActorID)
			if err != nil {
				return nil, err
			}
			goodsIDs = list
		}
	case commonv1.RecommendScene(_const.RECOMMEND_SCENE_PROFILE):
		// 个人中心优先读取收藏商品做兴趣上下文。
		if actor != nil && actor.IsUser() {
			// 当前主体是登录用户时，收藏商品更能代表长期兴趣偏好。
			var list []int64
			list, err = c.userCollectCase.listGoodsIDsByUserID(ctx, actor.ActorID, RECOMMEND_RECENT_HISTORY_LIMIT)
			if err != nil {
				return nil, err
			}
			goodsIDs = list
		}
	case commonv1.RecommendScene(_const.RECOMMEND_SCENE_ORDER_DETAIL), commonv1.RecommendScene(_const.RECOMMEND_SCENE_ORDER_PAID):
		// 订单上下文必须归属当前用户，支付成功页按交易聚合全部门店商品。
		if actor == nil || !actor.IsUser() {
			break
		}
		orderQuery := c.orderInfoCase.Query(ctx).OrderInfo
		orderOpts := make([]repository.QueryOption, 0, 2)
		orderOpts = append(orderOpts, repository.Where(orderQuery.UserID.Eq(actor.ActorID)))
		if req.TradeID > 0 {
			orderOpts = append(orderOpts, repository.Where(orderQuery.TradeID.Eq(req.TradeID)))
		} else if req.OrderID > 0 {
			orderOpts = append(orderOpts, repository.Where(orderQuery.ID.Eq(req.OrderID)))
		} else {
			break
		}
		var orderInfos []*models.OrderInfo
		orderInfos, err = c.orderInfoCase.List(ctx, orderOpts...)
		if err != nil {
			return nil, err
		}
		orderIDs := make([]int64, 0, len(orderInfos))
		for _, orderInfo := range orderInfos {
			orderIDs = append(orderIDs, orderInfo.ID)
		}
		if len(orderIDs) == 0 {
			break
		}
		goodsQuery := c.orderGoodsCase.Query(ctx).OrderGoods
		goodsOpts := make([]repository.QueryOption, 0, 1)
		goodsOpts = append(goodsOpts, repository.Where(goodsQuery.OrderID.In(orderIDs...)))
		var orderGoodsList []*models.OrderGoods
		orderGoodsList, err = c.orderGoodsCase.List(ctx, goodsOpts...)
		if err != nil {
			return nil, err
		}
		for _, orderGoods := range orderGoodsList {
			goodsIDs = append(goodsIDs, orderGoods.GoodsID)
		}
	default:
		// 业务场景没有稳定上下文时，再回退到最近推荐行为商品。
		if actor != nil && actor.IsValid() {
			// 当前主体可识别时，回放最近推荐行为涉及的商品作为弱上下文。
			var list []int64
			list, err = c.recommendEventCase.listRecentRecommendEventGoodsIDs(ctx, actor)
			if err != nil {
				return nil, err
			}
			goodsIDs = list
		}
	}

	return _slice.Unique(goodsIDs), nil
}

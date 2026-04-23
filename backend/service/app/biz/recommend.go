package biz

import (
	"context"
	"time"

	"shop/api/gen/go/app"
	"shop/api/gen/go/common"
	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/models"
	pkgRecommend "shop/pkg/recommend"

	_slice "github.com/liujitcn/go-utils/slice"
	"github.com/liujitcn/gorm-kit/repo"
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
	onlineChain                 *pkgRecommend.OnlineChainReceiver
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
	onlineChain *pkgRecommend.OnlineChainReceiver,
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
		onlineChain:                 onlineChain,
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

	contextGoodsIds := make([]int64, 0)
	contextGoodsIds, err = c.listRecommendContextGoodsIds(ctx, actor, req)
	if err != nil {
		return nil, err
	}

	var recommendGoodsIds []int64
	total := int64(0)
	requestSource := common.RecommendRequestSource_LOCAL
	requestStatus := common.RecommendRequestStatus_REQUEST_FALLBACK
	strategyType := common.RecommendRequestStrategyType_UNKNOWN_RRST
	recommendGoodsIds, total, requestSource, requestStatus, strategyType, err = c.listRecommendGoodsIds(ctx, req.GetScene(), req.GetGoodsId(), actor, contextGoodsIds, pageNum, pageSize)
	if err != nil {
		return nil, err
	}
	var goodsList []*app.GoodsInfo
	goodsList, err = c.goodsInfoCase.listByGoodsIds(ctx, recommendGoodsIds)
	if err != nil {
		return nil, err
	}

	var requestId int64
	requestId, err = c.recommendRequestCase.resolveRecommendRequestId(ctx, actor, req)
	if err != nil {
		return nil, err
	}
	contextRecord := &app.RecommendRequestContext{
		GoodsId:         req.GetGoodsId(),
		OrderId:         req.GetOrderId(),
		ContextGoodsIds: contextGoodsIds,
		StrategyType:    strategyType,
		Source:          requestSource,
		Status:          requestStatus,
	}
	err = c.recommendRequestCase.saveRecommendRequest(ctx, actor, requestId, req, contextRecord, goodsList, total, pageNum, pageSize)
	if err != nil {
		return nil, err
	}

	return &app.RecommendGoodsResponse{
		List:      goodsList,
		Total:     int32(total),
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
func (c *RecommendCase) resolveRecommendActor(ctx context.Context, allowAnonymous bool) (*app.RecommendActor, error) {
	authInfo, err := auth.FromContext(ctx)
	// 当前请求已登录时，优先使用登录用户作为推荐主体。
	if err == nil && authInfo != nil && authInfo.UserId > 0 {
		return &app.RecommendActor{
			ActorType: common.RecommendActorType_USER,
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
	return &app.RecommendActor{
		ActorType: common.RecommendActorType_ANONYMOUS,
		ActorId:   anonymousId,
	}, nil
}

// listRecommendContextGoodsIds 查询当前推荐请求的上下文商品编号列表。
func (c *RecommendCase) listRecommendContextGoodsIds(
	ctx context.Context,
	actor *app.RecommendActor,
	req *app.RecommendGoodsRequest,
) ([]int64, error) {
	goodsIds := make([]int64, 0)
	switch req.GetScene() {
	case common.RecommendScene_GOODS_DETAIL:
		// 商品详情场景优先以当前商品作为锚点。
		if req.GetGoodsId() > 0 {
			goodsIds = append(goodsIds, req.GetGoodsId())
		}
	case common.RecommendScene_CART:
		// 登录态购物车页优先读取购物车商品做上下文。
		if actor != nil && actor.GetActorType() == common.RecommendActorType_USER {
			list, loadErr := c.userCartCase.listGoodsIdsByUserId(ctx, actor.GetActorId())
			if loadErr != nil {
				return nil, loadErr
			}
			goodsIds = list
		}
	case common.RecommendScene_PROFILE:
		// 个人中心优先读取收藏商品做兴趣上下文。
		if actor != nil && actor.GetActorType() == common.RecommendActorType_USER {
			list, loadErr := c.userCollectCase.listGoodsIdsByUserId(ctx, actor.GetActorId(), recommendRecentHistoryLimit)
			if loadErr != nil {
				return nil, loadErr
			}
			goodsIds = list
		}
	case common.RecommendScene_ORDER_DETAIL, common.RecommendScene_ORDER_PAID:
		// 订单详情与支付成功页优先读取订单商品做上下文。
		if req.GetOrderId() > 0 {
			list, loadErr := c.orderGoodsCase.listGoodsIdsByOrderId(ctx, req.GetOrderId())
			if loadErr != nil {
				return nil, loadErr
			}
			goodsIds = list
		}
	default:
		// 业务场景没有稳定上下文时，再回退到最近推荐行为商品。
		if actor != nil && actor.GetActorId() > 0 {
			list, loadErr := c.recommendEventCase.listRecentRecommendEventGoodsIds(ctx, actor)
			if loadErr != nil {
				return nil, loadErr
			}
			goodsIds = list
		}
	}

	return _slice.Unique(goodsIds), nil
}

// listRecommendGoodsIds 优先查询在线推荐并在必要时回退到本地兜底。
func (c *RecommendCase) listRecommendGoodsIds(
	ctx context.Context,
	scene common.RecommendScene,
	goodsId int64,
	actor *app.RecommendActor,
	contextGoodsIds []int64,
	pageNum, pageSize int64,
) ([]int64, int64, common.RecommendRequestSource, common.RecommendRequestStatus, common.RecommendRequestStrategyType, error) {
	// 当前推荐系统链路已启用时，优先尝试走在线推荐结果。
	if c.onlineChain.Enabled() {
		result, err := c.onlineChain.ExecuteOnlinePlan(ctx, scene, actor, goodsId, contextGoodsIds, pageNum, pageSize)
		if err != nil {
			return nil, 0, common.RecommendRequestSource_RECOMMEND, common.RecommendRequestStatus_REQUEST_SUCCESS, common.RecommendRequestStrategyType_UNKNOWN_RRST, nil
		}
		// 推荐系统返回了有效结果时，优先使用在线推荐结果。
		if len(result.GoodsIds) > 0 {
			return result.GoodsIds, result.Total, common.RecommendRequestSource_RECOMMEND, common.RecommendRequestStatus_REQUEST_SUCCESS, common.RecommendRequestStrategyType_UNKNOWN_RRST, nil
		}
	}

	// 有上下文商品时，优先按同类目兜底推荐。
	if len(contextGoodsIds) > 0 {
		goodsIds, total, err := c.pageGoodsIdsByCategory(ctx, contextGoodsIds, pageNum, pageSize)
		if err != nil {
			return nil, 0, common.RecommendRequestSource_UNKNOWN_RRSO, common.RecommendRequestStatus_UNKNOWN_RRQS, common.RecommendRequestStrategyType_UNKNOWN_RRST, err
		}
		// 同类目存在可推荐商品时，直接返回当前策略结果。
		if total > 0 {
			return goodsIds, total, common.RecommendRequestSource_LOCAL, common.RecommendRequestStatus_REQUEST_FALLBACK, common.RecommendRequestStrategyType_CATEGORY_FALLBACK, nil
		}
	}

	goodsIds, total, err := c.pageGoodsIdsByLatest(ctx, contextGoodsIds, pageNum, pageSize)
	if err != nil {
		return nil, 0, common.RecommendRequestSource_UNKNOWN_RRSO, common.RecommendRequestStatus_UNKNOWN_RRQS, common.RecommendRequestStrategyType_UNKNOWN_RRST, err
	}
	return goodsIds, total, common.RecommendRequestSource_LOCAL, common.RecommendRequestStatus_REQUEST_FALLBACK, common.RecommendRequestStrategyType_LATEST_FALLBACK, nil
}

// pageGoodsIdsByCategory 按上下文商品类目分页查询推荐商品。
func (c *RecommendCase) pageGoodsIdsByCategory(ctx context.Context, contextGoodsIds []int64, pageNum, pageSize int64) ([]int64, int64, error) {
	categoryIds, err := c.goodsInfoCase.listCategoryIdsByGoodsIds(ctx, contextGoodsIds)
	if err != nil {
		return nil, 0, err
	}
	// 上下文商品未能解析出类目时，当前策略没有可用候选集。
	if len(categoryIds) == 0 {
		return []int64{}, 0, nil
	}
	var goodsIdList []int64
	goodsIdList, err = c.goodsInfoCase.findGoodsIdsByCategoryIds(ctx, categoryIds)
	if err != nil {
		return nil, 0, err
	}
	// 分类条件无命中商品时，当前策略没有可用候选集。
	if len(goodsIdList) == 0 {
		return []int64{}, 0, nil
	}

	query := c.goodsInfoCase.Query(ctx).GoodsInfo
	opts := make([]repo.QueryOption, 0, 5)
	opts = append(opts, repo.Order(query.RealSaleNum.Desc()))
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))
	opts = append(opts, repo.Where(query.Status.Eq(int32(common.GoodsStatus_PUT_ON))))
	opts = append(opts, repo.Where(query.ID.In(goodsIdList...)))
	opts = append(opts, repo.Where(query.ID.NotIn(contextGoodsIds...)))
	var list []*models.GoodsInfo
	total := int64(0)
	list, total, err = c.goodsInfoCase.Page(ctx, pageNum, pageSize, opts...)
	if err != nil {
		return nil, 0, err
	}

	goodsIds := make([]int64, 0, len(list))
	for _, item := range list {
		goodsIds = append(goodsIds, item.ID)
	}
	return goodsIds, total, nil
}

// pageGoodsIdsByLatest 按最新热度分页查询推荐商品。
func (c *RecommendCase) pageGoodsIdsByLatest(ctx context.Context, excludedGoodsIds []int64, pageNum, pageSize int64) ([]int64, int64, error) {
	query := c.goodsInfoCase.Query(ctx).GoodsInfo
	opts := make([]repo.QueryOption, 0, 4)
	opts = append(opts, repo.Order(query.RealSaleNum.Desc()))
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))
	opts = append(opts, repo.Where(query.Status.Eq(int32(common.GoodsStatus_PUT_ON))))
	// 存在上下文商品时，从全局兜底里排除当前上下文商品本身。
	if len(excludedGoodsIds) > 0 {
		opts = append(opts, repo.Where(query.ID.NotIn(excludedGoodsIds...)))
	}
	list, total, err := c.goodsInfoCase.Page(ctx, pageNum, pageSize, opts...)
	if err != nil {
		return nil, 0, err
	}

	goodsIds := make([]int64, 0, len(list))
	for _, item := range list {
		goodsIds = append(goodsIds, item.ID)
	}
	return goodsIds, total, nil
}

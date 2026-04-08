package biz

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"shop/api/gen/go/app"
	"shop/api/gen/go/common"
	"shop/pkg/biz"
	_const "shop/pkg/const"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"shop/pkg/utils"

	"github.com/liujitcn/go-utils/id"
	"github.com/liujitcn/gorm-kit/repo"
	auth "github.com/liujitcn/kratos-kit/auth"
	queueData "github.com/liujitcn/kratos-kit/queue/data"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"gorm.io/gorm"
)

const (
	recommendStrategyVersion        = "v1"
	recommendEventTypeExposure      = "recommend_exposure"
	recommendEventTypeClick         = "recommend_click"
	recommendEventTypeView          = "goods_view"
	recommendEventTypeCollect       = "goods_collect"
	recommendEventTypeCart          = "goods_cart"
	recommendEventTypeOrder         = "order_create"
	recommendEventTypePay           = "order_pay"
	recommendAggregateWindowDays    = 30
	recommendPreferenceTypeCategory = "category"
	recommendRelationTypeCoClick    = "co_click"
	recommendRelationTypeCoView     = "co_view"
	recommendRelationTypeCoOrder    = "co_order"
	recommendRelationTypeCoPay      = "co_pay"
	recommendActorTypeAnonymous     = int32(0)
	recommendActorTypeUser          = int32(1)
	recommendAnonymousRecallDays    = 30
)

// RecommendCase 推荐业务处理对象。
type RecommendCase struct {
	*biz.BaseCase
	*data.RecommendRequestRepo
	*data.RecommendUserGoodsPreferenceRepo
	*data.RecommendExposureRepo
	*data.RecommendClickRepo
	goodsInfoRepo             *data.GoodsInfoRepo
	orderGoodsRepo            *data.OrderGoodsRepo
	userCartRepo              *data.UserCartRepo
	goodsStatDayRepo          *data.GoodsStatDayRepo
	recommendGoodsStatDayRepo *data.RecommendGoodsStatDayRepo
	recommendProfile          *RecommendProfileCase
	recommendRelation         *RecommendRelationCase
}

// NewRecommendCase 创建推荐业务处理对象。
func NewRecommendCase(
	baseCase *biz.BaseCase,
	recommendRequestRepo *data.RecommendRequestRepo,
	recommendUserGoodsPreferenceRepo *data.RecommendUserGoodsPreferenceRepo,
	recommendExposureRepo *data.RecommendExposureRepo,
	recommendClickRepo *data.RecommendClickRepo,
	goodsInfoRepo *data.GoodsInfoRepo,
	orderGoodsRepo *data.OrderGoodsRepo,
	userCartRepo *data.UserCartRepo,
	goodsStatDayRepo *data.GoodsStatDayRepo,
	recommendGoodsStatDayRepo *data.RecommendGoodsStatDayRepo,
	recommendProfile *RecommendProfileCase,
	recommendRelation *RecommendRelationCase,
) *RecommendCase {
	return &RecommendCase{
		BaseCase:                         baseCase,
		RecommendRequestRepo:             recommendRequestRepo,
		RecommendUserGoodsPreferenceRepo: recommendUserGoodsPreferenceRepo,
		RecommendExposureRepo:            recommendExposureRepo,
		RecommendClickRepo:               recommendClickRepo,
		goodsInfoRepo:                    goodsInfoRepo,
		orderGoodsRepo:                   orderGoodsRepo,
		userCartRepo:                     userCartRepo,
		goodsStatDayRepo:                 goodsStatDayRepo,
		recommendGoodsStatDayRepo:        recommendGoodsStatDayRepo,
		recommendProfile:                 recommendProfile,
		recommendRelation:                recommendRelation,
	}
}

// RecommendEvent 推荐行为异步事件。
type RecommendEvent struct {
	EventType  string                     `json:"eventType"`
	UserID     int64                      `json:"userId"`
	ActorType  int32                      `json:"actorType"`
	ActorID    int64                      `json:"actorId"`
	RequestID  string                     `json:"requestId,omitempty"`
	Scene      int32                      `json:"scene,omitempty"`
	Source     int32                      `json:"source,omitempty"`
	GoodsID    int64                      `json:"goodsId,omitempty"`
	GoodsIDs   []int64                    `json:"goodsIds,omitempty"`
	GoodsNum   int64                      `json:"goodsNum,omitempty"`
	GoodsItems []*RecommendEventGoodsItem `json:"goodsItems,omitempty"`
	Position   int32                      `json:"position,omitempty"`
	ExposeMode string                     `json:"exposeMode,omitempty"`
	ViewMode   string                     `json:"viewMode,omitempty"`
	OccurredAt int64                      `json:"occurredAt,omitempty"`
}

// RecommendEventGoodsItem 推荐强行为事件中的商品项。
type RecommendEventGoodsItem struct {
	GoodsID   int64  `json:"goodsId,omitempty"`
	GoodsNum  int64  `json:"goodsNum,omitempty"`
	Source    int32  `json:"source,omitempty"`
	Scene     int32  `json:"scene,omitempty"`
	RequestID string `json:"requestId,omitempty"`
	Position  int32  `json:"position,omitempty"`
}

// RecommendEventCase 推荐行为事件消费者。
type RecommendEventCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.RecommendExposureRepo
	*data.RecommendClickRepo
	*data.RecommendGoodsViewRepo
	*data.RecommendGoodsActionRepo
	*data.RecommendRequestRepo
	*data.RecommendUserPreferenceRepo
	*data.RecommendUserGoodsPreferenceRepo
	*data.RecommendGoodsRelationRepo
	*data.GoodsInfoRepo
}

// NewRecommendEventCase 创建推荐行为事件消费者并注册队列。
func NewRecommendEventCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	recommendExposureRepo *data.RecommendExposureRepo,
	recommendClickRepo *data.RecommendClickRepo,
	recommendGoodsViewRepo *data.RecommendGoodsViewRepo,
	recommendGoodsActionRepo *data.RecommendGoodsActionRepo,
	recommendRequestRepo *data.RecommendRequestRepo,
	recommendUserPreferenceRepo *data.RecommendUserPreferenceRepo,
	recommendUserGoodsPreferenceRepo *data.RecommendUserGoodsPreferenceRepo,
	recommendGoodsRelationRepo *data.RecommendGoodsRelationRepo,
	goodsInfoRepo *data.GoodsInfoRepo,
) *RecommendEventCase {
	c := &RecommendEventCase{
		BaseCase:                         baseCase,
		tx:                               tx,
		RecommendExposureRepo:            recommendExposureRepo,
		RecommendClickRepo:               recommendClickRepo,
		RecommendGoodsViewRepo:           recommendGoodsViewRepo,
		RecommendGoodsActionRepo:         recommendGoodsActionRepo,
		RecommendRequestRepo:             recommendRequestRepo,
		RecommendUserPreferenceRepo:      recommendUserPreferenceRepo,
		RecommendUserGoodsPreferenceRepo: recommendUserGoodsPreferenceRepo,
		RecommendGoodsRelationRepo:       recommendGoodsRelationRepo,
		GoodsInfoRepo:                    goodsInfoRepo,
	}

	c.RegisterQueueConsumer(_const.RecommendEvent, c.SaveRecommendEvent)
	return c
}

// RecommendGoods 查询推荐商品列表。
func (c *RecommendCase) RecommendAnonymousActor(ctx context.Context, req *emptypb.Empty) (*wrapperspb.Int64Value, error) {
	actorId := id.GenSnowflakeID()
	return wrapperspb.Int64(actorId), nil
}

// RecommendGoods 查询推荐商品列表。
func (c *RecommendCase) RecommendGoods(ctx context.Context, req *app.RecommendGoodsRequest) (*app.RecommendGoodsResponse, error) {
	// 统一兜底分页参数，避免前端漏传导致查询异常。
	pageNum := req.GetPageNum()
	// 页码非法时回退到首页，保证分页查询始终可执行。
	if pageNum <= 0 {
		pageNum = 1
	}
	pageSize := req.GetPageSize()
	// 每页数量非法时使用默认值，避免查全表或空分页。
	if pageSize <= 0 {
		pageSize = 10
	}
	req.PageNum = pageNum
	req.PageSize = pageSize
	// 每次推荐请求都生成独立 requestID，用于后续曝光归因。
	requestID := id.NewShortUUID()
	actor := c.resolveRecommendActor(ctx)

	var list []*app.GoodsInfo
	var total int64
	sourceContext := map[string]any{
		"orderId": req.GetOrderId(),
	}
	recallSources := make([]string, 0, 4)
	err := error(nil)
	// 匿名主体统一走公共推荐池，减少首页、购物车、我的三端内容分裂。
	if actor.ActorType == recommendActorTypeAnonymous {
		list, total, recallSources, sourceContext, err = c.listAnonymousRecommendGoods(ctx, actor, req, pageNum, pageSize)
	} else {
		var sceneGoodsIds []int64
		var sceneCategoryIds []int64
		sceneGoodsIds, sceneCategoryIds, sourceContext, recallSources, err = c.resolveSceneContext(ctx, req, actor.UserId, int(pageSize))
		if err == nil {
			list, total, recallSources, sourceContext, err = c.listRecommendGoods(ctx, actor, req, actor.UserId, sceneGoodsIds, sceneCategoryIds, pageNum, pageSize)
		}
	}
	if err != nil {
		return nil, err
	}
	sourceContext["actorType"] = actor.ActorType
	sourceContext["actorId"] = actor.ActorId

	err = c.saveRecommendRequest(ctx, requestID, actor, req, sourceContext, list, recallSources)
	if err != nil {
		return nil, err
	}

	return &app.RecommendGoodsResponse{
		List:      list,
		Total:     int32(total),
		RequestId: requestID,
	}, nil
}

// RecommendExposureReport 接收独立推荐曝光接口并异步投递事件。
func (c *RecommendEventCase) RecommendExposureReport(ctx context.Context, req *app.RecommendExposureReportRequest) error {
	// 空请求直接忽略，避免埋点接口影响主业务流程。
	if req == nil {
		return nil
	}

	actor := resolveRecommendActor(ctx)
	publishRecommendExposureEvent(
		actor,
		strings.TrimSpace(req.GetRequestId()),
		parseRecommendScene(req.GetScene()),
		req.GetGoodsIds(),
	)
	return nil
}

// RecommendGoodsActionReport 接收独立推荐商品行为接口并异步投递事件。
func (c *RecommendEventCase) RecommendGoodsActionReport(ctx context.Context, req *app.RecommendGoodsActionReportRequest) error {
	// 空请求直接返回，异步埋点不做额外失败放大。
	if req == nil {
		return nil
	}

	actor := resolveRecommendActor(ctx)
	// 按商品行为类型拆分投递不同事件，保持曝光与商品行为链路独立。
	switch req.GetEventType() {
	case common.RecommendGoodsActionType_RECOMMEND_GOODS_ACTION_CLICK:
		c.publishTrackGoodsEvents(actor, req.GetGoodsItems(), publishRecommendClickEvent)
	case common.RecommendGoodsActionType_RECOMMEND_GOODS_ACTION_VIEW:
		c.publishTrackGoodsViewEvents(actor, req.GetGoodsItems())
	case common.RecommendGoodsActionType_RECOMMEND_GOODS_ACTION_COLLECT:
		c.publishTrackGoodsEvents(actor, req.GetGoodsItems(), publishGoodsCollectEvent)
	case common.RecommendGoodsActionType_RECOMMEND_GOODS_ACTION_CART:
		c.publishTrackGoodsCartEvents(actor, req.GetGoodsItems())
	case common.RecommendGoodsActionType_RECOMMEND_GOODS_ACTION_ORDER_CREATE:
		publishOrderCreateEvent(actor, c.buildTrackGoodsItems(req.GetGoodsItems()))
	case common.RecommendGoodsActionType_RECOMMEND_GOODS_ACTION_ORDER_PAY:
		publishOrderPayEvent(actor, c.buildTrackGoodsItems(req.GetGoodsItems()))
	}
	return nil
}

// SaveRecommendEvent 消费推荐行为事件。
func (c *RecommendEventCase) SaveRecommendEvent(message queueData.Message) error {
	rawBody, err := json.Marshal(message.Values)
	if err != nil {
		return err
	}

	var payload map[string]*RecommendEvent
	if err = json.Unmarshal(rawBody, &payload); err != nil {
		return err
	}

	event, ok := payload["data"]
	// 队列消息缺少业务体时直接丢弃，避免消费者重复报错。
	if !ok || event == nil {
		return nil
	}
	return c.consume(context.TODO(), event)
}

// resolveSceneContext 解析不同推荐场景下的上下文信息。
func (c *RecommendCase) resolveSceneContext(ctx context.Context, req *app.RecommendGoodsRequest, userID int64, limit int) ([]int64, []int64, map[string]any, []string, error) {
	sourceContext := map[string]any{
		"orderId": req.GetOrderId(),
	}
	relationGoodsIds, categoryIds, recallSources, err := c.resolveSceneRecall(ctx, req, userID, sourceContext, limit)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	profileCategoryIds := make([]int64, 0)
	profileCategoryIds, err = c.recommendProfile.ListPreferredCategoryIds(ctx, userID, 3)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	// 用户画像只作为补充召回来源，不与场景召回互斥。
	if len(profileCategoryIds) > 0 {
		// 用户画像作为补充召回来源，不直接覆盖场景召回结果。
		categoryIds = append(categoryIds, profileCategoryIds...)
		recallSources = append(recallSources, "profile")
	}
	// 当场景和画像都没有可用数据时，最终标记为最新商品兜底。
	if len(recallSources) == 0 {
		// 没有任何场景或画像数据时，退化到最新商品兜底。
		recallSources = append(recallSources, "latest")
	}

	return dedupeInt64s(relationGoodsIds), dedupeInt64s(categoryIds), sourceContext, dedupeStrings(recallSources), nil
}

// resolveSceneRecall 解析推荐场景对应的召回商品与类目。
func (c *RecommendCase) resolveSceneRecall(ctx context.Context, req *app.RecommendGoodsRequest, userID int64, sourceContext map[string]any, limit int) ([]int64, []int64, []string, error) {
	relationGoodsIds := make([]int64, 0)
	categoryIds := make([]int64, 0)
	recallSources := make([]string, 0, 3)

	// 根据推荐场景选择不同的召回入口，优先复用最强业务上下文。
	switch req.GetScene() {
	case common.RecommendScene_CART:
		cartGoodsIds, err := c.listCurrentUserCartGoodsIds(ctx, userID)
		if err != nil {
			return nil, nil, nil, err
		}
		sourceContext["cartGoodsIds"] = cartGoodsIds
		// 当前购物车为空时，不再执行关联召回，交给后续画像或兜底逻辑处理。
		if len(cartGoodsIds) == 0 {
			return relationGoodsIds, categoryIds, recallSources, nil
		}

		// 购物车场景优先取购物车商品的关联商品。
		relationGoodsIds, err = c.recommendRelation.ListRelatedGoodsIds(ctx, cartGoodsIds, limit)
		if err != nil {
			return nil, nil, nil, err
		}
		// 关联商品不足时，再用购物车商品所属类目补足候选集。
		categoryIds, err = c.listCategoryIdsByGoodsIds(ctx, cartGoodsIds)
		if err != nil {
			return nil, nil, nil, err
		}
		recallSources = append(recallSources, "cart")
	case common.RecommendScene_ORDER_DETAIL, common.RecommendScene_ORDER_PAID:
		// 订单场景没有订单号时无法做强关联召回，直接返回空场景结果。
		if req.GetOrderId() <= 0 {
			return relationGoodsIds, categoryIds, recallSources, nil
		}

		orderGoodsIds, err := c.listOrderGoodsIds(ctx, req.GetOrderId())
		if err != nil {
			return nil, nil, nil, err
		}
		// 订单详情和支付成功都优先基于订单商品做强关联召回。
		relationGoodsIds, err = c.recommendRelation.ListRelatedGoodsIds(ctx, orderGoodsIds, limit)
		if err != nil {
			return nil, nil, nil, err
		}
		categoryIds, err = c.listCategoryIdsByGoodsIds(ctx, orderGoodsIds)
		if err != nil {
			return nil, nil, nil, err
		}
		recallSources = append(recallSources, "order")
	}

	return relationGoodsIds, categoryIds, recallSources, nil
}

// listCurrentUserCartGoodsIds 查询当前用户购物车中的商品ID列表。
func (c *RecommendCase) listCurrentUserCartGoodsIds(ctx context.Context, userID int64) ([]int64, error) {
	// 未登录用户没有专属购物车，直接返回空集合。
	if userID == 0 {
		return []int64{}, nil
	}

	userCartQuery := c.userCartRepo.Query(ctx).UserCart
	list, err := c.userCartRepo.List(ctx,
		repo.Where(userCartQuery.UserID.Eq(userID)),
	)
	if err != nil {
		return nil, err
	}

	goodsIds := make([]int64, 0, len(list))
	for _, item := range list {
		goodsIds = append(goodsIds, item.GoodsID)
	}
	return dedupeInt64s(goodsIds), nil
}

// listOrderGoodsIds 查询订单中的商品ID列表。
func (c *RecommendCase) listOrderGoodsIds(ctx context.Context, orderID int64) ([]int64, error) {
	orderGoodsQuery := c.orderGoodsRepo.Query(ctx).OrderGoods
	list, err := c.orderGoodsRepo.List(ctx,
		repo.Where(orderGoodsQuery.OrderID.Eq(orderID)),
	)
	if err != nil {
		return nil, err
	}

	goodsIds := make([]int64, 0, len(list))
	for _, item := range list {
		goodsIds = append(goodsIds, item.GoodsID)
	}
	return dedupeInt64s(goodsIds), nil
}

// listCategoryIdsByGoodsIds 根据商品ID列表查询分类ID列表。
func (c *RecommendCase) listCategoryIdsByGoodsIds(ctx context.Context, goodsIds []int64) ([]int64, error) {
	// 没有商品上下文时无需访问数据库查询类目。
	if len(goodsIds) == 0 {
		return []int64{}, nil
	}

	goodsQuery := c.goodsInfoRepo.Query(ctx).GoodsInfo
	list, err := c.goodsInfoRepo.List(ctx,
		repo.Where(goodsQuery.ID.In(goodsIds...)),
	)
	if err != nil {
		return nil, err
	}

	categoryIds := make([]int64, 0, len(list))
	for _, item := range list {
		categoryIds = append(categoryIds, item.CategoryID)
	}
	return dedupeInt64s(categoryIds), nil
}

// listGoodsByIds 按商品ID顺序查询商品信息。
func (c *RecommendCase) listGoodsByIds(ctx context.Context, goodsIds []int64) ([]*models.GoodsInfo, error) {
	// 空商品集合不触发数据库查询，直接返回空结果。
	if len(goodsIds) == 0 {
		return []*models.GoodsInfo{}, nil
	}

	goodsQuery := c.goodsInfoRepo.Query(ctx).GoodsInfo
	list, err := c.goodsInfoRepo.List(ctx,
		repo.Where(goodsQuery.ID.In(goodsIds...)),
		repo.Where(goodsQuery.Status.Eq(int32(common.GoodsStatus_PUT_ON))),
		repo.Order(goodsQuery.CreatedAt.Desc()),
	)
	if err != nil {
		return nil, err
	}

	goodsMap := make(map[int64]*models.GoodsInfo, len(list))
	for _, item := range list {
		goodsMap[item.ID] = item
	}
	// 数据库 IN 查询不保证原顺序，这里按输入顺序重新组装结果。
	result := make([]*models.GoodsInfo, 0, len(goodsIds))
	for _, goodsID := range goodsIds {
		item, ok := goodsMap[goodsID]
		if !ok {
			continue
		}
		result = append(result, item)
	}
	return result, nil
}

// pageGoods 分页查询推荐商品。
func (c *RecommendCase) pageGoods(ctx context.Context, categoryIds []int64, excludeGoodsIds []int64, pageNum, pageSize int64) ([]*models.GoodsInfo, int64, error) {
	// 分页大小非法时不再继续查询，避免产生无意义 SQL。
	if pageSize <= 0 {
		return []*models.GoodsInfo{}, 0, nil
	}

	goodsQuery := c.goodsInfoRepo.Query(ctx).GoodsInfo
	opts := make([]repo.QueryOption, 0, 5)
	opts = append(opts, repo.Where(goodsQuery.Status.Eq(int32(common.GoodsStatus_PUT_ON))))
	opts = append(opts, repo.Order(goodsQuery.CreatedAt.Desc()))
	// 有类目召回时优先限制查询范围，保持推荐结果与场景相关。
	if len(categoryIds) > 0 {
		opts = append(opts, repo.Where(goodsQuery.CategoryID.In(categoryIds...)))
	}
	// 已输出的商品需要排除，避免同一页内重复命中。
	if len(excludeGoodsIds) > 0 {
		// 已被优先命中的商品不再重复进入补足列表。
		opts = append(opts, repo.Where(goodsQuery.ID.NotIn(excludeGoodsIds...)))
	}
	return c.goodsInfoRepo.Page(ctx, pageNum, pageSize, opts...)
}

// saveRecommendRequest 保存推荐请求记录。
func (c *RecommendCase) saveRecommendRequest(ctx context.Context, requestID string, actor *RecommendActor, req *app.RecommendGoodsRequest, sourceContext map[string]any, list []*app.GoodsInfo, recallSources []string) error {
	sourceContextJSON, err := json.Marshal(sourceContext)
	if err != nil {
		return err
	}

	goodsIds := make([]int64, 0, len(list))
	for _, item := range list {
		goodsIds = append(goodsIds, item.GetId())
	}

	goodsIdsJSON, err := json.Marshal(goodsIds)
	if err != nil {
		return err
	}
	recallSourcesJSON, err := json.Marshal(recallSources)
	if err != nil {
		return err
	}

	// 推荐请求表保存的是本次实际下发结果，供曝光与点击链路统一回查。
	entity := &models.RecommendRequest{
		RequestID:         requestID,
		ActorType:         actor.ActorType,
		ActorID:           actor.ActorId,
		Scene:             int32(req.GetScene()),
		SourceContextJSON: string(sourceContextJSON),
		PageNum:           int32(req.GetPageNum()),
		PageSize:          int32(req.GetPageSize()),
		GoodsIdsJSON:      string(goodsIdsJSON),
		StrategyVersion:   recommendStrategyVersion,
		RecallSourcesJSON: string(recallSourcesJSON),
	}
	return c.RecommendRequestRepo.Create(ctx, entity)
}

// convertGoodsToProto 将商品模型转换为推荐商品响应。
func (c *RecommendCase) convertGoodsToProto(item *models.GoodsInfo, member bool) *app.GoodsInfo {
	price := item.Price
	// 会员用户优先展示会员价，其余用户沿用原价。
	if member {
		price = item.DiscountPrice
	}
	return &app.GoodsInfo{
		Id:      item.ID,
		Name:    item.Name,
		Desc:    item.Desc,
		Picture: item.Picture,
		SaleNum: item.InitSaleNum + item.RealSaleNum,
		Price:   price,
	}
}

// consume 在同一事务中保存明细并执行聚合。
func (c *RecommendEventCase) consume(ctx context.Context, event *RecommendEvent) error {
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		err := c.saveEventDetail(ctx, event)
		if err != nil {
			return err
		}
		return c.aggregateEvent(ctx, event)
	})
}

// saveEventDetail 保存推荐行为明细。
func (c *RecommendEventCase) saveEventDetail(ctx context.Context, event *RecommendEvent) error {
	// 不同事件类型落不同明细表，保证曝光、点击、浏览各自独立存储。
	switch event.EventType {
	case recommendEventTypeExposure:
		goodsIdsJson, err := json.Marshal(event.GoodsIDs)
		if err != nil {
			return err
		}
		return c.RecommendExposureRepo.Create(ctx, &models.RecommendExposure{
			RequestID:    event.RequestID,
			ActorType:    event.ActorType,
			ActorID:      event.ActorID,
			Scene:        event.Scene,
			GoodsIdsJSON: string(goodsIdsJson),
			ExposeMode:   defaultString(event.ExposeMode, "viewport_once"),
		})
	case recommendEventTypeClick:
		err := c.RecommendClickRepo.Create(ctx, &models.RecommendClick{
			RequestID: event.RequestID,
			ActorType: event.ActorType,
			ActorID:   event.ActorID,
			Scene:     event.Scene,
			GoodsID:   event.GoodsID,
			Position:  event.Position,
			Source:    normalizeRecommendSourceCode(event.Source, common.RecommendSource_RECOMMEND),
		})
		if err != nil {
			return err
		}
		return c.RecommendGoodsActionRepo.Create(ctx, &models.RecommendGoodsAction{
			ActorType: event.ActorType,
			ActorID:   event.ActorID,
			EventType: event.EventType,
			GoodsID:   event.GoodsID,
			GoodsNum:  c.normalizeGoodsCount(event.GoodsNum),
			Source:    normalizeRecommendSourceCode(event.Source, common.RecommendSource_RECOMMEND),
			Scene:     event.Scene,
			RequestID: event.RequestID,
			Position:  event.Position,
		})
	case recommendEventTypeView:
		err := c.RecommendGoodsViewRepo.Create(ctx, &models.RecommendGoodsView{
			ActorType: event.ActorType,
			ActorID:   event.ActorID,
			GoodsID:   event.GoodsID,
			Source:    normalizeRecommendSourceCode(event.Source, common.RecommendSource_DIRECT),
			Scene:     event.Scene,
			RequestID: event.RequestID,
			Position:  event.Position,
			ViewMode:  defaultString(event.ViewMode, "detail_open"),
		})
		if err != nil {
			return err
		}
		return c.RecommendGoodsActionRepo.Create(ctx, &models.RecommendGoodsAction{
			ActorType: event.ActorType,
			ActorID:   event.ActorID,
			EventType: event.EventType,
			GoodsID:   event.GoodsID,
			GoodsNum:  c.normalizeGoodsCount(event.GoodsNum),
			Source:    normalizeRecommendSourceCode(event.Source, common.RecommendSource_DIRECT),
			Scene:     event.Scene,
			RequestID: event.RequestID,
			Position:  event.Position,
		})
	case recommendEventTypeCollect, recommendEventTypeCart:
		return c.RecommendGoodsActionRepo.Create(ctx, &models.RecommendGoodsAction{
			ActorType: event.ActorType,
			ActorID:   event.ActorID,
			EventType: event.EventType,
			GoodsID:   event.GoodsID,
			GoodsNum:  c.normalizeGoodsCount(event.GoodsNum),
			Source:    normalizeRecommendSourceCode(event.Source, common.RecommendSource_DIRECT),
			Scene:     event.Scene,
			RequestID: event.RequestID,
			Position:  event.Position,
		})
	case recommendEventTypeOrder, recommendEventTypePay:
		for _, goodsItem := range c.normalizeGoodsItems(event.GoodsItems) {
			err := c.RecommendGoodsActionRepo.Create(ctx, &models.RecommendGoodsAction{
				ActorType: event.ActorType,
				ActorID:   event.ActorID,
				EventType: event.EventType,
				GoodsID:   goodsItem.GoodsID,
				GoodsNum:  c.normalizeGoodsCount(goodsItem.GoodsNum),
				Source:    normalizeRecommendSourceCode(goodsItem.Source, common.RecommendSource_DIRECT),
				Scene:     goodsItem.Scene,
				RequestID: goodsItem.RequestID,
				Position:  goodsItem.Position,
			})
			if err != nil {
				return err
			}
		}
		return nil
	default:
		return nil
	}
}

// aggregateEvent 聚合推荐行为到画像与商品关联表。
func (c *RecommendEventCase) aggregateEvent(ctx context.Context, event *RecommendEvent) error {
	// 空事件无需聚合，避免事务内继续做无效操作。
	if event == nil {
		return nil
	}
	// 匿名用户不沉淀画像，只保留必要明细。
	if event.UserID <= 0 {
		return nil
	}
	// 单商品行为优先走单商品聚合链路。
	if c.isSingleGoodsEvent(event.EventType) {
		return c.aggregateSingleGoodsEvent(ctx, event)
	}
	// 订单级行为需要拆成多商品聚合并维护共现关系。
	if c.isOrderGoodsEvent(event.EventType) {
		return c.aggregateOrderGoodsEvent(ctx, event)
	}
	return nil
}

// aggregateSingleGoodsEvent 聚合单商品行为。
func (c *RecommendEventCase) aggregateSingleGoodsEvent(ctx context.Context, event *RecommendEvent) error {
	// 单商品行为缺少商品ID时无法聚合，直接忽略。
	if event.GoodsID <= 0 {
		return nil
	}

	goodsInfo, err := c.findGoodsInfo(ctx, event.GoodsID)
	if err != nil {
		return err
	}

	eventTime := c.getEventTime(event)
	err = c.upsertUserGoodsPreference(ctx, event, eventTime, event.GoodsNum)
	if err != nil {
		return err
	}
	err = c.upsertUserCategoryPreference(ctx, event, goodsInfo.CategoryID, eventTime, event.GoodsNum)
	if err != nil {
		return err
	}
	return c.upsertGoodsRelation(ctx, event, eventTime)
}

// aggregateOrderGoodsEvent 聚合订单级强行为。
func (c *RecommendEventCase) aggregateOrderGoodsEvent(ctx context.Context, event *RecommendEvent) error {
	eventTime := c.getEventTime(event)
	goodsItems := c.normalizeGoodsItems(event.GoodsItems)
	for _, goodsItem := range goodsItems {
		singleEvent := &RecommendEvent{
			EventType:  event.EventType,
			UserID:     event.UserID,
			GoodsID:    goodsItem.GoodsID,
			GoodsNum:   goodsItem.GoodsNum,
			Source:     goodsItem.Source,
			Scene:      goodsItem.Scene,
			RequestID:  goodsItem.RequestID,
			Position:   goodsItem.Position,
			OccurredAt: event.OccurredAt,
		}

		goodsInfo, err := c.findGoodsInfo(ctx, goodsItem.GoodsID)
		if err != nil {
			return err
		}
		err = c.upsertUserGoodsPreference(ctx, singleEvent, eventTime, goodsItem.GoodsNum)
		if err != nil {
			return err
		}
		err = c.upsertUserCategoryPreference(ctx, singleEvent, goodsInfo.CategoryID, eventTime, goodsItem.GoodsNum)
		if err != nil {
			return err
		}
	}
	return c.upsertOrderGoodsRelations(ctx, event, goodsItems, eventTime)
}

// upsertUserGoodsPreference 累计用户对具体商品的偏好得分。
func (c *RecommendEventCase) upsertUserGoodsPreference(ctx context.Context, event *RecommendEvent, eventTime time.Time, goodsNum int64) error {
	recommendUserGoodsPreferenceQuery := c.RecommendUserGoodsPreferenceRepo.Query(ctx).RecommendUserGoodsPreference
	entity, err := c.RecommendUserGoodsPreferenceRepo.Find(ctx,
		repo.Where(recommendUserGoodsPreferenceQuery.UserID.Eq(event.UserID)),
		repo.Where(recommendUserGoodsPreferenceQuery.GoodsID.Eq(event.GoodsID)),
		repo.Where(recommendUserGoodsPreferenceQuery.WindowDays.Eq(recommendAggregateWindowDays)),
	)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	summaryJSON := ""
	score := c.getEventWeight(event.EventType) * c.normalizeGoodsNum(goodsNum)
	// 已有画像记录时在原有得分与摘要上继续累加。
	if entity != nil {
		score += entity.Score
		summaryJSON = entity.BehaviorSummaryJSON
	}
	summaryJSON, err = addBehaviorSummaryCount(summaryJSON, c.getEventSummaryKey(event.EventType), c.normalizeGoodsCount(goodsNum))
	if err != nil {
		return err
	}

	if entity == nil || entity.ID == 0 {
		return c.RecommendUserGoodsPreferenceRepo.Create(ctx, &models.RecommendUserGoodsPreference{
			UserID:              event.UserID,
			GoodsID:             event.GoodsID,
			Score:               score,
			LastBehaviorType:    event.EventType,
			LastBehaviorAt:      eventTime,
			BehaviorSummaryJSON: summaryJSON,
			WindowDays:          recommendAggregateWindowDays,
			CreatedAt:           eventTime,
			UpdatedAt:           eventTime,
		})
	}

	entity.Score = score
	entity.LastBehaviorType = event.EventType
	entity.LastBehaviorAt = eventTime
	entity.BehaviorSummaryJSON = summaryJSON
	entity.UpdatedAt = eventTime
	return c.RecommendUserGoodsPreferenceRepo.UpdateById(ctx, entity)
}

// upsertUserCategoryPreference 累计用户对商品类目的偏好得分。
func (c *RecommendEventCase) upsertUserCategoryPreference(ctx context.Context, event *RecommendEvent, categoryID int64, eventTime time.Time, goodsNum int64) error {
	// 商品没有有效类目时，不生成类目偏好画像。
	if categoryID <= 0 {
		return nil
	}

	recommendUserPreferenceQuery := c.RecommendUserPreferenceRepo.Query(ctx).RecommendUserPreference
	entity, err := c.RecommendUserPreferenceRepo.Find(ctx,
		repo.Where(recommendUserPreferenceQuery.UserID.Eq(event.UserID)),
		repo.Where(recommendUserPreferenceQuery.PreferenceType.Eq(recommendPreferenceTypeCategory)),
		repo.Where(recommendUserPreferenceQuery.TargetID.Eq(categoryID)),
		repo.Where(recommendUserPreferenceQuery.WindowDays.Eq(recommendAggregateWindowDays)),
	)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	summaryJSON := ""
	score := c.getEventWeight(event.EventType) * c.normalizeGoodsNum(goodsNum)
	// 已有类目画像时继续在原值上累积行为分数。
	if entity != nil {
		score += entity.Score
		summaryJSON = entity.BehaviorSummaryJSON
	}
	summaryJSON, err = addBehaviorSummaryCount(summaryJSON, c.getEventSummaryKey(event.EventType), c.normalizeGoodsCount(goodsNum))
	if err != nil {
		return err
	}

	if entity == nil || entity.ID == 0 {
		return c.RecommendUserPreferenceRepo.Create(ctx, &models.RecommendUserPreference{
			UserID:              event.UserID,
			PreferenceType:      recommendPreferenceTypeCategory,
			TargetID:            categoryID,
			Score:               score,
			BehaviorSummaryJSON: summaryJSON,
			WindowDays:          recommendAggregateWindowDays,
			CreatedAt:           eventTime,
			UpdatedAt:           eventTime,
		})
	}

	entity.Score = score
	entity.BehaviorSummaryJSON = summaryJSON
	entity.UpdatedAt = eventTime
	return c.RecommendUserPreferenceRepo.UpdateById(ctx, entity)
}

// upsertGoodsRelation 按同一次推荐请求的共同出现结果累计商品关联度。
func (c *RecommendEventCase) upsertGoodsRelation(ctx context.Context, event *RecommendEvent, eventTime time.Time) error {
	// 只有来源于推荐位的请求才参与商品共现关系沉淀。
	if event.RequestID == "" || !isRecommendSource(normalizeRecommendSourceCode(event.Source, common.RecommendSource_DIRECT)) {
		return nil
	}

	relationType := c.getRelationType(event.EventType)
	// 无法映射为关系类型的行为不进入商品关联聚合。
	if relationType == "" {
		return nil
	}

	relatedGoodsIds, err := c.listRequestRelatedGoodsIds(ctx, event.RequestID, event.GoodsID)
	if err != nil {
		return err
	}
	// 共同出现在同一请求中的商品两两建立双向关联关系。
	for _, relatedGoodsID := range relatedGoodsIds {
		err = c.upsertSingleGoodsRelation(ctx, relatedGoodsID, event.GoodsID, relationType, eventTime, c.normalizeGoodsNum(event.GoodsNum))
		if err != nil {
			return err
		}
		err = c.upsertSingleGoodsRelation(ctx, event.GoodsID, relatedGoodsID, relationType, eventTime, c.normalizeGoodsNum(event.GoodsNum))
		if err != nil {
			return err
		}
	}
	return nil
}

// upsertOrderGoodsRelations 累计订单内商品的共购与共支付关系。
func (c *RecommendEventCase) upsertOrderGoodsRelations(ctx context.Context, event *RecommendEvent, goodsItems []*RecommendEventGoodsItem, eventTime time.Time) error {
	relationType := c.getRelationType(event.EventType)
	// 仅下单和支付这类订单级事件需要建立共购关系。
	if relationType == "" {
		return nil
	}

	// 同一订单内商品两两组合，沉淀双向关联强度。
	for i := 0; i < len(goodsItems); i++ {
		leftItem := goodsItems[i]
		for j := i + 1; j < len(goodsItems); j++ {
			rightItem := goodsItems[j]
			relationScore := c.normalizeGoodsNum(leftItem.GoodsNum) + c.normalizeGoodsNum(rightItem.GoodsNum)
			err := c.upsertSingleGoodsRelation(ctx, leftItem.GoodsID, rightItem.GoodsID, relationType, eventTime, relationScore)
			if err != nil {
				return err
			}
			err = c.upsertSingleGoodsRelation(ctx, rightItem.GoodsID, leftItem.GoodsID, relationType, eventTime, relationScore)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// findGoodsInfo 查询商品信息，供画像聚合读取类目。
func (c *RecommendEventCase) findGoodsInfo(ctx context.Context, goodsID int64) (*models.GoodsInfo, error) {
	return c.GoodsInfoRepo.FindById(ctx, goodsID)
}

// listRequestRelatedGoodsIds 读取推荐请求中与当前商品共同出现的其他商品。
func (c *RecommendEventCase) listRequestRelatedGoodsIds(ctx context.Context, requestID string, goodsID int64) ([]int64, error) {
	recommendRequestQuery := c.RecommendRequestRepo.Query(ctx).RecommendRequest
	entity, err := c.RecommendRequestRepo.Find(ctx,
		repo.Where(recommendRequestQuery.RequestID.Eq(requestID)),
	)
	if err != nil {
		// 历史请求不存在时不报错，说明当前事件无法回溯推荐列表。
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []int64{}, nil
		}
		return nil, err
	}

	goodsIds := make([]int64, 0)
	err = json.Unmarshal([]byte(entity.GoodsIdsJSON), &goodsIds)
	if err != nil {
		return nil, err
	}

	relatedGoodsIds := make([]int64, 0, len(goodsIds))
	// 过滤当前商品自身与非法值，只保留同请求下的其他商品。
	for _, item := range goodsIds {
		if item == 0 || item == goodsID {
			continue
		}
		relatedGoodsIds = append(relatedGoodsIds, item)
	}
	return dedupeInt64s(relatedGoodsIds), nil
}

// upsertSingleGoodsRelation 累计单个方向的商品关联强度。
func (c *RecommendEventCase) upsertSingleGoodsRelation(ctx context.Context, goodsID, relatedGoodsID int64, relationType string, eventTime time.Time, relationScore float64) error {
	// 非法商品对不进入关系计算，避免写入脏数据。
	if goodsID <= 0 || relatedGoodsID <= 0 || goodsID == relatedGoodsID {
		return nil
	}

	recommendGoodsRelationQuery := c.RecommendGoodsRelationRepo.Query(ctx).RecommendGoodsRelation
	entity, err := c.RecommendGoodsRelationRepo.Find(ctx,
		repo.Where(recommendGoodsRelationQuery.GoodsID.Eq(goodsID)),
		repo.Where(recommendGoodsRelationQuery.RelatedGoodsID.Eq(relatedGoodsID)),
		repo.Where(recommendGoodsRelationQuery.RelationType.Eq(relationType)),
		repo.Where(recommendGoodsRelationQuery.WindowDays.Eq(recommendAggregateWindowDays)),
	)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	evidenceJSON := ""
	score := relationScore
	// 外部未传分值时，回退到关系类型默认权重。
	if score <= 0 {
		score = c.getRelationWeight(relationType)
	}
	// 已存在关系记录时在原有强度上继续累加。
	if entity != nil {
		score += entity.Score
		evidenceJSON = entity.EvidenceJSON
	}
	evidenceJSON, err = addBehaviorSummaryCount(evidenceJSON, relationType, int64(score))
	if err != nil {
		return err
	}

	if entity == nil || entity.ID == 0 {
		return c.RecommendGoodsRelationRepo.Create(ctx, &models.RecommendGoodsRelation{
			GoodsID:        goodsID,
			RelatedGoodsID: relatedGoodsID,
			RelationType:   relationType,
			Score:          score,
			EvidenceJSON:   evidenceJSON,
			WindowDays:     recommendAggregateWindowDays,
			CreatedAt:      eventTime,
			UpdatedAt:      eventTime,
		})
	}

	entity.Score = score
	entity.EvidenceJSON = evidenceJSON
	entity.UpdatedAt = eventTime
	return c.RecommendGoodsRelationRepo.UpdateById(ctx, entity)
}

// publishRecommendExposureEvent 投递推荐曝光事件。
func publishRecommendExposureEvent(actor *RecommendActor, requestID string, scene int32, goodsIds []int64) {
	utils.AddQueue(_const.RecommendEvent, &RecommendEvent{
		EventType:  recommendEventTypeExposure,
		UserID:     actor.UserId,
		ActorType:  actor.ActorType,
		ActorID:    actor.ActorId,
		RequestID:  requestID,
		Scene:      scene,
		GoodsIDs:   goodsIds,
		ExposeMode: "viewport_once",
		OccurredAt: time.Now().Unix(),
	})
}

// publishRecommendClickEvent 投递推荐点击事件。
func publishRecommendClickEvent(actor *RecommendActor, goodsID int64, requestID string, scene, source int32, position int32) {
	utils.AddQueue(_const.RecommendEvent, &RecommendEvent{
		EventType:  recommendEventTypeClick,
		UserID:     actor.UserId,
		ActorType:  actor.ActorType,
		ActorID:    actor.ActorId,
		RequestID:  requestID,
		Scene:      scene,
		Source:     source,
		GoodsID:    goodsID,
		GoodsNum:   1,
		Position:   position,
		OccurredAt: time.Now().Unix(),
	})
}

// publishGoodsViewEvent 投递商品浏览事件。
func publishGoodsViewEvent(actor *RecommendActor, goodsID int64, position int32, requestID string, source, scene int32) {
	utils.AddQueue(_const.RecommendEvent, &RecommendEvent{
		EventType:  recommendEventTypeView,
		UserID:     actor.UserId,
		ActorType:  actor.ActorType,
		ActorID:    actor.ActorId,
		RequestID:  requestID,
		Scene:      scene,
		Source:     source,
		GoodsID:    goodsID,
		GoodsNum:   1,
		Position:   position,
		ViewMode:   "detail_open",
		OccurredAt: time.Now().Unix(),
	})
}

// publishGoodsCollectEvent 投递商品收藏事件。
func publishGoodsCollectEvent(actor *RecommendActor, goodsID int64, requestID string, scene, source int32, position int32) {
	utils.AddQueue(_const.RecommendEvent, &RecommendEvent{
		EventType:  recommendEventTypeCollect,
		UserID:     actor.UserId,
		ActorType:  actor.ActorType,
		ActorID:    actor.ActorId,
		RequestID:  requestID,
		Scene:      scene,
		Source:     source,
		GoodsID:    goodsID,
		GoodsNum:   1,
		Position:   position,
		OccurredAt: time.Now().Unix(),
	})
}

// publishGoodsCartEvent 投递商品加购事件。
func publishGoodsCartEvent(actor *RecommendActor, goodsID, goodsNum int64, requestID string, scene, source int32, position int32) {
	utils.AddQueue(_const.RecommendEvent, &RecommendEvent{
		EventType:  recommendEventTypeCart,
		UserID:     actor.UserId,
		ActorType:  actor.ActorType,
		ActorID:    actor.ActorId,
		RequestID:  requestID,
		Scene:      scene,
		Source:     source,
		GoodsID:    goodsID,
		GoodsNum:   goodsNum,
		Position:   position,
		OccurredAt: time.Now().Unix(),
	})
}

// publishOrderCreateEvent 投递下单事件。
func publishOrderCreateEvent(actor *RecommendActor, goodsItems []*RecommendEventGoodsItem) {
	utils.AddQueue(_const.RecommendEvent, &RecommendEvent{
		EventType:  recommendEventTypeOrder,
		UserID:     actor.UserId,
		ActorType:  actor.ActorType,
		ActorID:    actor.ActorId,
		GoodsItems: goodsItems,
		OccurredAt: time.Now().Unix(),
	})
}

// publishOrderPayEvent 投递支付成功事件。
func publishOrderPayEvent(actor *RecommendActor, goodsItems []*RecommendEventGoodsItem) {
	utils.AddQueue(_const.RecommendEvent, &RecommendEvent{
		EventType:  recommendEventTypePay,
		UserID:     actor.UserId,
		ActorType:  actor.ActorType,
		ActorID:    actor.ActorId,
		GoodsItems: goodsItems,
		OccurredAt: time.Now().Unix(),
	})
}

// buildRecommendEventGoodsItems 将订单商品转换为推荐事件商品项。
func buildRecommendEventGoodsItems(orderGoodsList []*models.OrderGoods) []*RecommendEventGoodsItem {
	goodsItems := make([]*RecommendEventGoodsItem, 0, len(orderGoodsList))
	for _, orderGoods := range orderGoodsList {
		if orderGoods == nil || orderGoods.GoodsID <= 0 {
			continue
		}
		goodsItems = append(goodsItems, &RecommendEventGoodsItem{
			GoodsID:   orderGoods.GoodsID,
			GoodsNum:  orderGoods.Num,
			Source:    orderGoods.Source,
			Scene:     orderGoods.Scene,
			RequestID: orderGoods.RequestID,
			Position:  orderGoods.Position,
		})
	}
	return goodsItems
}

// publishTrackGoodsEvents 批量投递单商品埋点事件。
func (c *RecommendEventCase) publishTrackGoodsEvents(actor *RecommendActor, goodsItems []*app.RecommendGoodsActionItem, publishFn func(actor *RecommendActor, goodsID int64, requestID string, scene, source int32, position int32)) {
	for _, goodsItem := range goodsItems {
		// 商品项为空或缺少商品ID时跳过，避免发送脏事件。
		if goodsItem == nil || goodsItem.GetGoodsId() <= 0 {
			continue
		}
		recommendContext := goodsItem.GetRecommendContext()
		publishFn(
			actor,
			goodsItem.GetGoodsId(),
			strings.TrimSpace(recommendContext.GetRequestId()),
			parseRecommendScene(recommendContext.GetScene()),
			normalizeRecommendSource(recommendContext.GetSource()),
			recommendContext.GetPosition(),
		)
	}
}

// publishTrackGoodsViewEvents 批量投递商品浏览埋点事件。
func (c *RecommendEventCase) publishTrackGoodsViewEvents(actor *RecommendActor, goodsItems []*app.RecommendGoodsActionItem) {
	for _, goodsItem := range goodsItems {
		// 无效商品项不参与浏览埋点投递。
		if goodsItem == nil || goodsItem.GetGoodsId() <= 0 {
			continue
		}
		recommendContext := goodsItem.GetRecommendContext()
		publishGoodsViewEvent(
			actor,
			goodsItem.GetGoodsId(),
			recommendContext.GetPosition(),
			strings.TrimSpace(recommendContext.GetRequestId()),
			normalizeRecommendSource(recommendContext.GetSource()),
			parseRecommendScene(recommendContext.GetScene()),
		)
	}
}

// publishTrackGoodsCartEvents 批量投递加购埋点事件。
func (c *RecommendEventCase) publishTrackGoodsCartEvents(actor *RecommendActor, goodsItems []*app.RecommendGoodsActionItem) {
	for _, goodsItem := range goodsItems {
		// 加购埋点只接受带有效商品ID的商品项。
		if goodsItem == nil || goodsItem.GetGoodsId() <= 0 {
			continue
		}
		recommendContext := goodsItem.GetRecommendContext()
		publishGoodsCartEvent(
			actor,
			goodsItem.GetGoodsId(),
			goodsItem.GetGoodsNum(),
			strings.TrimSpace(recommendContext.GetRequestId()),
			parseRecommendScene(recommendContext.GetScene()),
			normalizeRecommendSource(recommendContext.GetSource()),
			recommendContext.GetPosition(),
		)
	}
}

// buildTrackGoodsItems 将 proto 商品项转换为内部事件商品项。
func (c *RecommendEventCase) buildTrackGoodsItems(goodsItems []*app.RecommendGoodsActionItem) []*RecommendEventGoodsItem {
	list := make([]*RecommendEventGoodsItem, 0, len(goodsItems))
	for _, goodsItem := range goodsItems {
		// 过滤非法商品项，避免订单级事件混入空数据。
		if goodsItem == nil || goodsItem.GetGoodsId() <= 0 {
			continue
		}
		recommendContext := goodsItem.GetRecommendContext()
		list = append(list, &RecommendEventGoodsItem{
			GoodsID:   goodsItem.GetGoodsId(),
			GoodsNum:  goodsItem.GetGoodsNum(),
			Source:    normalizeRecommendSource(recommendContext.GetSource()),
			Scene:     parseRecommendScene(recommendContext.GetScene()),
			RequestID: strings.TrimSpace(recommendContext.GetRequestId()),
			Position:  recommendContext.GetPosition(),
		})
	}
	return list
}

func normalizeRecommendSourceCode(source int32, defaultSource common.RecommendSource) int32 {
	if source > 0 {
		return source
	}
	return int32(defaultSource)
}

// getRecommendUserID 获取推荐场景下的用户ID。
func (c *RecommendCase) getRecommendUserID(ctx context.Context) int64 {
	return getRecommendUserID(ctx)
}

// isSingleGoodsEvent 判断是否为单商品行为事件。
func (c *RecommendEventCase) isSingleGoodsEvent(eventType string) bool {
	// 点击、浏览、收藏、加购都只作用于单个商品。
	switch eventType {
	case recommendEventTypeClick, recommendEventTypeView, recommendEventTypeCollect, recommendEventTypeCart:
		return true
	default:
		return false
	}
}

// isOrderGoodsEvent 判断是否为订单级商品事件。
func (c *RecommendEventCase) isOrderGoodsEvent(eventType string) bool {
	// 下单和支付会同时影响订单内多件商品的关系。
	switch eventType {
	case recommendEventTypeOrder, recommendEventTypePay:
		return true
	default:
		return false
	}
}

// normalizeGoodsItems 过滤非法商品项并兜底数量。
func (c *RecommendEventCase) normalizeGoodsItems(goodsItems []*RecommendEventGoodsItem) []*RecommendEventGoodsItem {
	list := make([]*RecommendEventGoodsItem, 0, len(goodsItems))
	for _, goodsItem := range goodsItems {
		// 缺少商品ID的记录直接剔除，避免后续聚合失败。
		if goodsItem == nil || goodsItem.GoodsID <= 0 {
			continue
		}
		// 商品数量缺失时统一按 1 处理，保证画像权重可计算。
		if goodsItem.GoodsNum <= 0 {
			goodsItem.GoodsNum = 1
		}
		list = append(list, goodsItem)
	}
	return list
}

// normalizeGoodsNum 统一商品数量的下限。
func (c *RecommendEventCase) normalizeGoodsNum(goodsNum int64) float64 {
	// 数量非正时使用 1 兜底，避免权重被错误归零。
	if goodsNum <= 0 {
		return 1
	}
	return float64(goodsNum)
}

// normalizeGoodsCount 统一商品数量计数的下限。
func (c *RecommendEventCase) normalizeGoodsCount(goodsNum int64) int64 {
	// 计数字段与权重保持一致，也需要做最小值兜底。
	if goodsNum <= 0 {
		return 1
	}
	return goodsNum
}

// getEventTime 获取事件发生时间，未传时退化为当前时间。
func (c *RecommendEventCase) getEventTime(event *RecommendEvent) time.Time {
	// 事件未显式携带时间时，以消费时间作为聚合时间。
	if event == nil || event.OccurredAt <= 0 {
		return time.Now()
	}
	return time.Unix(event.OccurredAt, 0)
}

// getEventWeight 返回用户偏好聚合所使用的事件权重。
func (c *RecommendEventCase) getEventWeight(eventType string) float64 {
	// 不同行为强度不同，支付和下单权重高于浏览点击。
	switch eventType {
	case recommendEventTypeClick:
		return 3
	case recommendEventTypeView:
		return 2
	case recommendEventTypeCollect:
		return 4
	case recommendEventTypeCart:
		return 6
	case recommendEventTypeOrder:
		return 8
	case recommendEventTypePay:
		return 10
	default:
		return 0
	}
}

// getRelationWeight 返回商品关联聚合所使用的关系权重。
func (c *RecommendEventCase) getRelationWeight(relationType string) float64 {
	// 商品关系默认强度按行为价值分层，支付关系最高。
	switch relationType {
	case recommendRelationTypeCoClick:
		return 3
	case recommendRelationTypeCoView:
		return 2
	case recommendRelationTypeCoOrder:
		return 8
	case recommendRelationTypeCoPay:
		return 10
	default:
		return 0
	}
}

// getRelationType 根据事件类型映射商品关联类型。
func (c *RecommendEventCase) getRelationType(eventType string) string {
	// 仅支持能形成商品关系的行为类型映射。
	if eventType == recommendEventTypeClick {
		return recommendRelationTypeCoClick
	}
	if eventType == recommendEventTypeView {
		return recommendRelationTypeCoView
	}
	if eventType == recommendEventTypeOrder {
		return recommendRelationTypeCoOrder
	}
	if eventType == recommendEventTypePay {
		return recommendRelationTypeCoPay
	}
	return ""
}

// getEventSummaryKey 返回行为汇总 JSON 中的计数字段名。
func (c *RecommendEventCase) getEventSummaryKey(eventType string) string {
	// 行为摘要 JSON 使用稳定字段名记录各类行为次数。
	if eventType == recommendEventTypeClick {
		return "click_count"
	}
	if eventType == recommendEventTypeView {
		return "view_count"
	}
	if eventType == recommendEventTypeCollect {
		return "collect_count"
	}
	if eventType == recommendEventTypeCart {
		return "cart_count"
	}
	if eventType == recommendEventTypeOrder {
		return "order_count"
	}
	if eventType == recommendEventTypePay {
		return "pay_count"
	}
	return ""
}

// addBehaviorSummaryCount 累加 JSON 汇总中的行为计数。
func addBehaviorSummaryCount(summaryJSON, key string, delta int64) (string, error) {
	// 缺少有效字段名或增量时，不再生成新的 JSON 内容。
	if key == "" || delta == 0 {
		return summaryJSON, nil
	}

	summary := make(map[string]int64)
	// 已有摘要内容时先反序列化后再累加新计数。
	if summaryJSON != "" {
		err := json.Unmarshal([]byte(summaryJSON), &summary)
		if err != nil {
			return "", err
		}
	}
	summary[key] += delta
	rawBody, err := json.Marshal(summary)
	if err != nil {
		return "", err
	}
	return string(rawBody), nil
}

// getRecommendUserID 获取推荐场景下的用户ID。
func getRecommendUserID(ctx context.Context) int64 {
	authInfo, err := auth.FromContext(ctx)
	// 上下文中没有登录态时返回 0，保持推荐链路可匿名工作。
	if err != nil || authInfo == nil {
		return 0
	}
	return authInfo.UserId
}

// defaultString 返回优先值，为空时回退到默认值。
func defaultString(value, fallback string) string {
	// 空字符串统一回退默认值，避免落库字段缺失语义。
	if value == "" {
		return fallback
	}
	return value
}

// dedupeInt64s 去重整型切片。
func dedupeInt64s(values []int64) []int64 {
	result := make([]int64, 0, len(values))
	seen := make(map[int64]struct{}, len(values))
	for _, value := range values {
		// 无效主键值不参与去重结果。
		if value == 0 {
			continue
		}
		// 已处理过的值直接跳过，保持结果顺序稳定。
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

// dedupeStrings 去重字符串切片。
func dedupeStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		// 空字符串不写入结果，避免产生无意义标识。
		if value == "" {
			continue
		}
		// 重复字符串只保留首次出现的顺序。
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

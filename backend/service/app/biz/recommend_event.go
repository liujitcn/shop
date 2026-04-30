package biz

import (
	"context"
	"time"

	appv1 "shop/api/gen/go/app/v1"
	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/biz"
	_const "shop/pkg/const"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"shop/pkg/queue"
	"shop/pkg/recommend/dto"

	"github.com/liujitcn/gorm-kit/repository"
	queueData "github.com/liujitcn/kratos-kit/queue/data"
)

// RecommendEventCase 推荐事件业务处理对象。
type RecommendEventCase struct {
	*biz.BaseCase
	*data.RecommendEventRepository
	recommendRequestCase *RecommendRequestCase
}

// NewRecommendEventCase 创建推荐事件业务处理对象。
func NewRecommendEventCase(
	baseCase *biz.BaseCase,
	recommendEventRepo *data.RecommendEventRepository,
	recommendRequestCase *RecommendRequestCase,
) *RecommendEventCase {
	c := &RecommendEventCase{
		BaseCase:                 baseCase,
		RecommendEventRepository: recommendEventRepo,
		recommendRequestCase:     recommendRequestCase,
	}

	// 注册推荐事件异步消费者，统一承接后端事实回写。
	c.RegisterQueueConsumer(_const.RECOMMEND_EVENT_REPORT, c.saveRecommendEventReport)
	return c
}

// saveRecommendEventReport 消费推荐事件队列并持久化到本地。
func (c *RecommendEventCase) saveRecommendEventReport(message queueData.Message) error {
	recommendEvent, err := queue.DecodeQueueData[queue.RecommendEventReportEvent](message)
	if err != nil {
		return err
	}
	// 队列消息里没有推荐事件主体时，直接忽略当前消息。
	if recommendEvent == nil {
		return nil
	}

	items := make([]*appv1.RecommendEventItem, 0, len(recommendEvent.Items))
	for _, item := range recommendEvent.Items {
		// 非法商品项直接跳过，避免把脏数据写入推荐链路。
		if item == nil || item.GoodsID <= 0 {
			continue
		}
		items = append(items, &appv1.RecommendEventItem{
			GoodsId:  item.GoodsID,
			GoodsNum: item.GoodsNum,
			Position: item.Position,
		})
	}
	// 队列事件里没有有效商品项时，不再继续落库。
	if len(items) == 0 {
		return nil
	}

	recommendEventReport := &appv1.RecommendEventReportRequest{
		EventType: recommendEvent.EventType,
		RecommendContext: &appv1.RecommendEventContext{
			Scene:     commonv1.RecommendScene(recommendEvent.Scene),
			RequestId: recommendEvent.RequestID,
		},
		Items: items,
	}
	return c.persistRecommendEventReport(context.TODO(), recommendEvent.RecommendActor, recommendEventReport, recommendEvent.EventTime)
}

// persistRecommendEventReport 持久化推荐事件。
func (c *RecommendEventCase) persistRecommendEventReport(
	ctx context.Context,
	actor *dto.RecommendActor,
	req *appv1.RecommendEventReportRequest,
	eventTime time.Time,
) error {
	// 空请求直接忽略，避免埋点影响主流程。
	if req == nil {
		return nil
	}
	// 主体缺失或主体编号非法时，当前事件无法归因。
	if actor == nil || !actor.IsValid() {
		return errorsx.InvalidArgument("推荐主体不能为空")
	}
	// 事件类型未知时，不写入推荐事件表。
	if req.GetEventType() == commonv1.RecommendEventType(_const.RECOMMEND_EVENT_TYPE_UNKNOWN) {
		return errorsx.InvalidArgument("推荐事件类型不能为空")
	}
	// 调用方未显式传入事件时间时，统一回退到当前时间。
	if eventTime.IsZero() {
		eventTime = time.Now()
	}

	recommendContext := req.GetRecommendContext()
	scene := int32(0)
	requestID := int64(0)
	// 请求携带推荐归因上下文时，再补齐场景和请求编号。
	if recommendContext != nil {
		scene = int32(recommendContext.GetScene())
		requestID = recommendContext.GetRequestId()
	}

	eventList := make([]*models.RecommendEvent, 0, len(req.GetItems()))
	for _, item := range req.GetItems() {
		// 非法商品项直接跳过，避免把脏数据写入推荐链路。
		if item == nil || item.GetGoodsId() <= 0 {
			continue
		}

		goodsNum := item.GetGoodsNum()
		// 未显式传入商品数量时，统一按 1 处理。
		if goodsNum <= 0 {
			goodsNum = 1
		}

		eventList = append(eventList, &models.RecommendEvent{
			ActorType: int32(actor.ActorType),
			ActorID:   actor.ActorID,
			Scene:     scene,
			EventType: int32(req.GetEventType()),
			GoodsID:   item.GetGoodsId(),
			GoodsNum:  int32(goodsNum),
			RequestID: requestID,
			Position:  item.GetPosition(),
			EventAt:   eventTime,
		})
	}
	// 经过清洗后没有可写入事件时，直接结束。
	if len(eventList) == 0 {
		return nil
	}

	err := c.RecommendEventRepository.BatchCreate(ctx, eventList)
	if err != nil {
		return errorsx.Internal("保存推荐事件失败").WithCause(err)
	}

	// 本地推荐事件落库成功后，再异步投递到推荐系统，避免推荐系统异常阻塞主流程。
	queue.DispatchRecommendEventList(eventList)
	return nil
}

// listRecentRecommendEventGoodsIDs 查询当前主体最近的推荐行为商品编号列表。
func (c *RecommendEventCase) listRecentRecommendEventGoodsIDs(ctx context.Context, actor *dto.RecommendActor) ([]int64, error) {
	goodsIDs := make([]int64, 0)
	// 主体缺失或主体编号非法时，不存在可用的最近行为上下文。
	if actor == nil || !actor.IsValid() {
		return goodsIDs, nil
	}

	query := c.RecommendEventRepository.Query(ctx).RecommendEvent
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Order(query.EventAt.Desc()))
	opts = append(opts, repository.Where(query.ActorType.Eq(int32(actor.ActorType))))
	opts = append(opts, repository.Where(query.ActorID.Eq(actor.ActorID)))
	// 最近行为上下文仅使用能体现兴趣偏好的事件，不包含曝光。
	opts = append(opts, repository.Where(query.EventType.In(
		_const.RECOMMEND_EVENT_TYPE_CLICK,
		_const.RECOMMEND_EVENT_TYPE_VIEW,
		_const.RECOMMEND_EVENT_TYPE_COLLECT,
		_const.RECOMMEND_EVENT_TYPE_ADD_CART,
		_const.RECOMMEND_EVENT_TYPE_ORDER_CREATE,
		_const.RECOMMEND_EVENT_TYPE_ORDER_PAY,
	)))
	opts = append(opts, repository.Limit(RECOMMEND_RECENT_HISTORY_LIMIT))
	list, err := c.RecommendEventRepository.List(ctx, opts...)
	if err != nil {
		return nil, errorsx.Internal("查询最近推荐事件失败").WithCause(err)
	}

	seenGoods := make(map[int64]struct{}, len(list))
	for _, item := range list {
		// 商品编号非法或已加入结果集时，直接跳过。
		if item == nil || item.GoodsID <= 0 {
			continue
		}
		if _, ok := seenGoods[item.GoodsID]; ok {
			continue
		}
		seenGoods[item.GoodsID] = struct{}{}
		goodsIDs = append(goodsIDs, item.GoodsID)
	}
	return goodsIDs, nil
}

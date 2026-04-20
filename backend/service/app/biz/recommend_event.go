package biz

import (
	"context"
	"shop/pkg/gorse"
	"time"

	"shop/api/gen/go/app"
	"shop/api/gen/go/common"
	"shop/pkg/biz"
	_const "shop/pkg/const"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	pkgUtils "shop/pkg/utils"

	"github.com/liujitcn/gorm-kit/repo"
	queueData "github.com/liujitcn/kratos-kit/queue/data"
)

// RecommendEventCase 推荐事件业务处理对象。
type RecommendEventCase struct {
	*biz.BaseCase
	*data.RecommendEventRepo
	recommendRequestCase *RecommendRequestCase
	gorse                *gorse.Gorse
}

// NewRecommendEventCase 创建推荐事件业务处理对象。
func NewRecommendEventCase(
	baseCase *biz.BaseCase,
	recommendEventRepo *data.RecommendEventRepo,
	recommendRequestCase *RecommendRequestCase,
	gorse *gorse.Gorse,
) *RecommendEventCase {
	c := &RecommendEventCase{
		BaseCase:             baseCase,
		RecommendEventRepo:   recommendEventRepo,
		recommendRequestCase: recommendRequestCase,
		gorse:                gorse,
	}

	// 注册推荐事件异步消费者，统一承接后端事实回写。
	c.RegisterQueueConsumer(_const.RecommendEventReport, c.saveRecommendEventReport)
	return c
}

// saveRecommendEventReport 消费推荐事件队列并持久化到本地。
func (c *RecommendEventCase) saveRecommendEventReport(message queueData.Message) error {
	recommendEvent, err := pkgUtils.DecodeQueueData[pkgUtils.RecommendEventReportEvent](message)
	if err != nil {
		return err
	}
	// 队列消息里没有推荐事件主体时，直接忽略当前消息。
	if recommendEvent == nil {
		return nil
	}

	items := make([]*app.RecommendEventItem, 0, len(recommendEvent.Items))
	for _, item := range recommendEvent.Items {
		// 非法商品项直接跳过，避免把脏数据写入推荐链路。
		if item == nil || item.GoodsId <= 0 {
			continue
		}
		items = append(items, &app.RecommendEventItem{
			GoodsId:  item.GoodsId,
			GoodsNum: item.GoodsNum,
			Position: item.Position,
		})
	}
	// 队列事件里没有有效商品项时，不再继续落库。
	if len(items) == 0 {
		return nil
	}

	recommendEventReport := &app.RecommendEventReportRequest{
		EventType: recommendEvent.EventType,
		RecommendContext: &app.RecommendEventContext{
			Scene:     common.RecommendScene(recommendEvent.Scene),
			RequestId: recommendEvent.RequestId,
		},
		Items: items,
	}
	return c.persistRecommendEventReport(context.TODO(), recommendEvent.RecommendActor, recommendEventReport, recommendEvent.EventTime)
}

// persistRecommendEventReport 持久化推荐事件。
func (c *RecommendEventCase) persistRecommendEventReport(
	ctx context.Context,
	actor *app.RecommendActor,
	req *app.RecommendEventReportRequest,
	eventTime time.Time,
) error {
	// 空请求直接忽略，避免埋点影响主流程。
	if req == nil {
		return nil
	}
	// 主体缺失或主体编号非法时，当前事件无法归因。
	if actor == nil || actor.GetActorId() <= 0 {
		return errorsx.InvalidArgument("推荐主体不能为空")
	}
	// 事件类型未知时，不写入推荐事件表。
	if req.GetEventType() == common.RecommendEventType_UNKNOWN_RET {
		return errorsx.InvalidArgument("推荐事件类型不能为空")
	}
	// 调用方未显式传入事件时间时，统一回退到当前时间。
	if eventTime.IsZero() {
		eventTime = time.Now()
	}

	recommendContext := req.GetRecommendContext()
	scene := int32(0)
	requestId := int64(0)
	// 请求携带推荐归因上下文时，再补齐场景和请求编号。
	if recommendContext != nil {
		scene = int32(recommendContext.GetScene())
		requestId = recommendContext.GetRequestId()
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
			ActorType: int32(actor.GetActorType()),
			ActorID:   actor.GetActorId(),
			Scene:     scene,
			EventType: int32(req.GetEventType()),
			GoodsID:   item.GetGoodsId(),
			GoodsNum:  int32(goodsNum),
			RequestID: requestId,
			Position:  item.GetPosition(),
			EventAt:   eventTime,
		})
	}
	// 经过清洗后没有可写入事件时，直接结束。
	if len(eventList) == 0 {
		return nil
	}

	err := c.RecommendEventRepo.BatchCreate(ctx, eventList)
	if err != nil {
		return errorsx.Internal("保存推荐事件失败").WithCause(err)
	}
	return nil
}

// listRecentRecommendEventGoodsIds 查询当前主体最近的推荐行为商品编号列表。
func (c *RecommendEventCase) listRecentRecommendEventGoodsIds(ctx context.Context, actor *app.RecommendActor) ([]int64, error) {
	goodsIds := make([]int64, 0)
	// 主体缺失或主体编号非法时，不存在可用的最近行为上下文。
	if actor == nil || actor.GetActorId() <= 0 {
		return goodsIds, nil
	}

	query := c.RecommendEventRepo.Query(ctx).RecommendEvent
	opts := make([]repo.QueryOption, 0, 4)
	opts = append(opts, repo.Order(query.EventAt.Desc()))
	opts = append(opts, repo.Where(query.ActorType.Eq(int32(actor.GetActorType()))))
	opts = append(opts, repo.Where(query.ActorID.Eq(actor.GetActorId())))
	// 最近行为上下文仅使用能体现兴趣偏好的事件，不包含曝光。
	opts = append(opts, repo.Where(query.EventType.In(
		int32(common.RecommendEventType_CLICK),
		int32(common.RecommendEventType_VIEW),
		int32(common.RecommendEventType_COLLECT),
		int32(common.RecommendEventType_ADD_CART),
		int32(common.RecommendEventType_ORDER_CREATE),
		int32(common.RecommendEventType_ORDER_PAY),
	)))
	opts = append(opts, repo.Limit(recommendRecentHistoryLimit))
	list, err := c.RecommendEventRepo.List(ctx, opts...)
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
		goodsIds = append(goodsIds, item.GoodsID)
	}
	return goodsIds, nil
}

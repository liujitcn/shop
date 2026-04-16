package biz

import (
	"context"
	"encoding/json"
	"time"

	"shop/api/gen/go/app"
	"shop/pkg/biz"
	_const "shop/pkg/const"
	"shop/pkg/gen/data"
	recommendDomain "shop/pkg/recommend/domain"
	recommendEvent "shop/pkg/recommend/event"
	recommendAggregate "shop/pkg/recommend/offline/aggregate"
	"shop/pkg/utils"
	appDto "shop/service/app/dto"

	queueData "github.com/liujitcn/kratos-kit/queue/data"
)

// RecommendGoodsActionCase 推荐商品行为业务处理对象。
type RecommendGoodsActionCase struct {
	*biz.BaseCase
	tx data.Transaction // 事务执行器，用于保证事实落库与投影更新按同一事务提交。
	*data.RecommendGoodsActionRepo
	goodsActionProjector *recommendAggregate.GoodsActionProjector // 商品行为投影器，推荐聚合逻辑统一下沉到 pkg/recommend。
}

// NewRecommendGoodsActionCase 创建推荐商品行为业务处理对象。
func NewRecommendGoodsActionCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	recommendGoodsActionRepo *data.RecommendGoodsActionRepo,
	goodsActionProjector *recommendAggregate.GoodsActionProjector,
) *RecommendGoodsActionCase {
	recommendGoodsActionCase := &RecommendGoodsActionCase{
		BaseCase:                 baseCase,
		tx:                       tx,
		RecommendGoodsActionRepo: recommendGoodsActionRepo,
		goodsActionProjector:     goodsActionProjector,
	}
	recommendGoodsActionCase.RegisterQueueConsumer(_const.RecommendGoodsActionEvent, recommendGoodsActionCase.saveRecommendGoodsActionEvent)
	return recommendGoodsActionCase
}

// saveRecommendGoodsActionEvent 消费推荐商品行为事件。
func (c *RecommendGoodsActionCase) saveRecommendGoodsActionEvent(message queueData.Message) error {
	rawBody, err := json.Marshal(message.Values)
	if err != nil {
		return err
	}

	payload := make(map[string]*utils.RecommendGoodsActionEvent)
	err = json.Unmarshal(rawBody, &payload)
	if err != nil {
		return err
	}

	event, ok := payload["data"]
	// 队列消息缺少业务体时直接丢弃，避免消费者重复报错。
	if !ok || event == nil {
		return nil
	}

	list := event.GoodsItems
	// 队列消息没有商品行为明细时，不再继续落库和聚合。
	if len(list) == 0 {
		return nil
	}

	return c.tx.Transaction(context.TODO(), func(ctx context.Context) error {
		// 先写入行为事实，再由 pkg/recommend 中的投影器更新聚合结果。
		err = c.BatchCreate(ctx, list)
		if err != nil {
			return err
		}
		return c.goodsActionProjector.Project(ctx, c.buildGoodsActionProjectionEvent(event))
	})
}

// buildGoodsActionProjectionEvent 将队列事件收敛为推荐聚合领域事件。
func (c *RecommendGoodsActionCase) buildGoodsActionProjectionEvent(event *utils.RecommendGoodsActionEvent) *recommendDomain.GoodsActionProjectionEvent {
	// 队列事件为空时，不构造领域事件对象。
	if event == nil {
		return nil
	}

	actorType := int32(0)
	actorId := int64(0)
	// 队列消息包含主体时，继续补齐领域事件主体信息。
	if event.RecommendActor != nil {
		actorType = event.RecommendActor.ActorType
		actorId = event.RecommendActor.ActorId
	}
	return &recommendDomain.GoodsActionProjectionEvent{
		ActorType:  actorType,
		ActorId:    actorId,
		EventType:  event.EventType,
		EventTime:  event.EventTime,
		GoodsItems: event.GoodsItems,
	}
}

// bindRecommendGoodsActionActor 将匿名行为主体绑定为登录主体。
func (c *RecommendGoodsActionCase) bindRecommendGoodsActionActor(ctx context.Context, anonymousId, userId int64) error {
	query := c.RecommendGoodsActionRepo.Data.Query(ctx).RecommendGoodsAction
	_, err := query.WithContext(ctx).
		Where(
			query.ActorType.Eq(recommendEvent.ActorTypeAnonymous),
			query.ActorID.Eq(anonymousId),
		).
		Updates(map[string]interface{}{
			"actor_type": recommendEvent.ActorTypeUser,
			"actor_id":   userId,
		})
	return err
}

// publishRecommendGoodsActionEvent 投递单商品埋点事件。
func (c *RecommendGoodsActionCase) publishRecommendGoodsActionEvent(actor *appDto.RecommendActor, req *app.RecommendGoodsActionReportRequest) {
	// 空请求直接忽略，避免埋点接口影响主流程。
	if req == nil {
		return
	}

	utils.DispatchRecommendGoodsActionEvent(actor, req, time.Now())
}

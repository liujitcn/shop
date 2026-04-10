package biz

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"shop/api/gen/go/app"
	"shop/api/gen/go/common"
	"shop/pkg/biz"
	_const "shop/pkg/const"
	"shop/pkg/gen/data"
	recommendactor "shop/pkg/recommend/actor"
	recommendcontext "shop/pkg/recommend/context"
	recommendevent "shop/pkg/recommend/event"
	"shop/pkg/utils"
	appdto "shop/service/app/dto"

	queueData "github.com/liujitcn/kratos-kit/queue/data"
)

// RecommendGoodsActionCase 推荐商品行为业务处理对象。
type RecommendGoodsActionCase struct {
	*biz.BaseCase
	*data.RecommendGoodsActionRepo
	recommendExposureCase *RecommendExposureCase
}

// NewRecommendGoodsActionCase 创建推荐商品行为业务处理对象。
func NewRecommendGoodsActionCase(baseCase *biz.BaseCase, recommendGoodsActionRepo *data.RecommendGoodsActionRepo, recommendExposureCase *RecommendExposureCase) *RecommendGoodsActionCase {
	recommendGoodsActionCase := &RecommendGoodsActionCase{
		BaseCase:                 baseCase,
		RecommendGoodsActionRepo: recommendGoodsActionRepo,
		recommendExposureCase:    recommendExposureCase,
	}
	recommendGoodsActionCase.RegisterQueueConsumer(_const.RecommendGoodsActionEvent, recommendGoodsActionCase.SaveRecommendEvent)
	return recommendGoodsActionCase
}

// RecommendGoodsActionReport 接收独立推荐商品行为接口并异步投递事件。
func (c *RecommendGoodsActionCase) RecommendGoodsActionReport(ctx context.Context, req *app.RecommendGoodsActionReportRequest) error {
	// 空请求直接返回，异步埋点不做额外失败放大。
	if req == nil {
		return nil
	}

	actor := recommendactor.Resolve(ctx)
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
		publishOrderCreateEvent(actor, recommendevent.BuildGoodsItemsFromActionItems(req.GetGoodsItems()))
	case common.RecommendGoodsActionType_RECOMMEND_GOODS_ACTION_ORDER_PAY:
		publishOrderPayEvent(actor, recommendevent.BuildGoodsItemsFromActionItems(req.GetGoodsItems()))
	}
	return nil
}

// BindRecommendGoodsActionActor 将匿名行为主体绑定为登录主体。
func (c *RecommendGoodsActionCase) BindRecommendGoodsActionActor(ctx context.Context, anonymousId, userId int64) error {
	recommendGoodsActionQuery := c.RecommendGoodsActionRepo.Data.Query(ctx).RecommendGoodsAction
	_, err := recommendGoodsActionQuery.WithContext(ctx).
		Where(
			recommendGoodsActionQuery.ActorType.Eq(recommendevent.ActorTypeAnonymous),
			recommendGoodsActionQuery.ActorID.Eq(anonymousId),
		).
		Updates(map[string]interface{}{
			"actor_type": recommendevent.ActorTypeUser,
			"actor_id":   userId,
		})
	return err
}

// SaveRecommendEvent 消费推荐商品行为事件。
func (c *RecommendGoodsActionCase) SaveRecommendEvent(message queueData.Message) error {
	rawBody, err := json.Marshal(message.Values)
	if err != nil {
		return err
	}

	payload := make(map[string]*appdto.RecommendEvent)
	err = json.Unmarshal(rawBody, &payload)
	if err != nil {
		return err
	}

	event, ok := payload["data"]
	// 队列消息缺少业务体时直接丢弃，避免消费者重复报错。
	if !ok || event == nil {
		return nil
	}
	return c.recommendExposureCase.consume(context.TODO(), event)
}

// publishTrackGoodsEvents 批量投递单商品埋点事件。
func (c *RecommendGoodsActionCase) publishTrackGoodsEvents(actor *appdto.RecommendActor, goodsItems []*app.RecommendGoodsActionItem, publishFn func(actor *appdto.RecommendActor, goodsId int64, requestId string, scene int32, position int32)) {
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
			recommendcontext.NormalizeSceneEnum(recommendContext.GetScene()),
			recommendContext.GetPosition(),
		)
	}
}

// publishTrackGoodsViewEvents 批量投递商品浏览埋点事件。
func (c *RecommendGoodsActionCase) publishTrackGoodsViewEvents(actor *appdto.RecommendActor, goodsItems []*app.RecommendGoodsActionItem) {
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
			recommendcontext.NormalizeSceneEnum(recommendContext.GetScene()),
		)
	}
}

// publishTrackGoodsCartEvents 批量投递加购埋点事件。
func (c *RecommendGoodsActionCase) publishTrackGoodsCartEvents(actor *appdto.RecommendActor, goodsItems []*app.RecommendGoodsActionItem) {
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
			recommendcontext.NormalizeSceneEnum(recommendContext.GetScene()),
			recommendContext.GetPosition(),
		)
	}
}

// publishRecommendClickEvent 投递推荐点击事件。
func publishRecommendClickEvent(actor *appdto.RecommendActor, goodsId int64, requestId string, scene int32, position int32) {
	utils.AddQueue(_const.RecommendGoodsActionEvent, &appdto.RecommendEvent{
		EventType:  recommendevent.EventTypeClick,
		UserID:     recommendUserID(actor),
		ActorType:  actor.ActorType,
		ActorID:    actor.ActorId,
		RequestID:  requestId,
		Scene:      common.RecommendScene(scene),
		GoodsID:    goodsId,
		GoodsNum:   1,
		Position:   position,
		OccurredAt: time.Now().Unix(),
	})
}

// publishGoodsViewEvent 投递商品浏览事件。
func publishGoodsViewEvent(actor *appdto.RecommendActor, goodsId int64, position int32, requestId string, scene int32) {
	utils.AddQueue(_const.RecommendGoodsActionEvent, &appdto.RecommendEvent{
		EventType:  recommendevent.EventTypeView,
		UserID:     recommendUserID(actor),
		ActorType:  actor.ActorType,
		ActorID:    actor.ActorId,
		RequestID:  requestId,
		Scene:      common.RecommendScene(scene),
		GoodsID:    goodsId,
		GoodsNum:   1,
		Position:   position,
		OccurredAt: time.Now().Unix(),
	})
}

// publishGoodsCollectEvent 投递商品收藏事件。
func publishGoodsCollectEvent(actor *appdto.RecommendActor, goodsId int64, requestId string, scene int32, position int32) {
	utils.AddQueue(_const.RecommendGoodsActionEvent, &appdto.RecommendEvent{
		EventType:  recommendevent.EventTypeCollect,
		UserID:     recommendUserID(actor),
		ActorType:  actor.ActorType,
		ActorID:    actor.ActorId,
		RequestID:  requestId,
		Scene:      common.RecommendScene(scene),
		GoodsID:    goodsId,
		GoodsNum:   1,
		Position:   position,
		OccurredAt: time.Now().Unix(),
	})
}

// publishGoodsCartEvent 投递商品加购事件。
func publishGoodsCartEvent(actor *appdto.RecommendActor, goodsId, goodsNum int64, requestId string, scene int32, position int32) {
	utils.AddQueue(_const.RecommendGoodsActionEvent, &appdto.RecommendEvent{
		EventType:  recommendevent.EventTypeCart,
		UserID:     recommendUserID(actor),
		ActorType:  actor.ActorType,
		ActorID:    actor.ActorId,
		RequestID:  requestId,
		Scene:      common.RecommendScene(scene),
		GoodsID:    goodsId,
		GoodsNum:   goodsNum,
		Position:   position,
		OccurredAt: time.Now().Unix(),
	})
}

// publishOrderCreateEvent 投递下单事件。
func publishOrderCreateEvent(actor *appdto.RecommendActor, goodsItems []*appdto.RecommendEventGoodsItem) {
	utils.AddQueue(_const.RecommendGoodsActionEvent, &appdto.RecommendEvent{
		EventType:  recommendevent.EventTypeOrder,
		UserID:     recommendUserID(actor),
		ActorType:  actor.ActorType,
		ActorID:    actor.ActorId,
		GoodsItems: goodsItems,
		OccurredAt: time.Now().Unix(),
	})
}

// publishOrderPayEvent 投递支付成功事件。
func publishOrderPayEvent(actor *appdto.RecommendActor, goodsItems []*appdto.RecommendEventGoodsItem) {
	utils.AddQueue(_const.RecommendGoodsActionEvent, &appdto.RecommendEvent{
		EventType:  recommendevent.EventTypePay,
		UserID:     recommendUserID(actor),
		ActorType:  actor.ActorType,
		ActorID:    actor.ActorId,
		GoodsItems: goodsItems,
		OccurredAt: time.Now().Unix(),
	})
}

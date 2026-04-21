package queue

import (
	"time"

	"shop/api/gen/go/app"
	"shop/api/gen/go/common"
	_const "shop/pkg/const"
	"shop/pkg/gen/models"
)

// GorseBaseUserEvent 表示用户同步到 Gorse 的队列消息。
type GorseBaseUserEvent struct {
	User *models.BaseUser // 用户快照
}

// GorseDeleteBaseUserEvent 表示删除 Gorse 用户主体的队列消息。
type GorseDeleteBaseUserEvent struct {
	UserId int64 // 用户编号
}

// GorseGoodsInfoEvent 表示商品同步到 Gorse 的队列消息。
type GorseGoodsInfoEvent struct {
	Goods *models.GoodsInfo // 商品快照
}

// GorseDeleteGoodsInfoEvent 表示删除 Gorse 商品主体的队列消息。
type GorseDeleteGoodsInfoEvent struct {
	GoodsId int64 // 商品编号
}

// GorseReplayRecommendEventsEvent 表示推荐历史回放到 Gorse 的队列消息。
type GorseReplayRecommendEventsEvent struct {
	ActorType common.RecommendActorType // 推荐主体类型
	ActorId   int64                     // 推荐主体编号
	EventList []*models.RecommendEvent  // 历史事件列表
	EventTime time.Time                 // 投递时间
}

// DispatchGorseSyncBaseUser 将用户快照投递到 Gorse 异步同步链路。
func DispatchGorseSyncBaseUser(user *models.BaseUser) {
	// 用户为空或用户编号非法时，当前同步请求无效。
	if user == nil || user.ID <= 0 {
		return
	}
	AddQueue(_const.GorseSyncBaseUser, &GorseBaseUserEvent{User: user})
}

// DispatchGorseDeleteBaseUser 将用户删除事件投递到 Gorse 异步同步链路。
func DispatchGorseDeleteBaseUser(userId int64) {
	// 用户编号非法时，不再继续投递删除消息。
	if userId <= 0 {
		return
	}
	AddQueue(_const.GorseDeleteBaseUser, &GorseDeleteBaseUserEvent{UserId: userId})
}

// DispatchGorseSyncGoodsInfo 将商品快照投递到 Gorse 异步同步链路。
func DispatchGorseSyncGoodsInfo(goods *models.GoodsInfo) {
	// 商品为空或商品编号非法时，当前同步请求无效。
	if goods == nil || goods.ID <= 0 {
		return
	}
	AddQueue(_const.GorseSyncGoodsInfo, &GorseGoodsInfoEvent{Goods: goods})
}

// DispatchGorseDeleteGoodsInfo 将商品删除事件投递到 Gorse 异步同步链路。
func DispatchGorseDeleteGoodsInfo(goodsId int64) {
	// 商品编号非法时，不再继续投递删除消息。
	if goodsId <= 0 {
		return
	}
	AddQueue(_const.GorseDeleteGoodsInfo, &GorseDeleteGoodsInfoEvent{GoodsId: goodsId})
}

// DispatchGorseReplayRecommendEvents 将历史推荐事件投递到 Gorse 回放链路。
func DispatchGorseReplayRecommendEvents(actorType common.RecommendActorType, actorId int64, eventList []*models.RecommendEvent) {
	// 主体编号非法或历史事件为空时，不再继续投递回放消息。
	if actorId <= 0 || len(eventList) == 0 {
		return
	}
	AddQueue(_const.GorseReplayEvent, &GorseReplayRecommendEventsEvent{
		ActorType: actorType,
		ActorId:   actorId,
		EventList: eventList,
		EventTime: time.Now(),
	})
}

// DispatchGorseRecommendEvent 将推荐事件投递到 Gorse 异步同步链路。
func DispatchGorseRecommendEvent(actor *app.RecommendActor, req *app.RecommendEventReportRequest, eventTime time.Time) {
	event := buildRecommendEventReportEvent(actor, req, eventTime)
	// 当前请求无法构造成有效推荐事件时，不再继续投递。
	if event == nil {
		return
	}
	AddQueue(_const.GorseRecommendEvent, event)
}

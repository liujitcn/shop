package recommend

import (
	"context"
	"errors"
	"strconv"
	"time"

	"shop/api/gen/go/common"
	_const "shop/pkg/const"
	"shop/pkg/gen/models"
	pkgQueue "shop/pkg/queue"

	client "github.com/gorse-io/gorse-go"
	queueData "github.com/liujitcn/kratos-kit/queue/data"
	"github.com/liujitcn/kratos-kit/sdk"
)

// QueueReceiver 表示推荐系统队列消费接收器。
type QueueReceiver struct {
	recommend *Recommend
	userSync  *UserSyncReceiver
	goodsSync *GoodsSyncReceiver
}

// NewQueueReceiver 构建队列消费接收器并完成内部订阅初始化。
func NewQueueReceiver(recommend *Recommend, userSync *UserSyncReceiver, goodsSync *GoodsSyncReceiver) *QueueReceiver {
	receiver := &QueueReceiver{
		recommend: recommend,
		userSync:  userSync,
		goodsSync: goodsSync,
	}
	receiver.initSubscriber()
	return receiver
}

// Enabled 判断当前队列消费接收器是否可用。
func (r *QueueReceiver) Enabled() bool {
	return r.recommend.Enabled()
}

// initSubscriber 初始化推荐系统相关队列订阅。
func (r *QueueReceiver) initSubscriber() {
	// 推荐系统未启用时，直接跳过推荐队列订阅。
	if !r.Enabled() {
		return
	}

	queueRuntime := sdk.Runtime.GetQueue()
	// 运行时队列未初始化时，直接跳过推荐队列订阅。
	if queueRuntime == nil {
		return
	}

	queueRuntime.Register(string(_const.RecommendSyncBaseUser), r.consumeSyncBaseUser)
	queueRuntime.Register(string(_const.RecommendDeleteBaseUser), r.consumeDeleteBaseUser)
	queueRuntime.Register(string(_const.RecommendSyncGoodsInfo), r.consumeSyncGoodsInfo)
	queueRuntime.Register(string(_const.RecommendDeleteGoodsInfo), r.consumeDeleteGoodsInfo)
	queueRuntime.Register(string(_const.RecommendEvent), r.consumeRecommendEvent)
}

// consumeSyncBaseUser 消费用户同步队列并发送到推荐系统。
func (r *QueueReceiver) consumeSyncBaseUser(message queueData.Message) error {
	user, err := pkgQueue.DecodeQueueData[models.BaseUser](message)
	if err != nil {
		return err
	}
	// 推荐系统未启用或队列消息里没有有效用户快照时，直接忽略当前消息。
	if user == nil || user.ID <= 0 || !r.Enabled() {
		return nil
	}
	return r.userSync.sync(context.TODO(), user)
}

// consumeDeleteBaseUser 消费用户删除队列并发送到推荐系统。
func (r *QueueReceiver) consumeDeleteBaseUser(message queueData.Message) error {
	userIds, err := pkgQueue.DecodeQueueData[[]int64](message)
	if err != nil {
		return err
	}
	// 推荐系统未启用或队列消息里没有有效用户编号时，直接忽略当前消息。
	if userIds == nil || len(*userIds) <= 0 || !r.Enabled() {
		return nil
	}

	// 推荐系统接口会在删除用户主体时一并级联删除该用户下的反馈数据。
	var deleteErr error
	for _, userId := range *userIds {
		_, err = r.recommend.gorseClient.DeleteUser(context.TODO(), strconv.FormatInt(userId, 10))
		if err != nil {
			deleteErr = errors.Join(deleteErr, err)
		}
	}
	return deleteErr
}

// consumeSyncGoodsInfo 消费商品同步队列并发送到推荐系统。
func (r *QueueReceiver) consumeSyncGoodsInfo(message queueData.Message) error {
	goods, err := pkgQueue.DecodeQueueData[models.GoodsInfo](message)
	if err != nil {
		return err
	}
	// 推荐系统未启用或队列消息里没有有效商品快照时，直接忽略当前消息。
	if goods == nil || goods.ID <= 0 || !r.Enabled() {
		return nil
	}
	return r.goodsSync.sync(context.TODO(), goods)
}

// consumeDeleteGoodsInfo 消费商品删除队列并发送到推荐系统。
func (r *QueueReceiver) consumeDeleteGoodsInfo(message queueData.Message) error {
	goodsIds, err := pkgQueue.DecodeQueueData[[]int64](message)
	if err != nil {
		return err
	}
	// 推荐系统未启用或队列消息里没有有效商品编号时，直接忽略当前消息。
	if goodsIds == nil || len(*goodsIds) <= 0 || !r.Enabled() {
		return nil
	}

	// 推荐系统接口会在删除商品主体时一并级联删除该商品下的反馈数据。
	var deleteErr error
	for _, goodsId := range *goodsIds {
		_, err = r.recommend.gorseClient.DeleteItem(context.TODO(), strconv.FormatInt(goodsId, 10))
		if err != nil {
			deleteErr = errors.Join(deleteErr, err)
		}
	}
	return deleteErr
}

// consumeRecommendEvent 消费历史回放队列并发送到推荐系统。
func (r *QueueReceiver) consumeRecommendEvent(message queueData.Message) error {
	eventList, err := pkgQueue.DecodeQueueData[[]*models.RecommendEvent](message)
	if err != nil {
		return err
	}
	// 推荐系统未启用或队列消息里没有有效回放事件时，直接忽略当前消息。
	if eventList == nil || len(*eventList) == 0 || !r.Enabled() {
		return nil
	}

	ctx := context.TODO()
	feedbacks := make([]client.Feedback, 0, len(*eventList))
	for _, item := range *eventList {
		// 历史事件为空、商品编号非法或事件类型未知时，直接跳过当前无效事件。
		if item == nil || item.ActorID <= 0 || item.ActorType != int32(common.RecommendActorType_USER) || item.GoodsID <= 0 || item.EventType == int32(common.RecommendEventType_UNKNOWN_RET) {
			continue
		}

		value := float64(item.GoodsNum)
		// 商品数量未显式记录时，统一按 1 回放事件权重。
		if value <= 0 {
			value = 1
		}

		timestamp := item.EventAt
		// 事件时间为空时，统一回退到当前时间，避免远端因零值时间产生脏数据。
		if timestamp.IsZero() {
			timestamp = time.Now()
		}

		feedbacks = append(feedbacks, client.Feedback{
			FeedbackType: common.RecommendEventType(item.EventType).String(),
			// 回放事件必须写回原始登录用户，不能把所有反馈都错误归并到固定主体。
			UserId:    strconv.FormatInt(item.ActorID, 10),
			ItemId:    strconv.FormatInt(item.GoodsID, 10),
			Value:     value,
			Timestamp: timestamp,
		})
	}
	// 当前批次没有有效反馈时，直接结束回放，避免空请求打到远端。
	if len(feedbacks) == 0 {
		return nil
	}

	_, err = r.recommend.gorseClient.InsertFeedback(ctx, feedbacks)
	return err
}

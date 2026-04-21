package gorse

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"shop/api/gen/go/app"
	"shop/api/gen/go/common"
	"shop/api/gen/go/conf"
	_const "shop/pkg/const"
	"shop/pkg/gen/models"
	pkgQueue "shop/pkg/queue"

	client "github.com/gorse-io/gorse-go"
	queueData "github.com/liujitcn/kratos-kit/queue/data"
	"github.com/liujitcn/kratos-kit/sdk"
)

const (
	recommendActorUserPrefix = "user:"
)

type Gorse struct {
	gorseClient *client.GorseClient
}

// NewGorse 创建 Gorse 客户端包装。
func NewGorse(cfg *conf.Gorse) *Gorse {
	entryPoint := strings.TrimSpace(cfg.GetEntryPoint())
	// 未配置入口地址时，直接返回空客户端并在业务侧走本地兜底。
	if entryPoint == "" {
		return &Gorse{}
	}

	gorseClient := client.NewGorseClient(entryPoint, cfg.GetApiKey())
	g := &Gorse{
		gorseClient: gorseClient,
	}
	g.registerQueueConsumers()
	return g
}

// Enabled 判断当前 Gorse 客户端是否可用。
func (g *Gorse) Enabled() bool {
	return g != nil && g.gorseClient != nil
}

// SyncBaseUser 将后台用户资料异步投递到 Gorse。
func (g *Gorse) SyncBaseUser(_ context.Context, user *models.BaseUser) error {
	// 用户为空、用户编号非法或客户端未启用时，当前同步请求无效。
	if user == nil || user.ID <= 0 || !g.Enabled() {
		return nil
	}
	pkgQueue.DispatchGorseSyncBaseUser(user)
	return nil
}

// DeleteBaseUser 将后台用户删除事件异步投递到 Gorse。
func (g *Gorse) DeleteBaseUser(_ context.Context, userId int64) error {
	// 客户端未启用或用户编号非法时，无需继续删除主体。
	if !g.Enabled() || userId <= 0 {
		return nil
	}
	pkgQueue.DispatchGorseDeleteBaseUser(userId)
	return nil
}

// SyncGoodsInfo 将商品资料异步投递到 Gorse。
func (g *Gorse) SyncGoodsInfo(_ context.Context, goods *models.GoodsInfo) error {
	// 商品为空、商品编号非法或客户端未启用时，当前同步请求无效。
	if goods == nil || goods.ID <= 0 || !g.Enabled() {
		return nil
	}
	pkgQueue.DispatchGorseSyncGoodsInfo(goods)
	return nil
}

// DeleteGoodsInfo 将商品删除事件异步投递到 Gorse。
func (g *Gorse) DeleteGoodsInfo(_ context.Context, goodsId int64) error {
	// 客户端未启用或商品编号非法时，无需继续删除商品。
	if !g.Enabled() || goodsId <= 0 {
		return nil
	}
	pkgQueue.DispatchGorseDeleteGoodsInfo(goodsId)
	return nil
}

// InsertRecommendEvent 将推荐事件异步投递到 Gorse。
func (g *Gorse) InsertRecommendEvent(_ context.Context, actor *app.RecommendActor, req *app.RecommendEventReportRequest, eventTime time.Time) error {
	// 客户端未启用、主体无效、主体不是登录用户或事件请求为空时，直接跳过事件投递。
	if !g.Enabled() || actor == nil || actor.GetActorId() <= 0 || req == nil {
		return nil
	}
	// 匿名主体不写入 Gorse，避免匿名用户在 Gorse 中形成主体数据。
	if actor.GetActorType() != common.RecommendActorType_USER {
		return nil
	}
	// 事件类型未知时，当前事件不需要继续投递到 Gorse。
	if req.GetEventType() == common.RecommendEventType_UNKNOWN_RET {
		return nil
	}
	pkgQueue.DispatchGorseRecommendEvent(actor, req, eventTime)
	return nil
}

// ReplayRecommendEvents 将历史推荐事件异步投递到 Gorse 回放链路。
func (g *Gorse) ReplayRecommendEvents(_ context.Context, actorType common.RecommendActorType, actorId int64, eventList []*models.RecommendEvent) error {
	// 客户端未启用、主体编号非法、主体不是登录用户或历史事件为空时，直接跳过历史回放投递。
	if !g.Enabled() || actorId <= 0 || actorType != common.RecommendActorType_USER || len(eventList) == 0 {
		return nil
	}
	pkgQueue.DispatchGorseReplayRecommendEvents(actorType, actorId, eventList)
	return nil
}

// GetRecommendGoodsIds 查询用户维度推荐商品编号列表。
func (g *Gorse) GetRecommendGoodsIds(ctx context.Context, actor *app.RecommendActor, pageNum, pageSize int64) ([]int64, int64, error) {
	// 客户端未启用、推荐主体无效或主体不是登录用户时，直接返回空推荐结果。
	if !g.Enabled() || actor == nil || actor.GetActorId() <= 0 {
		return []int64{}, 0, nil
	}
	// 匿名主体不走用户维度的 Gorse 推荐。
	if actor.GetActorType() != common.RecommendActorType_USER {
		return []int64{}, 0, nil
	}

	offset := int((pageNum - 1) * pageSize)
	rawIds, err := g.gorseClient.GetRecommend(ctx, g.getUserId(actor.GetActorId()), "", int(pageSize)+1, offset)
	if err != nil {
		return nil, 0, err
	}
	return g.buildRecommendPageResult(rawIds, pageNum, pageSize)
}

// SessionRecommendGoodsIds 查询会话级推荐商品编号列表。
func (g *Gorse) SessionRecommendGoodsIds(ctx context.Context, contextGoodsIds []int64, pageNum, pageSize int64) ([]int64, int64, error) {
	// 客户端未启用或上下文商品为空时，直接返回空会话推荐结果。
	if !g.Enabled() || len(contextGoodsIds) == 0 {
		return []int64{}, 0, nil
	}

	cleanGoodsIds := make([]int64, 0, len(contextGoodsIds))
	excludedGoods := make(map[int64]struct{}, len(contextGoodsIds))
	for _, goodsId := range contextGoodsIds {
		// 非法商品编号或重复商品编号时，直接跳过当前无效值。
		if goodsId <= 0 {
			continue
		}
		if _, ok := excludedGoods[goodsId]; ok {
			continue
		}
		excludedGoods[goodsId] = struct{}{}
		cleanGoodsIds = append(cleanGoodsIds, goodsId)
	}
	if len(cleanGoodsIds) == 0 {
		return []int64{}, 0, nil
	}

	now := time.Now()
	feedbacks := make([]client.Feedback, 0, len(cleanGoodsIds))
	for _, goodsId := range cleanGoodsIds {
		feedbacks = append(feedbacks, client.Feedback{
			FeedbackType: common.RecommendEventType_VIEW.String(),
			ItemId:       strconv.FormatInt(goodsId, 10),
			Value:        1,
			Timestamp:    now,
		})
	}

	scores, err := g.gorseClient.SessionRecommend(ctx, feedbacks, int(pageNum*pageSize)+1)
	if err != nil {
		return nil, 0, err
	}

	rawIds := make([]string, 0, len(scores))
	for _, score := range scores {
		goodsId, convErr := strconv.ParseInt(score.Id, 10, 64)
		// Gorse 返回了非法商品编号或上下文商品本身时，直接跳过当前结果。
		if convErr != nil || goodsId <= 0 {
			continue
		}
		if _, ok := excludedGoods[goodsId]; ok {
			continue
		}
		rawIds = append(rawIds, score.Id)
	}
	return g.buildRecommendPageResult(rawIds, pageNum, pageSize)
}

// registerQueueConsumers 注册 Gorse 异步投递的队列订阅事件。
func (g *Gorse) registerQueueConsumers() {
	q := sdk.Runtime.GetQueue()
	// 运行时队列未初始化时，直接跳过 Gorse 队列订阅注册。
	if q == nil {
		return
	}

	q.Register(string(_const.GorseSyncBaseUser), g.consumeSyncBaseUser)
	q.Register(string(_const.GorseDeleteBaseUser), g.consumeDeleteBaseUser)
	q.Register(string(_const.GorseSyncGoodsInfo), g.consumeSyncGoodsInfo)
	q.Register(string(_const.GorseDeleteGoodsInfo), g.consumeDeleteGoodsInfo)
	q.Register(string(_const.GorseRecommendEvent), g.consumeRecommendEvent)
	q.Register(string(_const.GorseReplayEvent), g.consumeReplayRecommendEvents)
}

// consumeSyncBaseUser 消费用户同步队列并发送到 Gorse。
func (g *Gorse) consumeSyncBaseUser(message queueData.Message) error {
	event, err := pkgQueue.DecodeQueueData[pkgQueue.GorseBaseUserEvent](message)
	if err != nil {
		return err
	}
	// 队列消息里没有有效用户快照时，直接忽略当前消息。
	if event == nil || event.User == nil {
		return nil
	}
	return g.syncBaseUserNow(context.TODO(), event.User)
}

// consumeDeleteBaseUser 消费用户删除队列并发送到 Gorse。
func (g *Gorse) consumeDeleteBaseUser(message queueData.Message) error {
	event, err := pkgQueue.DecodeQueueData[pkgQueue.GorseDeleteBaseUserEvent](message)
	if err != nil {
		return err
	}
	// 队列消息里没有有效用户编号时，直接忽略当前消息。
	if event == nil || event.UserId <= 0 {
		return nil
	}
	return g.deleteBaseUserNow(context.TODO(), event.UserId)
}

// consumeSyncGoodsInfo 消费商品同步队列并发送到 Gorse。
func (g *Gorse) consumeSyncGoodsInfo(message queueData.Message) error {
	event, err := pkgQueue.DecodeQueueData[pkgQueue.GorseGoodsInfoEvent](message)
	if err != nil {
		return err
	}
	// 队列消息里没有有效商品快照时，直接忽略当前消息。
	if event == nil || event.Goods == nil {
		return nil
	}
	return g.syncGoodsInfoNow(context.TODO(), event.Goods)
}

// consumeDeleteGoodsInfo 消费商品删除队列并发送到 Gorse。
func (g *Gorse) consumeDeleteGoodsInfo(message queueData.Message) error {
	event, err := pkgQueue.DecodeQueueData[pkgQueue.GorseDeleteGoodsInfoEvent](message)
	if err != nil {
		return err
	}
	// 队列消息里没有有效商品编号时，直接忽略当前消息。
	if event == nil || event.GoodsId <= 0 {
		return nil
	}
	return g.deleteGoodsInfoNow(context.TODO(), event.GoodsId)
}

// consumeRecommendEvent 消费推荐事件队列并发送到 Gorse。
func (g *Gorse) consumeRecommendEvent(message queueData.Message) error {
	event, err := pkgQueue.DecodeQueueData[pkgQueue.RecommendEventReportEvent](message)
	if err != nil {
		return err
	}
	// 队列消息里没有推荐事件主体时，直接忽略当前消息。
	if event == nil {
		return nil
	}

	items := make([]*app.RecommendEventItem, 0, len(event.Items))
	for _, item := range event.Items {
		// 非法商品项直接跳过，避免把脏数据写入 Gorse。
		if item == nil || item.GoodsId <= 0 {
			continue
		}
		items = append(items, &app.RecommendEventItem{
			GoodsId:  item.GoodsId,
			GoodsNum: item.GoodsNum,
			Position: item.Position,
		})
	}
	// 队列事件里没有有效商品项时，不再继续发送到 Gorse。
	if len(items) == 0 {
		return nil
	}

	req := &app.RecommendEventReportRequest{
		EventType: event.EventType,
		RecommendContext: &app.RecommendEventContext{
			Scene:     common.RecommendScene(event.Scene),
			RequestId: event.RequestId,
		},
		Items: items,
	}
	return g.insertRecommendEventNow(context.TODO(), event.RecommendActor, req, event.EventTime)
}

// consumeReplayRecommendEvents 消费历史回放队列并发送到 Gorse。
func (g *Gorse) consumeReplayRecommendEvents(message queueData.Message) error {
	event, err := pkgQueue.DecodeQueueData[pkgQueue.GorseReplayRecommendEventsEvent](message)
	if err != nil {
		return err
	}
	// 队列消息里没有有效回放事件时，直接忽略当前消息。
	if event == nil || event.ActorId <= 0 || len(event.EventList) == 0 {
		return nil
	}
	return g.replayRecommendEventsNow(context.TODO(), event.ActorType, event.ActorId, event.EventList)
}

// syncBaseUserNow 立即同步后台用户资料到 Gorse。
func (g *Gorse) syncBaseUserNow(ctx context.Context, user *models.BaseUser) error {
	// 用户为空、用户编号非法或客户端未启用时，当前同步请求无效。
	if user == nil || user.ID <= 0 || !g.Enabled() {
		return nil
	}

	comment := user.NickName
	// 用户昵称为空时，回退到用户名作为注释信息。
	if comment == "" {
		comment = user.UserName
	}

	userId := g.getUserId(user.ID)
	labels := map[string]interface{}{
		"user_id":    user.ID,
		"user_name":  user.UserName,
		"role_id":    user.RoleID,
		"dept_id":    user.DeptID,
		"gender":     user.Gender,
		"status":     user.Status,
		"has_phone":  user.Phone != "",
		"has_openid": user.Openid != "",
	}
	_, err := g.gorseClient.InsertUser(ctx, client.User{
		UserId:  userId,
		Labels:  labels,
		Comment: comment,
	})
	if err == nil {
		return nil
	}

	commentPatch := comment
	_, updateErr := g.gorseClient.UpdateUser(ctx, userId, client.UserPatch{
		Labels:  labels,
		Comment: &commentPatch,
	})
	if updateErr == nil {
		return nil
	}
	return errors.Join(err, updateErr)
}

// deleteBaseUserNow 立即删除 Gorse 中的后台用户。
func (g *Gorse) deleteBaseUserNow(ctx context.Context, userId int64) error {
	// 客户端未启用或用户编号非法时，无需继续删除主体。
	if !g.Enabled() || userId <= 0 {
		return nil
	}

	_, err := g.gorseClient.DeleteUser(ctx, g.getUserId(userId))
	return err
}

// syncGoodsInfoNow 立即同步商品资料到 Gorse。
func (g *Gorse) syncGoodsInfoNow(ctx context.Context, goods *models.GoodsInfo) error {
	// 商品为空、商品编号非法或客户端未启用时，当前同步请求无效。
	if goods == nil || goods.ID <= 0 || !g.Enabled() {
		return nil
	}

	categories := make([]string, 0, 1)
	// 商品存在分类时，把分类编号作为 Gorse 分类维度同步。
	if goods.CategoryID > 0 {
		categories = append(categories, strconv.FormatInt(goods.CategoryID, 10))
	}

	timestamp := goods.UpdatedAt
	// 商品更新时间为空时，回退到创建时间，再不满足时使用当前时间。
	if timestamp.IsZero() {
		timestamp = goods.CreatedAt
	}
	if timestamp.IsZero() {
		timestamp = time.Now()
	}

	item := client.Item{
		ItemId:     strconv.FormatInt(goods.ID, 10),
		IsHidden:   goods.Status != int32(common.GoodsStatus_PUT_ON),
		Categories: categories,
		Timestamp:  timestamp,
		Comment:    goods.Name,
		Labels: map[string]interface{}{
			"goods_id":       goods.ID,
			"category_id":    goods.CategoryID,
			"status":         goods.Status,
			"price":          goods.Price,
			"discount_price": goods.DiscountPrice,
			"inventory":      goods.Inventory,
		},
	}
	_, err := g.gorseClient.InsertItem(ctx, item)
	if err == nil {
		return nil
	}

	isHidden := item.IsHidden
	comment := item.Comment
	_, updateErr := g.gorseClient.UpdateItem(ctx, item.ItemId, client.ItemPatch{
		IsHidden:   &isHidden,
		Categories: item.Categories,
		Timestamp:  &item.Timestamp,
		Labels:     item.Labels,
		Comment:    &comment,
	})
	if updateErr == nil {
		return nil
	}
	return errors.Join(err, updateErr)
}

// deleteGoodsInfoNow 立即删除 Gorse 中的商品。
func (g *Gorse) deleteGoodsInfoNow(ctx context.Context, goodsId int64) error {
	// 客户端未启用或商品编号非法时，无需继续删除商品。
	if !g.Enabled() || goodsId <= 0 {
		return nil
	}

	_, err := g.gorseClient.DeleteItem(ctx, strconv.FormatInt(goodsId, 10))
	return err
}

// insertRecommendEventNow 立即将推荐事件同步到 Gorse。
func (g *Gorse) insertRecommendEventNow(ctx context.Context, actor *app.RecommendActor, req *app.RecommendEventReportRequest, eventTime time.Time) error {
	// 客户端未启用、主体无效、主体不是登录用户或事件请求为空时，直接跳过事件同步。
	if !g.Enabled() || actor == nil || actor.GetActorId() <= 0 || req == nil {
		return nil
	}
	// 匿名主体不写入 Gorse，避免匿名用户在 Gorse 中形成主体数据。
	if actor.GetActorType() != common.RecommendActorType_USER {
		return nil
	}
	// 事件类型未知时，当前事件不需要继续同步到 Gorse。
	if req.GetEventType() == common.RecommendEventType_UNKNOWN_RET {
		return nil
	}
	if eventTime.IsZero() {
		eventTime = time.Now()
	}

	err := g.ensureRecommendUserNow(ctx, actor.GetActorId())
	if err != nil {
		return err
	}

	feedbacks := make([]client.Feedback, 0, len(req.GetItems()))
	for _, item := range req.GetItems() {
		// 商品项为空或商品编号非法时，直接跳过当前无效事件项。
		if item == nil || item.GetGoodsId() <= 0 {
			continue
		}

		value := float64(item.GetGoodsNum())
		// 事件未显式传入商品数量时，统一按 1 处理反馈值。
		if value <= 0 {
			value = 1
		}

		feedbacks = append(feedbacks, client.Feedback{
			FeedbackType: req.GetEventType().String(),
			UserId:       g.getUserId(actor.GetActorId()),
			ItemId:       strconv.FormatInt(item.GetGoodsId(), 10),
			Value:        value,
			Timestamp:    eventTime,
		})
	}
	if len(feedbacks) == 0 {
		return nil
	}

	_, err = g.gorseClient.InsertFeedback(ctx, feedbacks)
	return err
}

// replayRecommendEventsNow 立即将历史推荐事件重放到指定推荐主体。
func (g *Gorse) replayRecommendEventsNow(ctx context.Context, actorType common.RecommendActorType, actorId int64, eventList []*models.RecommendEvent) error {
	// 客户端未启用、主体编号非法、主体不是登录用户或历史事件为空时，直接跳过历史重放。
	if !g.Enabled() || actorId <= 0 || actorType != common.RecommendActorType_USER || len(eventList) == 0 {
		return nil
	}

	err := g.ensureRecommendUserNow(ctx, actorId)
	if err != nil {
		return err
	}

	feedbacks := make([]client.Feedback, 0, len(eventList))
	for _, item := range eventList {
		// 历史事件为空、商品编号非法或事件类型未知时，直接跳过当前无效事件。
		if item == nil || item.GoodsID <= 0 || item.EventType == int32(common.RecommendEventType_UNKNOWN_RET) {
			continue
		}

		value := float64(item.GoodsNum)
		// 商品数量未显式记录时，统一按 1 回放事件权重。
		if value <= 0 {
			value = 1
		}

		timestamp := item.EventAt
		if timestamp.IsZero() {
			timestamp = time.Now()
		}

		feedbacks = append(feedbacks, client.Feedback{
			FeedbackType: common.RecommendEventType(item.EventType).String(),
			UserId:       g.getUserId(actorId),
			ItemId:       strconv.FormatInt(item.GoodsID, 10),
			Value:        value,
			Timestamp:    timestamp,
		})
	}
	if len(feedbacks) == 0 {
		return nil
	}

	_, err = g.gorseClient.InsertFeedback(ctx, feedbacks)
	return err
}

// ensureRecommendUserNow 确保推荐用户主体已同步到 Gorse。
func (g *Gorse) ensureRecommendUserNow(ctx context.Context, userId int64) error {
	// 客户端未启用或用户编号非法时，无需继续同步推荐主体。
	if !g.Enabled() || userId <= 0 {
		return nil
	}

	gorseUserId := g.getUserId(userId)
	labels := map[string]interface{}{
		"actor_type": int32(common.RecommendActorType_USER),
		"actor_id":   userId,
	}
	_, err := g.gorseClient.InsertUser(ctx, client.User{
		UserId: gorseUserId,
		Labels: labels,
	})
	if err == nil {
		return nil
	}

	_, updateErr := g.gorseClient.UpdateUser(ctx, gorseUserId, client.UserPatch{
		Labels: labels,
	})
	if updateErr == nil {
		return nil
	}
	return errors.Join(err, updateErr)
}

// getUserId 构建项目内统一的 Gorse 登录用户编号。
func (g *Gorse) getUserId(userId int64) string {
	return recommendActorUserPrefix + strconv.FormatInt(userId, 10)
}

// buildRecommendPageResult 将 Gorse 返回结果转换为项目分页结果。
func (g *Gorse) buildRecommendPageResult(rawIds []string, pageNum, pageSize int64) ([]int64, int64, error) {
	offset := (pageNum - 1) * pageSize
	goodsIds := make([]int64, 0, len(rawIds))
	for _, rawId := range rawIds {
		goodsId, err := strconv.ParseInt(rawId, 10, 64)
		if err != nil {
			return nil, 0, err
		}
		// 返回结果里包含非法商品编号时，直接跳过当前无效值。
		if goodsId <= 0 {
			continue
		}
		goodsIds = append(goodsIds, goodsId)
	}

	hasMore := int64(0)
	// 当前页多取到 1 条时，说明后续仍存在下一页数据。
	if int64(len(goodsIds)) > pageSize {
		hasMore = 1
		goodsIds = goodsIds[:pageSize]
	}
	return goodsIds, offset + int64(len(goodsIds)) + hasMore, nil
}

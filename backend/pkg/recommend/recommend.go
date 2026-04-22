package recommend

import (
	"context"
	"encoding/json"
	"errors"
	_const "shop/pkg/const"
	"strconv"
	"strings"
	"time"

	"shop/api/gen/go/app"
	"shop/api/gen/go/common"
	"shop/api/gen/go/conf"
	"shop/pkg/gen/models"
	pkgQueue "shop/pkg/queue"

	mapset "github.com/deckarep/golang-set/v2"
	client "github.com/gorse-io/gorse-go"
	queueData "github.com/liujitcn/kratos-kit/queue/data"
	"github.com/liujitcn/kratos-kit/sdk"
)

type Recommend struct {
	gorseClient *client.GorseClient
}

// NewRecommend 创建推荐系统客户端包装。
func NewRecommend(cfg *conf.Recommend) *Recommend {
	// 推荐配置缺失时，直接关闭推荐系统链路并走本地兜底。
	if cfg == nil {
		pkgQueue.SetRecommendEnabled(false)
		return &Recommend{}
	}

	entryPoint := strings.TrimSpace(cfg.GetEntryPoint())
	// 未配置入口地址时，直接返回空客户端并在业务侧走本地兜底。
	if entryPoint == "" {
		pkgQueue.SetRecommendEnabled(false)
		return &Recommend{}
	}

	systemClient := client.NewGorseClient(entryPoint, cfg.GetApiKey())
	recommend := &Recommend{
		gorseClient: systemClient,
	}
	pkgQueue.SetRecommendEnabled(true)
	q := sdk.Runtime.GetQueue()
	// 运行时队列未初始化时，直接跳过推荐系统队列订阅注册。
	if q != nil {
		q.Register(string(_const.RecommendSyncBaseUser), recommend.consumeSyncBaseUser)
		q.Register(string(_const.RecommendDeleteBaseUser), recommend.consumeDeleteBaseUser)
		q.Register(string(_const.RecommendSyncGoodsInfo), recommend.consumeSyncGoodsInfo)
		q.Register(string(_const.RecommendDeleteGoodsInfo), recommend.consumeDeleteGoodsInfo)
		q.Register(string(_const.RecommendEvent), recommend.consumeRecommendEvent)
	}
	return recommend
}

// Enabled 判断当前推荐系统客户端是否可用。
func (g *Recommend) Enabled() bool {
	return g != nil && g.gorseClient != nil
}

// LoadUserIds 加载推荐系统中已存在的用户主体编号集合。
func (g *Recommend) LoadUserIds(ctx context.Context, pageSize int) (mapset.Set[string], error) {
	// 客户端未启用时，直接返回空用户集合。
	if !g.Enabled() {
		return mapset.NewThreadUnsafeSet[string](), nil
	}
	// 调用方未显式传入上下文时，回退到默认上下文继续查询。
	if ctx == nil {
		ctx = context.TODO()
	}
	// 分页大小非法时，回退到默认分页大小，避免远端接口收到无效参数。
	if pageSize <= 0 {
		pageSize = 100
	}

	userIdSet := mapset.NewThreadUnsafeSetWithSize[string](pageSize)
	cursor := ""
	for {
		iterator, err := g.gorseClient.GetUsers(ctx, pageSize, cursor)
		if err != nil {
			return nil, err
		}
		for _, item := range iterator.Users {
			// 远端返回空用户编号时，直接跳过当前无效数据。
			if item.UserId == "" {
				continue
			}
			userIdSet.Add(item.UserId)
		}
		// 当前页没有更多游标或下一页游标未发生变化时，说明远端集合已经遍历完成。
		if iterator.Cursor == "" || iterator.Cursor == cursor {
			break
		}
		cursor = iterator.Cursor
	}
	return userIdSet, nil
}

// SyncBaseUserList 同步一批后台用户快照到推荐系统。
func (g *Recommend) SyncBaseUserList(ctx context.Context, userList []*models.BaseUser, existingUserIds mapset.Set[string], staleUserIds mapset.Set[string]) error {
	// 客户端未启用时，直接跳过当前用户同步批次。
	if !g.Enabled() {
		return nil
	}
	// 调用方未显式传入上下文时，回退到默认上下文继续同步。
	if ctx == nil {
		ctx = context.TODO()
	}

	// 未传远端索引时，回退到单条 upsert 逻辑保证兼容性。
	if existingUserIds == nil {
		for _, user := range userList {
			// 无效用户快照不参与当前用户同步批次。
			if user == nil || user.ID <= 0 {
				continue
			}
			syncErr := g.syncBaseUser(ctx, user)
			if syncErr != nil {
				return syncErr
			}
		}
		return nil
	}

	insertUsers := make([]client.User, 0, len(userList))
	insertUserList := make([]*models.BaseUser, 0, len(userList))
	for _, user := range userList {
		// 无效用户快照不参与当前用户同步批次。
		if user == nil || user.ID <= 0 {
			continue
		}
		recommendUserId := strconv.FormatInt(user.ID, 10)
		// 当前用户在本地仍然存在时，先从远端待删除集合中移除，避免后续误删有效主体。
		if staleUserIds != nil {
			staleUserIds.Remove(recommendUserId)
		}
		recommendUser, userPatch := g.buildRecommendUser(user)
		// 远端已经存在时，直接走单条更新，避免重复插入失败后再回退。
		if existingUserIds.ContainsOne(recommendUser.UserId) {
			_, updateErr := g.gorseClient.UpdateUser(ctx, recommendUser.UserId, userPatch)
			if updateErr != nil {
				return updateErr
			}
			continue
		}
		insertUsers = append(insertUsers, recommendUser)
		insertUserList = append(insertUserList, user)
	}
	// 当前批次没有新增用户时，说明本轮只命中了更新数据。
	if len(insertUsers) == 0 {
		return nil
	}

	_, err := g.gorseClient.InsertUsers(ctx, insertUsers)
	// 批量插入失败时，回退到单条 upsert，避免因为索引陈旧或远端部分冲突导致整批失败。
	if err != nil {
		var fallbackErr error
		for _, user := range insertUserList {
			syncErr := g.syncBaseUser(ctx, user)
			if syncErr != nil {
				fallbackErr = errors.Join(fallbackErr, syncErr)
			}
		}
		if fallbackErr != nil {
			return errors.Join(err, fallbackErr)
		}
		return nil
	}

	for _, item := range insertUsers {
		existingUserIds.Add(item.UserId)
	}
	return nil
}

// DeleteUserIds 删除推荐系统中多余的用户主体。
func (g *Recommend) DeleteUserIds(ctx context.Context, staleUserIds mapset.Set[string]) error {
	// 客户端未启用或没有待删除用户时，直接跳过当前清理批次。
	if !g.Enabled() || staleUserIds == nil || staleUserIds.IsEmpty() {
		return nil
	}
	// 调用方未显式传入上下文时，回退到默认上下文继续删除。
	if ctx == nil {
		ctx = context.TODO()
	}

	var deleteErr error
	for userId := range staleUserIds.Iter() {
		// 待删除编号为空时，直接跳过当前无效主体。
		if userId == "" {
			continue
		}
		// 推荐系统接口会在删除用户主体时一并级联删除该用户下的反馈数据。
		_, err := g.gorseClient.DeleteUser(ctx, userId)
		if err != nil {
			deleteErr = errors.Join(deleteErr, err)
		}
	}
	return deleteErr
}

// LoadGoodsIds 加载推荐系统中已存在的商品主体编号集合。
func (g *Recommend) LoadGoodsIds(ctx context.Context, pageSize int) (mapset.Set[string], error) {
	// 客户端未启用时，直接返回空商品集合。
	if !g.Enabled() {
		return mapset.NewThreadUnsafeSet[string](), nil
	}
	// 调用方未显式传入上下文时，回退到默认上下文继续查询。
	if ctx == nil {
		ctx = context.TODO()
	}
	// 分页大小非法时，回退到默认分页大小，避免远端接口收到无效参数。
	if pageSize <= 0 {
		pageSize = 100
	}

	itemIdSet := mapset.NewThreadUnsafeSetWithSize[string](pageSize)
	cursor := ""
	for {
		iterator, err := g.gorseClient.GetItems(ctx, pageSize, cursor)
		if err != nil {
			return nil, err
		}
		for _, item := range iterator.Items {
			// 远端返回空商品编号时，直接跳过当前无效数据。
			if item.ItemId == "" {
				continue
			}
			itemIdSet.Add(item.ItemId)
		}
		// 当前页没有更多游标或下一页游标未发生变化时，说明远端集合已经遍历完成。
		if iterator.Cursor == "" || iterator.Cursor == cursor {
			break
		}
		cursor = iterator.Cursor
	}
	return itemIdSet, nil
}

// SyncGoodsInfoList 同步一批商品快照到推荐系统。
func (g *Recommend) SyncGoodsInfoList(ctx context.Context, goodsList []*models.GoodsInfo, existingItemIds mapset.Set[string], staleItemIds mapset.Set[string]) error {
	// 客户端未启用时，直接跳过当前商品同步批次。
	if !g.Enabled() {
		return nil
	}
	// 调用方未显式传入上下文时，回退到默认上下文继续同步。
	if ctx == nil {
		ctx = context.TODO()
	}

	// 未传远端索引时，回退到单条 upsert 逻辑保证兼容性。
	if existingItemIds == nil {
		for _, goods := range goodsList {
			// 无效商品快照不参与当前商品同步批次。
			if goods == nil || goods.ID <= 0 {
				continue
			}
			syncErr := g.syncGoodsInfo(ctx, goods)
			if syncErr != nil {
				return syncErr
			}
		}
		return nil
	}

	insertItems := make([]client.Item, 0, len(goodsList))
	insertGoodsList := make([]*models.GoodsInfo, 0, len(goodsList))
	for _, goods := range goodsList {
		// 无效商品快照不参与当前商品同步批次。
		if goods == nil || goods.ID <= 0 {
			continue
		}
		recommendItem, itemPatch := g.buildRecommendItem(goods)
		// 当前商品在本地仍然存在时，先从远端待删除集合中移除，避免后续误删有效主体。
		if staleItemIds != nil {
			staleItemIds.Remove(recommendItem.ItemId)
		}
		// 远端已经存在时，直接走单条更新，避免重复插入失败后再回退。
		if existingItemIds.ContainsOne(recommendItem.ItemId) {
			_, updateErr := g.gorseClient.UpdateItem(ctx, recommendItem.ItemId, itemPatch)
			if updateErr != nil {
				return updateErr
			}
			continue
		}
		insertItems = append(insertItems, recommendItem)
		insertGoodsList = append(insertGoodsList, goods)
	}
	// 当前批次没有新增商品时，说明本轮只命中了更新数据。
	if len(insertItems) == 0 {
		return nil
	}

	_, err := g.gorseClient.InsertItems(ctx, insertItems)
	// 批量插入失败时，回退到单条 upsert，避免因为索引陈旧或远端部分冲突导致整批失败。
	if err != nil {
		var fallbackErr error
		for _, goods := range insertGoodsList {
			syncErr := g.syncGoodsInfo(ctx, goods)
			if syncErr != nil {
				fallbackErr = errors.Join(fallbackErr, syncErr)
			}
		}
		if fallbackErr != nil {
			return errors.Join(err, fallbackErr)
		}
		return nil
	}

	for _, item := range insertItems {
		existingItemIds.Add(item.ItemId)
	}
	return nil
}

// DeleteGoodsIds 删除推荐系统中多余的商品主体。
func (g *Recommend) DeleteGoodsIds(ctx context.Context, staleItemIds mapset.Set[string]) error {
	// 客户端未启用或没有待删除商品时，直接跳过当前清理批次。
	if !g.Enabled() || staleItemIds == nil || staleItemIds.IsEmpty() {
		return nil
	}
	// 调用方未显式传入上下文时，回退到默认上下文继续删除。
	if ctx == nil {
		ctx = context.TODO()
	}

	var deleteErr error
	for itemId := range staleItemIds.Iter() {
		// 待删除编号为空时，直接跳过当前无效主体。
		if itemId == "" {
			continue
		}
		// 推荐系统接口会在删除商品主体时一并级联删除该商品下的反馈数据。
		_, err := g.gorseClient.DeleteItem(ctx, itemId)
		if err != nil {
			deleteErr = errors.Join(deleteErr, err)
		}
	}
	return deleteErr
}

// ListRecommendGoodsIds 查询当前用户前 N 条原始推荐商品编号。
func (g *Recommend) ListRecommendGoodsIds(ctx context.Context, actor *app.RecommendActor, limit int64) ([]int64, bool, error) {
	// 客户端未启用、推荐主体无效或主体不是登录用户时，直接返回空推荐结果。
	if !g.Enabled() || actor == nil || actor.GetActorId() <= 0 {
		return []int64{}, false, nil
	}
	// 匿名主体不走用户维度的推荐系统推荐。
	if actor.GetActorType() != common.RecommendActorType_USER {
		return []int64{}, false, nil
	}
	// 调用方未显式传入上下文时，回退到默认上下文继续查询。
	if ctx == nil {
		ctx = context.TODO()
	}
	// 请求上限非法时，直接返回空结果，避免远端收到无效参数。
	if limit <= 0 {
		return []int64{}, false, nil
	}

	rawIds, err := g.gorseClient.GetRecommend(ctx, strconv.FormatInt(actor.GetActorId(), 10), "", int(limit)+1, 0)
	if err != nil {
		return nil, false, err
	}
	return g.buildRecommendGoodsIds(rawIds, limit)
}

// GetRecommendGoodsIds 查询用户维度推荐商品编号列表。
func (g *Recommend) GetRecommendGoodsIds(ctx context.Context, actor *app.RecommendActor, pageNum, pageSize int64) ([]int64, int64, error) {
	limit := pageNum*pageSize + 1
	rawIds, hasMore, err := g.ListRecommendGoodsIds(ctx, actor, limit)
	if err != nil {
		return nil, 0, err
	}
	return g.buildRecommendPageResult(rawIds, hasMore, pageNum, pageSize)
}

// ListSessionRecommendGoodsIds 查询当前会话前 N 条原始推荐商品编号。
func (g *Recommend) ListSessionRecommendGoodsIds(ctx context.Context, contextGoodsIds []int64, limit int64) ([]int64, bool, error) {
	// 客户端未启用或上下文商品为空时，直接返回空会话推荐结果。
	if !g.Enabled() || len(contextGoodsIds) == 0 {
		return []int64{}, false, nil
	}
	// 调用方未显式传入上下文时，回退到默认上下文继续查询。
	if ctx == nil {
		ctx = context.TODO()
	}
	// 请求上限非法时，直接返回空结果，避免远端收到无效参数。
	if limit <= 0 {
		return []int64{}, false, nil
	}

	cleanGoodsIds := make([]int64, 0, len(contextGoodsIds))
	excludedGoods := make(map[int64]struct{}, len(contextGoodsIds))
	for _, goodsId := range contextGoodsIds {
		// 非法商品编号或重复商品编号时，直接跳过当前无效值。
		if goodsId <= 0 {
			continue
		}
		// 同一个上下文商品只保留一次，避免会话推荐被重复反馈放大。
		if _, ok := excludedGoods[goodsId]; ok {
			continue
		}
		excludedGoods[goodsId] = struct{}{}
		cleanGoodsIds = append(cleanGoodsIds, goodsId)
	}
	// 清洗后没有有效上下文商品时，无需继续调用远端推荐。
	if len(cleanGoodsIds) == 0 {
		return []int64{}, false, nil
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

	scores, err := g.gorseClient.SessionRecommend(ctx, feedbacks, int(limit)+1)
	if err != nil {
		return nil, false, err
	}

	rawIds := make([]string, 0, len(scores))
	for _, score := range scores {
		var goodsId int64
		goodsId, err = strconv.ParseInt(score.Id, 10, 64)
		// 推荐系统返回了非法商品编号或上下文商品本身时，直接跳过当前结果。
		if err != nil || goodsId <= 0 {
			continue
		}
		// 会话推荐结果不应该再次出现上下文商品本身，命中时直接过滤掉。
		if _, ok := excludedGoods[goodsId]; ok {
			continue
		}
		rawIds = append(rawIds, score.Id)
	}
	return g.buildRecommendGoodsIds(rawIds, limit)
}

// SessionRecommendGoodsIds 查询会话级推荐商品编号列表。
func (g *Recommend) SessionRecommendGoodsIds(ctx context.Context, contextGoodsIds []int64, pageNum, pageSize int64) ([]int64, int64, error) {
	limit := pageNum*pageSize + 1
	rawIds, hasMore, err := g.ListSessionRecommendGoodsIds(ctx, contextGoodsIds, limit)
	if err != nil {
		return nil, 0, err
	}
	return g.buildRecommendPageResult(rawIds, hasMore, pageNum, pageSize)
}

// consumeSyncBaseUser 消费用户同步队列并发送到推荐系统。
func (g *Recommend) consumeSyncBaseUser(message queueData.Message) error {
	user, err := pkgQueue.DecodeQueueData[models.BaseUser](message)
	if err != nil {
		return err
	}
	// 队列消息里没有有效用户快照时，直接忽略当前消息。
	if user == nil || user.ID <= 0 || !g.Enabled() {
		return nil
	}
	return g.syncBaseUser(context.TODO(), user)
}

// consumeDeleteBaseUser 消费用户删除队列并发送到推荐系统。
func (g *Recommend) consumeDeleteBaseUser(message queueData.Message) error {
	userIds, err := pkgQueue.DecodeQueueData[[]int64](message)
	if err != nil {
		return err
	}
	// 队列消息里没有有效用户编号时，直接忽略当前消息。
	if userIds == nil || len(*userIds) <= 0 || !g.Enabled() {
		return nil
	}

	// 推荐系统接口会在删除用户主体时一并级联删除该用户下的反馈数据。
	var deleteErr error
	for _, userId := range *userIds {
		_, err = g.gorseClient.DeleteUser(context.TODO(), strconv.FormatInt(userId, 10))
		if err != nil {
			deleteErr = errors.Join(deleteErr, err)
		}
	}
	return deleteErr
}

// consumeSyncGoodsInfo 消费商品同步队列并发送到推荐系统。
func (g *Recommend) consumeSyncGoodsInfo(message queueData.Message) error {
	goods, err := pkgQueue.DecodeQueueData[models.GoodsInfo](message)
	if err != nil {
		return err
	}
	// 队列消息里没有有效商品快照时，直接忽略当前消息。
	if goods == nil || goods.ID <= 0 || !g.Enabled() {
		return nil
	}
	return g.syncGoodsInfo(context.TODO(), goods)
}

// consumeDeleteGoodsInfo 消费商品删除队列并发送到推荐系统。
func (g *Recommend) consumeDeleteGoodsInfo(message queueData.Message) error {
	goodsIds, err := pkgQueue.DecodeQueueData[[]int64](message)
	if err != nil {
		return err
	}
	// 队列消息里没有有效商品编号时，直接忽略当前消息。
	if goodsIds == nil || len(*goodsIds) <= 0 || !g.Enabled() {
		return nil
	}

	// 推荐系统接口会在删除商品主体时一并级联删除该商品下的反馈数据。
	var deleteErr error
	for _, goodsId := range *goodsIds {
		_, err = g.gorseClient.DeleteItem(context.TODO(), strconv.FormatInt(goodsId, 10))
		if err != nil {
			deleteErr = errors.Join(deleteErr, err)
		}
	}
	return deleteErr
}

// consumeRecommendEvent 消费历史回放队列并发送到推荐系统。
func (g *Recommend) consumeRecommendEvent(message queueData.Message) error {
	eventList, err := pkgQueue.DecodeQueueData[[]*models.RecommendEvent](message)
	if err != nil {
		return err
	}
	// 队列消息里没有有效回放事件时，直接忽略当前消息。
	if eventList == nil || len(*eventList) == 0 || !g.Enabled() {
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
	if len(feedbacks) == 0 {
		return nil
	}

	_, err = g.gorseClient.InsertFeedback(ctx, feedbacks)
	return err
}

// syncBaseUser 将单个用户快照同步到推荐系统。
func (g *Recommend) syncBaseUser(ctx context.Context, user *models.BaseUser) error {
	// 客户端未启用或用户快照无效时，无需继续同步。
	if !g.Enabled() || user == nil || user.ID <= 0 {
		return nil
	}
	// 调用方未显式传入上下文时，回退到默认上下文继续同步。
	if ctx == nil {
		ctx = context.TODO()
	}

	recommendUser, userPatch := g.buildRecommendUser(user)
	_, err := g.gorseClient.InsertUser(ctx, recommendUser)
	if err == nil {
		return nil
	}

	_, updateErr := g.gorseClient.UpdateUser(ctx, recommendUser.UserId, userPatch)
	if updateErr == nil {
		return nil
	}
	return errors.Join(err, updateErr)
}

// buildRecommendUser 构建推荐系统用户写入载荷。
func (g *Recommend) buildRecommendUser(user *models.BaseUser) (client.User, client.UserPatch) {
	comment := user.NickName
	// 用户昵称为空时，回退到用户名作为注释信息。
	if comment == "" {
		comment = user.UserName
	}

	labels := map[string]interface{}{
		"user_id":   user.ID,
		"user_name": user.UserName,
		"role_id":   user.RoleID,
		"dept_id":   user.DeptID,
		"gender":    user.Gender,
		"status":    user.Status,
	}
	return client.User{
			UserId:  strconv.FormatInt(user.ID, 10),
			Labels:  labels,
			Comment: comment,
		}, client.UserPatch{
			Labels:  labels,
			Comment: new(comment),
		}
}

// syncGoodsInfo 将单个商品快照同步到推荐系统。
func (g *Recommend) syncGoodsInfo(ctx context.Context, goods *models.GoodsInfo) error {
	// 客户端未启用或商品快照无效时，无需继续同步。
	if !g.Enabled() || goods == nil || goods.ID <= 0 {
		return nil
	}
	// 调用方未显式传入上下文时，回退到默认上下文继续同步。
	if ctx == nil {
		ctx = context.TODO()
	}

	item, itemPatch := g.buildRecommendItem(goods)
	_, err := g.gorseClient.InsertItem(ctx, item)
	if err == nil {
		return nil
	}

	_, updateErr := g.gorseClient.UpdateItem(ctx, item.ItemId, itemPatch)
	if updateErr == nil {
		return nil
	}
	return errors.Join(err, updateErr)
}

// buildRecommendItem 构建推荐系统商品写入载荷。
func (g *Recommend) buildRecommendItem(goods *models.GoodsInfo) (client.Item, client.ItemPatch) {
	categoryIds := g.parseCategoryIds(goods.CategoryID)
	categories := make([]string, 0, len(categoryIds))
	for _, categoryId := range categoryIds {
		// 商品存在分类时，把分类编号作为推荐系统分类维度同步。
		if categoryId > 0 {
			categories = append(categories, strconv.FormatInt(categoryId, 10))
		}
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
			"category_id":    categoryIds,
			"status":         goods.Status,
			"price":          goods.Price,
			"discount_price": goods.DiscountPrice,
			"inventory":      goods.Inventory,
		},
	}
	return item, client.ItemPatch{
		IsHidden:   new(item.IsHidden),
		Categories: item.Categories,
		Timestamp:  &item.Timestamp,
		Labels:     item.Labels,
		Comment:    new(item.Comment),
	}
}

// parseCategoryIds 解析商品分类编号列表。
func (g *Recommend) parseCategoryIds(rawCategoryIds string) []int64 {
	// 分类字段为空时，直接返回空分类列表。
	if strings.TrimSpace(rawCategoryIds) == "" {
		return []int64{}
	}

	categoryIds := make([]int64, 0)
	// 分类 JSON 解析失败时，回退为空列表，避免单条商品脏数据阻塞整批推荐同步。
	if err := json.Unmarshal([]byte(rawCategoryIds), &categoryIds); err != nil {
		return []int64{}
	}
	return categoryIds
}

// buildRecommendGoodsIds 清洗推荐系统返回的原始商品编号列表。
func (g *Recommend) buildRecommendGoodsIds(rawIds []string, limit int64) ([]int64, bool, error) {
	goodsIds := make([]int64, 0, len(rawIds))
	seenGoodsIds := make(map[int64]struct{}, len(rawIds))
	for _, rawId := range rawIds {
		goodsId, err := strconv.ParseInt(rawId, 10, 64)
		// 推荐系统返回了非法商品编号时，直接跳过当前无效值，避免整批结果回退成本地兜底。
		if err != nil {
			continue
		}
		// 返回结果里包含非法商品编号时，直接跳过当前无效值。
		if goodsId <= 0 {
			continue
		}
		// 推荐系统偶发返回重复商品时，仅保留首次命中的结果，避免前端分页出现重复卡片。
		if _, ok := seenGoodsIds[goodsId]; ok {
			continue
		}
		seenGoodsIds[goodsId] = struct{}{}
		goodsIds = append(goodsIds, goodsId)
	}

	hasMore := false
	// 当前结果超过请求上限时，说明远端至少还存在一条后续原始推荐结果。
	if int64(len(goodsIds)) > limit {
		hasMore = true
		goodsIds = goodsIds[:limit]
	}
	return goodsIds, hasMore, nil
}

// buildRecommendPageResult 将推荐系统返回结果转换为项目分页结果。
func (g *Recommend) buildRecommendPageResult(goodsIds []int64, hasMore bool, pageNum, pageSize int64) ([]int64, int64, error) {
	startIndex := (pageNum - 1) * pageSize
	// 当前页起点已经超过已知结果时，仅在仍有后续数据时保留翻页信号。
	if startIndex >= int64(len(goodsIds)) {
		if hasMore {
			return []int64{}, startIndex + 1, nil
		}
		return []int64{}, int64(len(goodsIds)), nil
	}

	endIndex := startIndex + pageSize
	// 已知结果不足一整页时，只截取当前实际存在的数据范围。
	if endIndex > int64(len(goodsIds)) {
		endIndex = int64(len(goodsIds))
	}

	pageGoodsIds := append([]int64(nil), goodsIds[int(startIndex):int(endIndex)]...)
	total := startIndex + int64(len(pageGoodsIds))
	// 当前页后面仍有已知结果，或远端还存在未加载结果时，向前端保留“还有下一页”的信号。
	if int64(len(goodsIds)) > endIndex || hasMore {
		total++
	}
	return pageGoodsIds, total, nil
}

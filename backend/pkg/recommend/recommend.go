package recommend

import (
	"context"
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

	client "github.com/gorse-io/gorse-go"
	queueData "github.com/liujitcn/kratos-kit/queue/data"
	"github.com/liujitcn/kratos-kit/sdk"
)

const (
	recommendUserPrefix = "user:"
)

type Recommend struct {
	systemClient *client.GorseClient
}

// NewRecommend 创建推荐系统客户端包装。
func NewRecommend(cfg *conf.Recommend) *Recommend {
	entryPoint := strings.TrimSpace(cfg.GetEntryPoint())
	// 未配置入口地址时，直接返回空客户端并在业务侧走本地兜底。
	if entryPoint == "" {
		pkgQueue.SetRecommendEnabled(false)
		return &Recommend{}
	}

	systemClient := client.NewGorseClient(entryPoint, cfg.GetApiKey())
	recommend := &Recommend{
		systemClient: systemClient,
	}
	pkgQueue.SetRecommendEnabled(true)
	q := sdk.Runtime.GetQueue()
	// 运行时队列未初始化时，直接跳过推荐系统队列订阅注册。
	if q != nil {
		q.Register(string(_const.RecommendSyncBaseUser), recommend.consumeSyncBaseUser)
		q.Register(string(_const.RecommendDeleteBaseUser), recommend.consumeDeleteBaseUser)
		q.Register(string(_const.RecommendSyncGoodsInfo), recommend.consumeSyncGoodsInfo)
		q.Register(string(_const.RecommendDeleteGoodsInfo), recommend.consumeDeleteGoodsInfo)
		q.Register(string(_const.RecommendFeedbackEvent), recommend.consumeRecommendEvent)
		q.Register(string(_const.RecommendEvent), recommend.consumeReplayRecommendEvents)
	}
	return recommend
}

// Enabled 判断当前推荐系统客户端是否可用。
func (g *Recommend) Enabled() bool {
	return g != nil && g.systemClient != nil
}

// LoadBaseUserIds 加载推荐系统中已存在的用户主体编号集合。
func (g *Recommend) LoadBaseUserIds(ctx context.Context, pageSize int) (map[string]struct{}, error) {
	// 客户端未启用时，直接返回空用户集合。
	if !g.Enabled() {
		return map[string]struct{}{}, nil
	}
	// 调用方未显式传入上下文时，回退到默认上下文继续查询。
	if ctx == nil {
		ctx = context.TODO()
	}
	// 分页大小非法时，回退到默认分页大小，避免远端接口收到无效参数。
	if pageSize <= 0 {
		pageSize = 100
	}

	userIdSet := make(map[string]struct{})
	cursor := ""
	for {
		iterator, err := g.systemClient.GetUsers(ctx, pageSize, cursor)
		if err != nil {
			return nil, err
		}
		for _, item := range iterator.Users {
			// 远端返回空用户编号时，直接跳过当前无效数据。
			if item.UserId == "" {
				continue
			}
			userIdSet[item.UserId] = struct{}{}
		}
		// 当前页没有更多游标或下一页游标未发生变化时，说明远端集合已经遍历完成。
		if iterator.Cursor == "" || iterator.Cursor == cursor {
			break
		}
		cursor = iterator.Cursor
	}
	return userIdSet, nil
}

// SyncBaseUsers 同步一批后台用户快照到推荐系统。
func (g *Recommend) SyncBaseUsers(ctx context.Context, userList []*models.BaseUser, existingUserIds map[string]struct{}, staleUserIds map[string]struct{}) error {
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
			err := g.syncBaseUser(ctx, user)
			if err != nil {
				return err
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
		recommendUserId := g.getUserId(user.ID)
		// 当前用户在本地仍然存在时，先从远端待删除集合中移除，避免后续误删有效主体。
		delete(staleUserIds, recommendUserId)
		recommendUser, userPatch := g.buildRecommendUser(user)
		// 远端已经存在时，直接走单条更新，避免重复插入失败后再回退。
		if _, ok := existingUserIds[recommendUser.UserId]; ok {
			_, err := g.systemClient.UpdateUser(ctx, recommendUser.UserId, userPatch)
			if err != nil {
				return err
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

	_, err := g.systemClient.InsertUsers(ctx, insertUsers)
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
		existingUserIds[item.UserId] = struct{}{}
	}
	return nil
}

// DeleteBaseUsers 删除推荐系统中多余的用户主体。
func (g *Recommend) DeleteBaseUsers(ctx context.Context, staleUserIds map[string]struct{}) error {
	// 客户端未启用或没有待删除用户时，直接跳过当前清理批次。
	if !g.Enabled() || len(staleUserIds) == 0 {
		return nil
	}
	// 调用方未显式传入上下文时，回退到默认上下文继续删除。
	if ctx == nil {
		ctx = context.TODO()
	}

	var deleteErr error
	for userId := range staleUserIds {
		// 待删除编号为空时，直接跳过当前无效主体。
		if userId == "" {
			continue
		}
		// 推荐系统接口会在删除用户主体时一并级联删除该用户下的反馈数据。
		_, err := g.systemClient.DeleteUser(ctx, userId)
		if err != nil {
			deleteErr = errors.Join(deleteErr, err)
		}
	}
	return deleteErr
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
	_, err := g.systemClient.InsertUser(ctx, recommendUser)
	if err == nil {
		return nil
	}

	_, updateErr := g.systemClient.UpdateUser(ctx, recommendUser.UserId, userPatch)
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
		"user_id":    user.ID,
		"user_name":  user.UserName,
		"role_id":    user.RoleID,
		"dept_id":    user.DeptID,
		"gender":     user.Gender,
		"status":     user.Status,
		"has_phone":  user.Phone != "",
		"has_openid": user.Openid != "",
	}
	commentPatch := comment
	return client.User{
			UserId:  g.getUserId(user.ID),
			Labels:  labels,
			Comment: comment,
		}, client.UserPatch{
			Labels:  labels,
			Comment: &commentPatch,
		}
}

// LoadGoodsInfoIds 加载推荐系统中已存在的商品主体编号集合。
func (g *Recommend) LoadGoodsInfoIds(ctx context.Context, pageSize int) (map[string]struct{}, error) {
	// 客户端未启用时，直接返回空商品集合。
	if !g.Enabled() {
		return map[string]struct{}{}, nil
	}
	// 调用方未显式传入上下文时，回退到默认上下文继续查询。
	if ctx == nil {
		ctx = context.TODO()
	}
	// 分页大小非法时，回退到默认分页大小，避免远端接口收到无效参数。
	if pageSize <= 0 {
		pageSize = 100
	}

	itemIdSet := make(map[string]struct{})
	cursor := ""
	for {
		iterator, err := g.systemClient.GetItems(ctx, pageSize, cursor)
		if err != nil {
			return nil, err
		}
		for _, item := range iterator.Items {
			// 远端返回空商品编号时，直接跳过当前无效数据。
			if item.ItemId == "" {
				continue
			}
			itemIdSet[item.ItemId] = struct{}{}
		}
		// 当前页没有更多游标或下一页游标未发生变化时，说明远端集合已经遍历完成。
		if iterator.Cursor == "" || iterator.Cursor == cursor {
			break
		}
		cursor = iterator.Cursor
	}
	return itemIdSet, nil
}

// SyncGoodsInfos 同步一批商品快照到推荐系统。
func (g *Recommend) SyncGoodsInfos(ctx context.Context, goodsList []*models.GoodsInfo, existingItemIds map[string]struct{}, staleItemIds map[string]struct{}) error {
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
			err := g.syncGoodsInfo(ctx, goods)
			if err != nil {
				return err
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
		delete(staleItemIds, recommendItem.ItemId)
		// 远端已经存在时，直接走单条更新，避免重复插入失败后再回退。
		if _, ok := existingItemIds[recommendItem.ItemId]; ok {
			_, err := g.systemClient.UpdateItem(ctx, recommendItem.ItemId, itemPatch)
			if err != nil {
				return err
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

	_, err := g.systemClient.InsertItems(ctx, insertItems)
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
		existingItemIds[item.ItemId] = struct{}{}
	}
	return nil
}

// DeleteGoodsInfos 删除推荐系统中多余的商品主体。
func (g *Recommend) DeleteGoodsInfos(ctx context.Context, staleItemIds map[string]struct{}) error {
	// 客户端未启用或没有待删除商品时，直接跳过当前清理批次。
	if !g.Enabled() || len(staleItemIds) == 0 {
		return nil
	}
	// 调用方未显式传入上下文时，回退到默认上下文继续删除。
	if ctx == nil {
		ctx = context.TODO()
	}

	var deleteErr error
	for itemId := range staleItemIds {
		// 待删除编号为空时，直接跳过当前无效主体。
		if itemId == "" {
			continue
		}
		// 推荐系统接口会在删除商品主体时一并级联删除该商品下的反馈数据。
		_, err := g.systemClient.DeleteItem(ctx, itemId)
		if err != nil {
			deleteErr = errors.Join(deleteErr, err)
		}
	}
	return deleteErr
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
	_, err := g.systemClient.InsertItem(ctx, item)
	if err == nil {
		return nil
	}

	_, updateErr := g.systemClient.UpdateItem(ctx, item.ItemId, itemPatch)
	if updateErr == nil {
		return nil
	}
	return errors.Join(err, updateErr)
}

// buildRecommendItem 构建推荐系统商品写入载荷。
func (g *Recommend) buildRecommendItem(goods *models.GoodsInfo) (client.Item, client.ItemPatch) {
	categories := make([]string, 0, 1)
	// 商品存在分类时，把分类编号作为推荐系统分类维度同步。
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
	isHidden := item.IsHidden
	comment := item.Comment
	return item, client.ItemPatch{
		IsHidden:   &isHidden,
		Categories: item.Categories,
		Timestamp:  &item.Timestamp,
		Labels:     item.Labels,
		Comment:    &comment,
	}
}

// GetRecommendGoodsIds 查询用户维度推荐商品编号列表。
func (g *Recommend) GetRecommendGoodsIds(ctx context.Context, actor *app.RecommendActor, pageNum, pageSize int64) ([]int64, int64, error) {
	// 客户端未启用、推荐主体无效或主体不是登录用户时，直接返回空推荐结果。
	if !g.Enabled() || actor == nil || actor.GetActorId() <= 0 {
		return []int64{}, 0, nil
	}
	// 匿名主体不走用户维度的推荐系统推荐。
	if actor.GetActorType() != common.RecommendActorType_USER {
		return []int64{}, 0, nil
	}

	offset := int((pageNum - 1) * pageSize)
	rawIds, err := g.systemClient.GetRecommend(ctx, g.getUserId(actor.GetActorId()), "", int(pageSize)+1, offset)
	if err != nil {
		return nil, 0, err
	}
	return g.buildRecommendPageResult(rawIds, pageNum, pageSize)
}

// SessionRecommendGoodsIds 查询会话级推荐商品编号列表。
func (g *Recommend) SessionRecommendGoodsIds(ctx context.Context, contextGoodsIds []int64, pageNum, pageSize int64) ([]int64, int64, error) {
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

	scores, err := g.systemClient.SessionRecommend(ctx, feedbacks, int(pageNum*pageSize)+1)
	if err != nil {
		return nil, 0, err
	}

	rawIds := make([]string, 0, len(scores))
	for _, score := range scores {
		goodsId, convErr := strconv.ParseInt(score.Id, 10, 64)
		// 推荐系统返回了非法商品编号或上下文商品本身时，直接跳过当前结果。
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

// ensureRecommendUser 确保推荐用户主体已同步到推荐系统。
func (g *Recommend) ensureRecommendUser(ctx context.Context, userId int64) error {
	// 客户端未启用或用户编号非法时，无需继续同步推荐主体。
	if !g.Enabled() || userId <= 0 {
		return nil
	}

	recommendUserId := g.getUserId(userId)
	labels := map[string]interface{}{
		"actor_type": int32(common.RecommendActorType_USER),
		"actor_id":   userId,
	}
	_, err := g.systemClient.InsertUser(ctx, client.User{
		UserId: recommendUserId,
		Labels: labels,
	})
	if err == nil {
		return nil
	}

	_, updateErr := g.systemClient.UpdateUser(ctx, recommendUserId, client.UserPatch{
		Labels: labels,
	})
	if updateErr == nil {
		return nil
	}
	return errors.Join(err, updateErr)
}

// getUserId 构建项目内统一的推荐系统登录用户编号。
func (g *Recommend) getUserId(userId int64) string {
	return recommendUserPrefix + strconv.FormatInt(userId, 10)
}

// buildRecommendPageResult 将推荐系统返回结果转换为项目分页结果。
func (g *Recommend) buildRecommendPageResult(rawIds []string, pageNum, pageSize int64) ([]int64, int64, error) {
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
	userId, err := pkgQueue.DecodeQueueData[int64](message)
	if err != nil {
		return err
	}
	// 队列消息里没有有效用户编号时，直接忽略当前消息。
	if userId == nil || *userId <= 0 || !g.Enabled() {
		return nil
	}

	// 推荐系统接口会在删除用户主体时一并级联删除该用户下的反馈数据。
	_, err = g.systemClient.DeleteUser(context.TODO(), g.getUserId(*userId))
	return err
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
	goodsId, err := pkgQueue.DecodeQueueData[int64](message)
	if err != nil {
		return err
	}
	// 队列消息里没有有效商品编号时，直接忽略当前消息。
	if goodsId == nil || *goodsId <= 0 || !g.Enabled() {
		return nil
	}

	// 推荐系统接口会在删除商品主体时一并级联删除该商品下的反馈数据。
	_, err = g.systemClient.DeleteItem(context.TODO(), strconv.FormatInt(*goodsId, 10))
	return err
}

// consumeRecommendEvent 消费推荐事件队列并发送到推荐系统。
func (g *Recommend) consumeRecommendEvent(message queueData.Message) error {
	event, err := pkgQueue.DecodeQueueData[pkgQueue.RecommendEventReportEvent](message)
	if err != nil {
		return err
	}
	// 队列消息里没有推荐事件主体时，直接忽略当前消息。
	if event == nil || event.RecommendActor == nil || event.RecommendActor.GetActorId() <= 0 || !g.Enabled() {
		return nil
	}

	actor := event.RecommendActor
	// 匿名主体不写入推荐系统，避免匿名用户在推荐系统中形成主体数据。
	if actor.GetActorType() != common.RecommendActorType_USER {
		return nil
	}
	// 事件类型未知时，当前事件不需要继续同步到推荐系统。
	if event.EventType == common.RecommendEventType_UNKNOWN_RET {
		return nil
	}

	eventTime := event.EventTime
	if eventTime.IsZero() {
		eventTime = time.Now()
	}

	ctx := context.TODO()
	err = g.ensureRecommendUser(ctx, actor.GetActorId())
	if err != nil {
		return err
	}

	userId := g.getUserId(actor.GetActorId())
	feedbacks := make([]client.Feedback, 0, len(event.Items))
	for _, item := range event.Items {
		// 商品项为空或商品编号非法时，直接跳过当前无效事件项。
		if item == nil || item.GoodsId <= 0 {
			continue
		}

		value := float64(item.GoodsNum)
		// 事件未显式传入商品数量时，统一按 1 处理反馈值。
		if value <= 0 {
			value = 1
		}

		feedbacks = append(feedbacks, client.Feedback{
			FeedbackType: event.EventType.String(),
			UserId:       userId,
			ItemId:       strconv.FormatInt(item.GoodsId, 10),
			Value:        value,
			Timestamp:    eventTime,
		})
	}
	if len(feedbacks) == 0 {
		return nil
	}

	_, err = g.systemClient.InsertFeedback(ctx, feedbacks)
	return err
}

// consumeReplayRecommendEvents 消费历史回放队列并发送到推荐系统。
func (g *Recommend) consumeReplayRecommendEvents(message queueData.Message) error {
	eventList, err := pkgQueue.DecodeQueueData[[]*models.RecommendEvent](message)
	if err != nil {
		return err
	}
	// 队列消息里没有有效回放事件时，直接忽略当前消息。
	if eventList == nil || len(*eventList) == 0 || !g.Enabled() {
		return nil
	}

	ctx := context.TODO()
	targetUserId := int64(0)
	feedbacks := make([]client.Feedback, 0, len(*eventList))
	for _, item := range *eventList {
		// 历史事件为空、商品编号非法或事件类型未知时，直接跳过当前无效事件。
		if item == nil || item.ActorID <= 0 || item.ActorType != int32(common.RecommendActorType_USER) || item.GoodsID <= 0 || item.EventType == int32(common.RecommendEventType_UNKNOWN_RET) {
			continue
		}
		// 首条有效事件决定本次回放的目标用户，后续仅接受同一用户的历史。
		if targetUserId == 0 {
			targetUserId = item.ActorID
			err = g.ensureRecommendUser(ctx, targetUserId)
			if err != nil {
				return err
			}
		}
		if item.ActorID != targetUserId {
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
			UserId:       g.getUserId(targetUserId),
			ItemId:       strconv.FormatInt(item.GoodsID, 10),
			Value:        value,
			Timestamp:    timestamp,
		})
	}
	if len(feedbacks) == 0 {
		return nil
	}

	_, err = g.systemClient.InsertFeedback(ctx, feedbacks)
	return err
}

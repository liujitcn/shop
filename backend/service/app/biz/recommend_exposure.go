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
	recommendactor "shop/pkg/recommend/actor"
	recommendcandidate "shop/pkg/recommend/candidate"
	recommendcontext "shop/pkg/recommend/context"
	recommendcore "shop/pkg/recommend/core"
	recommendevent "shop/pkg/recommend/event"
	"shop/pkg/utils"

	"github.com/liujitcn/gorm-kit/repo"
	queueData "github.com/liujitcn/kratos-kit/queue/data"
	"gorm.io/gorm"
)

// RecommendExposureCase 推荐曝光业务处理对象。
type RecommendExposureCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.RecommendExposureRepo
	*data.RecommendGoodsActionRepo
	*data.RecommendRequestRepo
	*data.RecommendUserPreferenceRepo
	*data.RecommendUserGoodsPreferenceRepo
	*data.RecommendGoodsRelationRepo
	*data.GoodsInfoRepo
}

// NewRecommendExposureCase 创建推荐曝光业务处理对象。
func NewRecommendExposureCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	recommendExposureRepo *data.RecommendExposureRepo,
	recommendGoodsActionRepo *data.RecommendGoodsActionRepo,
	recommendRequestRepo *data.RecommendRequestRepo,
	recommendUserPreferenceRepo *data.RecommendUserPreferenceRepo,
	recommendUserGoodsPreferenceRepo *data.RecommendUserGoodsPreferenceRepo,
	recommendGoodsRelationRepo *data.RecommendGoodsRelationRepo,
	goodsInfoRepo *data.GoodsInfoRepo,
) *RecommendExposureCase {
	recommendExposureCase := &RecommendExposureCase{
		BaseCase:                         baseCase,
		tx:                               tx,
		RecommendExposureRepo:            recommendExposureRepo,
		RecommendGoodsActionRepo:         recommendGoodsActionRepo,
		RecommendRequestRepo:             recommendRequestRepo,
		RecommendUserPreferenceRepo:      recommendUserPreferenceRepo,
		RecommendUserGoodsPreferenceRepo: recommendUserGoodsPreferenceRepo,
		RecommendGoodsRelationRepo:       recommendGoodsRelationRepo,
		GoodsInfoRepo:                    goodsInfoRepo,
	}
	recommendExposureCase.RegisterQueueConsumer(_const.RecommendEvent, recommendExposureCase.SaveRecommendEvent)
	return recommendExposureCase
}

// RecommendExposureReport 接收独立推荐曝光接口并异步投递事件。
func (c *RecommendExposureCase) RecommendExposureReport(ctx context.Context, req *app.RecommendExposureReportRequest) error {
	// 空请求直接忽略，避免埋点接口影响主业务流程。
	if req == nil {
		return nil
	}

	actor := recommendactor.Resolve(ctx)
	publishRecommendExposureEvent(
		actor,
		strings.TrimSpace(req.GetRequestId()),
		recommendcontext.NormalizeSceneEnum(req.GetScene()),
		req.GetGoodsIds(),
	)
	return nil
}

// SaveRecommendEvent 消费推荐行为事件。
func (c *RecommendExposureCase) SaveRecommendEvent(message queueData.Message) error {
	rawBody, err := json.Marshal(message.Values)
	if err != nil {
		return err
	}

	payload := make(map[string]*RecommendEvent)
	err = json.Unmarshal(rawBody, &payload)
	if err != nil {
		return err
	}

	event, ok := payload["data"]
	// 队列消息缺少业务体时直接丢弃，避免消费者重复报错。
	if !ok || event == nil {
		return nil
	}
	return c.consume(context.TODO(), event)
}

// loadActorExposurePenalties 加载当前主体的曝光惩罚分。
func (c *RecommendExposureCase) loadActorExposurePenalties(ctx context.Context, actor *RecommendActor, scene int32, goodsIds []int64) (map[int64]float64, error) {
	// 主体、场景或候选商品缺失时，不计算曝光惩罚。
	if actor == nil || actor.ActorId <= 0 || scene == 0 || len(goodsIds) == 0 {
		return map[int64]float64{}, nil
	}
	exposureQuery := c.RecommendExposureRepo.Query(ctx).RecommendExposure
	actionQuery := c.RecommendGoodsActionRepo.Query(ctx).RecommendGoodsAction
	cutoff := time.Now().AddDate(0, 0, -recommendcandidate.ActorExposureLookbackDays)
	exposureList, err := c.RecommendExposureRepo.List(ctx,
		repo.Where(exposureQuery.ActorType.Eq(actor.ActorType)),
		repo.Where(exposureQuery.ActorID.Eq(actor.ActorId)),
		repo.Where(exposureQuery.Scene.Eq(scene)),
		repo.Where(exposureQuery.CreatedAt.Gte(cutoff)),
	)
	if err != nil {
		return nil, err
	}
	clickList, err := c.RecommendGoodsActionRepo.List(ctx,
		repo.Where(actionQuery.ActorType.Eq(actor.ActorType)),
		repo.Where(actionQuery.ActorID.Eq(actor.ActorId)),
		repo.Where(actionQuery.Scene.Eq(scene)),
		repo.Where(actionQuery.EventType.Eq(int32(common.RecommendGoodsActionType_RECOMMEND_GOODS_ACTION_CLICK))),
		repo.Where(actionQuery.CreatedAt.Gte(cutoff)),
		repo.Where(actionQuery.GoodsID.In(goodsIds...)),
	)
	if err != nil {
		return nil, err
	}

	exposureCountMap := make(map[int64]int64, len(goodsIds))
	for _, item := range exposureList {
		ids := make([]int64, 0)
		// 曝光商品列表反序列化失败时，直接跳过当前批次。
		if err = json.Unmarshal([]byte(item.GoodsIds), &ids); err != nil {
			continue
		}
		for _, goodsID := range ids {
			exposureCountMap[goodsID]++
		}
	}

	clickCountMap := make(map[int64]int64, len(clickList))
	for _, item := range clickList {
		clickCountMap[item.GoodsID]++
	}

	penalties := make(map[int64]float64, len(goodsIds))
	for _, goodsID := range goodsIds {
		exposureCount := exposureCountMap[goodsID]
		clickCount := clickCountMap[goodsID]
		// 曝光明显偏高且没有点击时，直接下调该商品权重。
		if exposureCount >= 3 && clickCount == 0 {
			penalties[goodsID] = 0.6
			continue
		}
		// 曝光很高但点击率极低时，施加更强的曝光惩罚。
		if exposureCount >= 5 && clickCount*20 < exposureCount {
			penalties[goodsID] = 0.3
		}
	}
	return penalties, nil
}

// BindRecommendExposureActor 将匿名曝光主体绑定为登录主体。
func (c *RecommendExposureCase) BindRecommendExposureActor(ctx context.Context, anonymousId, userId int64) error {
	recommendExposureQuery := c.RecommendExposureRepo.Data.Query(ctx).RecommendExposure
	_, err := recommendExposureQuery.WithContext(ctx).
		Where(
			recommendExposureQuery.ActorType.Eq(recommendevent.ActorTypeAnonymous),
			recommendExposureQuery.ActorID.Eq(anonymousId),
		).
		Updates(map[string]interface{}{
			"actor_type": recommendevent.ActorTypeUser,
			"actor_id":   userId,
		})
	return err
}

// consume 在同一事务中保存明细并执行聚合。
func (c *RecommendExposureCase) consume(ctx context.Context, event *RecommendEvent) error {
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		err := c.saveEventDetail(ctx, event)
		if err != nil {
			return err
		}
		return c.aggregateEvent(ctx, event)
	})
}

// saveEventDetail 保存推荐行为明细。
func (c *RecommendExposureCase) saveEventDetail(ctx context.Context, event *RecommendEvent) error {
	// 曝光单独保留批次明细，其余商品行为统一写入行为事实表。
	var err error
	switch event.EventType {
	case recommendevent.EventTypeExposure:
		goodsIdsJson, err := json.Marshal(event.GoodsIDs)
		if err != nil {
			return err
		}
		return c.RecommendExposureRepo.Create(ctx, &models.RecommendExposure{
			RequestID: event.RequestID,
			ActorType: event.ActorType,
			ActorID:   event.ActorID,
			Scene:     event.Scene,
			GoodsIds:  string(goodsIdsJson),
		})
	case recommendevent.EventTypeClick, recommendevent.EventTypeView:
		return c.RecommendGoodsActionRepo.Create(ctx, &models.RecommendGoodsAction{
			ActorType: event.ActorType,
			ActorID:   event.ActorID,
			EventType: int32(recommendevent.ConvertEventTypeToGoodsActionType(event.EventType)),
			GoodsID:   event.GoodsID,
			GoodsNum:  recommendevent.NormalizeGoodsCount(event.GoodsNum),
			Scene:     event.Scene,
			RequestID: event.RequestID,
			Position:  event.Position,
		})
	case recommendevent.EventTypeCollect, recommendevent.EventTypeCart:
		return c.RecommendGoodsActionRepo.Create(ctx, &models.RecommendGoodsAction{
			ActorType: event.ActorType,
			ActorID:   event.ActorID,
			EventType: int32(recommendevent.ConvertEventTypeToGoodsActionType(event.EventType)),
			GoodsID:   event.GoodsID,
			GoodsNum:  recommendevent.NormalizeGoodsCount(event.GoodsNum),
			Scene:     event.Scene,
			RequestID: event.RequestID,
			Position:  event.Position,
		})
	case recommendevent.EventTypeOrder, recommendevent.EventTypePay:
		for _, goodsItem := range recommendevent.NormalizeGoodsItems(event.GoodsItems) {
			err = c.RecommendGoodsActionRepo.Create(ctx, &models.RecommendGoodsAction{
				ActorType: event.ActorType,
				ActorID:   event.ActorID,
				EventType: int32(recommendevent.ConvertEventTypeToGoodsActionType(event.EventType)),
				GoodsID:   goodsItem.GoodsID,
				GoodsNum:  recommendevent.NormalizeGoodsCount(goodsItem.GoodsNum),
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
func (c *RecommendExposureCase) aggregateEvent(ctx context.Context, event *RecommendEvent) error {
	// 空事件无需聚合，避免事务内继续做无效操作。
	if event == nil {
		return nil
	}
	// 匿名用户不沉淀画像，只保留必要明细。
	if event.UserID <= 0 {
		return nil
	}
	// 单商品行为优先走单商品聚合链路。
	if recommendevent.IsSingleGoodsEvent(event.EventType) {
		return c.aggregateSingleGoodsEvent(ctx, event)
	}
	// 订单级行为需要拆成多商品聚合并维护共现关系。
	if recommendevent.IsOrderGoodsEvent(event.EventType) {
		return c.aggregateOrderGoodsEvent(ctx, event)
	}
	return nil
}

// aggregateSingleGoodsEvent 聚合单商品行为。
func (c *RecommendExposureCase) aggregateSingleGoodsEvent(ctx context.Context, event *RecommendEvent) error {
	// 单商品行为缺少商品ID时无法聚合，直接忽略。
	if event.GoodsID <= 0 {
		return nil
	}

	goodsInfo, err := c.findGoodsInfo(ctx, event.GoodsID)
	if err != nil {
		return err
	}

	eventTime := recommendevent.EventTime(event)
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
func (c *RecommendExposureCase) aggregateOrderGoodsEvent(ctx context.Context, event *RecommendEvent) error {
	eventTime := recommendevent.EventTime(event)
	goodsItems := recommendevent.NormalizeGoodsItems(event.GoodsItems)
	var err error
	for _, goodsItem := range goodsItems {
		singleEvent := &RecommendEvent{
			EventType:  event.EventType,
			UserID:     event.UserID,
			GoodsID:    goodsItem.GoodsID,
			GoodsNum:   goodsItem.GoodsNum,
			Scene:      goodsItem.Scene,
			RequestID:  goodsItem.RequestID,
			Position:   goodsItem.Position,
			OccurredAt: event.OccurredAt,
		}

		var goodsInfo *models.GoodsInfo
		goodsInfo, err = c.findGoodsInfo(ctx, goodsItem.GoodsID)
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
func (c *RecommendExposureCase) upsertUserGoodsPreference(ctx context.Context, event *RecommendEvent, eventTime time.Time, goodsNum int64) error {
	recommendUserGoodsPreferenceQuery := c.RecommendUserGoodsPreferenceRepo.Query(ctx).RecommendUserGoodsPreference
	entity, err := c.RecommendUserGoodsPreferenceRepo.Find(ctx,
		repo.Where(recommendUserGoodsPreferenceQuery.UserID.Eq(event.UserID)),
		repo.Where(recommendUserGoodsPreferenceQuery.GoodsID.Eq(event.GoodsID)),
		repo.Where(recommendUserGoodsPreferenceQuery.WindowDays.Eq(recommendevent.AggregateWindowDays)),
	)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	summaryJson := ""
	score := recommendevent.EventWeight(event.EventType) * recommendevent.NormalizeGoodsNum(goodsNum)
	// 已有画像记录时在原有得分与摘要上继续累加。
	if entity != nil {
		score += entity.Score
		summaryJson = entity.BehaviorSummary
	}
	summaryJson, err = recommendevent.AddBehaviorSummaryCount(summaryJson, recommendevent.EventSummaryKey(event.EventType), recommendevent.NormalizeGoodsCount(goodsNum))
	if err != nil {
		return err
	}

	// 不存在画像记录时，新建窗口内商品偏好画像。
	if entity == nil || entity.ID == 0 {
		return c.RecommendUserGoodsPreferenceRepo.Create(ctx, &models.RecommendUserGoodsPreference{
			UserID:           event.UserID,
			GoodsID:          event.GoodsID,
			Score:            score,
			LastBehaviorType: event.EventType,
			LastBehaviorAt:   eventTime,
			BehaviorSummary:  summaryJson,
			WindowDays:       recommendevent.AggregateWindowDays,
			CreatedAt:        eventTime,
			UpdatedAt:        eventTime,
		})
	}

	entity.Score = score
	entity.LastBehaviorType = event.EventType
	entity.LastBehaviorAt = eventTime
	entity.BehaviorSummary = summaryJson
	entity.UpdatedAt = eventTime
	return c.RecommendUserGoodsPreferenceRepo.UpdateById(ctx, entity)
}

// upsertUserCategoryPreference 累计用户对商品类目的偏好得分。
func (c *RecommendExposureCase) upsertUserCategoryPreference(ctx context.Context, event *RecommendEvent, categoryId int64, eventTime time.Time, goodsNum int64) error {
	// 商品没有有效类目时，不生成类目偏好画像。
	if categoryId <= 0 {
		return nil
	}

	recommendUserPreferenceQuery := c.RecommendUserPreferenceRepo.Query(ctx).RecommendUserPreference
	entity, err := c.RecommendUserPreferenceRepo.Find(ctx,
		repo.Where(recommendUserPreferenceQuery.UserID.Eq(event.UserID)),
		repo.Where(recommendUserPreferenceQuery.PreferenceType.Eq(recommendevent.PreferenceTypeCategory)),
		repo.Where(recommendUserPreferenceQuery.TargetID.Eq(categoryId)),
		repo.Where(recommendUserPreferenceQuery.WindowDays.Eq(recommendevent.AggregateWindowDays)),
	)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	summaryJson := ""
	score := recommendevent.EventWeight(event.EventType) * recommendevent.NormalizeGoodsNum(goodsNum)
	// 已有类目画像时继续在原值上累积行为分数。
	if entity != nil {
		score += entity.Score
		summaryJson = entity.BehaviorSummary
	}
	summaryJson, err = recommendevent.AddBehaviorSummaryCount(summaryJson, recommendevent.EventSummaryKey(event.EventType), recommendevent.NormalizeGoodsCount(goodsNum))
	if err != nil {
		return err
	}

	// 不存在类目画像时，新建当前类目的聚合结果。
	if entity == nil || entity.ID == 0 {
		return c.RecommendUserPreferenceRepo.Create(ctx, &models.RecommendUserPreference{
			UserID:          event.UserID,
			PreferenceType:  recommendevent.PreferenceTypeCategory,
			TargetID:        categoryId,
			Score:           score,
			BehaviorSummary: summaryJson,
			WindowDays:      recommendevent.AggregateWindowDays,
			CreatedAt:       eventTime,
			UpdatedAt:       eventTime,
		})
	}

	entity.Score = score
	entity.BehaviorSummary = summaryJson
	entity.UpdatedAt = eventTime
	return c.RecommendUserPreferenceRepo.UpdateById(ctx, entity)
}

// upsertGoodsRelation 按同一次推荐请求的共同出现结果累计商品关联度。
func (c *RecommendExposureCase) upsertGoodsRelation(ctx context.Context, event *RecommendEvent, eventTime time.Time) error {
	// 只有带 requestId 的请求才参与商品共现关系沉淀。
	if !recommendcontext.HasRequest(event.RequestID) {
		return nil
	}

	relationType := recommendevent.RelationType(event.EventType)
	// 无法映射为关系类型的行为不进入商品关联聚合。
	if relationType == "" {
		return nil
	}

	relatedGoodsIds, err := c.listRequestRelatedGoodsIds(ctx, event.RequestID, event.GoodsID)
	if err != nil {
		return err
	}
	// 共同出现在同一请求中的商品两两建立双向关联关系。
	for _, relatedGoodsId := range relatedGoodsIds {
		err = c.upsertSingleGoodsRelation(ctx, relatedGoodsId, event.GoodsID, relationType, eventTime, recommendevent.NormalizeGoodsNum(event.GoodsNum))
		if err != nil {
			return err
		}
		err = c.upsertSingleGoodsRelation(ctx, event.GoodsID, relatedGoodsId, relationType, eventTime, recommendevent.NormalizeGoodsNum(event.GoodsNum))
		if err != nil {
			return err
		}
	}
	return nil
}

// upsertOrderGoodsRelations 累计订单内商品的共购与共支付关系。
func (c *RecommendExposureCase) upsertOrderGoodsRelations(ctx context.Context, event *RecommendEvent, goodsItems []*RecommendEventGoodsItem, eventTime time.Time) error {
	relationType := recommendevent.RelationType(event.EventType)
	// 仅下单和支付这类订单级事件需要建立共购关系。
	if relationType == "" {
		return nil
	}

	// 同一订单内商品两两组合，沉淀双向关联强度。
	var err error
	for i := 0; i < len(goodsItems); i++ {
		leftItem := goodsItems[i]
		for j := i + 1; j < len(goodsItems); j++ {
			rightItem := goodsItems[j]
			relationScore := recommendevent.NormalizeGoodsNum(leftItem.GoodsNum) + recommendevent.NormalizeGoodsNum(rightItem.GoodsNum)
			err = c.upsertSingleGoodsRelation(ctx, leftItem.GoodsID, rightItem.GoodsID, relationType, eventTime, relationScore)
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
func (c *RecommendExposureCase) findGoodsInfo(ctx context.Context, goodsId int64) (*models.GoodsInfo, error) {
	return c.GoodsInfoRepo.FindById(ctx, goodsId)
}

// listRequestRelatedGoodsIds 读取推荐请求中与当前商品共同出现的其他商品。
func (c *RecommendExposureCase) listRequestRelatedGoodsIds(ctx context.Context, requestId string, goodsId int64) ([]int64, error) {
	recommendRequestQuery := c.RecommendRequestRepo.Query(ctx).RecommendRequest
	entity, err := c.RecommendRequestRepo.Find(ctx,
		repo.Where(recommendRequestQuery.RequestID.Eq(requestId)),
	)
	if err != nil {
		// 历史请求不存在时不报错，说明当前事件无法回溯推荐列表。
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []int64{}, nil
		}
		return nil, err
	}

	goodsIds := make([]int64, 0)
	err = json.Unmarshal([]byte(entity.GoodsIds), &goodsIds)
	if err != nil {
		return nil, err
	}

	relatedGoodsIds := make([]int64, 0, len(goodsIds))
	for _, item := range goodsIds {
		// 过滤当前商品自身与非法值，只保留同请求下的其他商品。
		if item == 0 || item == goodsId {
			continue
		}
		relatedGoodsIds = append(relatedGoodsIds, item)
	}
	return recommendcore.DedupeInt64s(relatedGoodsIds), nil
}

// upsertSingleGoodsRelation 累计单个方向的商品关联强度。
func (c *RecommendExposureCase) upsertSingleGoodsRelation(ctx context.Context, goodsId, relatedGoodsId int64, relationType string, eventTime time.Time, relationScore float64) error {
	// 非法商品对不进入关系计算，避免写入脏数据。
	if goodsId <= 0 || relatedGoodsId <= 0 || goodsId == relatedGoodsId {
		return nil
	}

	recommendGoodsRelationQuery := c.RecommendGoodsRelationRepo.Query(ctx).RecommendGoodsRelation
	entity, err := c.RecommendGoodsRelationRepo.Find(ctx,
		repo.Where(recommendGoodsRelationQuery.GoodsID.Eq(goodsId)),
		repo.Where(recommendGoodsRelationQuery.RelatedGoodsID.Eq(relatedGoodsId)),
		repo.Where(recommendGoodsRelationQuery.RelationType.Eq(relationType)),
		repo.Where(recommendGoodsRelationQuery.WindowDays.Eq(recommendevent.AggregateWindowDays)),
	)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	evidenceJson := ""
	score := relationScore
	// 外部未传分值时，回退到关系类型默认权重。
	if score <= 0 {
		score = recommendevent.RelationWeight(relationType)
	}
	// 已存在关系记录时在原有强度上继续累加。
	if entity != nil {
		score += entity.Score
		evidenceJson = entity.Evidence
	}
	evidenceJson, err = recommendevent.AddBehaviorSummaryCount(evidenceJson, relationType, int64(score))
	if err != nil {
		return err
	}

	// 关系记录不存在时，新建当前方向的商品关联结果。
	if entity == nil || entity.ID == 0 {
		return c.RecommendGoodsRelationRepo.Create(ctx, &models.RecommendGoodsRelation{
			GoodsID:        goodsId,
			RelatedGoodsID: relatedGoodsId,
			RelationType:   relationType,
			Score:          score,
			Evidence:       evidenceJson,
			WindowDays:     recommendevent.AggregateWindowDays,
			CreatedAt:      eventTime,
			UpdatedAt:      eventTime,
		})
	}

	entity.Score = score
	entity.Evidence = evidenceJson
	entity.UpdatedAt = eventTime
	return c.RecommendGoodsRelationRepo.UpdateById(ctx, entity)
}

// publishRecommendExposureEvent 投递推荐曝光事件。
func publishRecommendExposureEvent(actor *RecommendActor, requestId string, scene int32, goodsIds []int64) {
	utils.AddQueue(_const.RecommendEvent, &RecommendEvent{
		EventType:  recommendevent.EventTypeExposure,
		UserID:     actor.UserId,
		ActorType:  actor.ActorType,
		ActorID:    actor.ActorId,
		RequestID:  requestId,
		Scene:      scene,
		GoodsIDs:   goodsIds,
		OccurredAt: time.Now().Unix(),
	})
}

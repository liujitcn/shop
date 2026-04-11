package biz

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"shop/api/gen/go/app"
	"shop/api/gen/go/common"
	"shop/pkg/biz"
	_const "shop/pkg/const"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	recommendcontext "shop/pkg/recommend/context"
	recommendcore "shop/pkg/recommend/core"
	recommendevent "shop/pkg/recommend/event"
	"shop/pkg/utils"
	appdto "shop/service/app/dto"

	"github.com/liujitcn/gorm-kit/repo"
	queueData "github.com/liujitcn/kratos-kit/queue/data"
	"gorm.io/gorm"
)

// RecommendGoodsActionCase 推荐商品行为业务处理对象。
type RecommendGoodsActionCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.RecommendGoodsActionRepo
	recommendRequestCase             *RecommendRequestCase
	recommendUserPreferenceCase      *RecommendUserPreferenceCase
	recommendUserGoodsPreferenceCase *RecommendUserGoodsPreferenceCase
	recommendGoodsRelationCase       *RecommendGoodsRelationCase
	goodsInfoCase                    *GoodsInfoCase
}

// NewRecommendGoodsActionCase 创建推荐商品行为业务处理对象。
func NewRecommendGoodsActionCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	recommendGoodsActionRepo *data.RecommendGoodsActionRepo,
	recommendRequestCase *RecommendRequestCase,
	recommendUserPreferenceCase *RecommendUserPreferenceCase,
	recommendUserGoodsPreferenceCase *RecommendUserGoodsPreferenceCase,
	recommendGoodsRelationCase *RecommendGoodsRelationCase,
	goodsInfoCase *GoodsInfoCase,
) *RecommendGoodsActionCase {
	recommendGoodsActionCase := &RecommendGoodsActionCase{
		BaseCase:                         baseCase,
		tx:                               tx,
		RecommendGoodsActionRepo:         recommendGoodsActionRepo,
		recommendRequestCase:             recommendRequestCase,
		recommendUserPreferenceCase:      recommendUserPreferenceCase,
		recommendUserGoodsPreferenceCase: recommendUserGoodsPreferenceCase,
		recommendGoodsRelationCase:       recommendGoodsRelationCase,
		goodsInfoCase:                    goodsInfoCase,
	}
	recommendGoodsActionCase.RegisterQueueConsumer(_const.RecommendGoodsActionEvent, recommendGoodsActionCase.saveRecommendGoodsActionEvent)
	return recommendGoodsActionCase
}

// SaveRecommendEvent 消费推荐商品行为事件。
func (c *RecommendGoodsActionCase) saveRecommendGoodsActionEvent(message queueData.Message) error {
	rawBody, err := json.Marshal(message.Values)
	if err != nil {
		return err
	}

	payload := make(map[string][]*models.RecommendGoodsAction)
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

// bindRecommendGoodsActionActor 将匿名行为主体绑定为登录主体。
func (c *RecommendGoodsActionCase) bindRecommendGoodsActionActor(ctx context.Context, anonymousId, userId int64) error {
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

// publishRecommendGoodsActionEvent 投递单商品埋点事件。
func (c *RecommendGoodsActionCase) publishRecommendGoodsActionEvent(actor *appdto.RecommendActor, req *app.RecommendGoodsActionReportRequest) {
	goodsItems := req.GetGoodsItems()
	list := make([]*models.RecommendGoodsAction, len(goodsItems))
	for _, item := range goodsItems {
		recommendContext := item.GetRecommendContext()
		list = append(list, &models.RecommendGoodsAction{
			ActorType: actor.ActorType,
			ActorID:   actor.ActorId,
			EventType: int32(req.GetEventType()),
			GoodsID:   item.GoodsId,
			GoodsNum:  item.GoodsNum,
			Scene:     int32(recommendContext.Scene),
			RequestID: recommendContext.GetRequestId(),
			Position:  recommendContext.GetPosition(),
			CreatedAt: time.Now(),
		})
	}
	utils.AddQueue(_const.RecommendGoodsActionEvent, list)
}

// loadRecommendClickCountMap 查询当前主体在指定场景下的点击次数。
func (c *RecommendGoodsActionCase) loadRecommendClickCountMap(ctx context.Context, actor *appdto.RecommendActor, scene int32, cutoff time.Time, goodsIds []int64) (map[int64]int64, error) {
	actionQuery := c.RecommendGoodsActionRepo.Query(ctx).RecommendGoodsAction
	clickOpts := make([]repo.QueryOption, 0, 6)
	clickOpts = append(clickOpts, repo.Where(actionQuery.ActorType.Eq(actor.ActorType)))
	clickOpts = append(clickOpts, repo.Where(actionQuery.ActorID.Eq(actor.ActorId)))
	clickOpts = append(clickOpts, repo.Where(actionQuery.Scene.Eq(scene)))
	clickOpts = append(clickOpts, repo.Where(actionQuery.EventType.Eq(int32(common.RecommendGoodsActionType_RECOMMEND_GOODS_ACTION_CLICK))))
	clickOpts = append(clickOpts, repo.Where(actionQuery.CreatedAt.Gte(cutoff)))
	clickOpts = append(clickOpts, repo.Where(actionQuery.GoodsID.In(goodsIds...)))

	clickList, err := c.List(ctx, clickOpts...)
	// 查询点击行为失败时，直接返回错误交由调用方处理。
	if err != nil {
		return nil, err
	}

	clickCountMap := make(map[int64]int64, len(clickList))
	for _, item := range clickList {
		clickCountMap[item.GoodsID]++
	}
	return clickCountMap, nil
}

// consume 在同一事务中保存商品行为明细并执行聚合。
func (c *RecommendGoodsActionCase) consume(ctx context.Context, event *appdto.RecommendGoodsActionEvent) error {
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		err := c.saveEventDetail(ctx, event)
		if err != nil {
			return err
		}
		return c.aggregateEvent(ctx, event)
	})
}

// saveEventDetail 保存推荐商品行为明细。
func (c *RecommendGoodsActionCase) saveEventDetail(ctx context.Context, event *appdto.RecommendGoodsActionEvent) error {
	var err error
	switch event.EventType {
	case recommendevent.EventTypeClick, recommendevent.EventTypeView:
		return c.RecommendGoodsActionRepo.Create(ctx, &models.RecommendGoodsAction{
			ActorType: event.ActorType,
			ActorID:   event.ActorId,
			EventType: int32(recommendevent.ConvertEventTypeToGoodsActionType(event.EventType)),
			GoodsID:   event.GoodsId,
			GoodsNum:  recommendevent.NormalizeGoodsCount(event.GoodsNum),
			Scene:     int32(event.Scene),
			RequestID: event.RequestId,
			Position:  event.Position,
		})
	case recommendevent.EventTypeCollect, recommendevent.EventTypeCart:
		return c.RecommendGoodsActionRepo.Create(ctx, &models.RecommendGoodsAction{
			ActorType: event.ActorType,
			ActorID:   event.ActorId,
			EventType: int32(recommendevent.ConvertEventTypeToGoodsActionType(event.EventType)),
			GoodsID:   event.GoodsId,
			GoodsNum:  recommendevent.NormalizeGoodsCount(event.GoodsNum),
			Scene:     int32(event.Scene),
			RequestID: event.RequestId,
			Position:  event.Position,
		})
	case recommendevent.EventTypeOrder, recommendevent.EventTypePay:
		for _, goodsItem := range recommendevent.NormalizeGoodsItems(event.GoodsItems) {
			err = c.RecommendGoodsActionRepo.Create(ctx, &models.RecommendGoodsAction{
				ActorType: event.ActorType,
				ActorID:   event.ActorId,
				EventType: int32(recommendevent.ConvertEventTypeToGoodsActionType(event.EventType)),
				GoodsID:   goodsItem.GoodsId,
				GoodsNum:  recommendevent.NormalizeGoodsCount(goodsItem.GoodsNum),
				Scene:     int32(goodsItem.Scene),
				RequestID: goodsItem.RequestId,
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

// aggregateEvent 聚合商品行为到画像与商品关联表。
func (c *RecommendGoodsActionCase) aggregateEvent(ctx context.Context, event *appdto.RecommendGoodsActionEvent) error {
	if event == nil || event.UserId <= 0 {
		return nil
	}
	if recommendevent.IsSingleGoodsEvent(event.EventType) {
		return c.aggregateSingleGoodsEvent(ctx, event)
	}
	if recommendevent.IsOrderGoodsEvent(event.EventType) {
		return c.aggregateOrderGoodsEvent(ctx, event)
	}
	return nil
}

// aggregateSingleGoodsEvent 聚合单商品行为。
func (c *RecommendGoodsActionCase) aggregateSingleGoodsEvent(ctx context.Context, event *appdto.RecommendGoodsActionEvent) error {
	if event.GoodsId <= 0 {
		return nil
	}

	goodsInfo, err := c.findGoodsInfo(ctx, event.GoodsId)
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
func (c *RecommendGoodsActionCase) aggregateOrderGoodsEvent(ctx context.Context, event *appdto.RecommendGoodsActionEvent) error {
	eventTime := recommendevent.EventTime(event)
	goodsItems := recommendevent.NormalizeGoodsItems(event.GoodsItems)
	var err error
	for _, goodsItem := range goodsItems {
		singleEvent := &appdto.RecommendGoodsActionEvent{
			EventType:  event.EventType,
			UserId:     event.UserId,
			GoodsId:    goodsItem.GoodsId,
			GoodsNum:   goodsItem.GoodsNum,
			Scene:      goodsItem.Scene,
			RequestId:  goodsItem.RequestId,
			Position:   goodsItem.Position,
			OccurredAt: event.OccurredAt,
		}

		var goodsInfo *models.GoodsInfo
		goodsInfo, err = c.findGoodsInfo(ctx, goodsItem.GoodsId)
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
func (c *RecommendGoodsActionCase) upsertUserGoodsPreference(ctx context.Context, event *appdto.RecommendGoodsActionEvent, eventTime time.Time, goodsNum int64) error {
	recommendUserGoodsPreferenceQuery := c.recommendUserGoodsPreferenceCase.RecommendUserGoodsPreferenceRepo.Query(ctx).RecommendUserGoodsPreference
	entity, err := c.recommendUserGoodsPreferenceCase.RecommendUserGoodsPreferenceRepo.Find(ctx,
		repo.Where(recommendUserGoodsPreferenceQuery.UserID.Eq(event.UserId)),
		repo.Where(recommendUserGoodsPreferenceQuery.GoodsID.Eq(event.GoodsId)),
		repo.Where(recommendUserGoodsPreferenceQuery.WindowDays.Eq(recommendevent.AggregateWindowDays)),
	)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	summaryJson := ""
	score := recommendevent.EventWeight(event.EventType) * recommendevent.NormalizeGoodsNum(goodsNum)
	if entity != nil {
		score += entity.Score
		summaryJson = entity.BehaviorSummary
	}
	summaryJson, err = recommendevent.AddBehaviorSummaryCount(summaryJson, recommendevent.EventSummaryKey(event.EventType), recommendevent.NormalizeGoodsCount(goodsNum))
	if err != nil {
		return err
	}

	if entity == nil || entity.ID == 0 {
		return c.recommendUserGoodsPreferenceCase.RecommendUserGoodsPreferenceRepo.Create(ctx, &models.RecommendUserGoodsPreference{
			UserID:           event.UserId,
			GoodsID:          event.GoodsId,
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
	return c.recommendUserGoodsPreferenceCase.RecommendUserGoodsPreferenceRepo.UpdateById(ctx, entity)
}

// upsertUserCategoryPreference 累计用户对商品类目的偏好得分。
func (c *RecommendGoodsActionCase) upsertUserCategoryPreference(ctx context.Context, event *appdto.RecommendGoodsActionEvent, categoryId int64, eventTime time.Time, goodsNum int64) error {
	if categoryId <= 0 {
		return nil
	}

	recommendUserPreferenceQuery := c.recommendUserPreferenceCase.RecommendUserPreferenceRepo.Query(ctx).RecommendUserPreference
	entity, err := c.recommendUserPreferenceCase.RecommendUserPreferenceRepo.Find(ctx,
		repo.Where(recommendUserPreferenceQuery.UserID.Eq(event.UserId)),
		repo.Where(recommendUserPreferenceQuery.PreferenceType.Eq(recommendevent.PreferenceTypeCategory)),
		repo.Where(recommendUserPreferenceQuery.TargetID.Eq(categoryId)),
		repo.Where(recommendUserPreferenceQuery.WindowDays.Eq(recommendevent.AggregateWindowDays)),
	)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	summaryJson := ""
	score := recommendevent.EventWeight(event.EventType) * recommendevent.NormalizeGoodsNum(goodsNum)
	if entity != nil {
		score += entity.Score
		summaryJson = entity.BehaviorSummary
	}
	summaryJson, err = recommendevent.AddBehaviorSummaryCount(summaryJson, recommendevent.EventSummaryKey(event.EventType), recommendevent.NormalizeGoodsCount(goodsNum))
	if err != nil {
		return err
	}

	if entity == nil || entity.ID == 0 {
		return c.recommendUserPreferenceCase.RecommendUserPreferenceRepo.Create(ctx, &models.RecommendUserPreference{
			UserID:          event.UserId,
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
	return c.recommendUserPreferenceCase.RecommendUserPreferenceRepo.UpdateById(ctx, entity)
}

// upsertGoodsRelation 按同一次推荐请求的共同出现结果累计商品关联度。
func (c *RecommendGoodsActionCase) upsertGoodsRelation(ctx context.Context, event *appdto.RecommendGoodsActionEvent, eventTime time.Time) error {
	if !recommendcontext.HasRequest(event.RequestId) {
		return nil
	}

	relationType := recommendevent.RelationType(event.EventType)
	if relationType == "" {
		return nil
	}

	relatedGoodsIds, err := c.listRequestRelatedGoodsIds(ctx, event.RequestId, event.GoodsId)
	if err != nil {
		return err
	}
	for _, relatedGoodsId := range relatedGoodsIds {
		err = c.upsertSingleGoodsRelation(ctx, relatedGoodsId, event.GoodsId, relationType, eventTime, recommendevent.NormalizeGoodsNum(event.GoodsNum))
		if err != nil {
			return err
		}
		err = c.upsertSingleGoodsRelation(ctx, event.GoodsId, relatedGoodsId, relationType, eventTime, recommendevent.NormalizeGoodsNum(event.GoodsNum))
		if err != nil {
			return err
		}
	}
	return nil
}

// upsertOrderGoodsRelations 累计订单内商品的共购与共支付关系。
func (c *RecommendGoodsActionCase) upsertOrderGoodsRelations(ctx context.Context, event *appdto.RecommendGoodsActionEvent, goodsItems []*appdto.RecommendEventGoodsItem, eventTime time.Time) error {
	relationType := recommendevent.RelationType(event.EventType)
	if relationType == "" {
		return nil
	}

	var err error
	for i := 0; i < len(goodsItems); i++ {
		leftItem := goodsItems[i]
		for j := i + 1; j < len(goodsItems); j++ {
			rightItem := goodsItems[j]
			relationScore := recommendevent.NormalizeGoodsNum(leftItem.GoodsNum) + recommendevent.NormalizeGoodsNum(rightItem.GoodsNum)
			err = c.upsertSingleGoodsRelation(ctx, leftItem.GoodsId, rightItem.GoodsId, relationType, eventTime, relationScore)
			if err != nil {
				return err
			}
			err = c.upsertSingleGoodsRelation(ctx, rightItem.GoodsId, leftItem.GoodsId, relationType, eventTime, relationScore)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// findGoodsInfo 查询商品信息，供画像聚合读取类目。
func (c *RecommendGoodsActionCase) findGoodsInfo(ctx context.Context, goodsId int64) (*models.GoodsInfo, error) {
	return c.goodsInfoCase.GoodsInfoRepo.FindById(ctx, goodsId)
}

// listRequestRelatedGoodsIds 读取推荐请求中与当前商品共同出现的其他商品。
func (c *RecommendGoodsActionCase) listRequestRelatedGoodsIds(ctx context.Context, requestId string, goodsId int64) ([]int64, error) {
	recommendRequestQuery := c.recommendRequestCase.RecommendRequestRepo.Query(ctx).RecommendRequest
	entity, err := c.recommendRequestCase.RecommendRequestRepo.Find(ctx,
		repo.Where(recommendRequestQuery.RequestID.Eq(requestId)),
	)
	if err != nil {
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
		if item == 0 || item == goodsId {
			continue
		}
		relatedGoodsIds = append(relatedGoodsIds, item)
	}
	return recommendcore.DedupeInt64s(relatedGoodsIds), nil
}

// upsertSingleGoodsRelation 累计单个方向的商品关联强度。
func (c *RecommendGoodsActionCase) upsertSingleGoodsRelation(ctx context.Context, goodsId, relatedGoodsId int64, relationType string, eventTime time.Time, relationScore float64) error {
	if goodsId <= 0 || relatedGoodsId <= 0 || goodsId == relatedGoodsId {
		return nil
	}

	recommendGoodsRelationQuery := c.recommendGoodsRelationCase.RecommendGoodsRelationRepo.Query(ctx).RecommendGoodsRelation
	entity, err := c.recommendGoodsRelationCase.RecommendGoodsRelationRepo.Find(ctx,
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
	if score <= 0 {
		score = recommendevent.RelationWeight(relationType)
	}
	if entity != nil {
		score += entity.Score
		evidenceJson = entity.Evidence
	}
	evidenceJson, err = recommendevent.AddBehaviorSummaryCount(evidenceJson, relationType, int64(score))
	if err != nil {
		return err
	}

	if entity == nil || entity.ID == 0 {
		return c.recommendGoodsRelationCase.RecommendGoodsRelationRepo.Create(ctx, &models.RecommendGoodsRelation{
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
	return c.recommendGoodsRelationCase.RecommendGoodsRelationRepo.UpdateById(ctx, entity)
}

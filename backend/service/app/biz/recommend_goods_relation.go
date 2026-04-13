package biz

import (
	"context"
	"errors"
	"time"

	"shop/api/gen/go/common"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	recommendcore "shop/pkg/recommend/core"
	recommendEvent "shop/pkg/recommend/event"

	"github.com/liujitcn/gorm-kit/repo"
	"gorm.io/gorm"
)

// RecommendGoodsRelationCase 推荐商品关联业务处理对象。
type RecommendGoodsRelationCase struct {
	*biz.BaseCase
	*data.RecommendGoodsRelationRepo
	recommendGoodsActionRepo *data.RecommendGoodsActionRepo
	recommendRequestRepo     *data.RecommendRequestRepo
	recommendRequestItemRepo *data.RecommendRequestItemRepo
}

// NewRecommendGoodsRelationCase 创建推荐商品关联业务处理对象。
func NewRecommendGoodsRelationCase(
	baseCase *biz.BaseCase,
	recommendGoodsRelationRepo *data.RecommendGoodsRelationRepo,
	recommendGoodsActionRepo *data.RecommendGoodsActionRepo,
	recommendRequestRepo *data.RecommendRequestRepo,
	recommendRequestItemRepo *data.RecommendRequestItemRepo,
) *RecommendGoodsRelationCase {
	return &RecommendGoodsRelationCase{
		BaseCase:                   baseCase,
		RecommendGoodsRelationRepo: recommendGoodsRelationRepo,
		recommendGoodsActionRepo:   recommendGoodsActionRepo,
		recommendRequestRepo:       recommendRequestRepo,
		recommendRequestItemRepo:   recommendRequestItemRepo,
	}
}

type recommendGoodsRelationKey struct {
	goodsId        int64
	relatedGoodsId int64
	relationType   string
}

type recommendOrderRelationGroupKey struct {
	requestId string
	eventType int32
}

// RebuildRecommendGoodsRelation 重建商品关联聚合。
func (c *RecommendGoodsRelationCase) RebuildRecommendGoodsRelation(ctx context.Context, windowDays int32) error {
	endAt := time.Now()
	startAt := endAt.AddDate(0, 0, -int(windowDays))

	relationQuery := c.Query(ctx).RecommendGoodsRelation
	relationOpts := make([]repo.QueryOption, 0, 1)
	relationOpts = append(relationOpts, repo.Where(relationQuery.WindowDays.Eq(windowDays)))
	err := c.Delete(ctx, relationOpts...)
	if err != nil {
		return err
	}

	actionQuery := c.recommendGoodsActionRepo.Query(ctx).RecommendGoodsAction
	actionOpts := make([]repo.QueryOption, 0, 5)
	actionOpts = append(actionOpts, repo.Where(actionQuery.CreatedAt.Gte(startAt)))
	actionOpts = append(actionOpts, repo.Where(actionQuery.CreatedAt.Lte(endAt)))
	actionOpts = append(actionOpts, repo.Where(actionQuery.EventType.In(
		int32(common.RecommendGoodsActionType_CLICK),
		int32(common.RecommendGoodsActionType_VIEW),
		int32(common.RecommendGoodsActionType_ORDER_CREATE),
		int32(common.RecommendGoodsActionType_ORDER_PAY),
	)))
	actionOpts = append(actionOpts, repo.Order(actionQuery.CreatedAt.Asc()))
	actionOpts = append(actionOpts, repo.Order(actionQuery.ID.Asc()))
	actionList, err := c.recommendGoodsActionRepo.List(ctx, actionOpts...)
	if err != nil {
		return err
	}
	// 当前窗口没有关联行为时，只保留清理动作。
	if len(actionList) == 0 {
		return nil
	}

	requestGoodsMap, err := c.loadRequestGoodsMap(ctx, actionList)
	if err != nil {
		return err
	}

	relationMap := make(map[recommendGoodsRelationKey]*models.RecommendGoodsRelation)
	orderGroupMap := make(map[recommendOrderRelationGroupKey][]*models.RecommendGoodsAction)
	for _, item := range actionList {
		// 非法行为明细不参与商品关联重建。
		if item == nil || item.GoodsID <= 0 {
			continue
		}

		eventType := common.RecommendGoodsActionType(item.EventType)
		if recommendEvent.IsSingleGoodsEvent(eventType) {
			relatedGoodsIds := requestGoodsMap[item.RequestID]
			for _, relatedGoodsId := range relatedGoodsIds {
				err = c.accumulateGoodsRelationEntity(relationMap, item.GoodsID, relatedGoodsId, eventType, item.CreatedAt, recommendEvent.NormalizeGoodsNum(item.GoodsNum), windowDays)
				if err != nil {
					return err
				}
			}
			continue
		}

		// 订单级行为按请求编号分组后统一沉淀整单共现关系。
		if item.RequestID != "" {
			key := recommendOrderRelationGroupKey{requestId: item.RequestID, eventType: item.EventType}
			orderGroupMap[key] = append(orderGroupMap[key], item)
		}
	}

	for _, list := range orderGroupMap {
		// 订单级行为少于两个商品时，不生成共现关系。
		if len(list) < 2 {
			continue
		}
		eventType := common.RecommendGoodsActionType(list[0].EventType)
		for i := 0; i < len(list); i++ {
			leftItem := list[i]
			for j := i + 1; j < len(list); j++ {
				rightItem := list[j]
				relationScore := recommendEvent.NormalizeGoodsNum(leftItem.GoodsNum) + recommendEvent.NormalizeGoodsNum(rightItem.GoodsNum)
				err = c.accumulateGoodsRelationEntity(relationMap, leftItem.GoodsID, rightItem.GoodsID, eventType, rightItem.CreatedAt, relationScore, windowDays)
				if err != nil {
					return err
				}
				err = c.accumulateGoodsRelationEntity(relationMap, rightItem.GoodsID, leftItem.GoodsID, eventType, rightItem.CreatedAt, relationScore, windowDays)
				if err != nil {
					return err
				}
			}
		}
	}

	list := make([]*models.RecommendGoodsRelation, 0, len(relationMap))
	for _, item := range relationMap {
		list = append(list, item)
	}
	if len(list) == 0 {
		return nil
	}
	return c.BatchCreate(ctx, list)
}

// loadRequestGoodsMap 按请求编号预加载推荐请求内的商品集合。
func (c *RecommendGoodsRelationCase) loadRequestGoodsMap(ctx context.Context, actionList []*models.RecommendGoodsAction) (map[string][]int64, error) {
	requestIds := make([]string, 0, len(actionList))
	for _, item := range actionList {
		// 只有单商品行为才需要回查推荐请求明细。
		if item == nil || item.RequestID == "" || !recommendEvent.IsSingleGoodsEvent(common.RecommendGoodsActionType(item.EventType)) {
			continue
		}
		requestIds = append(requestIds, item.RequestID)
	}
	requestIds = recommendcore.DedupeStrings(requestIds)
	if len(requestIds) == 0 {
		return map[string][]int64{}, nil
	}

	requestQuery := c.recommendRequestRepo.Query(ctx).RecommendRequest
	requestOpts := make([]repo.QueryOption, 0, 1)
	requestOpts = append(requestOpts, repo.Where(requestQuery.RequestID.In(requestIds...)))
	requestList, err := c.recommendRequestRepo.List(ctx, requestOpts...)
	if err != nil {
		return nil, err
	}

	requestIdByRecordId := make(map[int64]string, len(requestList))
	requestRecordIds := make([]int64, 0, len(requestList))
	for _, item := range requestList {
		// 非法请求主记录不参与逐商品明细映射。
		if item == nil || item.ID <= 0 || item.RequestID == "" {
			continue
		}
		requestIdByRecordId[item.ID] = item.RequestID
		requestRecordIds = append(requestRecordIds, item.ID)
	}
	if len(requestRecordIds) == 0 {
		return map[string][]int64{}, nil
	}

	requestItemQuery := c.recommendRequestItemRepo.Query(ctx).RecommendRequestItem
	requestItemOpts := make([]repo.QueryOption, 0, 1)
	requestItemOpts = append(requestItemOpts, repo.Where(requestItemQuery.RecommendRequestID.In(requestRecordIds...)))
	requestItemList, err := c.recommendRequestItemRepo.List(ctx, requestItemOpts...)
	if err != nil {
		return nil, err
	}

	requestGoodsSetMap := make(map[string]map[int64]struct{}, len(requestItemList))
	for _, item := range requestItemList {
		requestId, ok := requestIdByRecordId[item.RecommendRequestID]
		// 逐商品明细无法匹配主请求或商品非法时，直接跳过。
		if !ok || item.GoodsID <= 0 {
			continue
		}
		if _, ok = requestGoodsSetMap[requestId]; !ok {
			requestGoodsSetMap[requestId] = make(map[int64]struct{}, 4)
		}
		requestGoodsSetMap[requestId][item.GoodsID] = struct{}{}
	}

	requestGoodsMap := make(map[string][]int64, len(requestGoodsSetMap))
	for requestId, goodsSet := range requestGoodsSetMap {
		goodsIds := make([]int64, 0, len(goodsSet))
		for goodsId := range goodsSet {
			goodsIds = append(goodsIds, goodsId)
		}
		requestGoodsMap[requestId] = recommendcore.DedupeInt64s(goodsIds)
	}
	return requestGoodsMap, nil
}

// accumulateGoodsRelationEntity 累加单个方向的商品关联实体。
func (c *RecommendGoodsRelationCase) accumulateGoodsRelationEntity(entityMap map[recommendGoodsRelationKey]*models.RecommendGoodsRelation, goodsId, relatedGoodsId int64, eventType common.RecommendGoodsActionType, eventTime time.Time, relationScore float64, windowDays int32) error {
	// 非法商品或自关联商品不生成商品关系。
	if goodsId <= 0 || relatedGoodsId <= 0 || goodsId == relatedGoodsId {
		return nil
	}

	relationType := eventType.String()
	key := recommendGoodsRelationKey{
		goodsId:        goodsId,
		relatedGoodsId: relatedGoodsId,
		relationType:   relationType,
	}
	entity, ok := entityMap[key]
	if !ok {
		entity = &models.RecommendGoodsRelation{
			GoodsID:        goodsId,
			RelatedGoodsID: relatedGoodsId,
			RelationType:   relationType,
			WindowDays:     windowDays,
			CreatedAt:      eventTime,
			UpdatedAt:      eventTime,
		}
		entityMap[key] = entity
	}

	// 调用方没有提供关联分时，回退到当前事件的默认关系权重。
	if relationScore <= 0 {
		relationScore = recommendEvent.RelationWeight(eventType)
	}
	entity.Score += relationScore
	var err error
	entity.Evidence, err = recommendEvent.AddBehaviorSummaryCount(entity.Evidence, eventType, int64(relationScore))
	if err != nil {
		return err
	}
	if eventTime.Before(entity.CreatedAt) {
		entity.CreatedAt = eventTime
	}
	if eventTime.After(entity.UpdatedAt) {
		entity.UpdatedAt = eventTime
	}
	return nil
}

// listRelatedGoodsIds 查询关联商品 ID 列表。
func (c *RecommendGoodsRelationCase) listRelatedGoodsIds(ctx context.Context, goodsIds []int64, limit int64) ([]int64, error) {
	// 商品集合为空或限制数量非法时，直接返回空结果。
	if len(goodsIds) == 0 || limit <= 0 {
		return []int64{}, nil
	}

	query := c.Query(ctx).RecommendGoodsRelation
	opts := make([]repo.QueryOption, 0, 3)
	opts = append(opts, repo.Where(query.GoodsID.In(goodsIds...)))
	opts = append(opts, repo.Order(query.Score.Desc()))
	opts = append(opts, repo.Order(query.UpdatedAt.Desc()))

	list, _, err := c.Page(ctx, 1, limit, opts...)
	if err != nil {
		return nil, err
	}

	relatedGoodsIds := make([]int64, 0, len(list))
	for _, item := range list {
		relatedGoodsIds = append(relatedGoodsIds, item.RelatedGoodsID)
	}
	return recommendcore.DedupeInt64s(relatedGoodsIds), nil
}

// loadRelationScores 加载候选商品的关联商品分数。
func (c *RecommendGoodsRelationCase) loadRelationScores(ctx context.Context, sourceGoodsIds []int64) (map[int64]float64, error) {
	// 源商品为空时，不需要继续查询关联分数。
	if len(sourceGoodsIds) == 0 {
		return map[int64]float64{}, nil
	}

	query := c.Query(ctx).RecommendGoodsRelation
	opts := make([]repo.QueryOption, 0, 2)
	opts = append(opts, repo.Where(query.GoodsID.In(sourceGoodsIds...)))
	opts = append(opts, repo.Where(query.WindowDays.Eq(recommendEvent.AggregateWindowDays)))

	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	scores := make(map[int64]float64, len(list))
	for _, item := range list {
		scores[item.RelatedGoodsID] += item.Score
	}
	return scores, nil
}

// upsertOrderGoodsRelations 累计订单内商品的共购与共支付关系。
func (c *RecommendGoodsRelationCase) upsertOrderGoodsRelations(ctx context.Context, list []*models.RecommendGoodsAction, eventType common.RecommendGoodsActionType, eventTime time.Time) error {
	// 订单商品不足两个时，不生成共购关系。
	if len(list) < 2 {
		return nil
	}
	// 非关联行为不继续写入关联关系。
	if !recommendEvent.IsRelationEvent(eventType) {
		return nil
	}

	for i := 0; i < len(list); i++ {
		leftItem := list[i]
		for j := i + 1; j < len(list); j++ {
			rightItem := list[j]
			relationScore := recommendEvent.NormalizeGoodsNum(leftItem.GoodsNum) + recommendEvent.NormalizeGoodsNum(rightItem.GoodsNum)
			relationErr := c.upsertSingleGoodsRelation(ctx, leftItem.GoodsID, rightItem.GoodsID, eventType, eventTime, relationScore)
			// 任一方向写入失败时，直接终止当前关联关系更新。
			if relationErr != nil {
				return relationErr
			}
			relationErr = c.upsertSingleGoodsRelation(ctx, rightItem.GoodsID, leftItem.GoodsID, eventType, eventTime, relationScore)
			// 反向关系写入失败时，直接终止当前关联关系更新。
			if relationErr != nil {
				return relationErr
			}
		}
	}
	return nil
}

// upsertSingleGoodsRelation 累计单个方向的商品关联强度。
func (c *RecommendGoodsRelationCase) upsertSingleGoodsRelation(ctx context.Context, goodsId, relatedGoodsId int64, eventType common.RecommendGoodsActionType, eventTime time.Time, relationScore float64) error {
	// 商品 ID 非法或同商品关联时，不生成关系记录。
	if goodsId <= 0 || relatedGoodsId <= 0 || goodsId == relatedGoodsId {
		return nil
	}
	// 非关联行为不继续写入关联关系。
	if !recommendEvent.IsRelationEvent(eventType) {
		return nil
	}
	relationType := eventType.String()

	query := c.Query(ctx).RecommendGoodsRelation
	opts := make([]repo.QueryOption, 0, 4)
	opts = append(opts, repo.Where(query.GoodsID.Eq(goodsId)))
	opts = append(opts, repo.Where(query.RelatedGoodsID.Eq(relatedGoodsId)))
	opts = append(opts, repo.Where(query.RelationType.Eq(relationType)))
	opts = append(opts, repo.Where(query.WindowDays.Eq(recommendEvent.AggregateWindowDays)))

	entity, err := c.Find(ctx, opts...)
	// 除记录不存在外的查询异常都应中断聚合，避免覆盖脏数据。
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	evidenceJson := ""
	score := relationScore
	// 调用方没有提供关系分时，回退到关系类型默认权重。
	if score <= 0 {
		score = recommendEvent.RelationWeight(eventType)
	}
	// 已有聚合记录时，在原有得分和证据上继续累加。
	if entity != nil {
		score += entity.Score
		evidenceJson = entity.Evidence
	}
	evidenceJson, err = recommendEvent.AddBehaviorSummaryCount(evidenceJson, eventType, int64(score))
	if err != nil {
		return err
	}

	// 不存在历史记录时，创建新的商品关联聚合数据。
	if entity == nil || entity.ID == 0 {
		return c.Create(ctx, &models.RecommendGoodsRelation{
			GoodsID:        goodsId,
			RelatedGoodsID: relatedGoodsId,
			RelationType:   relationType,
			Score:          score,
			Evidence:       evidenceJson,
			WindowDays:     recommendEvent.AggregateWindowDays,
			CreatedAt:      eventTime,
			UpdatedAt:      eventTime,
		})
	}

	// 命中历史记录时，更新累计分数和关联证据。
	entity.Score = score
	entity.Evidence = evidenceJson
	entity.UpdatedAt = eventTime
	return c.UpdateById(ctx, entity)
}

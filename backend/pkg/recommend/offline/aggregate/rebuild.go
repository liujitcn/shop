package aggregate

import (
	"context"
	"time"

	"shop/api/gen/go/common"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	recommendCore "shop/pkg/recommend/core"
	recommendEvent "shop/pkg/recommend/event"

	"github.com/liujitcn/gorm-kit/repo"
)

// userGoodsPreferenceKey 表示用户商品偏好重建时的聚合键。
type userGoodsPreferenceKey struct {
	userId  int64
	goodsId int64
}

// userCategoryPreferenceKey 表示用户类目偏好重建时的聚合键。
type userCategoryPreferenceKey struct {
	userId     int64
	categoryId int64
}

// goodsRelationKey 表示商品关联重建时的唯一键。
type goodsRelationKey struct {
	goodsId        int64
	relatedGoodsId int64
	relationType   string
}

// orderRelationGroupKey 表示订单级商品关联的分组键。
type orderRelationGroupKey struct {
	requestId string
	eventType int32
}

// RebuildUserGoodsPreferences 根据行为事实重建用户商品偏好快照。
func RebuildUserGoodsPreferences(actionList []*models.RecommendGoodsAction, windowDays int32) ([]*models.RecommendUserGoodsPreference, error) {
	// 当前窗口没有行为数据时，直接返回空快照。
	if len(actionList) == 0 {
		return []*models.RecommendUserGoodsPreference{}, nil
	}

	preferenceMap := make(map[userGoodsPreferenceKey]*models.RecommendUserGoodsPreference)
	for _, item := range actionList {
		// 非法行为明细不参与商品偏好重建。
		if item == nil || item.ActorID <= 0 || item.GoodsID <= 0 {
			continue
		}

		eventType := common.RecommendGoodsActionType(item.EventType)
		// 无法识别的行为类型不参与商品偏好重建。
		if eventType == common.RecommendGoodsActionType_UNKNOWN_RGAT {
			continue
		}

		key := userGoodsPreferenceKey{userId: item.ActorID, goodsId: item.GoodsID}
		entity, ok := preferenceMap[key]
		// 当前用户商品组合首次出现时，先初始化重建实体。
		if !ok {
			entity = &models.RecommendUserGoodsPreference{
				UserID:         item.ActorID,
				GoodsID:        item.GoodsID,
				LastBehaviorAt: item.CreatedAt,
				WindowDays:     windowDays,
				CreatedAt:      item.CreatedAt,
				UpdatedAt:      item.CreatedAt,
			}
			preferenceMap[key] = entity
		}

		entity.Score += recommendEvent.EventWeight(eventType) * recommendEvent.NormalizeGoodsNum(item.GoodsNum)
		var err error
		entity.BehaviorSummary, err = recommendEvent.AddBehaviorSummaryCount(entity.BehaviorSummary, eventType, recommendEvent.NormalizeGoodsCount(item.GoodsNum))
		if err != nil {
			return nil, err
		}
		// 当前行为时间更晚时，刷新最近行为信息。
		if !item.CreatedAt.Before(entity.LastBehaviorAt) {
			entity.LastBehaviorType = eventType.String()
			entity.LastBehaviorAt = item.CreatedAt
		}
		// 当前行为时间更晚时，同步刷新聚合更新时间。
		if item.CreatedAt.After(entity.UpdatedAt) {
			entity.UpdatedAt = item.CreatedAt
		}
	}

	list := make([]*models.RecommendUserGoodsPreference, 0, len(preferenceMap))
	for _, item := range preferenceMap {
		list = append(list, item)
	}
	return list, nil
}

// RebuildUserPreferences 根据行为事实重建用户类目偏好快照。
func RebuildUserPreferences(ctx context.Context, goodsInfoRepo *data.GoodsInfoRepo, actionList []*models.RecommendGoodsAction, windowDays int32) ([]*models.RecommendUserPreference, error) {
	// 当前窗口没有行为数据时，直接返回空快照。
	if len(actionList) == 0 {
		return []*models.RecommendUserPreference{}, nil
	}

	goodsInfoMap, err := loadGoodsInfoMap(ctx, goodsInfoRepo, actionList)
	if err != nil {
		return nil, err
	}

	preferenceMap := make(map[userCategoryPreferenceKey]*models.RecommendUserPreference)
	for _, item := range actionList {
		// 非法行为明细不参与类目偏好重建。
		if item == nil || item.ActorID <= 0 || item.GoodsID <= 0 {
			continue
		}

		goodsInfo, ok := goodsInfoMap[item.GoodsID]
		// 商品缺少类目时，不生成类目偏好。
		if !ok || goodsInfo == nil || goodsInfo.CategoryID <= 0 {
			continue
		}

		eventType := common.RecommendGoodsActionType(item.EventType)
		// 无法识别的行为类型不参与类目偏好重建。
		if eventType == common.RecommendGoodsActionType_UNKNOWN_RGAT {
			continue
		}

		key := userCategoryPreferenceKey{userId: item.ActorID, categoryId: goodsInfo.CategoryID}
		entity, ok := preferenceMap[key]
		// 当前用户类目组合首次出现时，先初始化重建实体。
		if !ok {
			entity = &models.RecommendUserPreference{
				UserID:         item.ActorID,
				PreferenceType: recommendEvent.PreferenceTypeCategory,
				TargetID:       goodsInfo.CategoryID,
				WindowDays:     windowDays,
				CreatedAt:      item.CreatedAt,
				UpdatedAt:      item.CreatedAt,
			}
			preferenceMap[key] = entity
		}

		entity.Score += recommendEvent.EventWeight(eventType) * recommendEvent.NormalizeGoodsNum(item.GoodsNum)
		entity.BehaviorSummary, err = recommendEvent.AddBehaviorSummaryCount(entity.BehaviorSummary, eventType, recommendEvent.NormalizeGoodsCount(item.GoodsNum))
		if err != nil {
			return nil, err
		}
		// 当前行为时间更晚时，同步刷新类目偏好更新时间。
		if item.CreatedAt.After(entity.UpdatedAt) {
			entity.UpdatedAt = item.CreatedAt
		}
	}

	list := make([]*models.RecommendUserPreference, 0, len(preferenceMap))
	for _, item := range preferenceMap {
		list = append(list, item)
	}
	return list, nil
}

// RebuildGoodsRelations 根据行为事实重建商品关联快照。
func RebuildGoodsRelations(ctx context.Context, recommendRequestRepo *data.RecommendRequestRepo, recommendRequestItemRepo *data.RecommendRequestItemRepo, actionList []*models.RecommendGoodsAction, windowDays int32) ([]*models.RecommendGoodsRelation, error) {
	// 当前窗口没有关联行为时，直接返回空快照。
	if len(actionList) == 0 {
		return []*models.RecommendGoodsRelation{}, nil
	}

	requestGoodsMap, err := loadRequestGoodsMap(ctx, recommendRequestRepo, recommendRequestItemRepo, actionList)
	if err != nil {
		return nil, err
	}

	relationMap := make(map[goodsRelationKey]*models.RecommendGoodsRelation)
	orderGroupMap := make(map[orderRelationGroupKey][]*models.RecommendGoodsAction)
	for _, item := range actionList {
		// 非法行为明细不参与商品关联重建。
		if item == nil || item.GoodsID <= 0 {
			continue
		}

		eventType := common.RecommendGoodsActionType(item.EventType)
		// 单商品行为按推荐请求中的共同出现结果累计关联。
		if recommendEvent.IsSingleGoodsEvent(eventType) {
			relatedGoodsIds := requestGoodsMap[item.RequestID]
			for _, relatedGoodsId := range relatedGoodsIds {
				err = accumulateGoodsRelationEntity(relationMap, item.GoodsID, relatedGoodsId, eventType, item.CreatedAt, recommendEvent.NormalizeGoodsNum(item.GoodsNum), windowDays)
				if err != nil {
					return nil, err
				}
			}
			continue
		}

		// 订单级行为按请求编号分组后统一沉淀整单共现关系。
		if item.RequestID != "" {
			key := orderRelationGroupKey{requestId: item.RequestID, eventType: item.EventType}
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
				err = accumulateGoodsRelationEntity(relationMap, leftItem.GoodsID, rightItem.GoodsID, eventType, rightItem.CreatedAt, relationScore, windowDays)
				if err != nil {
					return nil, err
				}
				err = accumulateGoodsRelationEntity(relationMap, rightItem.GoodsID, leftItem.GoodsID, eventType, rightItem.CreatedAt, relationScore, windowDays)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	list := make([]*models.RecommendGoodsRelation, 0, len(relationMap))
	for _, item := range relationMap {
		list = append(list, item)
	}
	return list, nil
}

// ListUserActionFacts 读取用户偏好重建所需的行为事实。
func ListUserActionFacts(ctx context.Context, recommendGoodsActionRepo *data.RecommendGoodsActionRepo, userIds []int64, windowDays int32) ([]*models.RecommendGoodsAction, error) {
	// 没有命中用户集合时，不需要继续查询行为事实。
	if len(userIds) == 0 {
		return []*models.RecommendGoodsAction{}, nil
	}

	endAt := time.Now()
	startAt := endAt.AddDate(0, 0, -int(windowDays))
	query := recommendGoodsActionRepo.Query(ctx).RecommendGoodsAction
	opts := make([]repo.QueryOption, 0, 5)
	opts = append(opts, repo.Where(query.ActorType.Eq(recommendEvent.ActorTypeUser)))
	opts = append(opts, repo.Where(query.ActorID.In(userIds...)))
	opts = append(opts, repo.Where(query.CreatedAt.Gte(startAt)))
	opts = append(opts, repo.Where(query.CreatedAt.Lte(endAt)))
	opts = append(opts, repo.Order(query.CreatedAt.Asc()))
	return recommendGoodsActionRepo.List(ctx, opts...)
}

// ListRelationActionFacts 读取商品关联重建所需的行为事实。
func ListRelationActionFacts(ctx context.Context, recommendGoodsActionRepo *data.RecommendGoodsActionRepo, windowDays int32) ([]*models.RecommendGoodsAction, error) {
	endAt := time.Now()
	startAt := endAt.AddDate(0, 0, -int(windowDays))
	query := recommendGoodsActionRepo.Query(ctx).RecommendGoodsAction
	opts := make([]repo.QueryOption, 0, 5)
	opts = append(opts, repo.Where(query.CreatedAt.Gte(startAt)))
	opts = append(opts, repo.Where(query.CreatedAt.Lte(endAt)))
	opts = append(opts, repo.Where(query.EventType.In(
		int32(common.RecommendGoodsActionType_CLICK),
		int32(common.RecommendGoodsActionType_VIEW),
		int32(common.RecommendGoodsActionType_ORDER_CREATE),
		int32(common.RecommendGoodsActionType_ORDER_PAY),
	)))
	opts = append(opts, repo.Order(query.CreatedAt.Asc()))
	opts = append(opts, repo.Order(query.ID.Asc()))
	return recommendGoodsActionRepo.List(ctx, opts...)
}

// loadGoodsInfoMap 按行为事实补齐商品到类目的映射关系。
func loadGoodsInfoMap(ctx context.Context, goodsInfoRepo *data.GoodsInfoRepo, actionList []*models.RecommendGoodsAction) (map[int64]*models.GoodsInfo, error) {
	goodsIds := make([]int64, 0, len(actionList))
	for _, item := range actionList {
		// 非法商品不参与类目映射补齐。
		if item == nil || item.GoodsID <= 0 {
			continue
		}
		goodsIds = append(goodsIds, item.GoodsID)
	}
	goodsIds = recommendCore.DedupeInt64s(goodsIds)
	// 当前窗口没有有效商品时，不需要继续查询商品信息。
	if len(goodsIds) == 0 {
		return map[int64]*models.GoodsInfo{}, nil
	}

	query := goodsInfoRepo.Query(ctx).GoodsInfo
	opts := make([]repo.QueryOption, 0, 1)
	opts = append(opts, repo.Where(query.ID.In(goodsIds...)))
	goodsList, err := goodsInfoRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	goodsInfoMap := make(map[int64]*models.GoodsInfo, len(goodsList))
	for _, item := range goodsList {
		// 空商品信息不参与类目映射。
		if item == nil || item.ID <= 0 {
			continue
		}
		goodsInfoMap[item.ID] = item
	}
	return goodsInfoMap, nil
}

// loadRequestGoodsMap 按请求编号预加载推荐请求内的商品集合。
func loadRequestGoodsMap(ctx context.Context, recommendRequestRepo *data.RecommendRequestRepo, recommendRequestItemRepo *data.RecommendRequestItemRepo, actionList []*models.RecommendGoodsAction) (map[string][]int64, error) {
	requestIds := make([]string, 0, len(actionList))
	for _, item := range actionList {
		// 只有单商品行为才需要回查推荐请求明细。
		if item == nil || item.RequestID == "" || !recommendEvent.IsSingleGoodsEvent(common.RecommendGoodsActionType(item.EventType)) {
			continue
		}
		requestIds = append(requestIds, item.RequestID)
	}
	requestIds = recommendCore.DedupeStrings(requestIds)
	// 当前窗口没有需要回查的请求编号时，直接返回空映射。
	if len(requestIds) == 0 {
		return map[string][]int64{}, nil
	}

	requestQuery := recommendRequestRepo.Query(ctx).RecommendRequest
	requestOpts := make([]repo.QueryOption, 0, 1)
	requestOpts = append(requestOpts, repo.Where(requestQuery.RequestID.In(requestIds...)))
	requestList, err := recommendRequestRepo.List(ctx, requestOpts...)
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
	// 当前窗口没有有效请求主记录时，直接返回空映射。
	if len(requestRecordIds) == 0 {
		return map[string][]int64{}, nil
	}

	requestItemQuery := recommendRequestItemRepo.Query(ctx).RecommendRequestItem
	requestItemOpts := make([]repo.QueryOption, 0, 1)
	requestItemOpts = append(requestItemOpts, repo.Where(requestItemQuery.RecommendRequestID.In(requestRecordIds...)))
	requestItemList, err := recommendRequestItemRepo.List(ctx, requestItemOpts...)
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
		// 当前请求编号首次出现时，先初始化商品集合。
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
		requestGoodsMap[requestId] = recommendCore.DedupeInt64s(goodsIds)
	}
	return requestGoodsMap, nil
}

// accumulateGoodsRelationEntity 累加单个方向的商品关联实体。
func accumulateGoodsRelationEntity(entityMap map[goodsRelationKey]*models.RecommendGoodsRelation, goodsId, relatedGoodsId int64, eventType common.RecommendGoodsActionType, eventTime time.Time, relationScore float64, windowDays int32) error {
	// 非法商品或自关联商品不生成商品关系。
	if goodsId <= 0 || relatedGoodsId <= 0 || goodsId == relatedGoodsId {
		return nil
	}

	relationType := eventType.String()
	key := goodsRelationKey{goodsId: goodsId, relatedGoodsId: relatedGoodsId, relationType: relationType}
	entity, ok := entityMap[key]
	// 当前方向关系首次出现时，先初始化聚合实体。
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
	evidence, err := recommendEvent.AddBehaviorSummaryCount(entity.Evidence, eventType, int64(relationScore))
	if err != nil {
		return err
	}
	entity.Evidence = evidence
	// 命中更早事件时间时，需要刷新聚合起始时间。
	if eventTime.Before(entity.CreatedAt) {
		entity.CreatedAt = eventTime
	}
	// 命中更晚事件时间时，需要刷新聚合更新时间。
	if eventTime.After(entity.UpdatedAt) {
		entity.UpdatedAt = eventTime
	}
	return nil
}

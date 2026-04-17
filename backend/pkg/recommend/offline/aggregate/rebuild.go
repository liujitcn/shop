package aggregate

import (
	"time"

	"shop/api/gen/go/common"
	"shop/pkg/gen/models"
	recommendEvent "shop/pkg/recommend/event"
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
func RebuildUserPreferences(goodsInfoMap map[int64]*models.GoodsInfo, actionList []*models.RecommendGoodsAction, windowDays int32) ([]*models.RecommendUserPreference, error) {
	// 当前窗口没有行为数据时，直接返回空快照。
	if len(actionList) == 0 {
		return []*models.RecommendUserPreference{}, nil
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
		var err error
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
func RebuildGoodsRelations(actionList []*models.RecommendGoodsAction, requestGoodsMap map[string][]int64, windowDays int32) ([]*models.RecommendGoodsRelation, error) {
	// 当前窗口没有关联行为时，直接返回空快照。
	if len(actionList) == 0 {
		return []*models.RecommendGoodsRelation{}, nil
	}

	relationMap := make(map[goodsRelationKey]*models.RecommendGoodsRelation)
	orderGroupMap := make(map[orderRelationGroupKey][]*models.RecommendGoodsAction)
	for _, item := range actionList {
		// 非法行为明细不参与商品关联重建。
		if item == nil || item.GoodsID <= 0 {
			continue
		}

		eventType := common.RecommendGoodsActionType(item.EventType)
		// 非关联行为不参与商品关系重建。
		if !recommendEvent.IsRelationEvent(eventType) {
			continue
		}

		// 单商品行为按推荐请求中的共同出现结果累计关联。
		if recommendEvent.IsSingleGoodsEvent(eventType) {
			relatedGoodsIds := requestGoodsMap[item.RequestID]
			for _, relatedGoodsId := range relatedGoodsIds {
				err := accumulateGoodsRelationEntity(relationMap, item.GoodsID, relatedGoodsId, eventType, item.CreatedAt, recommendEvent.NormalizeGoodsNum(item.GoodsNum), windowDays)
				if err != nil {
					return nil, err
				}
			}
			continue
		}

		// 订单级行为按请求编号分组后统一沉淀整单共现关系。
		if item.RequestID != "" && recommendEvent.IsOrderGoodsEvent(eventType) {
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
				err := accumulateGoodsRelationEntity(relationMap, leftItem.GoodsID, rightItem.GoodsID, eventType, rightItem.CreatedAt, relationScore, windowDays)
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

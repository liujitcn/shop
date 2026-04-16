package aggregate

import (
	"context"
	"errors"
	"time"

	"shop/api/gen/go/common"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	recommendCore "shop/pkg/recommend/core"
	recommendDomain "shop/pkg/recommend/domain"
	recommendEvent "shop/pkg/recommend/event"

	"github.com/liujitcn/gorm-kit/repo"
	"gorm.io/gorm"
)

// GoodsActionProjector 负责将商品行为事实投影到推荐聚合结果。
type GoodsActionProjector struct {
	recommendUserPreferenceRepo      *data.RecommendUserPreferenceRepo      // 用户类目偏好仓储。
	recommendUserGoodsPreferenceRepo *data.RecommendUserGoodsPreferenceRepo // 用户商品偏好仓储。
	recommendGoodsRelationRepo       *data.RecommendGoodsRelationRepo       // 商品关联仓储。
	recommendRequestRepo             *data.RecommendRequestRepo             // 推荐请求主表仓储。
	recommendRequestItemRepo         *data.RecommendRequestItemRepo         // 推荐请求逐商品明细仓储。
	goodsInfoRepo                    *data.GoodsInfoRepo                    // 商品信息仓储，用于补齐类目归属。
}

// NewGoodsActionProjector 创建商品行为投影器。
func NewGoodsActionProjector(
	recommendUserPreferenceRepo *data.RecommendUserPreferenceRepo,
	recommendUserGoodsPreferenceRepo *data.RecommendUserGoodsPreferenceRepo,
	recommendGoodsRelationRepo *data.RecommendGoodsRelationRepo,
	recommendRequestRepo *data.RecommendRequestRepo,
	recommendRequestItemRepo *data.RecommendRequestItemRepo,
	goodsInfoRepo *data.GoodsInfoRepo,
) *GoodsActionProjector {
	return &GoodsActionProjector{
		recommendUserPreferenceRepo:      recommendUserPreferenceRepo,
		recommendUserGoodsPreferenceRepo: recommendUserGoodsPreferenceRepo,
		recommendGoodsRelationRepo:       recommendGoodsRelationRepo,
		recommendRequestRepo:             recommendRequestRepo,
		recommendRequestItemRepo:         recommendRequestItemRepo,
		goodsInfoRepo:                    goodsInfoRepo,
	}
}

// Project 将行为事实投影到偏好和关系聚合结果。
func (p *GoodsActionProjector) Project(ctx context.Context, event *recommendDomain.GoodsActionProjectionEvent) error {
	// 事件为空时，没有可继续投影的业务内容。
	if event == nil {
		return nil
	}

	// 匿名主体当前阶段只保留事实，不写入用户画像投影。
	if event.ActorType != recommendEvent.ActorTypeUser || event.ActorId <= 0 {
		return nil
	}

	// 无法识别的行为类型不参与后续投影聚合。
	if event.EventType == common.RecommendGoodsActionType_UNKNOWN_RGAT {
		return nil
	}

	isSingleGoodsEvent := recommendEvent.IsSingleGoodsEvent(event.EventType)
	isOrderGoodsEvent := recommendEvent.IsOrderGoodsEvent(event.EventType)
	// 非单商品且非订单级行为，当前阶段仍只保留事实。
	if !isSingleGoodsEvent && !isOrderGoodsEvent {
		return nil
	}

	for _, item := range event.GoodsItems {
		err := p.projectActionItem(ctx, event.ActorId, event.EventType, item)
		if err != nil {
			return err
		}
		// 单商品行为逐条按推荐请求沉淀商品关联。
		if isSingleGoodsEvent {
			err = p.projectSingleGoodsActionRelation(ctx, event.EventType, item)
			if err != nil {
				return err
			}
		}
	}

	// 订单级行为统一按整单商品集合沉淀共现关系。
	if isOrderGoodsEvent {
		return p.upsertOrderGoodsRelations(ctx, event.GoodsItems, event.EventType, event.EventTime)
	}
	return nil
}

// projectActionItem 将单条行为事实投影到用户偏好聚合。
func (p *GoodsActionProjector) projectActionItem(ctx context.Context, userId int64, eventType common.RecommendGoodsActionType, item *models.RecommendGoodsAction) error {
	// 空行为或非法商品编号不参与后续投影。
	if item == nil || item.GoodsID <= 0 {
		return nil
	}

	goodsInfo, err := p.goodsInfoRepo.FindById(ctx, item.GoodsID)
	if err != nil {
		return err
	}

	err = p.upsertUserGoodsPreference(ctx, userId, item.GoodsID, eventType, item.CreatedAt, item.GoodsNum)
	if err != nil {
		return err
	}
	return p.upsertUserCategoryPreference(ctx, userId, goodsInfo.CategoryID, eventType, item.CreatedAt, item.GoodsNum)
}

// projectSingleGoodsActionRelation 将单商品行为投影到同批候选商品关系。
func (p *GoodsActionProjector) projectSingleGoodsActionRelation(ctx context.Context, eventType common.RecommendGoodsActionType, item *models.RecommendGoodsAction) error {
	// 空行为或非法商品编号时，不继续沉淀商品关联。
	if item == nil || item.GoodsID <= 0 {
		return nil
	}
	return p.upsertGoodsRelationByRequest(ctx, eventType, item.RequestID, item.GoodsID, item.GoodsNum, item.CreatedAt)
}

// upsertGoodsRelationByRequest 按同一次推荐请求的共同出现结果累计商品关联度。
func (p *GoodsActionProjector) upsertGoodsRelationByRequest(ctx context.Context, eventType common.RecommendGoodsActionType, requestId string, goodsId, goodsNum int64, eventTime time.Time) error {
	// 请求编号为空时，无法回查同批推荐商品，不做关联聚合。
	if requestId == "" {
		return nil
	}

	relatedGoodsIds, err := p.listRelatedGoodsIdsByRequestId(ctx, requestId, goodsId)
	if err != nil {
		return err
	}
	for _, relatedGoodsId := range relatedGoodsIds {
		err = p.upsertSingleGoodsRelation(ctx, relatedGoodsId, goodsId, eventType, eventTime, recommendEvent.NormalizeGoodsNum(goodsNum))
		if err != nil {
			return err
		}
		err = p.upsertSingleGoodsRelation(ctx, goodsId, relatedGoodsId, eventType, eventTime, recommendEvent.NormalizeGoodsNum(goodsNum))
		if err != nil {
			return err
		}
	}
	return nil
}

// listRelatedGoodsIdsByRequestId 读取推荐请求中与当前商品共同出现的其他商品。
func (p *GoodsActionProjector) listRelatedGoodsIdsByRequestId(ctx context.Context, requestId string, goodsId int64) ([]int64, error) {
	// 请求编号为空时，不需要继续回查逐商品明细。
	if requestId == "" {
		return []int64{}, nil
	}

	recommendRequestQuery := p.recommendRequestRepo.Query(ctx).RecommendRequest
	requestOpts := make([]repo.QueryOption, 0, 1)
	requestOpts = append(requestOpts, repo.Where(recommendRequestQuery.RequestID.Eq(requestId)))
	requestEntity, err := p.recommendRequestRepo.Find(ctx, requestOpts...)
	// 请求主表不存在时，说明当前行为无法回查推荐链路。
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return []int64{}, nil
	}
	if err != nil {
		return nil, err
	}

	recommendRequestItemQuery := p.recommendRequestItemRepo.Query(ctx).RecommendRequestItem
	requestItemOpts := make([]repo.QueryOption, 0, 1)
	requestItemOpts = append(requestItemOpts, repo.Where(recommendRequestItemQuery.RecommendRequestID.Eq(requestEntity.ID)))
	requestItemList, err := p.recommendRequestItemRepo.List(ctx, requestItemOpts...)
	if err != nil {
		return nil, err
	}

	relatedGoodsIds := make([]int64, 0, len(requestItemList))
	for _, item := range requestItemList {
		// 非法商品或当前商品自身都不参与关联商品集合。
		if item.GoodsID <= 0 || item.GoodsID == goodsId {
			continue
		}
		relatedGoodsIds = append(relatedGoodsIds, item.GoodsID)
	}
	return recommendCore.DedupeInt64s(relatedGoodsIds), nil
}

// upsertUserGoodsPreference 累计用户对具体商品的偏好得分。
func (p *GoodsActionProjector) upsertUserGoodsPreference(ctx context.Context, userId, goodsId int64, eventType common.RecommendGoodsActionType, eventTime time.Time, goodsNum int64) error {
	query := p.recommendUserGoodsPreferenceRepo.Query(ctx).RecommendUserGoodsPreference
	opts := make([]repo.QueryOption, 0, 3)
	opts = append(opts, repo.Where(query.UserID.Eq(userId)))
	opts = append(opts, repo.Where(query.GoodsID.Eq(goodsId)))
	opts = append(opts, repo.Where(query.WindowDays.Eq(recommendEvent.AggregateWindowDays)))

	entity, err := p.recommendUserGoodsPreferenceRepo.Find(ctx, opts...)
	// 除记录不存在外的查询异常都应中断聚合，避免覆盖脏数据。
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	summaryJson := ""
	score := recommendEvent.EventWeight(eventType) * recommendEvent.NormalizeGoodsNum(goodsNum)
	// 已有聚合记录时，在原有分数和行为汇总上继续累加。
	if entity != nil {
		score += entity.Score
		summaryJson = entity.BehaviorSummary
	}
	summaryJson, err = recommendEvent.AddBehaviorSummaryCount(summaryJson, eventType, recommendEvent.NormalizeGoodsCount(goodsNum))
	if err != nil {
		return err
	}

	// 不存在历史记录时，创建新的商品偏好聚合数据。
	if entity == nil || entity.ID == 0 {
		return p.recommendUserGoodsPreferenceRepo.Create(ctx, &models.RecommendUserGoodsPreference{
			UserID:           userId,
			GoodsID:          goodsId,
			Score:            score,
			LastBehaviorType: eventType.String(),
			LastBehaviorAt:   eventTime,
			BehaviorSummary:  summaryJson,
			WindowDays:       recommendEvent.AggregateWindowDays,
			CreatedAt:        eventTime,
			UpdatedAt:        eventTime,
		})
	}

	// 命中历史记录时，更新累计分数和最近行为信息。
	entity.Score = score
	entity.LastBehaviorType = eventType.String()
	entity.LastBehaviorAt = eventTime
	entity.BehaviorSummary = summaryJson
	entity.UpdatedAt = eventTime
	return p.recommendUserGoodsPreferenceRepo.UpdateById(ctx, entity)
}

// upsertUserCategoryPreference 累计用户对商品类目的偏好得分。
func (p *GoodsActionProjector) upsertUserCategoryPreference(ctx context.Context, userId, categoryId int64, eventType common.RecommendGoodsActionType, eventTime time.Time, goodsNum int64) error {
	// 类目编号非法时，不产生类目偏好聚合记录。
	if categoryId <= 0 {
		return nil
	}

	query := p.recommendUserPreferenceRepo.Query(ctx).RecommendUserPreference
	opts := make([]repo.QueryOption, 0, 4)
	opts = append(opts, repo.Where(query.UserID.Eq(userId)))
	opts = append(opts, repo.Where(query.PreferenceType.Eq(recommendEvent.PreferenceTypeCategory)))
	opts = append(opts, repo.Where(query.TargetID.Eq(categoryId)))
	opts = append(opts, repo.Where(query.WindowDays.Eq(recommendEvent.AggregateWindowDays)))

	entity, err := p.recommendUserPreferenceRepo.Find(ctx, opts...)
	// 除记录不存在外的查询异常都应中断聚合，避免覆盖脏数据。
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	summaryJson := ""
	score := recommendEvent.EventWeight(eventType) * recommendEvent.NormalizeGoodsNum(goodsNum)
	// 已有聚合记录时，在原有分数和行为汇总上继续累加。
	if entity != nil {
		score += entity.Score
		summaryJson = entity.BehaviorSummary
	}
	summaryJson, err = recommendEvent.AddBehaviorSummaryCount(summaryJson, eventType, recommendEvent.NormalizeGoodsCount(goodsNum))
	if err != nil {
		return err
	}

	// 不存在历史记录时，创建新的类目偏好聚合数据。
	if entity == nil || entity.ID == 0 {
		return p.recommendUserPreferenceRepo.Create(ctx, &models.RecommendUserPreference{
			UserID:          userId,
			PreferenceType:  recommendEvent.PreferenceTypeCategory,
			TargetID:        categoryId,
			Score:           score,
			BehaviorSummary: summaryJson,
			WindowDays:      recommendEvent.AggregateWindowDays,
			CreatedAt:       eventTime,
			UpdatedAt:       eventTime,
		})
	}

	// 命中历史记录时，更新累计分数和行为汇总。
	entity.Score = score
	entity.BehaviorSummary = summaryJson
	entity.UpdatedAt = eventTime
	return p.recommendUserPreferenceRepo.UpdateById(ctx, entity)
}

// upsertOrderGoodsRelations 累计订单内商品的共购与共支付关系。
func (p *GoodsActionProjector) upsertOrderGoodsRelations(ctx context.Context, list []*models.RecommendGoodsAction, eventType common.RecommendGoodsActionType, eventTime time.Time) error {
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
			relationErr := p.upsertSingleGoodsRelation(ctx, leftItem.GoodsID, rightItem.GoodsID, eventType, eventTime, relationScore)
			// 任一方向写入失败时，直接终止当前关联关系更新。
			if relationErr != nil {
				return relationErr
			}
			relationErr = p.upsertSingleGoodsRelation(ctx, rightItem.GoodsID, leftItem.GoodsID, eventType, eventTime, relationScore)
			// 反向关系写入失败时，直接终止当前关联关系更新。
			if relationErr != nil {
				return relationErr
			}
		}
	}
	return nil
}

// upsertSingleGoodsRelation 累计单个方向的商品关联强度。
func (p *GoodsActionProjector) upsertSingleGoodsRelation(ctx context.Context, goodsId, relatedGoodsId int64, eventType common.RecommendGoodsActionType, eventTime time.Time, relationScore float64) error {
	// 商品编号非法或同商品关联时，不生成关系记录。
	if goodsId <= 0 || relatedGoodsId <= 0 || goodsId == relatedGoodsId {
		return nil
	}
	// 非关联行为不继续写入关联关系。
	if !recommendEvent.IsRelationEvent(eventType) {
		return nil
	}
	relationType := eventType.String()

	query := p.recommendGoodsRelationRepo.Query(ctx).RecommendGoodsRelation
	opts := make([]repo.QueryOption, 0, 4)
	opts = append(opts, repo.Where(query.GoodsID.Eq(goodsId)))
	opts = append(opts, repo.Where(query.RelatedGoodsID.Eq(relatedGoodsId)))
	opts = append(opts, repo.Where(query.RelationType.Eq(relationType)))
	opts = append(opts, repo.Where(query.WindowDays.Eq(recommendEvent.AggregateWindowDays)))

	entity, err := p.recommendGoodsRelationRepo.Find(ctx, opts...)
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
		return p.recommendGoodsRelationRepo.Create(ctx, &models.RecommendGoodsRelation{
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
	return p.recommendGoodsRelationRepo.UpdateById(ctx, entity)
}

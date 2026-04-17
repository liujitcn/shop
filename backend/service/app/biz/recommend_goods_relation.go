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
	recommendAggregate "shop/pkg/recommend/offline/aggregate"

	"github.com/liujitcn/gorm-kit/repo"
	"gorm.io/gorm"
)

// RecommendGoodsRelationCase 推荐商品关联业务处理对象。
type RecommendGoodsRelationCase struct {
	*biz.BaseCase
	*data.RecommendGoodsRelationRepo
	recommendGoodsActionRepo *data.RecommendGoodsActionRepo
	recommendRequestItemCase *RecommendRequestItemCase
}

// NewRecommendGoodsRelationCase 创建推荐商品关联业务处理对象。
func NewRecommendGoodsRelationCase(
	baseCase *biz.BaseCase,
	recommendGoodsRelationRepo *data.RecommendGoodsRelationRepo,
	recommendGoodsActionRepo *data.RecommendGoodsActionRepo,
	recommendRequestItemCase *RecommendRequestItemCase,
) *RecommendGoodsRelationCase {
	return &RecommendGoodsRelationCase{
		BaseCase:                   baseCase,
		RecommendGoodsRelationRepo: recommendGoodsRelationRepo,
		recommendGoodsActionRepo:   recommendGoodsActionRepo,
		recommendRequestItemCase:   recommendRequestItemCase,
	}
}

// RebuildRecommendGoodsRelation 重建商品关联聚合。
func (c *RecommendGoodsRelationCase) RebuildRecommendGoodsRelation(ctx context.Context, windowDays int32) error {
	actionList, err := c.listRelationActionFacts(ctx, windowDays)
	if err != nil {
		return err
	}

	relationQuery := c.Query(ctx).RecommendGoodsRelation
	relationOpts := make([]repo.QueryOption, 0, 1)
	relationOpts = append(relationOpts, repo.Where(relationQuery.WindowDays.Eq(windowDays)))
	err = c.Delete(ctx, relationOpts...)
	if err != nil {
		return err
	}

	requestIds := make([]string, 0, len(actionList))
	for _, item := range actionList {
		// 只有单商品行为才需要回查推荐请求明细。
		if item == nil || item.RequestID == "" || !recommendEvent.IsSingleGoodsEvent(common.RecommendGoodsActionType(item.EventType)) {
			continue
		}
		requestIds = append(requestIds, item.RequestID)
	}
	requestGoodsMap, err := c.recommendRequestItemCase.loadRequestGoodsMapByRequestIds(ctx, requestIds)
	if err != nil {
		return err
	}

	list, err := recommendAggregate.RebuildGoodsRelations(actionList, requestGoodsMap, windowDays)
	if err != nil {
		return err
	}
	// 当前窗口没有沉淀出有效关联结果时，直接结束重建。
	if len(list) == 0 {
		return nil
	}
	return c.BatchCreate(ctx, list)
}

// projectSingleGoodsAction 将单商品行为投影到同批候选商品关系。
func (c *RecommendGoodsRelationCase) projectSingleGoodsAction(ctx context.Context, eventType common.RecommendGoodsActionType, item *models.RecommendGoodsAction) error {
	// 空行为或非法商品编号时，不继续沉淀商品关联。
	if item == nil || item.GoodsID <= 0 {
		return nil
	}
	return c.upsertGoodsRelationByRequest(ctx, eventType, item.RequestID, item.GoodsID, item.GoodsNum, item.CreatedAt)
}

// projectOrderGoodsActions 将订单级商品行为投影到共现关系。
func (c *RecommendGoodsRelationCase) projectOrderGoodsActions(ctx context.Context, list []*models.RecommendGoodsAction, eventType common.RecommendGoodsActionType, eventTime time.Time) error {
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
		// 非法行为明细不参与订单级共现关系更新。
		if leftItem == nil || leftItem.GoodsID <= 0 {
			continue
		}
		for j := i + 1; j < len(list); j++ {
			rightItem := list[j]
			// 非法行为明细不参与订单级共现关系更新。
			if rightItem == nil || rightItem.GoodsID <= 0 {
				continue
			}
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

// upsertGoodsRelationByRequest 按同一次推荐请求的共同出现结果累计商品关联度。
func (c *RecommendGoodsRelationCase) upsertGoodsRelationByRequest(ctx context.Context, eventType common.RecommendGoodsActionType, requestId string, goodsId, goodsNum int64, eventTime time.Time) error {
	// 请求编号为空时，无法回查同批推荐商品，不做关联聚合。
	if requestId == "" {
		return nil
	}

	relatedGoodsIds, err := c.recommendRequestItemCase.listRelatedGoodsIdsByRequestId(ctx, requestId, goodsId)
	if err != nil {
		return err
	}
	for _, relatedGoodsId := range relatedGoodsIds {
		err = c.upsertSingleGoodsRelation(ctx, relatedGoodsId, goodsId, eventType, eventTime, recommendEvent.NormalizeGoodsNum(goodsNum))
		if err != nil {
			return err
		}
		err = c.upsertSingleGoodsRelation(ctx, goodsId, relatedGoodsId, eventType, eventTime, recommendEvent.NormalizeGoodsNum(goodsNum))
		if err != nil {
			return err
		}
	}
	return nil
}

// upsertSingleGoodsRelation 累计单个方向的商品关联强度。
func (c *RecommendGoodsRelationCase) upsertSingleGoodsRelation(ctx context.Context, goodsId, relatedGoodsId int64, eventType common.RecommendGoodsActionType, eventTime time.Time, relationScore float64) error {
	// 商品编号非法或同商品关联时，不生成关系记录。
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

// listRelationActionFacts 读取商品关联重建所需的行为事实。
func (c *RecommendGoodsRelationCase) listRelationActionFacts(ctx context.Context, windowDays int32) ([]*models.RecommendGoodsAction, error) {
	endAt := time.Now()
	startAt := endAt.AddDate(0, 0, -int(windowDays))
	query := c.recommendGoodsActionRepo.Query(ctx).RecommendGoodsAction
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
	return c.recommendGoodsActionRepo.List(ctx, opts...)
}

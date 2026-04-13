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
}

// NewRecommendGoodsRelationCase 创建推荐商品关联业务处理对象。
func NewRecommendGoodsRelationCase(baseCase *biz.BaseCase, recommendGoodsRelationRepo *data.RecommendGoodsRelationRepo) *RecommendGoodsRelationCase {
	return &RecommendGoodsRelationCase{
		BaseCase:                   baseCase,
		RecommendGoodsRelationRepo: recommendGoodsRelationRepo,
	}
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

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

// RecommendUserPreferenceCase 推荐用户偏好业务处理对象。
type RecommendUserPreferenceCase struct {
	*biz.BaseCase
	*data.RecommendUserPreferenceRepo
	recommendGoodsActionRepo *data.RecommendGoodsActionRepo
	goodsInfoCase            *GoodsInfoCase
}

// NewRecommendUserPreferenceCase 创建推荐用户偏好业务处理对象。
func NewRecommendUserPreferenceCase(
	baseCase *biz.BaseCase,
	recommendUserPreferenceRepo *data.RecommendUserPreferenceRepo,
	recommendGoodsActionRepo *data.RecommendGoodsActionRepo,
	goodsInfoCase *GoodsInfoCase,
) *RecommendUserPreferenceCase {
	return &RecommendUserPreferenceCase{
		BaseCase:                    baseCase,
		RecommendUserPreferenceRepo: recommendUserPreferenceRepo,
		recommendGoodsActionRepo:    recommendGoodsActionRepo,
		goodsInfoCase:               goodsInfoCase,
	}
}

// RebuildRecommendUserPreference 重建用户类目偏好聚合。
func (c *RecommendUserPreferenceCase) RebuildRecommendUserPreference(ctx context.Context, userIds []int64, windowDays int32) error {
	// 没有命中重建用户时，无需继续重建类目偏好。
	if len(userIds) == 0 {
		return nil
	}

	actionList, err := c.listUserActionFacts(ctx, userIds, windowDays)
	if err != nil {
		return err
	}

	preferenceQuery := c.Query(ctx).RecommendUserPreference
	preferenceOpts := make([]repo.QueryOption, 0, 2)
	preferenceOpts = append(preferenceOpts, repo.Where(preferenceQuery.UserID.In(userIds...)))
	preferenceOpts = append(preferenceOpts, repo.Where(preferenceQuery.WindowDays.Eq(windowDays)))
	err = c.Delete(ctx, preferenceOpts...)
	if err != nil {
		return err
	}

	goodsInfoMap, err := c.loadGoodsInfoMapByActionList(ctx, actionList)
	if err != nil {
		return err
	}
	list, err := recommendAggregate.RebuildUserPreferences(goodsInfoMap, actionList, windowDays)
	if err != nil {
		return err
	}
	// 重建后没有生成有效类目偏好时，直接结束。
	if len(list) == 0 {
		return nil
	}
	return c.BatchCreate(ctx, list)
}

// projectGoodsAction 将单条商品行为投影到用户类目偏好表。
func (c *RecommendUserPreferenceCase) projectGoodsAction(ctx context.Context, userId int64, eventType common.RecommendGoodsActionType, item *models.RecommendGoodsAction) error {
	// 空行为或非法商品编号时，不继续沉淀类目偏好。
	if item == nil || item.GoodsID <= 0 {
		return nil
	}

	goodsInfo, err := c.goodsInfoCase.FindById(ctx, item.GoodsID)
	if err != nil {
		return err
	}
	// 类目编号非法时，不产生类目偏好聚合记录。
	if goodsInfo == nil || goodsInfo.CategoryID <= 0 {
		return nil
	}

	query := c.Query(ctx).RecommendUserPreference
	opts := make([]repo.QueryOption, 0, 4)
	opts = append(opts, repo.Where(query.UserID.Eq(userId)))
	opts = append(opts, repo.Where(query.PreferenceType.Eq(recommendEvent.PreferenceTypeCategory)))
	opts = append(opts, repo.Where(query.TargetID.Eq(goodsInfo.CategoryID)))
	opts = append(opts, repo.Where(query.WindowDays.Eq(recommendEvent.AggregateWindowDays)))

	entity, err := c.Find(ctx, opts...)
	// 除记录不存在外的查询异常都应中断聚合，避免覆盖脏数据。
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	summaryJson := ""
	score := recommendEvent.EventWeight(eventType) * recommendEvent.NormalizeGoodsNum(item.GoodsNum)
	// 已有聚合记录时，在原有分数和行为汇总上继续累加。
	if entity != nil {
		score += entity.Score
		summaryJson = entity.BehaviorSummary
	}
	summaryJson, err = recommendEvent.AddBehaviorSummaryCount(summaryJson, eventType, recommendEvent.NormalizeGoodsCount(item.GoodsNum))
	if err != nil {
		return err
	}

	// 不存在历史记录时，创建新的类目偏好聚合数据。
	if entity == nil || entity.ID == 0 {
		return c.Create(ctx, &models.RecommendUserPreference{
			UserID:          userId,
			PreferenceType:  recommendEvent.PreferenceTypeCategory,
			TargetID:        goodsInfo.CategoryID,
			Score:           score,
			BehaviorSummary: summaryJson,
			WindowDays:      recommendEvent.AggregateWindowDays,
			CreatedAt:       item.CreatedAt,
			UpdatedAt:       item.CreatedAt,
		})
	}

	// 命中历史记录时，更新累计分数和行为汇总。
	entity.Score = score
	entity.BehaviorSummary = summaryJson
	entity.UpdatedAt = item.CreatedAt
	return c.UpdateById(ctx, entity)
}

// listPreferredCategoryIds 查询用户偏好的分类 ID 列表。
func (c *RecommendUserPreferenceCase) listPreferredCategoryIds(ctx context.Context, userId int64, limit int64) ([]int64, error) {
	// 用户 ID 或限制数量非法时，直接返回空结果。
	if userId == 0 || limit <= 0 {
		return []int64{}, nil
	}

	query := c.Query(ctx).RecommendUserPreference
	opts := make([]repo.QueryOption, 0, 4)
	opts = append(opts, repo.Where(query.UserID.Eq(userId)))
	opts = append(opts, repo.Where(query.PreferenceType.Eq(recommendEvent.PreferenceTypeCategory)))
	opts = append(opts, repo.Order(query.Score.Desc()))
	opts = append(opts, repo.Order(query.UpdatedAt.Desc()))

	list, _, err := c.Page(ctx, 1, limit, opts...)
	if err != nil {
		return nil, err
	}

	categoryIds := make([]int64, 0, len(list))
	for _, item := range list {
		categoryIds = append(categoryIds, item.TargetID)
	}
	return categoryIds, nil
}

// loadProfileScores 加载用户类目画像分数。
func (c *RecommendUserPreferenceCase) loadProfileScores(ctx context.Context, userId int64, categoryIds []int64) (map[int64]float64, error) {
	// 用户 ID 或候选类目为空时，不需要继续查询画像分数。
	if userId == 0 || len(categoryIds) == 0 {
		return map[int64]float64{}, nil
	}

	query := c.Query(ctx).RecommendUserPreference
	opts := make([]repo.QueryOption, 0, 4)
	opts = append(opts, repo.Where(query.UserID.Eq(userId)))
	opts = append(opts, repo.Where(query.PreferenceType.Eq(recommendEvent.PreferenceTypeCategory)))
	opts = append(opts, repo.Where(query.TargetID.In(recommendcore.DedupeInt64s(categoryIds)...)))
	opts = append(opts, repo.Where(query.WindowDays.Eq(recommendEvent.AggregateWindowDays)))

	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	scores := make(map[int64]float64, len(list))
	for _, item := range list {
		scores[item.TargetID] = item.Score
	}
	return scores, nil
}

// listUserActionFacts 读取用户类目偏好重建所需的行为事实。
func (c *RecommendUserPreferenceCase) listUserActionFacts(ctx context.Context, userIds []int64, windowDays int32) ([]*models.RecommendGoodsAction, error) {
	// 没有命中用户集合时，不需要继续查询行为事实。
	if len(userIds) == 0 {
		return []*models.RecommendGoodsAction{}, nil
	}

	endAt := time.Now()
	startAt := endAt.AddDate(0, 0, -int(windowDays))
	query := c.recommendGoodsActionRepo.Query(ctx).RecommendGoodsAction
	opts := make([]repo.QueryOption, 0, 5)
	opts = append(opts, repo.Where(query.ActorType.Eq(recommendEvent.ActorTypeUser)))
	opts = append(opts, repo.Where(query.ActorID.In(userIds...)))
	opts = append(opts, repo.Where(query.CreatedAt.Gte(startAt)))
	opts = append(opts, repo.Where(query.CreatedAt.Lte(endAt)))
	opts = append(opts, repo.Order(query.CreatedAt.Asc()))
	return c.recommendGoodsActionRepo.List(ctx, opts...)
}

// loadGoodsInfoMapByActionList 按行为事实补齐商品到类目的映射关系。
func (c *RecommendUserPreferenceCase) loadGoodsInfoMapByActionList(ctx context.Context, actionList []*models.RecommendGoodsAction) (map[int64]*models.GoodsInfo, error) {
	goodsIds := make([]int64, 0, len(actionList))
	for _, item := range actionList {
		// 非法商品不参与类目映射补齐。
		if item == nil || item.GoodsID <= 0 {
			continue
		}
		goodsIds = append(goodsIds, item.GoodsID)
	}
	goodsInfoMap, err := c.goodsInfoCase.mapByGoodsIds(ctx, recommendcore.DedupeInt64s(goodsIds))
	if err != nil {
		return nil, err
	}
	if goodsInfoMap == nil {
		return map[int64]*models.GoodsInfo{}, nil
	}
	return goodsInfoMap, nil
}

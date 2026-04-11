package biz

import (
	"context"
	"errors"
	"shop/api/gen/go/common"
	"time"

	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	recommendcore "shop/pkg/recommend/core"
	recommendEvent "shop/pkg/recommend/event"

	"github.com/liujitcn/gorm-kit/repo"
	"gorm.io/gorm"
)

// RecommendUserPreferenceCase 推荐用户偏好业务处理对象。
type RecommendUserPreferenceCase struct {
	*biz.BaseCase
	*data.RecommendUserPreferenceRepo
}

// NewRecommendUserPreferenceCase 创建推荐用户偏好业务处理对象。
func NewRecommendUserPreferenceCase(baseCase *biz.BaseCase, recommendUserPreferenceRepo *data.RecommendUserPreferenceRepo) *RecommendUserPreferenceCase {
	return &RecommendUserPreferenceCase{
		BaseCase:                    baseCase,
		RecommendUserPreferenceRepo: recommendUserPreferenceRepo,
	}
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
	// 查询用户类目偏好失败时，直接返回错误交由上层处理。
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
	// 查询画像分数失败时，直接返回错误交由上层处理。
	if err != nil {
		return nil, err
	}

	scores := make(map[int64]float64, len(list))
	for _, item := range list {
		scores[item.TargetID] = item.Score
	}
	return scores, nil
}

// upsertUserCategoryPreference 累计用户对商品类目的偏好得分。
func (c *RecommendUserPreferenceCase) upsertUserCategoryPreference(ctx context.Context, userId, categoryId int64, eventType common.RecommendGoodsActionType, eventTime time.Time, goodsNum int64) error {
	// 类目 ID 非法时，不产生类目偏好聚合记录。
	if categoryId <= 0 {
		return nil
	}

	query := c.Query(ctx).RecommendUserPreference
	opts := make([]repo.QueryOption, 0, 4)
	opts = append(opts, repo.Where(query.UserID.Eq(userId)))
	opts = append(opts, repo.Where(query.PreferenceType.Eq(recommendEvent.PreferenceTypeCategory)))
	opts = append(opts, repo.Where(query.TargetID.Eq(categoryId)))
	opts = append(opts, repo.Where(query.WindowDays.Eq(recommendEvent.AggregateWindowDays)))

	entity, err := c.Find(ctx, opts...)
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
	// 行为汇总 JSON 更新失败时，直接返回错误避免写入不一致数据。
	if err != nil {
		return err
	}

	// 不存在历史记录时，创建新的类目偏好聚合数据。
	if entity == nil || entity.ID == 0 {
		return c.Create(ctx, &models.RecommendUserPreference{
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
	return c.UpdateById(ctx, entity)
}

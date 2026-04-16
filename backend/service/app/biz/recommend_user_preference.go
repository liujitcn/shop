package biz

import (
	"context"

	"shop/pkg/biz"
	"shop/pkg/gen/data"
	recommendcore "shop/pkg/recommend/core"
	recommendEvent "shop/pkg/recommend/event"
	recommendAggregate "shop/pkg/recommend/offline/aggregate"

	"github.com/liujitcn/gorm-kit/repo"
)

// RecommendUserPreferenceCase 推荐用户偏好业务处理对象。
type RecommendUserPreferenceCase struct {
	*biz.BaseCase
	*data.RecommendUserPreferenceRepo
	recommendGoodsActionRepo *data.RecommendGoodsActionRepo
	goodsInfoRepo            *data.GoodsInfoRepo
}

// NewRecommendUserPreferenceCase 创建推荐用户偏好业务处理对象。
func NewRecommendUserPreferenceCase(
	baseCase *biz.BaseCase,
	recommendUserPreferenceRepo *data.RecommendUserPreferenceRepo,
	recommendGoodsActionRepo *data.RecommendGoodsActionRepo,
	goodsInfoRepo *data.GoodsInfoRepo,
) *RecommendUserPreferenceCase {
	return &RecommendUserPreferenceCase{
		BaseCase:                    baseCase,
		RecommendUserPreferenceRepo: recommendUserPreferenceRepo,
		recommendGoodsActionRepo:    recommendGoodsActionRepo,
		goodsInfoRepo:               goodsInfoRepo,
	}
}

// RebuildRecommendUserPreference 重建用户类目偏好聚合。
func (c *RecommendUserPreferenceCase) RebuildRecommendUserPreference(ctx context.Context, userIds []int64, windowDays int32) error {
	// 没有命中重建用户时，无需继续重建类目偏好。
	if len(userIds) == 0 {
		return nil
	}

	actionList, err := recommendAggregate.ListUserActionFacts(ctx, c.recommendGoodsActionRepo, userIds, windowDays)
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

	list, err := recommendAggregate.RebuildUserPreferences(ctx, c.goodsInfoRepo, actionList, windowDays)
	if err != nil {
		return err
	}
	// 重建后没有生成有效类目偏好时，直接结束。
	if len(list) == 0 {
		return nil
	}
	return c.BatchCreate(ctx, list)
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

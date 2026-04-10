package biz

import (
	"context"

	"shop/pkg/biz"
	"shop/pkg/gen/data"
	recommendcore "shop/pkg/recommend/core"
	recommendevent "shop/pkg/recommend/event"

	"github.com/liujitcn/gorm-kit/repo"
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
func (c *RecommendUserPreferenceCase) listPreferredCategoryIds(ctx context.Context, userID int64, limit int) ([]int64, error) {
	if userID == 0 || limit <= 0 {
		return []int64{}, nil
	}
	preferenceQuery := c.RecommendUserPreferenceRepo.Query(ctx).RecommendUserPreference
	list, _, err := c.RecommendUserPreferenceRepo.Page(
		ctx,
		1,
		int64(limit),
		repo.Where(preferenceQuery.UserID.Eq(userID)),
		repo.Where(preferenceQuery.PreferenceType.Eq(recommendevent.PreferenceTypeCategory)),
		repo.Order(preferenceQuery.Score.Desc()),
		repo.Order(preferenceQuery.UpdatedAt.Desc()),
	)
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
func (c *RecommendUserPreferenceCase) loadProfileScores(ctx context.Context, userID int64, categoryIds []int64) (map[int64]float64, error) {
	if userID == 0 || len(categoryIds) == 0 {
		return map[int64]float64{}, nil
	}
	preferenceQuery := c.RecommendUserPreferenceRepo.Query(ctx).RecommendUserPreference
	list, err := c.RecommendUserPreferenceRepo.List(ctx,
		repo.Where(preferenceQuery.UserID.Eq(userID)),
		repo.Where(preferenceQuery.PreferenceType.Eq(recommendevent.PreferenceTypeCategory)),
		repo.Where(preferenceQuery.TargetID.In(recommendcore.DedupeInt64s(categoryIds)...)),
		repo.Where(preferenceQuery.WindowDays.Eq(recommendevent.AggregateWindowDays)),
	)
	if err != nil {
		return nil, err
	}

	scores := make(map[int64]float64, len(list))
	for _, item := range list {
		scores[item.TargetID] = item.Score
	}
	return scores, nil
}

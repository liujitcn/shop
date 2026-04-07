package biz

import (
	"context"

	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/gorm-kit/repo"
)

// RecommendProfileCase 推荐画像业务处理对象。
type RecommendProfileCase struct {
	*biz.BaseCase
	*data.RecommendUserPreferenceRepo
}

// NewRecommendProfileCase 创建推荐画像业务处理对象。
func NewRecommendProfileCase(baseCase *biz.BaseCase, recommendUserPreferenceRepo *data.RecommendUserPreferenceRepo) *RecommendProfileCase {
	return &RecommendProfileCase{
		BaseCase:                    baseCase,
		RecommendUserPreferenceRepo: recommendUserPreferenceRepo,
	}
}

// ListPreferredCategoryIds 查询用户偏好的分类ID列表。
func (c *RecommendProfileCase) ListPreferredCategoryIds(ctx context.Context, userID int64, limit int) ([]int64, error) {
	if userID == 0 || limit <= 0 {
		return []int64{}, nil
	}
	preferenceQuery := c.RecommendUserPreferenceRepo.Query(ctx).RecommendUserPreference
	opts := make([]repo.QueryOption, 0, 4)
	opts = append(opts, repo.Where(preferenceQuery.UserID.Eq(userID)))
	opts = append(opts, repo.Where(preferenceQuery.PreferenceType.Eq("category")))
	opts = append(opts, repo.Order(preferenceQuery.Score.Desc()))
	opts = append(opts, repo.Order(preferenceQuery.UpdatedAt.Desc()))
	list, _, err := c.Page(ctx, 1, int64(limit), opts...)
	if err != nil {
		return nil, err
	}
	categoryIds := make([]int64, 0, len(list))
	for _, item := range list {
		categoryIds = append(categoryIds, item.TargetID)
	}
	return categoryIds, nil
}

// GetTopPreference 查询用户画像首选偏好。
func (c *RecommendProfileCase) GetTopPreference(ctx context.Context, userID int64) (*models.RecommendUserPreference, error) {
	if userID == 0 {
		return nil, nil
	}
	preferenceQuery := c.RecommendUserPreferenceRepo.Query(ctx).RecommendUserPreference
	return c.Find(ctx,
		repo.Where(preferenceQuery.UserID.Eq(userID)),
		repo.Order(preferenceQuery.Score.Desc()),
		repo.Order(preferenceQuery.UpdatedAt.Desc()),
	)
}

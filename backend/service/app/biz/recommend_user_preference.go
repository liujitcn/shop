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
	appDto "shop/service/app/dto"

	"github.com/liujitcn/gorm-kit/repo"
	"gorm.io/gorm"
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

	endAt := time.Now()
	startAt := endAt.AddDate(0, 0, -int(windowDays))

	preferenceQuery := c.Query(ctx).RecommendUserPreference
	preferenceOpts := make([]repo.QueryOption, 0, 2)
	preferenceOpts = append(preferenceOpts, repo.Where(preferenceQuery.UserID.In(userIds...)))
	preferenceOpts = append(preferenceOpts, repo.Where(preferenceQuery.WindowDays.Eq(windowDays)))
	err := c.Delete(ctx, preferenceOpts...)
	if err != nil {
		return err
	}

	actionQuery := c.recommendGoodsActionRepo.Query(ctx).RecommendGoodsAction
	actionOpts := make([]repo.QueryOption, 0, 5)
	actionOpts = append(actionOpts, repo.Where(actionQuery.ActorType.Eq(recommendEvent.ActorTypeUser)))
	actionOpts = append(actionOpts, repo.Where(actionQuery.ActorID.In(userIds...)))
	actionOpts = append(actionOpts, repo.Where(actionQuery.CreatedAt.Gte(startAt)))
	actionOpts = append(actionOpts, repo.Where(actionQuery.CreatedAt.Lte(endAt)))
	actionOpts = append(actionOpts, repo.Order(actionQuery.CreatedAt.Asc()))
	var actionList []*models.RecommendGoodsAction
	actionList, err = c.recommendGoodsActionRepo.List(ctx, actionOpts...)
	if err != nil {
		return err
	}
	// 当前窗口没有行为数据时，只保留清理动作。
	if len(actionList) == 0 {
		return nil
	}

	goodsIds := make([]int64, 0, len(actionList))
	for _, item := range actionList {
		// 非法商品不参与类目偏好重建。
		if item == nil || item.GoodsID <= 0 {
			continue
		}
		goodsIds = append(goodsIds, item.GoodsID)
	}
	goodsIds = recommendcore.DedupeInt64s(goodsIds)
	goodsInfoMap := make(map[int64]*models.GoodsInfo, len(goodsIds))
	// 当前窗口存在商品行为时，继续补齐商品到类目的映射关系。
	if len(goodsIds) > 0 {
		goodsQuery := c.goodsInfoRepo.Query(ctx).GoodsInfo
		goodsOpts := make([]repo.QueryOption, 0, 1)
		goodsOpts = append(goodsOpts, repo.Where(goodsQuery.ID.In(goodsIds...)))
		goodsList, listErr := c.goodsInfoRepo.List(ctx, goodsOpts...)
		if listErr != nil {
			return listErr
		}
		for _, item := range goodsList {
			goodsInfoMap[item.ID] = item
		}
	}

	preferenceMap := make(map[appDto.RecommendUserPreferenceKey]*models.RecommendUserPreference)
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

		key := appDto.RecommendUserPreferenceKey{
			UserId:     item.ActorID,
			CategoryId: goodsInfo.CategoryID,
		}
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
			return err
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

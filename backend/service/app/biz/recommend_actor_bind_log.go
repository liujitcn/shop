package biz

import (
	"context"
	"fmt"
	"time"

	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	recommendCore "shop/pkg/recommend/core"
	recommendEvent "shop/pkg/recommend/event"
	appDto "shop/service/app/dto"
)

// RecommendActorBindLogCase 推荐主体绑定日志业务处理对象。
type RecommendActorBindLogCase struct {
	*biz.BaseCase
	data *data.Data
	tx   data.Transaction
	*data.RecommendActorBindLogRepo
	recommendUserPreferenceCase      *RecommendUserPreferenceCase
	recommendUserGoodsPreferenceCase *RecommendUserGoodsPreferenceCase
	recommendGoodsRelationCase       *RecommendGoodsRelationCase
}

// NewRecommendActorBindLogCase 创建推荐主体绑定日志业务处理对象。
func NewRecommendActorBindLogCase(
	baseCase *biz.BaseCase,
	data *data.Data,
	tx data.Transaction,
	recommendActorBindLogRepo *data.RecommendActorBindLogRepo,
	recommendUserPreferenceCase *RecommendUserPreferenceCase,
	recommendUserGoodsPreferenceCase *RecommendUserGoodsPreferenceCase,
	recommendGoodsRelationCase *RecommendGoodsRelationCase,
) *RecommendActorBindLogCase {
	return &RecommendActorBindLogCase{
		BaseCase:                         baseCase,
		data:                             data,
		tx:                               tx,
		RecommendActorBindLogRepo:        recommendActorBindLogRepo,
		recommendUserPreferenceCase:      recommendUserPreferenceCase,
		recommendUserGoodsPreferenceCase: recommendUserGoodsPreferenceCase,
		recommendGoodsRelationCase:       recommendGoodsRelationCase,
	}
}

// SaveRecommendActorBindLog 保存推荐主体绑定日志。
func (c *RecommendActorBindLogCase) SaveRecommendActorBindLog(ctx context.Context, anonymousId, userId int64) error {
	// 绑定双方任一非法时，不写入绑定日志。
	if anonymousId <= 0 || userId <= 0 {
		return nil
	}

	err := c.Create(ctx, &models.RecommendActorBindLog{
		AnonymousID: anonymousId,
		UserID:      userId,
	})
	if err != nil {
		return err
	}
	err = c.RebuildRecommendUserPreference(ctx, []int64{userId}, 0)
	if err != nil {
		return err
	}
	return c.RebuildRecommendGoodsRelation(ctx, []int64{userId}, 0)
}

// RebuildRecommendUserPreference 重建绑定用户的推荐偏好聚合。
func (c *RecommendActorBindLogCase) RebuildRecommendUserPreference(ctx context.Context, userIds []int64, windowDays int32) error {
	var err error
	windowDays, err = normalizeRecommendWindowDays(windowDays)
	if err != nil {
		return err
	}

	var rebuildUserIds []int64
	rebuildUserIds, err = c.resolveBindUserIds(ctx, userIds, windowDays)
	if err != nil {
		return err
	}
	// 当前窗口内没有命中待重建用户时，直接跳过。
	if len(rebuildUserIds) == 0 {
		return nil
	}

	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		err = c.recommendUserGoodsPreferenceCase.RebuildRecommendUserGoodsPreference(ctx, rebuildUserIds, windowDays)
		if err != nil {
			return err
		}
		return c.recommendUserPreferenceCase.RebuildRecommendUserPreference(ctx, rebuildUserIds, windowDays)
	})
}

// RebuildRecommendGoodsRelation 重建推荐商品关联聚合。
func (c *RecommendActorBindLogCase) RebuildRecommendGoodsRelation(ctx context.Context, userIds []int64, windowDays int32) error {
	var err error
	windowDays, err = normalizeRecommendWindowDays(windowDays)
	if err != nil {
		return err
	}

	var rebuildUserIds []int64
	rebuildUserIds, err = c.resolveBindUserIds(ctx, userIds, windowDays)
	if err != nil {
		return err
	}
	// 当前窗口没有绑定用户触发时，不执行商品关联重建。
	if len(rebuildUserIds) == 0 {
		return nil
	}

	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		return c.recommendGoodsRelationCase.RebuildRecommendGoodsRelation(ctx, windowDays)
	})
}

// normalizeRecommendWindowDays 归一化推荐重建窗口参数。
func normalizeRecommendWindowDays(windowDays int32) (int32, error) {
	// 未传窗口参数时，默认回退到固定 30 天窗口。
	if windowDays <= 0 {
		return recommendEvent.AggregateWindowDays, nil
	}
	// 当前推荐链路只支持固定 30 天窗口，避免重建结果与在线查询口径分叉。
	if windowDays != recommendEvent.AggregateWindowDays {
		return 0, errorsx.InvalidArgument(fmt.Sprintf("当前仅支持 %d 天窗口", recommendEvent.AggregateWindowDays))
	}
	return windowDays, nil
}

// resolveBindUserIds 解析本次重建需要处理的用户集合。
func (c *RecommendActorBindLogCase) resolveBindUserIds(ctx context.Context, userIds []int64, windowDays int32) ([]int64, error) {
	// 调用方显式指定用户集合时，优先按指定用户执行重建。
	if len(userIds) > 0 {
		rebuildUserIds := make([]int64, 0, len(userIds))
		for _, userId := range userIds {
			// 非法用户编号不参与重建集合。
			if userId <= 0 {
				continue
			}
			rebuildUserIds = append(rebuildUserIds, userId)
		}
		return recommendCore.DedupeInt64s(rebuildUserIds), nil
	}

	endAt := time.Now()
	startAt := endAt.AddDate(0, 0, -int(windowDays))
	query := c.data.Query(ctx).RecommendActorBindLog
	rows := make([]*appDto.RecommendActorBindLogUserRow, 0)
	err := query.WithContext(ctx).
		Select(query.UserID).
		Where(
			query.CreatedAt.Gte(startAt),
			query.CreatedAt.Lte(endAt),
		).
		Group(query.UserID).
		Scan(&rows)
	if err != nil {
		return nil, err
	}

	rebuildUserIds := make([]int64, 0, len(rows))
	for _, item := range rows {
		// 非法绑定日志不参与重建用户集合。
		if item == nil || item.UserId <= 0 {
			continue
		}
		rebuildUserIds = append(rebuildUserIds, item.UserId)
	}
	return recommendCore.DedupeInt64s(rebuildUserIds), nil
}

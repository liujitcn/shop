package task

import (
	"context"
	"fmt"
	"time"

	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/go-kratos/kratos/v2/log"
)

const (
	recommendRebuildWindowDays   = 30
	recommendActorTypeUser       = int32(1)
	recommendEventTypeClick      = "recommend_click"
	recommendEventTypeView       = "goods_view"
	recommendEventTypeCollect    = "goods_collect"
	recommendEventTypeCart       = "goods_cart"
	recommendEventTypeOrder      = "order_create"
	recommendEventTypePay        = "order_pay"
	recommendPreferenceTypeCat   = "category"
	recommendRelationTypeCoClick = "co_click"
	recommendRelationTypeCoView  = "co_view"
	recommendRelationTypeCoOrder = "co_order"
	recommendRelationTypeCoPay   = "co_pay"
	recommendRebuildArgStartDate = "startDate"
	recommendRebuildArgEndDate   = "endDate"
)

// RecommendUserPreferenceRebuild 推荐用户偏好重建任务。
type RecommendUserPreferenceRebuild struct {
	data *data.Data
	tx   data.Transaction
	ctx  context.Context
}

// NewRecommendUserPreferenceRebuild 创建推荐用户偏好重建任务实例。
func NewRecommendUserPreferenceRebuild(dataStore *data.Data, tx data.Transaction) *RecommendUserPreferenceRebuild {
	return &RecommendUserPreferenceRebuild{
		data: dataStore,
		tx:   tx,
		ctx:  context.Background(),
	}
}

// RecommendGoodsRelationRebuild 推荐商品关联重建任务。
type RecommendGoodsRelationRebuild struct {
	data *data.Data
	tx   data.Transaction
	ctx  context.Context
}

// NewRecommendGoodsRelationRebuild 创建推荐商品关联重建任务实例。
func NewRecommendGoodsRelationRebuild(dataStore *data.Data, tx data.Transaction) *RecommendGoodsRelationRebuild {
	return &RecommendGoodsRelationRebuild{
		data: dataStore,
		tx:   tx,
		ctx:  context.Background(),
	}
}

// Exec 执行推荐用户偏好重建。
func (t *RecommendUserPreferenceRebuild) Exec(args map[string]string) ([]string, error) {
	log.Infof("Job RecommendUserPreferenceRebuild Exec %+v", args)

	startAt, endAt, err := parseRecommendRebuildDateRange(args[recommendRebuildArgStartDate], args[recommendRebuildArgEndDate])
	if err != nil {
		return []string{err.Error()}, err
	}

	err = t.tx.Transaction(t.ctx, func(ctx context.Context) error {
		db := t.data.Query(ctx).RecommendUserPreference.WithContext(ctx).UnderlyingDB()
		if err := db.Exec(
			"DELETE FROM recommend_user_preference WHERE window_days = ?",
			recommendRebuildWindowDays,
		).Error; err != nil {
			return err
		}
		if err := db.Exec(
			"DELETE FROM recommend_user_goods_preference WHERE window_days = ?",
			recommendRebuildWindowDays,
		).Error; err != nil {
			return err
		}

		if err := db.Exec(t.buildUserGoodsPreferenceSQL(), t.userGoodsPreferenceArgs(startAt, endAt)...).Error; err != nil {
			return err
		}
		return db.Exec(t.buildUserCategoryPreferenceSQL(), t.userCategoryPreferenceArgs(startAt, endAt)...).Error
	})
	if err != nil {
		return []string{err.Error()}, err
	}

	return []string{
		fmt.Sprintf(
			"推荐用户偏好重建完成: windowDays=%d range=%s~%s",
			recommendRebuildWindowDays,
			startAt.Format("2006-01-02"),
			endAt.Add(-time.Nanosecond).Format("2006-01-02"),
		),
	}, nil
}

// Exec 执行推荐商品关联重建。
func (t *RecommendGoodsRelationRebuild) Exec(args map[string]string) ([]string, error) {
	log.Infof("Job RecommendGoodsRelationRebuild Exec %+v", args)

	startAt, endAt, err := parseRecommendRebuildDateRange(args[recommendRebuildArgStartDate], args[recommendRebuildArgEndDate])
	if err != nil {
		return []string{err.Error()}, err
	}

	err = t.tx.Transaction(t.ctx, func(ctx context.Context) error {
		db := t.data.Query(ctx).RecommendGoodsRelation.WithContext(ctx).UnderlyingDB()
		if err := db.Exec(
			"DELETE FROM recommend_goods_relation WHERE window_days = ?",
			recommendRebuildWindowDays,
		).Error; err != nil {
			return err
		}
		return db.Exec(t.buildGoodsRelationSQL(), t.goodsRelationArgs(startAt, endAt)...).Error
	})
	if err != nil {
		return []string{err.Error()}, err
	}

	return []string{
		fmt.Sprintf(
			"推荐商品关联重建完成: windowDays=%d range=%s~%s",
			recommendRebuildWindowDays,
			startAt.Format("2006-01-02"),
			endAt.Add(-time.Nanosecond).Format("2006-01-02"),
		),
	}, nil
}

func parseRecommendRebuildDateRange(startDate string, endDate string) (time.Time, time.Time, error) {
	location := time.Now().Location()
	if startDate == "" && endDate == "" {
		end := time.Now()
		end = time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, location).AddDate(0, 0, 1)
		start := end.AddDate(0, 0, -recommendRebuildWindowDays)
		return start, end, nil
	}

	if startDate == "" {
		return time.Time{}, time.Time{}, fmt.Errorf("startDate 不能为空")
	}
	start, err := time.ParseInLocation("2006-01-02", startDate, location)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("startDate 格式错误，应为 2006-01-02")
	}

	if endDate == "" {
		endDate = startDate
	}
	end, err := time.ParseInLocation("2006-01-02", endDate, location)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("endDate 格式错误，应为 2006-01-02")
	}
	if end.Before(start) {
		return time.Time{}, time.Time{}, fmt.Errorf("endDate 不能早于 startDate")
	}

	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, location)
	end = time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, location).AddDate(0, 0, 1)
	windowDays := int(end.Sub(start).Hours() / 24)
	if windowDays != recommendRebuildWindowDays {
		return time.Time{}, time.Time{}, fmt.Errorf("推荐重建任务要求时间窗口固定为 %d 天", recommendRebuildWindowDays)
	}
	return start, end, nil
}

func normalizeGoodsNumExpr(expr string) string {
	return "CASE WHEN " + expr + " <= 0 THEN 1 ELSE " + expr + " END"
}

func (t *RecommendUserPreferenceRebuild) userGoodsPreferenceArgs(startAt time.Time, endAt time.Time) []any {
	return []any{
		recommendRebuildWindowDays,
		recommendGoodsActionTypeClick,
		recommendGoodsActionTypeView,
		recommendGoodsActionTypeCollect,
		recommendGoodsActionTypeCart,
		recommendGoodsActionTypeOrder,
		recommendGoodsActionTypePay,
		recommendGoodsActionTypeClick,
		recommendGoodsActionTypeView,
		recommendGoodsActionTypeCollect,
		recommendGoodsActionTypeCart,
		recommendGoodsActionTypeOrder,
		recommendGoodsActionTypePay,
		startAt, endAt,
		recommendActorTypeUser,
	}
}

func (t *RecommendUserPreferenceRebuild) userCategoryPreferenceArgs(startAt time.Time, endAt time.Time) []any {
	return []any{
		recommendPreferenceTypeCat,
		recommendRebuildWindowDays,
		recommendGoodsActionTypeClick,
		recommendGoodsActionTypeView,
		recommendGoodsActionTypeCollect,
		recommendGoodsActionTypeCart,
		recommendGoodsActionTypeOrder,
		recommendGoodsActionTypePay,
		recommendGoodsActionTypeClick,
		recommendGoodsActionTypeView,
		recommendGoodsActionTypeCollect,
		recommendGoodsActionTypeCart,
		recommendGoodsActionTypeOrder,
		recommendGoodsActionTypePay,
		startAt, endAt,
		recommendActorTypeUser,
	}
}

func (t *RecommendUserPreferenceRebuild) buildUserGoodsPreferenceSQL() string {
	normalizedNum := normalizeGoodsNumExpr("rga.goods_num")
	eventTypeNameExpr := buildRecommendGoodsActionTypeNameExpr("rga.event_type")
	return `
INSERT INTO recommend_user_goods_preference (
  user_id,
  goods_id,
  score,
  last_behavior_type,
  last_behavior_at,
  behavior_summary,
  window_days,
  created_at,
  updated_at
)
SELECT
  agg.user_id,
  agg.goods_id,
  agg.score,
  agg.last_behavior_type,
  agg.last_behavior_at,
  JSON_OBJECT(
    'click_count', agg.click_count,
    'view_count', agg.view_count,
    'collect_count', agg.collect_count,
    'cart_count', agg.cart_count,
    'order_count', agg.order_count,
    'pay_count', agg.pay_count
  ) AS behavior_summary,
  ? AS window_days,
  agg.first_behavior_at AS created_at,
  agg.last_behavior_at AS updated_at
FROM (
  SELECT
    rga.actor_id AS user_id,
    rga.goods_id,
    SUM(
      CASE
        WHEN rga.event_type = ? THEN 3 * ` + normalizedNum + `
        WHEN rga.event_type = ? THEN 2 * ` + normalizedNum + `
        WHEN rga.event_type = ? THEN 4 * ` + normalizedNum + `
        WHEN rga.event_type = ? THEN 6 * ` + normalizedNum + `
        WHEN rga.event_type = ? THEN 8 * ` + normalizedNum + `
        WHEN rga.event_type = ? THEN 10 * ` + normalizedNum + `
        ELSE 0
      END
    ) AS score,
    SUBSTRING_INDEX(GROUP_CONCAT(` + eventTypeNameExpr + ` ORDER BY rga.created_at DESC, rga.id DESC), ',', 1) AS last_behavior_type,
    MAX(rga.created_at) AS last_behavior_at,
    MIN(rga.created_at) AS first_behavior_at,
    SUM(CASE WHEN rga.event_type = ? THEN ` + normalizedNum + ` ELSE 0 END) AS click_count,
    SUM(CASE WHEN rga.event_type = ? THEN ` + normalizedNum + ` ELSE 0 END) AS view_count,
    SUM(CASE WHEN rga.event_type = ? THEN ` + normalizedNum + ` ELSE 0 END) AS collect_count,
    SUM(CASE WHEN rga.event_type = ? THEN ` + normalizedNum + ` ELSE 0 END) AS cart_count,
    SUM(CASE WHEN rga.event_type = ? THEN ` + normalizedNum + ` ELSE 0 END) AS order_count,
    SUM(CASE WHEN rga.event_type = ? THEN ` + normalizedNum + ` ELSE 0 END) AS pay_count
  FROM ` + "`" + models.TableNameRecommendGoodsAction + "`" + ` rga
  WHERE rga.created_at >= ?
    AND rga.created_at < ?
    AND rga.actor_type = ?
  GROUP BY rga.actor_id, rga.goods_id
) agg
WHERE agg.user_id > 0
  AND agg.goods_id > 0
  AND agg.score > 0
`
}

func (t *RecommendUserPreferenceRebuild) buildUserCategoryPreferenceSQL() string {
	normalizedNum := normalizeGoodsNumExpr("rga.goods_num")
	return `
INSERT INTO recommend_user_preference (
  user_id,
  preference_type,
  target_id,
  score,
  behavior_summary,
  window_days,
  created_at,
  updated_at
)
SELECT
  agg.user_id,
  ? AS preference_type,
  agg.category_id AS target_id,
  agg.score,
  JSON_OBJECT(
    'click_count', agg.click_count,
    'view_count', agg.view_count,
    'collect_count', agg.collect_count,
    'cart_count', agg.cart_count,
    'order_count', agg.order_count,
    'pay_count', agg.pay_count
  ) AS behavior_summary,
  ? AS window_days,
  agg.first_behavior_at AS created_at,
  agg.last_behavior_at AS updated_at
FROM (
  SELECT
    rga.actor_id AS user_id,
    gi.category_id,
    SUM(
      CASE
        WHEN rga.event_type = ? THEN 3 * ` + normalizedNum + `
        WHEN rga.event_type = ? THEN 2 * ` + normalizedNum + `
        WHEN rga.event_type = ? THEN 4 * ` + normalizedNum + `
        WHEN rga.event_type = ? THEN 6 * ` + normalizedNum + `
        WHEN rga.event_type = ? THEN 8 * ` + normalizedNum + `
        WHEN rga.event_type = ? THEN 10 * ` + normalizedNum + `
        ELSE 0
      END
    ) AS score,
    MIN(rga.created_at) AS first_behavior_at,
    MAX(rga.created_at) AS last_behavior_at,
    SUM(CASE WHEN rga.event_type = ? THEN ` + normalizedNum + ` ELSE 0 END) AS click_count,
    SUM(CASE WHEN rga.event_type = ? THEN ` + normalizedNum + ` ELSE 0 END) AS view_count,
    SUM(CASE WHEN rga.event_type = ? THEN ` + normalizedNum + ` ELSE 0 END) AS collect_count,
    SUM(CASE WHEN rga.event_type = ? THEN ` + normalizedNum + ` ELSE 0 END) AS cart_count,
    SUM(CASE WHEN rga.event_type = ? THEN ` + normalizedNum + ` ELSE 0 END) AS order_count,
    SUM(CASE WHEN rga.event_type = ? THEN ` + normalizedNum + ` ELSE 0 END) AS pay_count
  FROM ` + "`" + models.TableNameRecommendGoodsAction + "`" + ` rga
  INNER JOIN ` + "`" + models.TableNameGoodsInfo + "`" + ` gi ON gi.id = rga.goods_id
  WHERE rga.created_at >= ?
    AND rga.created_at < ?
    AND rga.actor_type = ?
    AND gi.category_id > 0
  GROUP BY rga.actor_id, gi.category_id
) agg
WHERE agg.user_id > 0
  AND agg.category_id > 0
  AND agg.score > 0
`
}

func (t *RecommendGoodsRelationRebuild) goodsRelationArgs(startAt time.Time, endAt time.Time) []any {
	return []any{
		recommendRebuildWindowDays,
		recommendRelationTypeCoClick,
		startAt, endAt,
		recommendRelationTypeCoView,
		startAt, endAt,
		recommendRelationTypeCoOrder,
		startAt, endAt,
		recommendRelationTypeCoPay,
		startAt, endAt,
	}
}

func (t *RecommendGoodsRelationRebuild) buildGoodsRelationSQL() string {
	leftNum := normalizeGoodsNumExpr("left_og.num")
	rightNum := normalizeGoodsNumExpr("right_og.num")
	return `
INSERT INTO recommend_goods_relation (
  goods_id,
  related_goods_id,
  relation_type,
  score,
  evidence,
  window_days,
  created_at,
  updated_at
)
SELECT
  agg.goods_id,
  agg.related_goods_id,
  agg.relation_type,
  agg.score,
  JSON_OBJECT(agg.relation_type, CAST(agg.score AS SIGNED)) AS evidence,
  ? AS window_days,
  agg.first_seen_at AS created_at,
  agg.last_seen_at AS updated_at
FROM (
  SELECT
    pairs.goods_id,
    pairs.related_goods_id,
    pairs.relation_type,
    SUM(pairs.score) AS score,
    MIN(pairs.event_time) AS first_seen_at,
    MAX(pairs.event_time) AS last_seen_at
  FROM (
    SELECT
      rga.goods_id,
      jt.related_goods_id,
      ? AS relation_type,
      1 AS score,
      rga.created_at AS event_time
    FROM ` + "`" + models.TableNameRecommendGoodsAction + "`" + ` rga
    INNER JOIN ` + "`" + models.TableNameRecommendRequest + "`" + ` rr ON rr.request_id = rga.request_id
    INNER JOIN JSON_TABLE(rr.goods_ids, '$[*]' COLUMNS(related_goods_id BIGINT PATH '$')) jt
    WHERE rga.created_at >= ?
      AND rga.created_at < ?
      AND rga.event_type = ` + fmt.Sprintf("%d", recommendGoodsActionTypeClick) + `
      AND jt.related_goods_id <> rga.goods_id
    UNION ALL
    SELECT
      rga.goods_id,
      jt.related_goods_id,
      ? AS relation_type,
      1 AS score,
      rga.created_at AS event_time
    FROM ` + "`" + models.TableNameRecommendGoodsAction + "`" + ` rga
    INNER JOIN ` + "`" + models.TableNameRecommendRequest + "`" + ` rr ON rr.request_id = rga.request_id
    INNER JOIN JSON_TABLE(rr.goods_ids, '$[*]' COLUMNS(related_goods_id BIGINT PATH '$')) jt
    WHERE rga.created_at >= ?
      AND rga.created_at < ?
      AND rga.event_type = ` + fmt.Sprintf("%d", recommendGoodsActionTypeView) + `
      AND rga.source = 2
      AND jt.related_goods_id <> rga.goods_id
    UNION ALL
    SELECT
      left_og.goods_id,
      right_og.goods_id AS related_goods_id,
      ? AS relation_type,
      (` + leftNum + ` + ` + rightNum + `) AS score,
      oi.created_at AS event_time
    FROM ` + "`" + models.TableNameOrderGoods + "`" + ` left_og
    INNER JOIN ` + "`" + models.TableNameOrderGoods + "`" + ` right_og ON right_og.order_id = left_og.order_id
    INNER JOIN ` + "`" + models.TableNameOrderInfo + "`" + ` oi ON oi.id = left_og.order_id
    WHERE left_og.deleted_at IS NULL
      AND right_og.deleted_at IS NULL
      AND oi.deleted_at IS NULL
      AND oi.created_at >= ?
      AND oi.created_at < ?
      AND left_og.goods_id <> right_og.goods_id
    UNION ALL
    SELECT
      left_og.goods_id,
      right_og.goods_id AS related_goods_id,
      ? AS relation_type,
      (` + leftNum + ` + ` + rightNum + `) AS score,
      op.success_time AS event_time
    FROM ` + "`" + models.TableNameOrderGoods + "`" + ` left_og
    INNER JOIN ` + "`" + models.TableNameOrderGoods + "`" + ` right_og ON right_og.order_id = left_og.order_id
    INNER JOIN ` + "`" + models.TableNameOrderPayment + "`" + ` op ON op.order_id = left_og.order_id
    WHERE left_og.deleted_at IS NULL
      AND right_og.deleted_at IS NULL
      AND op.deleted_at IS NULL
      AND op.trade_state = 'SUCCESS'
      AND op.success_time >= ?
      AND op.success_time < ?
      AND left_og.goods_id <> right_og.goods_id
  ) pairs
  GROUP BY pairs.goods_id, pairs.related_goods_id, pairs.relation_type
) agg
WHERE agg.goods_id > 0
  AND agg.related_goods_id > 0
  AND agg.goods_id <> agg.related_goods_id
  AND agg.score > 0
`
}

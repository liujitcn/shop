package task

import (
	"context"
	"fmt"
	"time"

	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/go-kratos/kratos/v2/log"
)

// GoodsStatDay 商品日统计任务。
type GoodsStatDay struct {
	data *data.Data
	tx   data.Transaction
	ctx  context.Context
}

// NewGoodsStatDay 创建商品日统计任务实例。
func NewGoodsStatDay(dataStore *data.Data, tx data.Transaction) *GoodsStatDay {
	return &GoodsStatDay{
		data: dataStore,
		tx:   tx,
		ctx:  context.Background(),
	}
}

// Exec 执行商品日统计。
func (t *GoodsStatDay) Exec(args map[string]string) ([]string, error) {
	log.Infof("Job GoodsStatDay Exec %+v", args)

	statTime, err := t.parseStatDate(args["statDate"])
	if err != nil {
		return []string{err.Error()}, err
	}

	statDate := time.Date(statTime.Year(), statTime.Month(), statTime.Day(), 0, 0, 0, 0, statTime.Location())
	startAt := statDate
	endAt := statDate.AddDate(0, 0, 1)

	err = t.tx.Transaction(t.ctx, func(ctx context.Context) error {
		db := t.data.Query(ctx).GoodsStatDay.WithContext(ctx).UnderlyingDB()
		err := db.Exec("DELETE FROM goods_stat_day WHERE stat_date = ?", statDate).Error
		if err != nil {
			return err
		}

		sql := `
INSERT INTO goods_stat_day (
  stat_date,
  goods_id,
  view_count,
  collect_count,
  cart_count,
  order_count,
  pay_count,
  pay_goods_num,
  pay_amount,
  score
)
SELECT
  ? AS stat_date,
  dim.goods_id,
  COALESCE(view_stat.view_count, 0) AS view_count,
  COALESCE(collect_stat.collect_count, 0) AS collect_count,
  COALESCE(cart_stat.cart_count, 0) AS cart_count,
  COALESCE(order_stat.order_count, 0) AS order_count,
  COALESCE(pay_stat.pay_count, 0) AS pay_count,
  COALESCE(pay_stat.pay_goods_num, 0) AS pay_goods_num,
  COALESCE(pay_stat.pay_amount, 0) AS pay_amount,
  (
    COALESCE(view_stat.view_count, 0) * 1.0 +
    COALESCE(collect_stat.collect_count, 0) * 3.0 +
    COALESCE(cart_stat.cart_count, 0) * 4.0 +
    COALESCE(order_stat.order_count, 0) * 6.0 +
    COALESCE(pay_stat.pay_count, 0) * 8.0 +
    COALESCE(pay_stat.pay_goods_num, 0) * 1.0 +
    COALESCE(pay_stat.pay_amount, 0) / 10000.0
  ) AS score
FROM (
  SELECT goods_id
  FROM ` + "`" + models.TableNameRecommendGoodsAction + "`" + `
  WHERE created_at >= ? AND created_at < ?
    AND event_type = 'goods_view'
  UNION
  SELECT goods_id
  FROM user_collect
  WHERE deleted_at IS NULL
    AND created_at >= ?
    AND created_at < ?
  UNION
  SELECT goods_id
  FROM user_cart
  WHERE deleted_at IS NULL
    AND created_at >= ?
    AND created_at < ?
  UNION
  SELECT og.goods_id
  FROM ` + "`" + models.TableNameOrderGoods + "`" + ` og
  INNER JOIN ` + "`" + models.TableNameOrderInfo + "`" + ` oi ON oi.id = og.order_id
  WHERE og.deleted_at IS NULL
    AND oi.deleted_at IS NULL
    AND oi.created_at >= ?
    AND oi.created_at < ?
  UNION
  SELECT og.goods_id
  FROM ` + "`" + models.TableNameOrderGoods + "`" + ` og
  INNER JOIN order_payment op ON op.order_id = og.order_id
  WHERE og.deleted_at IS NULL
    AND op.deleted_at IS NULL
    AND op.trade_state = 'SUCCESS'
    AND op.success_time >= ?
    AND op.success_time < ?
) dim
LEFT JOIN (
  SELECT goods_id, COUNT(*) AS view_count
  FROM ` + "`" + models.TableNameRecommendGoodsAction + "`" + `
  WHERE created_at >= ?
    AND created_at < ?
    AND event_type = 'goods_view'
  GROUP BY goods_id
) view_stat ON view_stat.goods_id = dim.goods_id
LEFT JOIN (
  SELECT goods_id, COUNT(*) AS collect_count
  FROM user_collect
  WHERE deleted_at IS NULL
    AND created_at >= ?
    AND created_at < ?
  GROUP BY goods_id
) collect_stat ON collect_stat.goods_id = dim.goods_id
LEFT JOIN (
  SELECT goods_id, COALESCE(SUM(num), 0) AS cart_count
  FROM user_cart
  WHERE deleted_at IS NULL
    AND created_at >= ?
    AND created_at < ?
  GROUP BY goods_id
) cart_stat ON cart_stat.goods_id = dim.goods_id
LEFT JOIN (
  SELECT og.goods_id, COUNT(DISTINCT og.order_id) AS order_count
  FROM ` + "`" + models.TableNameOrderGoods + "`" + ` og
  INNER JOIN ` + "`" + models.TableNameOrderInfo + "`" + ` oi ON oi.id = og.order_id
  WHERE og.deleted_at IS NULL
    AND oi.deleted_at IS NULL
    AND oi.created_at >= ?
    AND oi.created_at < ?
  GROUP BY og.goods_id
) order_stat ON order_stat.goods_id = dim.goods_id
LEFT JOIN (
  SELECT
    og.goods_id,
    COUNT(DISTINCT og.order_id) AS pay_count,
    COALESCE(SUM(og.num), 0) AS pay_goods_num,
    COALESCE(SUM(og.total_pay_price), 0) AS pay_amount
  FROM ` + "`" + models.TableNameOrderGoods + "`" + ` og
  INNER JOIN order_payment op ON op.order_id = og.order_id
  WHERE og.deleted_at IS NULL
    AND op.deleted_at IS NULL
    AND op.trade_state = 'SUCCESS'
    AND op.success_time >= ?
    AND op.success_time < ?
  GROUP BY og.goods_id
) pay_stat ON pay_stat.goods_id = dim.goods_id
ORDER BY dim.goods_id ASC
`
		args := []any{
			statDate,
			startAt, endAt,
			startAt, endAt,
			startAt, endAt,
			startAt, endAt,
			startAt, endAt,
			startAt, endAt,
			startAt, endAt,
			startAt, endAt,
			startAt, endAt,
			startAt, endAt,
		}
		return db.Exec(sql, args...).Error
	})
	if err != nil {
		return []string{err.Error()}, err
	}

	return []string{fmt.Sprintf("商品日统计完成: %s", statDate.Format("2006-01-02"))}, nil
}

// parseStatDate 解析统计日期。
func (t *GoodsStatDay) parseStatDate(value string) (time.Time, error) {
	if value == "" {
		now := time.Now().AddDate(0, 0, -1)
		return now, nil
	}

	statTime, err := time.ParseInLocation("2006-01-02", value, time.Now().Location())
	if err != nil {
		return time.Time{}, fmt.Errorf("statDate 格式错误，应为 2006-01-02")
	}
	return statTime, nil
}

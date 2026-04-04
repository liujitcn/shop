package task

import (
	"context"
	"fmt"
	"time"

	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/go-kratos/kratos/v2/log"
)

// OrderStatDay 订单日汇总任务
type OrderStatDay struct {
	data *data.Data
	tx   data.Transaction
	ctx  context.Context
}

// NewOrderStatDay 创建订单日汇总任务实例
func NewOrderStatDay(dataStore *data.Data, tx data.Transaction) *OrderStatDay {
	return &OrderStatDay{
		data: dataStore,
		tx:   tx,
		ctx:  context.Background(),
	}
}

// Exec 执行订单日汇总
func (t *OrderStatDay) Exec(args map[string]string) ([]string, error) {
	log.Infof("Job OrderStatDay Exec %+v", args)

	statTime, err := t.parseStatDate(args["statDate"])
	if err != nil {
		return []string{err.Error()}, err
	}

	statDate := time.Date(statTime.Year(), statTime.Month(), statTime.Day(), 0, 0, 0, 0, statTime.Location())
	startAt := statDate
	endAt := statDate.AddDate(0, 0, 1)

	err = t.tx.Transaction(t.ctx, func(ctx context.Context) error {
		db := t.data.Query(ctx).OrderStatDay.WithContext(ctx).UnderlyingDB()
		if execErr := db.Exec("DELETE FROM order_stat_day WHERE stat_date = ?", statDate).Error; execErr != nil {
			return execErr
		}

		sql := `
INSERT INTO order_stat_day (
  stat_date,
  pay_type,
  pay_channel,
  paid_order_count,
  paid_order_amount,
  paid_user_count,
  goods_count,
  refund_order_count,
  refund_order_amount,
  canceled_order_count,
  canceled_order_amount
)
SELECT
  ? AS stat_date,
  dim.pay_type,
  dim.pay_channel,
  COALESCE(pay.paid_order_count, 0) AS paid_order_count,
  COALESCE(pay.paid_order_amount, 0) AS paid_order_amount,
  COALESCE(pay.paid_user_count, 0) AS paid_user_count,
  COALESCE(pay.goods_count, 0) AS goods_count,
  COALESCE(refund.refund_order_count, 0) AS refund_order_count,
  COALESCE(refund.refund_order_amount, 0) AS refund_order_amount,
  COALESCE(cancel.canceled_order_count, 0) AS canceled_order_count,
  COALESCE(cancel.canceled_order_amount, 0) AS canceled_order_amount
FROM (
  SELECT DISTINCT o.pay_type, o.pay_channel
  FROM order_payment op
  INNER JOIN ` + "`" + models.TableNameOrderInfo + "`" + ` o ON o.id = op.order_id
  WHERE op.deleted_at IS NULL
    AND o.deleted_at IS NULL
    AND op.trade_state = 'SUCCESS'
    AND op.success_time >= ?
    AND op.success_time < ?
  UNION
  SELECT DISTINCT o.pay_type, o.pay_channel
  FROM order_refund orf
  INNER JOIN ` + "`" + models.TableNameOrderInfo + "`" + ` o ON o.id = orf.order_id
  WHERE orf.deleted_at IS NULL
    AND o.deleted_at IS NULL
    AND orf.refund_state = 'SUCCESS'
    AND orf.success_time >= ?
    AND orf.success_time < ?
  UNION
  SELECT DISTINCT o.pay_type, o.pay_channel
  FROM order_cancel oc
  INNER JOIN ` + "`" + models.TableNameOrderInfo + "`" + ` o ON o.id = oc.order_id
  WHERE oc.deleted_at IS NULL
    AND o.deleted_at IS NULL
    AND oc.created_at >= ?
    AND oc.created_at < ?
) dim
LEFT JOIN (
  SELECT
    o.pay_type,
    o.pay_channel,
    COUNT(*) AS paid_order_count,
    COALESCE(SUM(o.pay_money), 0) AS paid_order_amount,
    COUNT(DISTINCT o.user_id) AS paid_user_count,
    COALESCE(SUM(COALESCE(og.goods_count, 0)), 0) AS goods_count
  FROM order_payment op
  INNER JOIN ` + "`" + models.TableNameOrderInfo + "`" + ` o ON o.id = op.order_id
  LEFT JOIN (
    SELECT order_id, SUM(num) AS goods_count
    FROM order_goods
    WHERE deleted_at IS NULL
    GROUP BY order_id
  ) og ON og.order_id = o.id
  WHERE op.deleted_at IS NULL
    AND o.deleted_at IS NULL
    AND op.trade_state = 'SUCCESS'
    AND op.success_time >= ?
    AND op.success_time < ?
  GROUP BY o.pay_type, o.pay_channel
) pay ON pay.pay_type = dim.pay_type AND pay.pay_channel = dim.pay_channel
LEFT JOIN (
  SELECT
    o.pay_type,
    o.pay_channel,
    COUNT(*) AS refund_order_count,
    COALESCE(SUM(CAST(JSON_UNQUOTE(JSON_EXTRACT(orf.amount, '$.payer_refund')) AS SIGNED)), 0) AS refund_order_amount
  FROM order_refund orf
  INNER JOIN ` + "`" + models.TableNameOrderInfo + "`" + ` o ON o.id = orf.order_id
  WHERE orf.deleted_at IS NULL
    AND o.deleted_at IS NULL
    AND orf.refund_state = 'SUCCESS'
    AND orf.success_time >= ?
    AND orf.success_time < ?
  GROUP BY o.pay_type, o.pay_channel
) refund ON refund.pay_type = dim.pay_type AND refund.pay_channel = dim.pay_channel
LEFT JOIN (
  SELECT
    o.pay_type,
    o.pay_channel,
    COUNT(*) AS canceled_order_count,
    COALESCE(SUM(o.total_money), 0) AS canceled_order_amount
  FROM order_cancel oc
  INNER JOIN ` + "`" + models.TableNameOrderInfo + "`" + ` o ON o.id = oc.order_id
  WHERE oc.deleted_at IS NULL
    AND o.deleted_at IS NULL
    AND oc.created_at >= ?
    AND oc.created_at < ?
  GROUP BY o.pay_type, o.pay_channel
) cancel ON cancel.pay_type = dim.pay_type AND cancel.pay_channel = dim.pay_channel
ORDER BY dim.pay_type ASC, dim.pay_channel ASC
`
		args := []any{
			statDate,
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

	return []string{fmt.Sprintf("订单日汇总完成: %s", statDate.Format("2006-01-02"))}, nil
}

func (t *OrderStatDay) parseStatDate(value string) (time.Time, error) {
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

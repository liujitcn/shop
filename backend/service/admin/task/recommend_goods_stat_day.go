package task

import (
	"context"
	"fmt"
	"time"

	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/go-kratos/kratos/v2/log"
)

// RecommendGoodsStatDay 推荐商品日统计任务。
type RecommendGoodsStatDay struct {
	data *data.Data
	tx   data.Transaction
	ctx  context.Context
}

// NewRecommendGoodsStatDay 创建推荐商品日统计任务实例。
func NewRecommendGoodsStatDay(dataStore *data.Data, tx data.Transaction) *RecommendGoodsStatDay {
	return &RecommendGoodsStatDay{
		data: dataStore,
		tx:   tx,
		ctx:  context.Background(),
	}
}

// Exec 执行推荐商品日统计。
func (t *RecommendGoodsStatDay) Exec(args map[string]string) ([]string, error) {
	log.Infof("Job RecommendGoodsStatDay Exec %+v", args)

	statTime, err := t.parseStatDate(args["statDate"])
	if err != nil {
		return []string{err.Error()}, err
	}

	statDate := time.Date(statTime.Year(), statTime.Month(), statTime.Day(), 0, 0, 0, 0, statTime.Location())
	startAt := statDate
	endAt := statDate.AddDate(0, 0, 1)

	err = t.tx.Transaction(t.ctx, func(ctx context.Context) error {
		db := t.data.Query(ctx).RecommendGoodsStatDay.WithContext(ctx).UnderlyingDB()
		err := db.Exec("DELETE FROM recommend_goods_stat_day WHERE stat_date = ?", statDate).Error
		if err != nil {
			return err
		}

		sql := `
INSERT INTO recommend_goods_stat_day (
  stat_date,
  scene,
  goods_id,
  request_count,
  exposure_count,
  click_count,
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
  dim.scene,
  dim.goods_id,
  COALESCE(request_stat.request_count, 0) AS request_count,
  COALESCE(exposure_stat.exposure_count, 0) AS exposure_count,
  COALESCE(click_stat.click_count, 0) AS click_count,
  COALESCE(action_stat.view_count, 0) AS view_count,
  COALESCE(action_stat.collect_count, 0) AS collect_count,
  COALESCE(action_stat.cart_count, 0) AS cart_count,
  COALESCE(action_stat.order_count, 0) AS order_count,
  COALESCE(action_stat.pay_count, 0) AS pay_count,
  COALESCE(action_stat.pay_goods_num, 0) AS pay_goods_num,
  COALESCE(action_stat.pay_amount, 0) AS pay_amount,
  (
    COALESCE(exposure_stat.exposure_count, 0) * 0.5 +
    COALESCE(click_stat.click_count, 0) * 2.0 +
    COALESCE(action_stat.view_count, 0) * 2.0 +
    COALESCE(action_stat.collect_count, 0) * 4.0 +
    COALESCE(action_stat.cart_count, 0) * 6.0 +
    COALESCE(action_stat.order_count, 0) * 8.0 +
    COALESCE(action_stat.pay_count, 0) * 10.0 +
    COALESCE(action_stat.pay_goods_num, 0) * 1.0 +
    COALESCE(action_stat.pay_amount, 0) / 10000.0
  ) AS score
FROM (
  SELECT DISTINCT rr.scene, jt.goods_id
  FROM ` + "`" + models.TableNameRecommendRequest + "`" + ` rr,
       JSON_TABLE(rr.goods_ids, '$[*]' COLUMNS(goods_id BIGINT PATH '$')) jt
  WHERE rr.created_at >= ?
    AND rr.created_at < ?
  UNION
  SELECT DISTINCT re.scene, jt.goods_id
  FROM ` + "`" + models.TableNameRecommendExposure + "`" + ` re,
       JSON_TABLE(re.goods_ids, '$[*]' COLUMNS(goods_id BIGINT PATH '$')) jt
  WHERE re.created_at >= ?
    AND re.created_at < ?
  UNION
  SELECT DISTINCT rga.scene, rga.goods_id
  FROM ` + "`" + models.TableNameRecommendGoodsAction + "`" + ` rga
  WHERE rga.created_at >= ?
    AND rga.created_at < ?
) dim
LEFT JOIN (
  SELECT rr.scene, jt.goods_id, COUNT(*) AS request_count
  FROM ` + "`" + models.TableNameRecommendRequest + "`" + ` rr,
       JSON_TABLE(rr.goods_ids, '$[*]' COLUMNS(goods_id BIGINT PATH '$')) jt
  WHERE rr.created_at >= ?
    AND rr.created_at < ?
  GROUP BY rr.scene, jt.goods_id
) request_stat ON request_stat.scene = dim.scene AND request_stat.goods_id = dim.goods_id
LEFT JOIN (
  SELECT re.scene, jt.goods_id, COUNT(*) AS exposure_count
  FROM ` + "`" + models.TableNameRecommendExposure + "`" + ` re,
       JSON_TABLE(re.goods_ids, '$[*]' COLUMNS(goods_id BIGINT PATH '$')) jt
  WHERE re.created_at >= ?
    AND re.created_at < ?
  GROUP BY re.scene, jt.goods_id
) exposure_stat ON exposure_stat.scene = dim.scene AND exposure_stat.goods_id = dim.goods_id
LEFT JOIN (
  SELECT scene, goods_id, COUNT(*) AS click_count
  FROM ` + "`" + models.TableNameRecommendGoodsAction + "`" + `
  WHERE created_at >= ?
    AND created_at < ?
    AND event_type = 'recommend_click'
  GROUP BY scene, goods_id
) click_stat ON click_stat.scene = dim.scene AND click_stat.goods_id = dim.goods_id
LEFT JOIN (
  SELECT
    scene,
    goods_id,
    SUM(CASE WHEN event_type = 'goods_view' THEN 1 ELSE 0 END) AS view_count,
    SUM(CASE WHEN event_type = 'goods_collect' THEN 1 ELSE 0 END) AS collect_count,
    SUM(CASE WHEN event_type = 'goods_cart' THEN goods_num ELSE 0 END) AS cart_count,
    SUM(CASE WHEN event_type = 'order_create' THEN 1 ELSE 0 END) AS order_count,
    SUM(CASE WHEN event_type = 'order_pay' THEN 1 ELSE 0 END) AS pay_count,
    SUM(CASE WHEN event_type = 'order_pay' THEN goods_num ELSE 0 END) AS pay_goods_num,
    SUM(CASE WHEN event_type = 'order_pay' THEN pay_info.total_pay_amount ELSE 0 END) AS pay_amount
  FROM ` + "`" + models.TableNameRecommendGoodsAction + "`" + ` rga
  LEFT JOIN (
    SELECT request_id, goods_id, SUM(total_pay_price) AS total_pay_amount
    FROM ` + "`" + models.TableNameOrderGoods + "`" + `
    WHERE deleted_at IS NULL
    GROUP BY request_id, goods_id
  ) pay_info ON pay_info.request_id = rga.request_id AND pay_info.goods_id = rga.goods_id
  WHERE rga.created_at >= ?
    AND rga.created_at < ?
  GROUP BY scene, goods_id
) action_stat ON action_stat.scene = dim.scene AND action_stat.goods_id = dim.goods_id
ORDER BY dim.scene ASC, dim.goods_id ASC
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
		}
		return db.Exec(sql, args...).Error
	})
	if err != nil {
		return []string{err.Error()}, err
	}

	return []string{fmt.Sprintf("推荐商品日统计完成: %s", statDate.Format("2006-01-02"))}, nil
}

// parseStatDate 解析统计日期。
func (t *RecommendGoodsStatDay) parseStatDate(value string) (time.Time, error) {
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

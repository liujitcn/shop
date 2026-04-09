package task

import (
	"context"
	"fmt"
	"strings"
	"time"

	"shop/api/gen/go/common"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	appBiz "shop/service/app/biz"

	"github.com/go-kratos/kratos/v2/log"
)

// RecommendEvalReport 推荐离线评估报告任务。
type RecommendEvalReport struct {
	data *data.Data
	ctx  context.Context
}

const (
	recommendEvalArgStartDate = "startDate"
	recommendEvalArgEndDate   = "endDate"
	recommendEvalArgScene     = "scene"
)

// NewRecommendEvalReport 创建推荐离线评估报告任务实例。
func NewRecommendEvalReport(dataStore *data.Data) *RecommendEvalReport {
	return &RecommendEvalReport{
		data: dataStore,
		ctx:  context.Background(),
	}
}

type recommendEvalSummaryRow struct {
	Scene               int32   `gorm:"column:scene"`
	RequestCount        int64   `gorm:"column:request_count"`
	ExposedRequestCount int64   `gorm:"column:exposed_request_count"`
	ExposedGoodsCount   int64   `gorm:"column:exposed_goods_count"`
	ClickRequestCount   int64   `gorm:"column:click_request_count"`
	ClickCount          int64   `gorm:"column:click_count"`
	PayRequestCount     int64   `gorm:"column:pay_request_count"`
	PayCount            int64   `gorm:"column:pay_count"`
	RequestCTR          float64 `gorm:"column:request_ctr"`
	ExposureCTR         float64 `gorm:"column:exposure_ctr"`
	RequestPayCVR       float64 `gorm:"column:request_pay_cvr"`
	ClickPayCVR         float64 `gorm:"column:click_pay_cvr"`
}

type recommendEvalSourceRow struct {
	Scene         int32   `gorm:"column:scene"`
	RecallSource  string  `gorm:"column:recall_source"`
	RequestCount  int64   `gorm:"column:request_count"`
	ClickReqCount int64   `gorm:"column:click_request_count"`
	PayReqCount   int64   `gorm:"column:pay_request_count"`
	RequestCTR    float64 `gorm:"column:request_ctr"`
	RequestPayCVR float64 `gorm:"column:request_pay_cvr"`
	ClickToPayCVR float64 `gorm:"column:click_to_pay_cvr"`
}

// Exec 执行推荐离线评估报告。
func (t *RecommendEvalReport) Exec(args map[string]string) ([]string, error) {
	log.Infof("Job RecommendEvalReport Exec %+v", args)

	startAt, endAt, err := t.parseDateRange(args[recommendEvalArgStartDate], args[recommendEvalArgEndDate])
	if err != nil {
		return []string{err.Error()}, err
	}

	scene := strings.TrimSpace(args[recommendEvalArgScene])
	sceneClause, queryArgs := t.buildSceneFilter(startAt, endAt, scene)

	db := t.data.Query(t.ctx).RecommendRequest.WithContext(t.ctx).UnderlyingDB()

	var summaryRows []*recommendEvalSummaryRow
	if err = db.Raw(t.buildSummarySQL(sceneClause), queryArgs...).Scan(&summaryRows).Error; err != nil {
		return []string{err.Error()}, err
	}

	var sourceRows []*recommendEvalSourceRow
	if err = db.Raw(t.buildSourceSQL(sceneClause), queryArgs...).Scan(&sourceRows).Error; err != nil {
		return []string{err.Error()}, err
	}

	return t.buildResult(startAt, endAt, scene, summaryRows, sourceRows), nil
}

func (t *RecommendEvalReport) parseDateRange(startDate string, endDate string) (time.Time, time.Time, error) {
	location := time.Now().Location()
	if startDate == "" && endDate == "" {
		start := time.Now().AddDate(0, 0, -1)
		start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, location)
		return start, start.AddDate(0, 0, 1), nil
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
	return start, end, nil
}

func (t *RecommendEvalReport) buildSceneFilter(startAt time.Time, endAt time.Time, scene string) (string, []any) {
	sceneClause := ""
	args := []any{startAt, endAt}
	if scene != "" {
		sceneValue := appBiz.ParseRecommendSceneForTask(scene)
		if sceneValue == int32(common.RecommendScene_RECOMMEND_SCENE_UNKNOWN) {
			return sceneClause, args
		}
		sceneClause = " AND rr.scene = ?"
		args = append(args, sceneValue)
	}
	return sceneClause, args
}

func (t *RecommendEvalReport) buildSummarySQL(sceneClause string) string {
	return `
WITH base_requests AS (
  SELECT
    rr.request_id,
    rr.scene
  FROM ` + "`" + models.TableNameRecommendRequest + "`" + ` rr
  WHERE rr.created_at >= ?
    AND rr.created_at < ?` + sceneClause + `
),
exposure_request AS (
  SELECT
    re.request_id,
    COUNT(*) AS exposure_batch_count,
    SUM(JSON_LENGTH(re.goods_ids)) AS exposed_goods_count
  FROM ` + "`" + models.TableNameRecommendExposure + "`" + ` re
  INNER JOIN base_requests br ON br.request_id = re.request_id
  GROUP BY re.request_id
),
click_request AS (
  SELECT
    rga.request_id,
    COUNT(*) AS click_count
  FROM ` + "`" + models.TableNameRecommendGoodsAction + "`" + ` rga
  INNER JOIN base_requests br ON br.request_id = rga.request_id
  WHERE rga.event_type = 'recommend_click'
  GROUP BY rga.request_id
),
pay_request AS (
  SELECT
    rga.request_id,
    COUNT(*) AS pay_count
  FROM ` + "`" + models.TableNameRecommendGoodsAction + "`" + ` rga
  INNER JOIN base_requests br ON br.request_id = rga.request_id
  WHERE rga.event_type = 'order_pay'
  GROUP BY rga.request_id
)
SELECT
  br.scene AS scene,
  COUNT(*) AS request_count,
  COUNT(er.request_id) AS exposed_request_count,
  COALESCE(SUM(er.exposed_goods_count), 0) AS exposed_goods_count,
  COUNT(cr.request_id) AS click_request_count,
  COALESCE(SUM(cr.click_count), 0) AS click_count,
  COUNT(pr.request_id) AS pay_request_count,
  COALESCE(SUM(pr.pay_count), 0) AS pay_count,
  CASE WHEN COUNT(*) = 0 THEN 0 ELSE COUNT(cr.request_id) * 1.0 / COUNT(*) END AS request_ctr,
  CASE WHEN COALESCE(SUM(er.exposed_goods_count), 0) = 0 THEN 0 ELSE COALESCE(SUM(cr.click_count), 0) * 1.0 / COALESCE(SUM(er.exposed_goods_count), 0) END AS exposure_ctr,
  CASE WHEN COUNT(*) = 0 THEN 0 ELSE COUNT(pr.request_id) * 1.0 / COUNT(*) END AS request_pay_cvr,
  CASE WHEN COUNT(cr.request_id) = 0 THEN 0 ELSE COUNT(pr.request_id) * 1.0 / COUNT(cr.request_id) END AS click_pay_cvr
FROM base_requests br
LEFT JOIN exposure_request er ON er.request_id = br.request_id
LEFT JOIN click_request cr ON cr.request_id = br.request_id
LEFT JOIN pay_request pr ON pr.request_id = br.request_id
GROUP BY br.scene
ORDER BY request_count DESC, br.scene ASC
`
}

func (t *RecommendEvalReport) buildSourceSQL(sceneClause string) string {
	return `
WITH request_scope AS (
  SELECT
    rr.request_id,
    rr.scene,
    COALESCE(NULLIF(rr.recall_sources, ''), '[]') AS recall_sources
  FROM ` + "`" + models.TableNameRecommendRequest + "`" + ` rr
  WHERE rr.created_at >= ?
    AND rr.created_at < ?` + sceneClause + `
),
base_requests AS (
  SELECT
    rs.request_id,
    rs.scene,
    jt.recall_source
  FROM request_scope rs,
       JSON_TABLE(rs.recall_sources, '$[*]' COLUMNS(recall_source VARCHAR(64) PATH '$')) jt
),
click_request AS (
  SELECT DISTINCT rga.request_id
  FROM ` + "`" + models.TableNameRecommendGoodsAction + "`" + ` rga
  INNER JOIN request_scope rs ON rs.request_id = rga.request_id
  WHERE rga.event_type = 'recommend_click'
),
pay_request AS (
  SELECT DISTINCT rga.request_id
  FROM ` + "`" + models.TableNameRecommendGoodsAction + "`" + ` rga
  INNER JOIN request_scope rs ON rs.request_id = rga.request_id
  WHERE rga.event_type = 'order_pay'
)
SELECT
  br.scene AS scene,
  br.recall_source AS recall_source,
  COUNT(*) AS request_count,
  COUNT(cr.request_id) AS click_request_count,
  COUNT(pr.request_id) AS pay_request_count,
  CASE WHEN COUNT(*) = 0 THEN 0 ELSE COUNT(cr.request_id) * 1.0 / COUNT(*) END AS request_ctr,
  CASE WHEN COUNT(*) = 0 THEN 0 ELSE COUNT(pr.request_id) * 1.0 / COUNT(*) END AS request_pay_cvr,
  CASE WHEN COUNT(cr.request_id) = 0 THEN 0 ELSE COUNT(pr.request_id) * 1.0 / COUNT(cr.request_id) END AS click_to_pay_cvr
FROM base_requests br
LEFT JOIN click_request cr ON cr.request_id = br.request_id
LEFT JOIN pay_request pr ON pr.request_id = br.request_id
GROUP BY br.scene, br.recall_source
ORDER BY br.scene ASC, request_count DESC, br.recall_source ASC
`
}

func (t *RecommendEvalReport) buildResult(
	startAt time.Time,
	endAt time.Time,
	scene string,
	summaryRows []*recommendEvalSummaryRow,
	sourceRows []*recommendEvalSourceRow,
) []string {
	result := []string{
		fmt.Sprintf("推荐离线评估范围: %s ~ %s", startAt.Format("2006-01-02"), endAt.Add(-time.Nanosecond).Format("2006-01-02")),
	}
	if scene != "" {
		result = append(result, fmt.Sprintf("场景过滤: %s", scene))
	}
	if len(summaryRows) == 0 {
		return append(result, "未找到推荐请求数据")
	}

	result = append(result, "场景汇总:")
	for _, row := range summaryRows {
		result = append(result, fmt.Sprintf(
			"scene=%s requests=%d exposedReq=%d exposedGoods=%d clickReq=%d clicks=%d payReq=%d pays=%d requestCTR=%.4f exposureCTR=%.4f requestPayCVR=%.4f clickPayCVR=%.4f",
			appBiz.FormatRecommendSceneForTask(row.Scene),
			row.RequestCount,
			row.ExposedRequestCount,
			row.ExposedGoodsCount,
			row.ClickRequestCount,
			row.ClickCount,
			row.PayRequestCount,
			row.PayCount,
			row.RequestCTR,
			row.ExposureCTR,
			row.RequestPayCVR,
			row.ClickPayCVR,
		))
	}

	if len(sourceRows) == 0 {
		return result
	}

	result = append(result, "召回来源拆分:")
	for _, row := range sourceRows {
		result = append(result, fmt.Sprintf(
			"scene=%s source=%s requests=%d clickReq=%d payReq=%d requestCTR=%.4f requestPayCVR=%.4f clickPayCVR=%.4f",
			appBiz.FormatRecommendSceneForTask(row.Scene),
			row.RecallSource,
			row.RequestCount,
			row.ClickReqCount,
			row.PayReqCount,
			row.RequestCTR,
			row.RequestPayCVR,
			row.ClickToPayCVR,
		))
	}

	return result
}

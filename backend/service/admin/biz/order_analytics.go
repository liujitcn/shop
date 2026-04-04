package biz

import (
	"context"
	"fmt"
	"time"

	adminApi "shop/api/gen/go/admin"
	commonApi "shop/api/gen/go/common"
	"shop/pkg/gen/models"
	"shop/service/admin/dto"
	"shop/service/admin/utils"

	"github.com/liujitcn/gorm-kit/repo"
)

// OrderAnalyticsCase 订单分析业务
type OrderAnalyticsCase struct {
	orderCase        *OrderInfoCase
	baseDictCase     *BaseDictCase
	baseDictItemCase *BaseDictItemCase
}

// NewOrderAnalyticsCase 创建订单分析业务
func NewOrderAnalyticsCase(orderCase *OrderInfoCase, baseDictCase *BaseDictCase, baseDictItemCase *BaseDictItemCase) *OrderAnalyticsCase {
	return &OrderAnalyticsCase{
		orderCase:        orderCase,
		baseDictCase:     baseDictCase,
		baseDictItemCase: baseDictItemCase,
	}
}

// GetOrderAnalyticsSummary 查询订单摘要指标
func (c *OrderAnalyticsCase) GetOrderAnalyticsSummary(ctx context.Context, req *commonApi.AnalyticsTimeRequest) (*adminApi.OrderAnalyticsSummaryResponse, error) {
	startAt, endAt := utils.GetAnalyticsTimeRange(req.GetTimeType())
	prevStartAt, prevEndAt := utils.GetPreviousAnalyticsTimeRange(req.GetTimeType(), startAt)

	newOrderCount, saleAmount, err := c.countOrderBaseSummary(ctx, startAt, endAt)
	if err != nil {
		return nil, err
	}
	var prevOrderCount int64
	prevOrderCount, _, err = c.countOrderBaseSummary(ctx, prevStartAt, prevEndAt)
	if err != nil {
		return nil, err
	}
	var orderUserCount int64
	orderUserCount, err = c.countDistinctOrderUsers(ctx, startAt, endAt)
	if err != nil {
		return nil, err
	}
	var repurchaseUserCount int64
	repurchaseUserCount, err = c.countRepurchaseUsers(ctx, startAt, endAt)
	if err != nil {
		return nil, err
	}

	return &adminApi.OrderAnalyticsSummaryResponse{
		NewOrderCount:      newOrderCount,
		NewOrderGrowthRate: utils.CalcGrowthRate(prevOrderCount, newOrderCount),
		SaleAmount:         saleAmount,
		AverageOrderAmount: utils.CalcPerUnit(saleAmount, newOrderCount),
		OrderUserCount:     orderUserCount,
		RepurchaseRate:     utils.CalcRatio(repurchaseUserCount, orderUserCount),
	}, nil
}

// GetOrderAnalyticsTrend 查询订单趋势
func (c *OrderAnalyticsCase) GetOrderAnalyticsTrend(ctx context.Context, req *commonApi.AnalyticsTimeRequest) (*commonApi.AnalyticsTrendResponse, error) {
	startAt, endAt := utils.GetAnalyticsTimeRange(req.GetTimeType())
	summary, axis, err := c.queryOrderSummary(ctx, req.GetTimeType(), startAt, endAt)
	if err != nil {
		return nil, err
	}

	orderRow := make([]int64, 0, len(axis))
	saleRow := make([]int64, 0, len(axis))
	for i := range axis {
		key := int64(i + 1)
		orderRow = append(orderRow, summary[key].OrderCount)
		saleRow = append(saleRow, summary[key].SaleAmount)
	}

	return &commonApi.AnalyticsTrendResponse{
		Axis: axis,
		Series: []*commonApi.AnalyticsTrendSeries{
			{Name: "订单量", Type: commonApi.AnalyticsSeriesType_BAR, Data: orderRow},
			{Name: "销售额", Type: commonApi.AnalyticsSeriesType_LINE, Data: saleRow, YAxisIndex: 1},
		},
		YAxisNames: []string{"订单量", "销售额"},
	}, nil
}

// GetOrderAnalyticsPie 查询订单状态分布
func (c *OrderAnalyticsCase) GetOrderAnalyticsPie(ctx context.Context, req *commonApi.AnalyticsTimeRequest) (*commonApi.AnalyticsPieResponse, error) {
	startAt, endAt := utils.GetAnalyticsTimeRange(req.GetTimeType())
	summary, err := c.queryOrderStatusSummary(ctx, startAt, endAt)
	if err != nil {
		return nil, err
	}
	var statusLabelMap map[int32]string
	statusLabelMap, err = c.getOrderStatusLabelMap(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]*commonApi.AnalyticsPieItem, 0, len(summary))
	for _, item := range summary {
		label, ok := statusLabelMap[item.Status]
		if !ok {
			label = fmt.Sprintf("状态%d", item.Status)
		}
		items = append(items, &commonApi.AnalyticsPieItem{Name: label, Value: item.OrderCount})
	}
	return &commonApi.AnalyticsPieResponse{Items: items}, nil
}

func (c *OrderAnalyticsCase) countOrderBaseSummary(ctx context.Context, startAt, endAt time.Time) (int64, int64, error) {
	type row struct {
		OrderCount int64 `gorm:"column:order_count"`
		SaleAmount int64 `gorm:"column:sale_amount"`
	}
	var result row
	err := c.orderCase.Query(ctx).OrderInfo.WithContext(ctx).UnderlyingDB().
		Model(&models.OrderInfo{}).
		Select("COUNT(*) AS order_count, COALESCE(SUM(pay_money),0) AS sale_amount").
		Where("created_at >= ? AND created_at < ?", startAt, endAt).
		Scan(&result).Error
	return result.OrderCount, result.SaleAmount, err
}

func (c *OrderAnalyticsCase) countDistinctOrderUsers(ctx context.Context, startAt, endAt time.Time) (int64, error) {
	var count int64
	err := c.orderCase.Query(ctx).OrderInfo.WithContext(ctx).UnderlyingDB().
		Model(&models.OrderInfo{}).
		Where("created_at >= ? AND created_at < ?", startAt, endAt).
		Distinct("user_id").
		Count(&count).Error
	return count, err
}

func (c *OrderAnalyticsCase) countRepurchaseUsers(ctx context.Context, startAt, endAt time.Time) (int64, error) {
	type row struct {
		Total int64 `gorm:"column:total"`
	}
	var result row
	sql := "" +
		"SELECT COUNT(*) AS total FROM (" +
		" SELECT user_id" +
		" FROM `" + models.TableNameOrderInfo + "`" +
		" WHERE created_at >= ? AND created_at < ?" +
		" GROUP BY user_id" +
		" HAVING COUNT(*) >= 2" +
		") t"
	err := c.orderCase.Query(ctx).OrderInfo.WithContext(ctx).UnderlyingDB().Raw(sql, startAt, endAt).Scan(&result).Error
	return result.Total, err
}

func (c *OrderAnalyticsCase) getOrderStatusLabelMap(ctx context.Context) (map[int32]string, error) {
	dictQuery := c.baseDictCase.Query(ctx).BaseDict
	baseDict, err := c.baseDictCase.Find(ctx, repo.Where(dictQuery.Code.Eq("order_status")))
	if err != nil {
		return nil, err
	}

	dictItemQuery := c.baseDictItemCase.Query(ctx).BaseDictItem
	baseDictItemList, err := c.baseDictItemCase.List(ctx, repo.Where(dictItemQuery.DictID.Eq(baseDict.ID)))
	if err != nil {
		return nil, err
	}

	statusLabelMap := make(map[int32]string, len(baseDictItemList))
	for _, item := range baseDictItemList {
		var statusValue int32
		_, err = fmt.Sscanf(item.Value, "%d", &statusValue)
		if err != nil {
			continue
		}
		statusLabelMap[statusValue] = item.Label
	}
	return statusLabelMap, nil
}

// queryOrderSummary 查询订单趋势汇总。
func (c *OrderAnalyticsCase) queryOrderSummary(ctx context.Context, timeType commonApi.AnalyticsTimeType, startAt, endAt time.Time) (map[int64]*dto.OrderSummary, []string, error) {
	summaryMap := make(map[int64]*dto.OrderSummary)
	axisData := make([]string, 0)
	db := c.orderCase.Query(ctx).OrderInfo.WithContext(ctx).UnderlyingDB()

	switch timeType {
	case commonApi.AnalyticsTimeType_YEAR:
		var rows []*dto.OrderSummary
		err := db.Model(&models.OrderInfo{}).
			Select("MONTH(created_at) AS `key`, COUNT(*) AS order_count, COALESCE(SUM(pay_money),0) AS sale_amount").
			Where("created_at >= ? AND created_at < ?", startAt, endAt).
			Group("MONTH(created_at)").
			Scan(&rows).Error
		if err != nil {
			return nil, nil, err
		}
		for _, item := range rows {
			summaryMap[item.Key] = item
		}
		for i := 0; i < 12; i++ {
			axisData = append(axisData, utils.FormatAnalyticsAxis(timeType, i, startAt))
		}
	case commonApi.AnalyticsTimeType_MONTH:
		var rows []*dto.OrderSummary
		err := db.Model(&models.OrderInfo{}).
			Select("DAY(created_at) AS `key`, COUNT(*) AS order_count, COALESCE(SUM(pay_money),0) AS sale_amount").
			Where("created_at >= ? AND created_at < ?", startAt, endAt).
			Group("DAY(created_at)").
			Scan(&rows).Error
		if err != nil {
			return nil, nil, err
		}
		for _, item := range rows {
			summaryMap[item.Key] = item
		}
		monthDays := endAt.AddDate(0, 0, -1).Day()
		for i := 0; i < monthDays; i++ {
			axisData = append(axisData, utils.FormatAnalyticsAxis(timeType, i, startAt))
		}
	default:
		var rows []*dto.OrderSummary
		err := db.Model(&models.OrderInfo{}).
			Select("WEEKDAY(created_at)+1 AS `key`, COUNT(*) AS order_count, COALESCE(SUM(pay_money),0) AS sale_amount").
			Where("created_at >= ? AND created_at < ?", startAt, endAt).
			Group("WEEKDAY(created_at)+1").
			Scan(&rows).Error
		if err != nil {
			return nil, nil, err
		}
		for _, item := range rows {
			summaryMap[item.Key] = item
		}
		for i := 0; i < 7; i++ {
			axisData = append(axisData, utils.FormatAnalyticsAxis(timeType, i, startAt))
		}
	}
	// 补齐缺失桶位，避免前端图表在空数据时出现断层。
	for i := range axisData {
		key := int64(i + 1)
		if _, ok := summaryMap[key]; !ok {
			summaryMap[key] = &dto.OrderSummary{Key: key}
		}
	}
	return summaryMap, axisData, nil
}

func (c *OrderAnalyticsCase) queryOrderStatusSummary(ctx context.Context, startAt, endAt time.Time) ([]*dto.OrderStatusSummary, error) {
	res := make([]*dto.OrderStatusSummary, 0)
	err := c.orderCase.Query(ctx).OrderInfo.WithContext(ctx).UnderlyingDB().
		Model(&models.OrderInfo{}).
		Select("status, COUNT(*) AS order_count").
		Where("created_at >= ? AND created_at < ?", startAt, endAt).
		Group("status").
		Scan(&res).Error
	return res, err
}

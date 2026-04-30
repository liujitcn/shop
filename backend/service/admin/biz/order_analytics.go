package biz

import (
	"context"
	"strconv"
	"time"

	_const "shop/pkg/const"

	adminv1 "shop/api/gen/go/admin/v1"
	commonv1 "shop/api/gen/go/common/v1"
	"shop/service/admin/dto"
	"shop/service/admin/utils"
)

// OrderAnalyticsCase 订单分析业务
type OrderAnalyticsCase struct {
	orderInfoCase *OrderInfoCase
}

// NewOrderAnalyticsCase 创建订单分析业务
func NewOrderAnalyticsCase(orderInfoCase *OrderInfoCase) *OrderAnalyticsCase {
	return &OrderAnalyticsCase{
		orderInfoCase: orderInfoCase,
	}
}

// SummaryOrderAnalytics 查询订单摘要指标
func (c *OrderAnalyticsCase) SummaryOrderAnalytics(ctx context.Context, req *adminv1.SummaryOrderAnalyticsRequest) (*adminv1.SummaryOrderAnalyticsResponse, error) {
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

	return &adminv1.SummaryOrderAnalyticsResponse{
		NewOrderCount:      newOrderCount,
		NewOrderGrowthRate: utils.CalcGrowthRate(prevOrderCount, newOrderCount),
		SaleAmount:         saleAmount,
		AverageOrderAmount: utils.CalcPerUnit(saleAmount, newOrderCount),
		OrderUserCount:     orderUserCount,
		RepurchaseRate:     utils.CalcRatio(repurchaseUserCount, orderUserCount),
	}, nil
}

// TrendOrderAnalytics 查询订单趋势
func (c *OrderAnalyticsCase) TrendOrderAnalytics(ctx context.Context, req *adminv1.TrendOrderAnalyticsRequest) (*commonv1.AnalyticsTrendResponse, error) {
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

	return &commonv1.AnalyticsTrendResponse{
		Axis: axis,
		Series: []*commonv1.AnalyticsTrendSeries{
			{Name: "订单量", Type: commonv1.AnalyticsSeriesType(_const.ANALYTICS_SERIES_TYPE_BAR), Data: orderRow},
			{Name: "销售额", Type: commonv1.AnalyticsSeriesType(_const.ANALYTICS_SERIES_TYPE_LINE), Data: saleRow, YAxisIndex: 1},
		},
		YAxisNames: []string{"订单量", "销售额"},
	}, nil
}

// PieOrderAnalytics 查询订单状态分布
func (c *OrderAnalyticsCase) PieOrderAnalytics(ctx context.Context, req *adminv1.PieOrderAnalyticsRequest) (*commonv1.AnalyticsPieResponse, error) {
	startAt, endAt := utils.GetAnalyticsTimeRange(req.GetTimeType())
	summary, err := c.queryOrderStatusSummary(ctx, startAt, endAt)
	if err != nil {
		return nil, err
	}

	items := make([]*commonv1.AnalyticsPieItem, 0, len(summary))
	for _, item := range summary {
		// 状态分布只返回状态值，展示文案统一交由前端按字典转换。
		items = append(items, &commonv1.AnalyticsPieItem{Name: strconv.FormatInt(int64(item.Status), 10), Value: item.OrderCount})
	}
	return &commonv1.AnalyticsPieResponse{Items: items}, nil
}

// countOrderBaseSummary 统计时间范围内订单数和销售额。
func (c *OrderAnalyticsCase) countOrderBaseSummary(ctx context.Context, startAt, endAt time.Time) (int64, int64, error) {
	type row struct {
		OrderCount int64 `gorm:"column:order_count"`
		SaleAmount int64 `gorm:"column:sale_amount"`
	}
	var result row
	query := c.orderInfoCase.Query(ctx).OrderInfo
	err := query.WithContext(ctx).
		Select(
			query.ID.Count().As("order_count"),
			query.PayMoney.Sum().FloorDiv(1).IfNull(0).As("sale_amount"),
		).
		Where(
			query.CreatedAt.Gte(startAt),
			query.CreatedAt.Lt(endAt),
		).
		Scan(&result)
	return result.OrderCount, result.SaleAmount, err
}

// countDistinctOrderUsers 统计时间范围内下单用户数。
func (c *OrderAnalyticsCase) countDistinctOrderUsers(ctx context.Context, startAt, endAt time.Time) (int64, error) {
	query := c.orderInfoCase.Query(ctx).OrderInfo
	count, err := query.WithContext(ctx).
		Where(
			query.CreatedAt.Gte(startAt),
			query.CreatedAt.Lt(endAt),
		).
		Distinct(query.UserID).
		Count()
	return count, err
}

// countRepurchaseUsers 统计时间范围内复购用户数。
func (c *OrderAnalyticsCase) countRepurchaseUsers(ctx context.Context, startAt, endAt time.Time) (int64, error) {
	query := c.orderInfoCase.Query(ctx).OrderInfo
	userIDs := make([]int64, 0)
	err := query.WithContext(ctx).
		Where(
			query.CreatedAt.Gte(startAt),
			query.CreatedAt.Lt(endAt),
		).
		Pluck(query.UserID, &userIDs)
	if err != nil {
		return 0, err
	}
	return utils.CountAtLeastOccurrences(userIDs, 2), nil
}

// queryOrderSummary 查询订单趋势汇总。
func (c *OrderAnalyticsCase) queryOrderSummary(ctx context.Context, timeType commonv1.AnalyticsTimeType, startAt, endAt time.Time) (map[int64]*dto.OrderSummary, []string, error) {
	summaryMap := make(map[int64]*dto.OrderSummary)
	rows := make([]*dto.OrderSummary, 0)
	query := c.orderInfoCase.Query(ctx).OrderInfo
	groupField, axisData := utils.GetAnalyticsGroupFieldByColumn(timeType, startAt, endAt, query.CreatedAt)
	err := query.WithContext(ctx).
		Select(
			groupField.As("key"),
			query.ID.Count().As("order_count"),
			query.PayMoney.Sum().FloorDiv(1).IfNull(0).As("sale_amount"),
		).
		Where(
			query.CreatedAt.Gte(startAt),
			query.CreatedAt.Lt(endAt),
		).
		Group(utils.AnalyticsGroupAliasField()).
		Scan(&rows)
	if err != nil {
		return nil, nil, err
	}
	for _, item := range rows {
		summaryMap[item.Key] = item
	}
	// 补齐缺失桶位，避免前端图表在空数据时出现断层。
	for i := range axisData {
		key := int64(i + 1)
		// 当前桶位缺少聚合结果时，补一个空对象保证序列完整。
		if _, ok := summaryMap[key]; !ok {
			summaryMap[key] = &dto.OrderSummary{Key: key}
		}
	}
	return summaryMap, axisData, nil
}

// queryOrderStatusSummary 查询指定时间范围内的订单状态分布。
func (c *OrderAnalyticsCase) queryOrderStatusSummary(ctx context.Context, startAt, endAt time.Time) ([]*dto.OrderStatusSummary, error) {
	res := make([]*dto.OrderStatusSummary, 0)
	query := c.orderInfoCase.Query(ctx).OrderInfo
	err := query.WithContext(ctx).
		Select(
			query.Status,
			query.ID.Count().As("order_count"),
		).
		Where(
			query.CreatedAt.Gte(startAt),
			query.CreatedAt.Lt(endAt),
		).
		Group(query.Status).
		Scan(&res)
	return res, err
}

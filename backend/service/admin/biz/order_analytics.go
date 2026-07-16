package biz

import (
	"context"
	"strconv"
	"time"

	"github.com/liujitcn/gorm-kit/repository"

	adminv1 "shop/api/gen/go/admin/v1"
	commonv1 "shop/api/gen/go/common/v1"
	_const "shop/pkg/const"
	orderutils "shop/pkg/utils"
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

	newOrderCount, saleAmount, err := c.countOrderBaseSummary(ctx, req.GetTenantId(), req.GetTenantStoreId(), startAt, endAt)
	if err != nil {
		return nil, err
	}
	var prevOrderCount int64
	prevOrderCount, _, err = c.countOrderBaseSummary(ctx, req.GetTenantId(), req.GetTenantStoreId(), prevStartAt, prevEndAt)
	if err != nil {
		return nil, err
	}
	var orderUserCount int64
	orderUserCount, err = c.countDistinctOrderUsers(ctx, req.GetTenantId(), req.GetTenantStoreId(), startAt, endAt)
	if err != nil {
		return nil, err
	}
	var repurchaseUserCount int64
	repurchaseUserCount, err = c.countRepurchaseUsers(ctx, req.GetTenantId(), req.GetTenantStoreId(), startAt, endAt)
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
	summary, axis, err := c.queryOrderSummary(ctx, req.GetTimeType(), req.GetTenantId(), req.GetTenantStoreId(), startAt, endAt)
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
	summary, err := c.queryOrderStatusSummary(ctx, req.GetTenantId(), req.GetTenantStoreId(), startAt, endAt)
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

// countDistinctOrderUsers 统计时间范围内下单用户数。
func (c *OrderAnalyticsCase) countDistinctOrderUsers(ctx context.Context, tenantID, tenantStoreID int64, startAt, endAt time.Time) (int64, error) {
	query := c.orderInfoCase.Query(ctx).OrderInfo
	dao := query.WithContext(ctx).
		Where(
			query.CreatedAt.Gte(startAt),
			query.CreatedAt.Lt(endAt),
		)
	// 默认租户可按租户筛选，普通租户仍受数据库租户隔离约束。
	if tenantID > 0 {
		dao = dao.Where(query.TenantID.Eq(tenantID))
	}
	if tenantStoreID > 0 {
		dao = dao.Where(query.TenantStoreID.Eq(tenantStoreID))
	}
	count, err := dao.
		Distinct(query.UserID).
		Count()
	return count, err
}

// countRepurchaseUsers 统计时间范围内复购用户数。
func (c *OrderAnalyticsCase) countRepurchaseUsers(ctx context.Context, tenantID, tenantStoreID int64, startAt, endAt time.Time) (int64, error) {
	query := c.orderInfoCase.Query(ctx).OrderInfo
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Where(query.CreatedAt.Gte(startAt)))
	opts = append(opts, repository.Where(query.CreatedAt.Lt(endAt)))
	if tenantID > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(tenantID)))
	}
	if tenantStoreID > 0 {
		opts = append(opts, repository.Where(query.TenantStoreID.Eq(tenantStoreID)))
	}
	orderInfos, err := c.orderInfoCase.List(ctx, opts...)
	if err != nil {
		return 0, err
	}
	tradeMap, err := c.orderInfoCase.getOrderTradeMap(ctx, orderInfos)
	if err != nil {
		return 0, err
	}
	userTrades := make(map[int64]map[int64]struct{})
	for _, orderInfo := range orderInfos {
		orderTrade := tradeMap[orderInfo.TradeID]
		// 复购只统计已经形成支付事实的交易，待支付或已关闭交易不参与计算。
		if orderTrade == nil || !orderutils.IsPaidTradeStatus(orderTrade.Status) {
			continue
		}
		if userTrades[orderInfo.UserID] == nil {
			userTrades[orderInfo.UserID] = make(map[int64]struct{})
		}
		userTrades[orderInfo.UserID][orderInfo.TradeID] = struct{}{}
	}
	var count int64
	for _, tradeSet := range userTrades {
		if len(tradeSet) >= 2 {
			count++
		}
	}
	return count, nil
}

// countOrderBaseSummary 统计时间范围内订单数和销售额。
func (c *OrderAnalyticsCase) countOrderBaseSummary(ctx context.Context, tenantID, tenantStoreID int64, startAt, endAt time.Time) (int64, int64, error) {
	query := c.orderInfoCase.Query(ctx).OrderInfo
	tradeQuery := c.orderInfoCase.orderTradeRepo.Query(ctx).OrderTrade
	dao := query.WithContext(ctx).
		Where(
			query.CreatedAt.Gte(startAt),
			query.CreatedAt.Lt(endAt),
		)
	if tenantID > 0 {
		dao = dao.Where(query.TenantID.Eq(tenantID))
	}
	if tenantStoreID > 0 {
		dao = dao.Where(query.TenantStoreID.Eq(tenantStoreID))
	}
	orderCount, err := dao.Count()
	if err != nil {
		return 0, 0, err
	}
	var result dto.OrderSummary
	err = dao.
		Select(query.PayMoney.Sum().FloorDiv(1).IfNull(0).As("sale_amount")).
		Join(tradeQuery, query.TradeID.EqCol(tradeQuery.ID)).
		Where(tradeQuery.Status.In(orderutils.PaidTradeStatuses()...)).
		Scan(&result)
	return orderCount, result.SaleAmount, err
}

// queryOrderSummary 查询订单趋势汇总。
func (c *OrderAnalyticsCase) queryOrderSummary(ctx context.Context, timeType commonv1.AnalyticsTimeType, tenantID, tenantStoreID int64, startAt, endAt time.Time) (map[int64]*dto.OrderSummary, []string, error) {
	summaryMap := make(map[int64]*dto.OrderSummary)
	query := c.orderInfoCase.Query(ctx).OrderInfo
	groupField, axisData := utils.GetAnalyticsGroupFieldByColumn(timeType, startAt, endAt, query.CreatedAt)
	dao := query.WithContext(ctx).
		Where(
			query.CreatedAt.Gte(startAt),
			query.CreatedAt.Lt(endAt),
		)
	if tenantID > 0 {
		dao = dao.Where(query.TenantID.Eq(tenantID))
	}
	if tenantStoreID > 0 {
		dao = dao.Where(query.TenantStoreID.Eq(tenantStoreID))
	}
	countRows := make([]*dto.OrderSummary, 0)
	err := dao.
		Select(
			groupField.As("key"),
			query.ID.Count().As("order_count"),
		).
		Group(utils.AnalyticsGroupAliasField()).
		Scan(&countRows)
	if err != nil {
		return nil, nil, err
	}
	for _, item := range countRows {
		summaryMap[item.Key] = item
	}
	saleRows := make([]*dto.OrderSummary, 0)
	tradeQuery := c.orderInfoCase.orderTradeRepo.Query(ctx).OrderTrade
	err = dao.
		Select(
			groupField.As("key"),
			query.PayMoney.Sum().FloorDiv(1).IfNull(0).As("sale_amount"),
		).
		Join(tradeQuery, query.TradeID.EqCol(tradeQuery.ID)).
		Where(tradeQuery.Status.In(orderutils.PaidTradeStatuses()...)).
		Group(utils.AnalyticsGroupAliasField()).
		Scan(&saleRows)
	if err != nil {
		return nil, nil, err
	}
	for _, item := range saleRows {
		if summaryMap[item.Key] == nil {
			summaryMap[item.Key] = &dto.OrderSummary{Key: item.Key}
		}
		summaryMap[item.Key].SaleAmount = item.SaleAmount
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
func (c *OrderAnalyticsCase) queryOrderStatusSummary(ctx context.Context, tenantID, tenantStoreID int64, startAt, endAt time.Time) ([]*dto.OrderStatusSummary, error) {
	res := make([]*dto.OrderStatusSummary, 0)
	query := c.orderInfoCase.Query(ctx).OrderInfo
	dao := query.WithContext(ctx).
		Where(
			query.CreatedAt.Gte(startAt),
			query.CreatedAt.Lt(endAt),
		)
	if tenantID > 0 {
		dao = dao.Where(query.TenantID.Eq(tenantID))
	}
	if tenantStoreID > 0 {
		dao = dao.Where(query.TenantStoreID.Eq(tenantStoreID))
	}
	err := dao.
		Select(
			query.Status,
			query.ID.Count().As("order_count"),
		).
		Group(query.Status).
		Scan(&res)
	return res, err
}

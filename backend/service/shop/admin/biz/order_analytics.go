package biz

import (
	"context"
	"strconv"
	"time"

	commonv1 "shop/api/gen/go/common/v1"
	shopadminv1 "shop/api/gen/go/shop/admin/v1"
	_const "shop/service/shop/consts"
	"shop/service/shop/admin/dto"
	"shop/service/shop/admin/utils"
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
func (c *OrderAnalyticsCase) SummaryOrderAnalytics(ctx context.Context, req *shopadminv1.SummaryOrderAnalyticsRequest) (*shopadminv1.SummaryOrderAnalyticsResponse, error) {
	startAt, endAt := utils.GetAnalyticsTimeRange(req.GetTimeType())
	prevStartAt, prevEndAt := utils.GetPreviousAnalyticsTimeRange(req.GetTimeType(), startAt)

	useGlobalTradeScope, err := c.orderInfoCase.useGlobalOrderTradeScope(ctx, req.GetTenantId(), req.GetTenantStoreId())
	if err != nil {
		return nil, err
	}
	var newOrderCount int64
	var paidOrderCount int64
	var saleAmount int64
	newOrderCount, paidOrderCount, saleAmount, err = c.countOrderBaseSummary(ctx, req.GetTenantId(), req.GetTenantStoreId(), startAt, endAt, useGlobalTradeScope)
	if err != nil {
		return nil, err
	}
	var prevOrderCount int64
	prevOrderCount, _, _, err = c.countOrderBaseSummary(ctx, req.GetTenantId(), req.GetTenantStoreId(), prevStartAt, prevEndAt, useGlobalTradeScope)
	if err != nil {
		return nil, err
	}
	var orderUserCount int64
	orderUserCount, err = c.countDistinctOrderUsers(ctx, req.GetTenantId(), req.GetTenantStoreId(), startAt, endAt, useGlobalTradeScope)
	if err != nil {
		return nil, err
	}
	var repurchaseUserCount int64
	repurchaseUserCount, err = c.countRepurchaseUsers(ctx, req.GetTenantId(), req.GetTenantStoreId(), startAt, endAt, useGlobalTradeScope)
	if err != nil {
		return nil, err
	}

	return &shopadminv1.SummaryOrderAnalyticsResponse{
		NewOrderCount:      newOrderCount,
		NewOrderGrowthRate: utils.CalcGrowthRate(prevOrderCount, newOrderCount),
		SaleAmount:         saleAmount,
		AverageOrderAmount: utils.CalcPerUnit(saleAmount, paidOrderCount),
		OrderUserCount:     orderUserCount,
		RepurchaseRate:     utils.CalcRatio(repurchaseUserCount, orderUserCount),
	}, nil
}

// TrendOrderAnalytics 查询订单趋势
func (c *OrderAnalyticsCase) TrendOrderAnalytics(ctx context.Context, req *shopadminv1.TrendOrderAnalyticsRequest) (*commonv1.AnalyticsTrendResponse, error) {
	startAt, endAt := utils.GetAnalyticsTimeRange(req.GetTimeType())
	useGlobalTradeScope, err := c.orderInfoCase.useGlobalOrderTradeScope(ctx, req.GetTenantId(), req.GetTenantStoreId())
	if err != nil {
		return nil, err
	}
	var summary map[int64]*dto.OrderSummary
	var axis []string
	summary, axis, err = c.queryOrderSummary(ctx, req.GetTimeType(), req.GetTenantId(), req.GetTenantStoreId(), startAt, endAt, useGlobalTradeScope)
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
func (c *OrderAnalyticsCase) PieOrderAnalytics(ctx context.Context, req *shopadminv1.PieOrderAnalyticsRequest) (*commonv1.AnalyticsPieResponse, error) {
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

// countDistinctOrderUsers 按当前订单口径统计时间范围内下单用户数。
func (c *OrderAnalyticsCase) countDistinctOrderUsers(ctx context.Context, tenantID, tenantStoreID int64, startAt, endAt time.Time, useGlobalTradeScope bool) (int64, error) {
	// 默认租户全局视角直接按交易单用户去重，避免依赖多门店子单。
	if useGlobalTradeScope {
		query := c.orderInfoCase.orderTradeRepo.Query(ctx).OrderTrade
		return query.WithContext(ctx).
			Where(
				query.CreatedAt.Gte(startAt),
				query.CreatedAt.Lt(endAt),
			).
			Distinct(query.UserID).
			Count()
	}
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

// countRepurchaseUsers 按当前订单口径统计时间范围内复购用户数。
func (c *OrderAnalyticsCase) countRepurchaseUsers(ctx context.Context, tenantID, tenantStoreID int64, startAt, endAt time.Time, useGlobalTradeScope bool) (int64, error) {
	paidFacts, err := c.orderInfoCase.queryPaidOrderFacts(ctx, 0, 0, tenantID, tenantStoreID, startAt, endAt, useGlobalTradeScope)
	if err != nil {
		return 0, err
	}
	userTrades := make(map[int64]map[int64]struct{})
	for _, fact := range paidFacts {
		if userTrades[fact.UserID] == nil {
			userTrades[fact.UserID] = make(map[int64]struct{})
		}
		userTrades[fact.UserID][fact.TradeID] = struct{}{}
	}
	var count int64
	for _, tradeSet := range userTrades {
		if len(tradeSet) >= 2 {
			count++
		}
	}
	return count, nil
}

// countOrderBaseSummary 按当前订单口径统计时间范围内订单数和销售额。
func (c *OrderAnalyticsCase) countOrderBaseSummary(ctx context.Context, tenantID, tenantStoreID int64, startAt, endAt time.Time, useGlobalTradeScope bool) (int64, int64, int64, error) {
	var orderCount int64
	// 默认租户全局视角直接聚合交易单，避免多门店子单重复增加订单数。
	if useGlobalTradeScope {
		query := c.orderInfoCase.orderTradeRepo.Query(ctx).OrderTrade
		var err error
		orderCount, err = query.WithContext(ctx).
			Where(
				query.CreatedAt.Gte(startAt),
				query.CreatedAt.Lt(endAt),
			).
			Count()
		if err != nil {
			return 0, 0, 0, err
		}
	} else {
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
		var err error
		orderCount, err = dao.Count()
		if err != nil {
			return 0, 0, 0, err
		}
	}

	paidFacts, err := c.orderInfoCase.queryPaidOrderFacts(ctx, 0, 0, tenantID, tenantStoreID, startAt, endAt, useGlobalTradeScope)
	if err != nil {
		return 0, 0, 0, err
	}
	var saleAmount int64
	for _, fact := range paidFacts {
		saleAmount += fact.PayMoney
	}
	return orderCount, int64(len(paidFacts)), saleAmount, nil
}

// queryOrderSummary 按当前订单口径查询订单趋势汇总。
func (c *OrderAnalyticsCase) queryOrderSummary(ctx context.Context, timeType commonv1.AnalyticsTimeType, tenantID, tenantStoreID int64, startAt, endAt time.Time, useGlobalTradeScope bool) (map[int64]*dto.OrderSummary, []string, error) {
	// 默认租户全局视角使用交易单趋势，状态饼图仍由独立的门店订单查询负责。
	if useGlobalTradeScope {
		return c.queryGlobalOrderTradeSummary(ctx, timeType, startAt, endAt)
	}
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
	var paidFacts []*dto.OrderPaidFact
	paidFacts, err = c.orderInfoCase.queryPaidOrderFacts(ctx, 0, 0, tenantID, tenantStoreID, startAt, endAt, false)
	if err != nil {
		return nil, nil, err
	}
	appendPaidFactSales(summaryMap, timeType, paidFacts)
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

// queryGlobalOrderTradeSummary 查询默认租户全局交易单趋势汇总。
func (c *OrderAnalyticsCase) queryGlobalOrderTradeSummary(ctx context.Context, timeType commonv1.AnalyticsTimeType, startAt, endAt time.Time) (map[int64]*dto.OrderSummary, []string, error) {
	summaryMap := make(map[int64]*dto.OrderSummary)
	query := c.orderInfoCase.orderTradeRepo.Query(ctx).OrderTrade
	groupField, axisData := utils.GetAnalyticsGroupFieldByColumn(timeType, startAt, endAt, query.CreatedAt)
	dao := query.WithContext(ctx).
		Where(
			query.CreatedAt.Gte(startAt),
			query.CreatedAt.Lt(endAt),
		)
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
	var paidFacts []*dto.OrderPaidFact
	paidFacts, err = c.orderInfoCase.queryPaidOrderFacts(ctx, 0, 0, 0, 0, startAt, endAt, true)
	if err != nil {
		return nil, nil, err
	}
	appendPaidFactSales(summaryMap, timeType, paidFacts)
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

// appendPaidFactSales 将支付事实金额写入对应的分析趋势桶位。
func appendPaidFactSales(summaryMap map[int64]*dto.OrderSummary, timeType commonv1.AnalyticsTimeType, paidFacts []*dto.OrderPaidFact) {
	for _, fact := range paidFacts {
		key := utils.GetAnalyticsTimeKey(timeType, fact.PaidAt)
		if summaryMap[key] == nil {
			summaryMap[key] = &dto.OrderSummary{Key: key}
		}
		summaryMap[key].SaleAmount += fact.PayMoney
	}
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

package biz

import (
	"context"
	"fmt"
	"time"

	shopadminv1 "shop/api/gen/go/shop/admin/v1"
	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/service/shop/admin/dto"
	"shop/service/shop/admin/utils"

	"github.com/liujitcn/go-utils/mapper"
)

// OrderReportCase 订单报表业务
type OrderReportCase struct {
	*biz.BaseCase
	*data.OrderStatDayRepository
	orderInfoCase *OrderInfoCase
	monthMapper   *mapper.CopierMapper[shopadminv1.OrderMonthReportItem, dto.OrderMonthReportRow]
	dayMapper     *mapper.CopierMapper[shopadminv1.OrderDayReportItem, dto.OrderDayReportRow]
}

// NewOrderReportCase 创建订单报表业务
func NewOrderReportCase(
	baseCase *biz.BaseCase,
	orderStatDayRepo *data.OrderStatDayRepository,
	orderInfoCase *OrderInfoCase,
) *OrderReportCase {
	return &OrderReportCase{
		BaseCase:               baseCase,
		OrderStatDayRepository: orderStatDayRepo,
		orderInfoCase:          orderInfoCase,
		monthMapper:            mapper.NewCopierMapper[shopadminv1.OrderMonthReportItem, dto.OrderMonthReportRow](),
		dayMapper:              mapper.NewCopierMapper[shopadminv1.OrderDayReportItem, dto.OrderDayReportRow](),
	}
}

// ListOrderDayReport 查询订单日报明细
func (c *OrderReportCase) ListOrderDayReport(ctx context.Context, req *shopadminv1.ListOrderDayReportRequest) (*shopadminv1.ListOrderDayReportResponse, error) {
	startDate, err := c.parseDate(req.GetStartDate())
	if err != nil {
		return nil, err
	}

	var endDate time.Time
	endDate, err = c.parseDate(req.GetEndDate())
	if err != nil {
		return nil, err
	}

	// 结束日期早于开始日期时，不允许继续统计日报。
	if endDate.Before(startDate) {
		return nil, errorsx.InvalidArgument("结束日期不能早于开始日期")
	}

	var rows []*dto.OrderDayReportRow
	rows, err = c.queryOrderDayReportRows(ctx, req.GetPayType(), req.GetPayChannel(), req.GetTenantId(), req.GetTenantStoreId(), startDate, endDate.AddDate(0, 0, 1))
	if err != nil {
		return nil, err
	}
	var paidMetrics *dto.OrderReportPaidMetrics
	paidMetrics, err = c.queryPaidOrderMetrics(ctx, req.GetPayType(), req.GetPayChannel(), req.GetTenantId(), req.GetTenantStoreId(), startDate, endDate.AddDate(0, 0, 1), "2006-01-02")
	if err != nil {
		return nil, err
	}

	rowMap := make(map[string]*dto.OrderDayReportRow, len(rows))
	for _, item := range rows {
		rowMap[item.Day] = item
	}

	items := make([]*shopadminv1.OrderDayReportItem, 0)
	for cursor := startDate; !cursor.After(endDate); cursor = cursor.AddDate(0, 0, 1) {
		dayKey := cursor.Format("2006-01-02")
		row, ok := rowMap[dayKey]
		// 当前日期没有统计数据时，补空行保证日期连续。
		if !ok {
			row = &dto.OrderDayReportRow{Day: dayKey}
		}
		row.PaidOrderCount = paidMetrics.PeriodOrderCounts[dayKey]
		row.PaidUserCount = paidMetrics.PeriodUserCounts[dayKey]
		items = append(items, c.toOrderDayReportItem(row))
	}

	return &shopadminv1.ListOrderDayReportResponse{
		OrderDayReports: items,
	}, nil
}

// ListOrderMonthReport 查询订单月报名细
func (c *OrderReportCase) ListOrderMonthReport(ctx context.Context, req *shopadminv1.ListOrderMonthReportRequest) (*shopadminv1.ListOrderMonthReportResponse, error) {
	startMonth, err := c.parseMonth(req.GetStartMonth())
	if err != nil {
		return nil, err
	}

	var endMonth time.Time
	endMonth, err = c.parseMonth(req.GetEndMonth())
	if err != nil {
		return nil, err
	}

	// 结束月份早于开始月份时，不允许继续统计月报。
	if endMonth.Before(startMonth) {
		return nil, errorsx.InvalidArgument("结束月份不能早于开始月份")
	}

	var rows []*dto.OrderMonthReportRow
	rows, err = c.queryOrderMonthReportRows(ctx, req.GetPayType(), req.GetPayChannel(), req.GetTenantId(), req.GetTenantStoreId(), startMonth, endMonth.AddDate(0, 1, 0))
	if err != nil {
		return nil, err
	}
	var paidMetrics *dto.OrderReportPaidMetrics
	paidMetrics, err = c.queryPaidOrderMetrics(ctx, req.GetPayType(), req.GetPayChannel(), req.GetTenantId(), req.GetTenantStoreId(), startMonth, endMonth.AddDate(0, 1, 0), "2006-01")
	if err != nil {
		return nil, err
	}

	rowMap := make(map[string]*dto.OrderMonthReportRow, len(rows))
	for _, item := range rows {
		rowMap[item.Month] = item
	}

	items := make([]*shopadminv1.OrderMonthReportItem, 0)
	for cursor := startMonth; !cursor.After(endMonth); cursor = cursor.AddDate(0, 1, 0) {
		monthKey := cursor.Format("2006-01")
		row, ok := rowMap[monthKey]
		// 当前月份没有统计数据时，补空行保证月份连续。
		if !ok {
			row = &dto.OrderMonthReportRow{Month: monthKey}
		}
		row.PaidOrderCount = paidMetrics.PeriodOrderCounts[monthKey]
		row.PaidUserCount = paidMetrics.PeriodUserCounts[monthKey]
		items = append(items, c.toOrderMonthReportItem(row))
	}

	return &shopadminv1.ListOrderMonthReportResponse{
		OrderMonthReports: items,
	}, nil
}

// SummaryOrderMonthReport 查询订单月报汇总
func (c *OrderReportCase) SummaryOrderMonthReport(ctx context.Context, req *shopadminv1.SummaryOrderMonthReportRequest) (*shopadminv1.SummaryOrderMonthReportResponse, error) {
	startMonth, err := c.parseMonth(req.GetStartMonth())
	if err != nil {
		return nil, err
	}

	var endMonth time.Time
	endMonth, err = c.parseMonth(req.GetEndMonth())
	if err != nil {
		return nil, err
	}

	// 结束月份早于开始月份时，不允许继续统计月报。
	if endMonth.Before(startMonth) {
		return nil, errorsx.InvalidArgument("结束月份不能早于开始月份")
	}

	var rows []*dto.OrderMonthReportRow
	rows, err = c.queryOrderMonthReportRows(ctx, req.GetPayType(), req.GetPayChannel(), req.GetTenantId(), req.GetTenantStoreId(), startMonth, endMonth.AddDate(0, 1, 0))
	if err != nil {
		return nil, err
	}
	var paidMetrics *dto.OrderReportPaidMetrics
	paidMetrics, err = c.queryPaidOrderMetrics(ctx, req.GetPayType(), req.GetPayChannel(), req.GetTenantId(), req.GetTenantStoreId(), startMonth, endMonth.AddDate(0, 1, 0), "2006-01")
	if err != nil {
		return nil, err
	}

	summary := &shopadminv1.SummaryOrderMonthReportResponse{}
	for _, row := range rows {
		item := c.toOrderMonthReportItem(row)
		c.appendMonthReportSummary(summary, item)
	}

	summary.PaidOrderCount = paidMetrics.TotalOrderCount
	summary.PaidUserCount = paidMetrics.TotalUserCount
	summary.NetOrderAmount = summary.PaidOrderAmount - summary.RefundOrderAmount
	summary.CustomerUnitPrice = utils.CalcPerUnit(summary.PaidOrderAmount, summary.PaidOrderCount)
	return summary, nil
}

// SummaryOrderDayReport 查询订单日报汇总
func (c *OrderReportCase) SummaryOrderDayReport(ctx context.Context, req *shopadminv1.SummaryOrderDayReportRequest) (*shopadminv1.SummaryOrderDayReportResponse, error) {
	startDate, err := c.parseDate(req.GetStartDate())
	if err != nil {
		return nil, err
	}

	var endDate time.Time
	endDate, err = c.parseDate(req.GetEndDate())
	if err != nil {
		return nil, err
	}

	// 结束日期早于开始日期时，不允许继续统计日报。
	if endDate.Before(startDate) {
		return nil, errorsx.InvalidArgument("结束日期不能早于开始日期")
	}

	var rows []*dto.OrderDayReportRow
	rows, err = c.queryOrderDayReportRows(ctx, req.GetPayType(), req.GetPayChannel(), req.GetTenantId(), req.GetTenantStoreId(), startDate, endDate.AddDate(0, 0, 1))
	if err != nil {
		return nil, err
	}
	var paidMetrics *dto.OrderReportPaidMetrics
	paidMetrics, err = c.queryPaidOrderMetrics(ctx, req.GetPayType(), req.GetPayChannel(), req.GetTenantId(), req.GetTenantStoreId(), startDate, endDate.AddDate(0, 0, 1), "2006-01-02")
	if err != nil {
		return nil, err
	}

	summary := &shopadminv1.SummaryOrderDayReportResponse{}
	for _, row := range rows {
		item := c.toOrderDayReportItem(row)
		c.appendDayReportSummary(summary, item)
	}

	summary.PaidOrderCount = paidMetrics.TotalOrderCount
	summary.PaidUserCount = paidMetrics.TotalUserCount
	summary.NetOrderAmount = summary.PaidOrderAmount - summary.RefundOrderAmount
	summary.CustomerUnitPrice = utils.CalcPerUnit(summary.PaidOrderAmount, summary.PaidOrderCount)
	return summary, nil
}

// parseMonth 解析月份字符串并归一化到当月第一天。
func (c *OrderReportCase) parseMonth(month string) (time.Time, error) {
	// 月份为空时，无法继续解析月报范围。
	if month == "" {
		return time.Time{}, errorsx.InvalidArgument("月份不能为空")
	}

	location := time.Now().Location()
	parsedTime, err := time.ParseInLocation("2006-01", month, location)
	if err != nil {
		return time.Time{}, errorsx.InvalidArgument(fmt.Sprintf("月份格式错误：%s", month))
	}
	return time.Date(parsedTime.Year(), parsedTime.Month(), 1, 0, 0, 0, 0, location), nil
}

// appendMonthReportSummary 累加月报区间汇总。
func (c *OrderReportCase) appendMonthReportSummary(summary *shopadminv1.SummaryOrderMonthReportResponse, item *shopadminv1.OrderMonthReportItem) {
	summary.PaidOrderCount += item.PaidOrderCount
	summary.PaidOrderAmount += item.PaidOrderAmount
	summary.RefundOrderCount += item.RefundOrderCount
	summary.RefundOrderAmount += item.RefundOrderAmount
	summary.GoodsCount += item.GoodsCount
}

// parseDate 解析日期字符串并归一化到当天零点。
func (c *OrderReportCase) parseDate(date string) (time.Time, error) {
	// 日期为空时，无法继续解析日报范围。
	if date == "" {
		return time.Time{}, errorsx.InvalidArgument("日期不能为空")
	}

	location := time.Now().Location()
	parsedTime, err := time.ParseInLocation("2006-01-02", date, location)
	if err != nil {
		return time.Time{}, errorsx.InvalidArgument(fmt.Sprintf("日期格式错误：%s", date))
	}
	return time.Date(parsedTime.Year(), parsedTime.Month(), parsedTime.Day(), 0, 0, 0, 0, location), nil
}

// appendDayReportSummary 累加日报区间汇总。
func (c *OrderReportCase) appendDayReportSummary(summary *shopadminv1.SummaryOrderDayReportResponse, item *shopadminv1.OrderDayReportItem) {
	summary.PaidOrderCount += item.PaidOrderCount
	summary.PaidOrderAmount += item.PaidOrderAmount
	summary.RefundOrderCount += item.RefundOrderCount
	summary.RefundOrderAmount += item.RefundOrderAmount
	summary.GoodsCount += item.GoodsCount
}

// queryPaidOrderMetrics 按支付事实时间统计报表订单数和唯一支付用户数。
func (c *OrderReportCase) queryPaidOrderMetrics(
	ctx context.Context,
	payType, payChannel int32,
	tenantID, tenantStoreID int64,
	startAt, endAt time.Time,
	periodLayout string,
) (*dto.OrderReportPaidMetrics, error) {
	useGlobalTradeScope, err := c.orderInfoCase.useGlobalOrderTradeScope(ctx, tenantID, tenantStoreID)
	if err != nil {
		return nil, err
	}
	var paidFacts []*dto.OrderPaidFact
	paidFacts, err = c.orderInfoCase.queryPaidOrderFacts(ctx, payType, payChannel, tenantID, tenantStoreID, startAt, endAt, useGlobalTradeScope)
	if err != nil {
		return nil, err
	}

	metrics := &dto.OrderReportPaidMetrics{
		PeriodOrderCounts: make(map[string]int64),
		PeriodUserCounts:  make(map[string]int64),
	}
	periodUsers := make(map[string]map[int64]struct{})
	totalUsers := make(map[int64]struct{})
	appendMetric := func(fact *dto.OrderPaidFact, userID int64) {
		period := fact.PaidAt.Format(periodLayout)
		metrics.PeriodOrderCounts[period]++
		metrics.TotalOrderCount++
		if periodUsers[period] == nil {
			periodUsers[period] = make(map[int64]struct{})
		}
		periodUsers[period][userID] = struct{}{}
		totalUsers[userID] = struct{}{}
	}
	for _, fact := range paidFacts {
		appendMetric(fact, fact.UserID)
	}
	for period, users := range periodUsers {
		metrics.PeriodUserCounts[period] = int64(len(users))
	}
	metrics.TotalUserCount = int64(len(totalUsers))
	return metrics, nil
}

// queryOrderMonthReportRows 查询月报聚合数据。
func (c *OrderReportCase) queryOrderMonthReportRows(ctx context.Context, payType, payChannel int32, tenantID, tenantStoreID int64, startAt, endAt time.Time) ([]*dto.OrderMonthReportRow, error) {
	rows := make([]*dto.OrderMonthReportRow, 0)
	query := c.Query(ctx).OrderStatDay
	groupField := utils.MonthReportGroupField(query.StatDate)
	dao := query.WithContext(ctx).
		Select(
			groupField.As("month"),
			query.PaidOrderCount.Sum().FloorDiv(1).IfNull(0).As("paid_order_count"),
			query.PaidOrderAmount.Sum().FloorDiv(1).IfNull(0).As("paid_order_amount"),
			query.RefundOrderCount.Sum().FloorDiv(1).IfNull(0).As("refund_order_count"),
			query.RefundOrderAmount.Sum().FloorDiv(1).IfNull(0).As("refund_order_amount"),
			query.PaidUserCount.Sum().FloorDiv(1).IfNull(0).As("paid_user_count"),
			query.GoodsCount.Sum().FloorDiv(1).IfNull(0).As("goods_count"),
		).
		Where(
			query.StatDate.Gte(startAt),
			query.StatDate.Lt(endAt),
		)
	// 传入支付类型时，按支付类型缩小月报统计范围。
	if payType > 0 {
		dao = dao.Where(query.PayType.Eq(payType))
	}
	// 传入支付渠道时，按支付渠道缩小月报统计范围。
	if payChannel > 0 {
		dao = dao.Where(query.PayChannel.Eq(payChannel))
	}
	// 默认租户可按租户筛选，普通租户继续受数据库租户隔离约束。
	if tenantID > 0 {
		dao = dao.Where(query.TenantID.Eq(tenantID))
	}
	if tenantStoreID > 0 {
		dao = dao.Where(query.TenantStoreID.Eq(tenantStoreID))
	}
	err := dao.Group(utils.MonthReportAliasField()).Order(utils.MonthReportAliasField()).Scan(&rows)
	return rows, err
}

// toOrderMonthReportItem 转换月报行数据。
func (c *OrderReportCase) toOrderMonthReportItem(row *dto.OrderMonthReportRow) *shopadminv1.OrderMonthReportItem {
	item := c.monthMapper.ToDTO(row)
	item.NetOrderAmount = row.PaidOrderAmount - row.RefundOrderAmount
	item.CustomerUnitPrice = utils.CalcPerUnit(row.PaidOrderAmount, row.PaidOrderCount)
	return item
}

// queryOrderDayReportRows 查询日报聚合数据。
func (c *OrderReportCase) queryOrderDayReportRows(ctx context.Context, payType, payChannel int32, tenantID, tenantStoreID int64, startAt, endAt time.Time) ([]*dto.OrderDayReportRow, error) {
	rows := make([]*dto.OrderDayReportRow, 0)
	query := c.Query(ctx).OrderStatDay
	groupField := utils.DayReportGroupField(query.StatDate)
	dao := query.WithContext(ctx).
		Select(
			groupField.As("day"),
			query.PaidOrderCount.Sum().FloorDiv(1).IfNull(0).As("paid_order_count"),
			query.PaidOrderAmount.Sum().FloorDiv(1).IfNull(0).As("paid_order_amount"),
			query.RefundOrderCount.Sum().FloorDiv(1).IfNull(0).As("refund_order_count"),
			query.RefundOrderAmount.Sum().FloorDiv(1).IfNull(0).As("refund_order_amount"),
			query.PaidUserCount.Sum().FloorDiv(1).IfNull(0).As("paid_user_count"),
			query.GoodsCount.Sum().FloorDiv(1).IfNull(0).As("goods_count"),
		).
		Where(
			query.StatDate.Gte(startAt),
			query.StatDate.Lt(endAt),
		)
	// 传入支付类型时，按支付类型缩小日报统计范围。
	if payType > 0 {
		dao = dao.Where(query.PayType.Eq(payType))
	}
	// 传入支付渠道时，按支付渠道缩小日报统计范围。
	if payChannel > 0 {
		dao = dao.Where(query.PayChannel.Eq(payChannel))
	}
	// 默认租户可按租户筛选，普通租户继续受数据库租户隔离约束。
	if tenantID > 0 {
		dao = dao.Where(query.TenantID.Eq(tenantID))
	}
	if tenantStoreID > 0 {
		dao = dao.Where(query.TenantStoreID.Eq(tenantStoreID))
	}
	err := dao.Group(utils.DayReportAliasField()).Order(utils.DayReportAliasField()).Scan(&rows)
	return rows, err
}

// toOrderDayReportItem 转换日报行数据。
func (c *OrderReportCase) toOrderDayReportItem(row *dto.OrderDayReportRow) *shopadminv1.OrderDayReportItem {
	item := c.dayMapper.ToDTO(row)
	item.NetOrderAmount = row.PaidOrderAmount - row.RefundOrderAmount
	item.CustomerUnitPrice = utils.CalcPerUnit(row.PaidOrderAmount, row.PaidOrderCount)
	return item
}

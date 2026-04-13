package biz

import (
	"context"
	"fmt"
	"time"

	adminApi "shop/api/gen/go/admin"
	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/service/admin/dto"
	"shop/service/admin/utils"

	"github.com/liujitcn/go-utils/mapper"
)

// OrderReportCase 订单报表业务
type OrderReportCase struct {
	*biz.BaseCase
	*data.OrderStatDayRepo
	monthMapper *mapper.CopierMapper[adminApi.OrderMonthReportItem, dto.OrderMonthReportRow]
	dayMapper   *mapper.CopierMapper[adminApi.OrderDayReportItem, dto.OrderDayReportRow]
}

// NewOrderReportCase 创建订单报表业务
func NewOrderReportCase(baseCase *biz.BaseCase, orderStatDayRepo *data.OrderStatDayRepo) *OrderReportCase {
	return &OrderReportCase{
		BaseCase:         baseCase,
		OrderStatDayRepo: orderStatDayRepo,
		monthMapper:      mapper.NewCopierMapper[adminApi.OrderMonthReportItem, dto.OrderMonthReportRow](),
		dayMapper:        mapper.NewCopierMapper[adminApi.OrderDayReportItem, dto.OrderDayReportRow](),
	}
}

// OrderMonthReportSummary 查询订单月报汇总
func (c *OrderReportCase) OrderMonthReportSummary(ctx context.Context, req *adminApi.OrderMonthReportSummaryRequest) (*adminApi.OrderMonthReportSummaryResponse, error) {
	startMonth, err := c.parseMonth(req.GetStartMonth())
	if err != nil {
		return nil, err
	}

	endMonth, err := c.parseMonth(req.GetEndMonth())
	if err != nil {
		return nil, err
	}

	// 结束月份早于开始月份时，不允许继续统计月报。
	if endMonth.Before(startMonth) {
		return nil, errorsx.InvalidArgument("结束月份不能早于开始月份")
	}

	rows, err := c.queryOrderMonthReportRows(ctx, req.GetPayType(), req.GetPayChannel(), startMonth, endMonth.AddDate(0, 1, 0))
	if err != nil {
		return nil, err
	}

	summary := &adminApi.OrderMonthReportSummaryResponse{}
	for _, row := range rows {
		item := c.toOrderMonthReportItem(row)
		c.appendMonthReportSummary(summary, item)
	}

	summary.NetOrderAmount = summary.PaidOrderAmount - summary.RefundOrderAmount
	summary.CustomerUnitPrice = utils.CalcPerUnit(summary.PaidOrderAmount, summary.PaidOrderCount)
	return summary, nil
}

// OrderMonthReportList 查询订单月报名细
func (c *OrderReportCase) OrderMonthReportList(ctx context.Context, req *adminApi.OrderMonthReportListRequest) (*adminApi.OrderMonthReportListResponse, error) {
	startMonth, err := c.parseMonth(req.GetStartMonth())
	if err != nil {
		return nil, err
	}

	endMonth, err := c.parseMonth(req.GetEndMonth())
	if err != nil {
		return nil, err
	}

	// 结束月份早于开始月份时，不允许继续统计月报。
	if endMonth.Before(startMonth) {
		return nil, errorsx.InvalidArgument("结束月份不能早于开始月份")
	}

	rows, err := c.queryOrderMonthReportRows(ctx, req.GetPayType(), req.GetPayChannel(), startMonth, endMonth.AddDate(0, 1, 0))
	if err != nil {
		return nil, err
	}

	rowMap := make(map[string]*dto.OrderMonthReportRow, len(rows))
	for _, item := range rows {
		rowMap[item.Month] = item
	}

	items := make([]*adminApi.OrderMonthReportItem, 0)
	cursor := startMonth
	for !cursor.After(endMonth) {
		monthKey := cursor.Format("2006-01")
		row, ok := rowMap[monthKey]
		// 当前月份没有统计数据时，补空行保证月份连续。
		if !ok {
			row = &dto.OrderMonthReportRow{Month: monthKey}
		}
		items = append(items, c.toOrderMonthReportItem(row))
		cursor = cursor.AddDate(0, 1, 0)
	}

	return &adminApi.OrderMonthReportListResponse{
		Items: items,
	}, nil
}

// OrderDayReportSummary 查询订单日报汇总
func (c *OrderReportCase) OrderDayReportSummary(ctx context.Context, req *adminApi.OrderDayReportSummaryRequest) (*adminApi.OrderDayReportSummaryResponse, error) {
	startDate, err := c.parseDate(req.GetStartDate())
	if err != nil {
		return nil, err
	}

	endDate, err := c.parseDate(req.GetEndDate())
	if err != nil {
		return nil, err
	}

	// 结束日期早于开始日期时，不允许继续统计日报。
	if endDate.Before(startDate) {
		return nil, errorsx.InvalidArgument("结束日期不能早于开始日期")
	}

	rows, err := c.queryOrderDayReportRows(ctx, req.GetPayType(), req.GetPayChannel(), startDate, endDate.AddDate(0, 0, 1))
	if err != nil {
		return nil, err
	}

	summary := &adminApi.OrderDayReportSummaryResponse{}
	for _, row := range rows {
		item := c.toOrderDayReportItem(row)
		c.appendDayReportSummary(summary, item)
	}

	summary.NetOrderAmount = summary.PaidOrderAmount - summary.RefundOrderAmount
	summary.CustomerUnitPrice = utils.CalcPerUnit(summary.PaidOrderAmount, summary.PaidOrderCount)
	return summary, nil
}

// OrderDayReportList 查询订单日报明细
func (c *OrderReportCase) OrderDayReportList(ctx context.Context, req *adminApi.OrderDayReportListRequest) (*adminApi.OrderDayReportListResponse, error) {
	startDate, err := c.parseDate(req.GetStartDate())
	if err != nil {
		return nil, err
	}

	endDate, err := c.parseDate(req.GetEndDate())
	if err != nil {
		return nil, err
	}

	// 结束日期早于开始日期时，不允许继续统计日报。
	if endDate.Before(startDate) {
		return nil, errorsx.InvalidArgument("结束日期不能早于开始日期")
	}

	rows, err := c.queryOrderDayReportRows(ctx, req.GetPayType(), req.GetPayChannel(), startDate, endDate.AddDate(0, 0, 1))
	if err != nil {
		return nil, err
	}

	rowMap := make(map[string]*dto.OrderDayReportRow, len(rows))
	for _, item := range rows {
		rowMap[item.Day] = item
	}

	items := make([]*adminApi.OrderDayReportItem, 0)
	cursor := startDate
	for !cursor.After(endDate) {
		dayKey := cursor.Format("2006-01-02")
		row, ok := rowMap[dayKey]
		// 当前日期没有统计数据时，补空行保证日期连续。
		if !ok {
			row = &dto.OrderDayReportRow{Day: dayKey}
		}
		items = append(items, c.toOrderDayReportItem(row))
		cursor = cursor.AddDate(0, 0, 1)
	}

	return &adminApi.OrderDayReportListResponse{
		Items: items,
	}, nil
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

// queryOrderMonthReportRows 查询月报聚合数据。
func (c *OrderReportCase) queryOrderMonthReportRows(ctx context.Context, payType, payChannel int32, startAt, endAt time.Time) ([]*dto.OrderMonthReportRow, error) {
	rows := make([]*dto.OrderMonthReportRow, 0)
	sql := "" +
		"SELECT DATE_FORMAT(stat_date, '%Y-%m') AS month," +
		" COALESCE(SUM(paid_order_count), 0) AS paid_order_count," +
		" COALESCE(SUM(paid_order_amount), 0) AS paid_order_amount," +
		" COALESCE(SUM(refund_order_count), 0) AS refund_order_count," +
		" COALESCE(SUM(refund_order_amount), 0) AS refund_order_amount," +
		" COALESCE(SUM(paid_user_count), 0) AS paid_user_count," +
		" COALESCE(SUM(goods_count), 0) AS goods_count" +
		" FROM order_stat_day" +
		" WHERE deleted_at IS NULL AND stat_date >= ? AND stat_date < ?"
	args := []any{startAt, endAt}
	// 传入支付类型时，按支付类型缩小月报统计范围。
	if payType > 0 {
		sql += " AND pay_type = ?"
		args = append(args, payType)
	}
	// 传入支付渠道时，按支付渠道缩小月报统计范围。
	if payChannel > 0 {
		sql += " AND pay_channel = ?"
		args = append(args, payChannel)
	}
	sql += "" +
		" GROUP BY DATE_FORMAT(stat_date, '%Y-%m')" +
		" ORDER BY month ASC"
	err := c.Query(ctx).OrderStatDay.WithContext(ctx).UnderlyingDB().Raw(sql, args...).Scan(&rows).Error
	return rows, err
}

// queryOrderDayReportRows 查询日报聚合数据。
func (c *OrderReportCase) queryOrderDayReportRows(ctx context.Context, payType, payChannel int32, startAt, endAt time.Time) ([]*dto.OrderDayReportRow, error) {
	rows := make([]*dto.OrderDayReportRow, 0)
	sql := "" +
		"SELECT DATE_FORMAT(stat_date, '%Y-%m-%d') AS day," +
		" COALESCE(SUM(paid_order_count), 0) AS paid_order_count," +
		" COALESCE(SUM(paid_order_amount), 0) AS paid_order_amount," +
		" COALESCE(SUM(refund_order_count), 0) AS refund_order_count," +
		" COALESCE(SUM(refund_order_amount), 0) AS refund_order_amount," +
		" COALESCE(SUM(paid_user_count), 0) AS paid_user_count," +
		" COALESCE(SUM(goods_count), 0) AS goods_count" +
		" FROM order_stat_day" +
		" WHERE deleted_at IS NULL AND stat_date >= ? AND stat_date < ?"
	args := []any{startAt, endAt}
	// 传入支付类型时，按支付类型缩小日报统计范围。
	if payType > 0 {
		sql += " AND pay_type = ?"
		args = append(args, payType)
	}
	// 传入支付渠道时，按支付渠道缩小日报统计范围。
	if payChannel > 0 {
		sql += " AND pay_channel = ?"
		args = append(args, payChannel)
	}
	sql += "" +
		" GROUP BY DATE_FORMAT(stat_date, '%Y-%m-%d')" +
		" ORDER BY day ASC"
	err := c.Query(ctx).OrderStatDay.WithContext(ctx).UnderlyingDB().Raw(sql, args...).Scan(&rows).Error
	return rows, err
}

// appendMonthReportSummary 累加月报区间汇总。
func (c *OrderReportCase) appendMonthReportSummary(summary *adminApi.OrderMonthReportSummaryResponse, item *adminApi.OrderMonthReportItem) {
	summary.PaidOrderCount += item.PaidOrderCount
	summary.PaidOrderAmount += item.PaidOrderAmount
	summary.RefundOrderCount += item.RefundOrderCount
	summary.RefundOrderAmount += item.RefundOrderAmount
	summary.PaidUserCount += item.PaidUserCount
	summary.GoodsCount += item.GoodsCount
}

// toOrderMonthReportItem 转换月报行数据。
func (c *OrderReportCase) toOrderMonthReportItem(row *dto.OrderMonthReportRow) *adminApi.OrderMonthReportItem {
	item := c.monthMapper.ToDTO(row)
	item.NetOrderAmount = row.PaidOrderAmount - row.RefundOrderAmount
	item.CustomerUnitPrice = utils.CalcPerUnit(row.PaidOrderAmount, row.PaidOrderCount)
	return item
}

// appendDayReportSummary 累加日报区间汇总。
func (c *OrderReportCase) appendDayReportSummary(summary *adminApi.OrderDayReportSummaryResponse, item *adminApi.OrderDayReportItem) {
	summary.PaidOrderCount += item.PaidOrderCount
	summary.PaidOrderAmount += item.PaidOrderAmount
	summary.RefundOrderCount += item.RefundOrderCount
	summary.RefundOrderAmount += item.RefundOrderAmount
	summary.PaidUserCount += item.PaidUserCount
	summary.GoodsCount += item.GoodsCount
}

// toOrderDayReportItem 转换日报行数据。
func (c *OrderReportCase) toOrderDayReportItem(row *dto.OrderDayReportRow) *adminApi.OrderDayReportItem {
	item := c.dayMapper.ToDTO(row)
	item.NetOrderAmount = row.PaidOrderAmount - row.RefundOrderAmount
	item.CustomerUnitPrice = utils.CalcPerUnit(row.PaidOrderAmount, row.PaidOrderCount)
	return item
}

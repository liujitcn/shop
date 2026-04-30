package biz

import (
	"context"
	"fmt"
	"time"

	adminv1 "shop/api/gen/go/admin/v1"
	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/service/admin/dto"
	"shop/service/admin/utils"

	"github.com/liujitcn/go-utils/mapper"
)

// GoodsReportCase 商品报表业务。
type GoodsReportCase struct {
	*biz.BaseCase
	*data.GoodsStatDayRepository
	monthMapper *mapper.CopierMapper[adminv1.GoodsMonthReportItem, dto.GoodsMonthReportRow]
	dayMapper   *mapper.CopierMapper[adminv1.GoodsDayReportItem, dto.GoodsDayReportRow]
}

// NewGoodsReportCase 创建商品报表业务。
func NewGoodsReportCase(baseCase *biz.BaseCase, goodsStatDayRepo *data.GoodsStatDayRepository) *GoodsReportCase {
	return &GoodsReportCase{
		BaseCase:               baseCase,
		GoodsStatDayRepository: goodsStatDayRepo,
		monthMapper:            mapper.NewCopierMapper[adminv1.GoodsMonthReportItem, dto.GoodsMonthReportRow](),
		dayMapper:              mapper.NewCopierMapper[adminv1.GoodsDayReportItem, dto.GoodsDayReportRow](),
	}
}

// SummaryGoodsMonthReport 查询商品月报汇总。
func (c *GoodsReportCase) SummaryGoodsMonthReport(ctx context.Context, req *adminv1.SummaryGoodsMonthReportRequest) (*adminv1.SummaryGoodsMonthReportResponse, error) {
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

	var rows []*dto.GoodsMonthReportRow
	rows, err = c.queryGoodsMonthReportRows(ctx, startMonth, endMonth.AddDate(0, 1, 0))
	if err != nil {
		return nil, err
	}

	summary := &adminv1.SummaryGoodsMonthReportResponse{}
	for _, row := range rows {
		item := c.toGoodsMonthReportItem(row)
		c.appendMonthReportSummary(summary, item)
	}
	summary.CartConversionRate = utils.CalcRatio(summary.CartCount, summary.ViewCount)
	summary.OrderConversionRate = utils.CalcRatio(summary.OrderCount, summary.CartCount)
	summary.PayConversionRate = utils.CalcRatio(summary.PayCount, summary.ViewCount)
	summary.PayUnitPrice = utils.CalcPerUnit(summary.PayAmount, summary.PayGoodsNum)
	return summary, nil
}

// ListGoodsMonthReports 查询商品月报名细。
func (c *GoodsReportCase) ListGoodsMonthReports(ctx context.Context, req *adminv1.ListGoodsMonthReportsRequest) (*adminv1.ListGoodsMonthReportsResponse, error) {
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

	var rows []*dto.GoodsMonthReportRow
	rows, err = c.queryGoodsMonthReportRows(ctx, startMonth, endMonth.AddDate(0, 1, 0))
	if err != nil {
		return nil, err
	}

	rowMap := make(map[string]*dto.GoodsMonthReportRow, len(rows))
	for _, item := range rows {
		rowMap[item.Month] = item
	}

	items := make([]*adminv1.GoodsMonthReportItem, 0)
	for cursor := startMonth; !cursor.After(endMonth); cursor = cursor.AddDate(0, 1, 0) {
		monthKey := cursor.Format("2006-01")
		row, ok := rowMap[monthKey]
		// 当前月份没有统计数据时，补空行保证月份连续。
		if !ok {
			row = &dto.GoodsMonthReportRow{Month: monthKey}
		}
		items = append(items, c.toGoodsMonthReportItem(row))
	}

	return &adminv1.ListGoodsMonthReportsResponse{GoodsMonthReports: items}, nil
}

// SummaryGoodsDayReport 查询商品日报汇总。
func (c *GoodsReportCase) SummaryGoodsDayReport(ctx context.Context, req *adminv1.SummaryGoodsDayReportRequest) (*adminv1.SummaryGoodsDayReportResponse, error) {
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

	var rows []*dto.GoodsDayReportRow
	rows, err = c.queryGoodsDayReportRows(ctx, startDate, endDate.AddDate(0, 0, 1))
	if err != nil {
		return nil, err
	}

	summary := &adminv1.SummaryGoodsDayReportResponse{}
	for _, row := range rows {
		item := c.toGoodsDayReportItem(row)
		c.appendDayReportSummary(summary, item)
	}
	summary.CartConversionRate = utils.CalcRatio(summary.CartCount, summary.ViewCount)
	summary.OrderConversionRate = utils.CalcRatio(summary.OrderCount, summary.CartCount)
	summary.PayConversionRate = utils.CalcRatio(summary.PayCount, summary.ViewCount)
	summary.PayUnitPrice = utils.CalcPerUnit(summary.PayAmount, summary.PayGoodsNum)
	return summary, nil
}

// ListGoodsDayReports 查询商品日报明细。
func (c *GoodsReportCase) ListGoodsDayReports(ctx context.Context, req *adminv1.ListGoodsDayReportsRequest) (*adminv1.ListGoodsDayReportsResponse, error) {
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

	var rows []*dto.GoodsDayReportRow
	rows, err = c.queryGoodsDayReportRows(ctx, startDate, endDate.AddDate(0, 0, 1))
	if err != nil {
		return nil, err
	}

	rowMap := make(map[string]*dto.GoodsDayReportRow, len(rows))
	for _, item := range rows {
		rowMap[item.Day] = item
	}

	items := make([]*adminv1.GoodsDayReportItem, 0)
	for cursor := startDate; !cursor.After(endDate); cursor = cursor.AddDate(0, 0, 1) {
		dayKey := cursor.Format("2006-01-02")
		row, ok := rowMap[dayKey]
		// 当前日期没有统计数据时，补空行保证日期连续。
		if !ok {
			row = &dto.GoodsDayReportRow{Day: dayKey}
		}
		items = append(items, c.toGoodsDayReportItem(row))
	}

	return &adminv1.ListGoodsDayReportsResponse{GoodsDayReports: items}, nil
}

// parseMonth 解析月份字符串并归一化到当月第一天。
func (c *GoodsReportCase) parseMonth(month string) (time.Time, error) {
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
func (c *GoodsReportCase) parseDate(date string) (time.Time, error) {
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

// queryGoodsMonthReportRows 查询商品月报聚合数据。
func (c *GoodsReportCase) queryGoodsMonthReportRows(ctx context.Context, startAt, endAt time.Time) ([]*dto.GoodsMonthReportRow, error) {
	rows := make([]*dto.GoodsMonthReportRow, 0)
	query := c.Query(ctx).GoodsStatDay
	groupField := utils.MonthReportGroupField(query.StatDate)
	err := query.WithContext(ctx).
		Select(
			groupField.As("month"),
			query.ViewCount.Sum().FloorDiv(1).IfNull(0).As("view_count"),
			query.CollectCount.Sum().FloorDiv(1).IfNull(0).As("collect_count"),
			query.CartCount.Sum().FloorDiv(1).IfNull(0).As("cart_count"),
			query.OrderCount.Sum().FloorDiv(1).IfNull(0).As("order_count"),
			query.PayCount.Sum().FloorDiv(1).IfNull(0).As("pay_count"),
			query.PayGoodsNum.Sum().FloorDiv(1).IfNull(0).As("pay_goods_num"),
			query.PayAmount.Sum().FloorDiv(1).IfNull(0).As("pay_amount"),
		).
		Where(
			query.DeletedAt.IsNull(),
			query.StatDate.Gte(startAt),
			query.StatDate.Lt(endAt),
		).
		Group(utils.MonthReportAliasField()).
		Order(utils.MonthReportAliasField()).
		Scan(&rows)
	return rows, err
}

// queryGoodsDayReportRows 查询商品日报聚合数据。
func (c *GoodsReportCase) queryGoodsDayReportRows(ctx context.Context, startAt, endAt time.Time) ([]*dto.GoodsDayReportRow, error) {
	rows := make([]*dto.GoodsDayReportRow, 0)
	query := c.Query(ctx).GoodsStatDay
	groupField := utils.DayReportGroupField(query.StatDate)
	err := query.WithContext(ctx).
		Select(
			groupField.As("day"),
			query.ViewCount.Sum().FloorDiv(1).IfNull(0).As("view_count"),
			query.CollectCount.Sum().FloorDiv(1).IfNull(0).As("collect_count"),
			query.CartCount.Sum().FloorDiv(1).IfNull(0).As("cart_count"),
			query.OrderCount.Sum().FloorDiv(1).IfNull(0).As("order_count"),
			query.PayCount.Sum().FloorDiv(1).IfNull(0).As("pay_count"),
			query.PayGoodsNum.Sum().FloorDiv(1).IfNull(0).As("pay_goods_num"),
			query.PayAmount.Sum().FloorDiv(1).IfNull(0).As("pay_amount"),
		).
		Where(
			query.DeletedAt.IsNull(),
			query.StatDate.Gte(startAt),
			query.StatDate.Lt(endAt),
		).
		Group(utils.DayReportAliasField()).
		Order(utils.DayReportAliasField()).
		Scan(&rows)
	return rows, err
}

// appendMonthReportSummary 累加商品月报区间汇总。
func (c *GoodsReportCase) appendMonthReportSummary(summary *adminv1.SummaryGoodsMonthReportResponse, item *adminv1.GoodsMonthReportItem) {
	summary.ViewCount += item.ViewCount
	summary.CollectCount += item.CollectCount
	summary.CartCount += item.CartCount
	summary.OrderCount += item.OrderCount
	summary.PayCount += item.PayCount
	summary.PayGoodsNum += item.PayGoodsNum
	summary.PayAmount += item.PayAmount
}

// toGoodsMonthReportItem 转换商品月报行数据。
func (c *GoodsReportCase) toGoodsMonthReportItem(row *dto.GoodsMonthReportRow) *adminv1.GoodsMonthReportItem {
	item := c.monthMapper.ToDTO(row)
	item.CartConversionRate = utils.CalcRatio(row.CartCount, row.ViewCount)
	item.OrderConversionRate = utils.CalcRatio(row.OrderCount, row.CartCount)
	item.PayConversionRate = utils.CalcRatio(row.PayCount, row.ViewCount)
	item.PayUnitPrice = utils.CalcPerUnit(row.PayAmount, row.PayGoodsNum)
	return item
}

// appendDayReportSummary 累加商品日报区间汇总。
func (c *GoodsReportCase) appendDayReportSummary(summary *adminv1.SummaryGoodsDayReportResponse, item *adminv1.GoodsDayReportItem) {
	summary.ViewCount += item.ViewCount
	summary.CollectCount += item.CollectCount
	summary.CartCount += item.CartCount
	summary.OrderCount += item.OrderCount
	summary.PayCount += item.PayCount
	summary.PayGoodsNum += item.PayGoodsNum
	summary.PayAmount += item.PayAmount
}

// toGoodsDayReportItem 转换商品日报行数据。
func (c *GoodsReportCase) toGoodsDayReportItem(row *dto.GoodsDayReportRow) *adminv1.GoodsDayReportItem {
	item := c.dayMapper.ToDTO(row)
	item.CartConversionRate = utils.CalcRatio(row.CartCount, row.ViewCount)
	item.OrderConversionRate = utils.CalcRatio(row.OrderCount, row.CartCount)
	item.PayConversionRate = utils.CalcRatio(row.PayCount, row.ViewCount)
	item.PayUnitPrice = utils.CalcPerUnit(row.PayAmount, row.PayGoodsNum)
	return item
}

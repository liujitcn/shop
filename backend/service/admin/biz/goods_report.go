package biz

import (
	"context"
	"fmt"
	"time"

	adminApi "shop/api/gen/go/admin"
	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"shop/service/admin/dto"
	"shop/service/admin/utils"

	"github.com/liujitcn/go-utils/mapper"
)

// GoodsReportCase 商品报表业务。
type GoodsReportCase struct {
	*biz.BaseCase
	*data.GoodsStatDayRepo
	monthMapper *mapper.CopierMapper[adminApi.GoodsMonthReportItem, dto.GoodsMonthReportRow]
	dayMapper   *mapper.CopierMapper[adminApi.GoodsDayReportItem, dto.GoodsDayReportRow]
}

// NewGoodsReportCase 创建商品报表业务。
func NewGoodsReportCase(baseCase *biz.BaseCase, goodsStatDayRepo *data.GoodsStatDayRepo) *GoodsReportCase {
	return &GoodsReportCase{
		BaseCase:         baseCase,
		GoodsStatDayRepo: goodsStatDayRepo,
		monthMapper:      mapper.NewCopierMapper[adminApi.GoodsMonthReportItem, dto.GoodsMonthReportRow](),
		dayMapper:        mapper.NewCopierMapper[adminApi.GoodsDayReportItem, dto.GoodsDayReportRow](),
	}
}

// GoodsMonthReportSummary 查询商品月报汇总。
func (c *GoodsReportCase) GoodsMonthReportSummary(ctx context.Context, req *adminApi.GoodsMonthReportSummaryRequest) (*adminApi.GoodsMonthReportSummaryResponse, error) {
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

	rows, err := c.queryGoodsMonthReportRows(ctx, startMonth, endMonth.AddDate(0, 1, 0))
	if err != nil {
		return nil, err
	}

	summary := &adminApi.GoodsMonthReportSummaryResponse{}
	for _, row := range rows {
		item := c.toGoodsMonthReportItem(row)
		c.appendMonthReportSummary(summary, item)
	}
	c.fillMonthReportDerived(summary)
	return summary, nil
}

// GoodsMonthReportList 查询商品月报名细。
func (c *GoodsReportCase) GoodsMonthReportList(ctx context.Context, req *adminApi.GoodsMonthReportListRequest) (*adminApi.GoodsMonthReportListResponse, error) {
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

	rows, err := c.queryGoodsMonthReportRows(ctx, startMonth, endMonth.AddDate(0, 1, 0))
	if err != nil {
		return nil, err
	}

	rowMap := make(map[string]*dto.GoodsMonthReportRow, len(rows))
	for _, item := range rows {
		rowMap[item.Month] = item
	}

	items := make([]*adminApi.GoodsMonthReportItem, 0)
	for cursor := startMonth; !cursor.After(endMonth); cursor = cursor.AddDate(0, 1, 0) {
		monthKey := cursor.Format("2006-01")
		row, ok := rowMap[monthKey]
		// 当前月份没有统计数据时，补空行保证月份连续。
		if !ok {
			row = &dto.GoodsMonthReportRow{Month: monthKey}
		}
		items = append(items, c.toGoodsMonthReportItem(row))
	}

	return &adminApi.GoodsMonthReportListResponse{Items: items}, nil
}

// GoodsDayReportSummary 查询商品日报汇总。
func (c *GoodsReportCase) GoodsDayReportSummary(ctx context.Context, req *adminApi.GoodsDayReportSummaryRequest) (*adminApi.GoodsDayReportSummaryResponse, error) {
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

	rows, err := c.queryGoodsDayReportRows(ctx, startDate, endDate.AddDate(0, 0, 1))
	if err != nil {
		return nil, err
	}

	summary := &adminApi.GoodsDayReportSummaryResponse{}
	for _, row := range rows {
		item := c.toGoodsDayReportItem(row)
		c.appendDayReportSummary(summary, item)
	}
	c.fillDayReportDerived(summary)
	return summary, nil
}

// GoodsDayReportList 查询商品日报明细。
func (c *GoodsReportCase) GoodsDayReportList(ctx context.Context, req *adminApi.GoodsDayReportListRequest) (*adminApi.GoodsDayReportListResponse, error) {
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

	rows, err := c.queryGoodsDayReportRows(ctx, startDate, endDate.AddDate(0, 0, 1))
	if err != nil {
		return nil, err
	}

	rowMap := make(map[string]*dto.GoodsDayReportRow, len(rows))
	for _, item := range rows {
		rowMap[item.Day] = item
	}

	items := make([]*adminApi.GoodsDayReportItem, 0)
	for cursor := startDate; !cursor.After(endDate); cursor = cursor.AddDate(0, 0, 1) {
		dayKey := cursor.Format("2006-01-02")
		row, ok := rowMap[dayKey]
		// 当前日期没有统计数据时，补空行保证日期连续。
		if !ok {
			row = &dto.GoodsDayReportRow{Day: dayKey}
		}
		items = append(items, c.toGoodsDayReportItem(row))
	}

	return &adminApi.GoodsDayReportListResponse{Items: items}, nil
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
	sql, args := c.buildGoodsMonthReportQuery(startAt, endAt)
	err := c.Query(ctx).GoodsStatDay.WithContext(ctx).UnderlyingDB().Raw(sql, args...).Scan(&rows).Error
	return rows, err
}

// queryGoodsDayReportRows 查询商品日报聚合数据。
func (c *GoodsReportCase) queryGoodsDayReportRows(ctx context.Context, startAt, endAt time.Time) ([]*dto.GoodsDayReportRow, error) {
	rows := make([]*dto.GoodsDayReportRow, 0)
	sql, args := c.buildGoodsDayReportQuery(startAt, endAt)
	err := c.Query(ctx).GoodsStatDay.WithContext(ctx).UnderlyingDB().Raw(sql, args...).Scan(&rows).Error
	return rows, err
}

// buildGoodsMonthReportQuery 构建商品月报聚合查询。
func (c *GoodsReportCase) buildGoodsMonthReportQuery(startAt, endAt time.Time) (string, []any) {
	sql := "" +
		"SELECT DATE_FORMAT(stat_date, '%Y-%m') AS month," +
		" COALESCE(SUM(view_count), 0) AS view_count," +
		" COALESCE(SUM(collect_count), 0) AS collect_count," +
		" COALESCE(SUM(cart_count), 0) AS cart_count," +
		" COALESCE(SUM(order_count), 0) AS order_count," +
		" COALESCE(SUM(pay_count), 0) AS pay_count," +
		" COALESCE(SUM(pay_goods_num), 0) AS pay_goods_num," +
		" COALESCE(SUM(pay_amount), 0) AS pay_amount" +
		" FROM " + models.TableNameGoodsStatDay +
		" WHERE deleted_at IS NULL AND stat_date >= ? AND stat_date < ?" +
		" GROUP BY DATE_FORMAT(stat_date, '%Y-%m')" +
		" ORDER BY month"
	return sql, []any{startAt, endAt}
}

// buildGoodsDayReportQuery 构建商品日报聚合查询。
func (c *GoodsReportCase) buildGoodsDayReportQuery(startAt, endAt time.Time) (string, []any) {
	sql := "" +
		"SELECT DATE_FORMAT(stat_date, '%Y-%m-%d') AS day," +
		" COALESCE(SUM(view_count), 0) AS view_count," +
		" COALESCE(SUM(collect_count), 0) AS collect_count," +
		" COALESCE(SUM(cart_count), 0) AS cart_count," +
		" COALESCE(SUM(order_count), 0) AS order_count," +
		" COALESCE(SUM(pay_count), 0) AS pay_count," +
		" COALESCE(SUM(pay_goods_num), 0) AS pay_goods_num," +
		" COALESCE(SUM(pay_amount), 0) AS pay_amount" +
		" FROM " + models.TableNameGoodsStatDay +
		" WHERE deleted_at IS NULL AND stat_date >= ? AND stat_date < ?" +
		" GROUP BY DATE_FORMAT(stat_date, '%Y-%m-%d')" +
		" ORDER BY day"
	return sql, []any{startAt, endAt}
}

// appendMonthReportSummary 累加商品月报区间汇总。
func (c *GoodsReportCase) appendMonthReportSummary(summary *adminApi.GoodsMonthReportSummaryResponse, item *adminApi.GoodsMonthReportItem) {
	summary.ViewCount += item.ViewCount
	summary.CollectCount += item.CollectCount
	summary.CartCount += item.CartCount
	summary.OrderCount += item.OrderCount
	summary.PayCount += item.PayCount
	summary.PayGoodsNum += item.PayGoodsNum
	summary.PayAmount += item.PayAmount
}

// toGoodsMonthReportItem 转换商品月报行数据。
func (c *GoodsReportCase) toGoodsMonthReportItem(row *dto.GoodsMonthReportRow) *adminApi.GoodsMonthReportItem {
	item := c.monthMapper.ToDTO(row)
	item.CartConversionRate = utils.CalcRatio(row.CartCount, row.ViewCount)
	item.OrderConversionRate = utils.CalcRatio(row.OrderCount, row.CartCount)
	item.PayConversionRate = utils.CalcRatio(row.PayCount, row.ViewCount)
	item.PayUnitPrice = utils.CalcPerUnit(row.PayAmount, row.PayGoodsNum)
	return item
}

// fillMonthReportDerived 补齐商品月报汇总派生字段。
func (c *GoodsReportCase) fillMonthReportDerived(summary *adminApi.GoodsMonthReportSummaryResponse) {
	summary.CartConversionRate = utils.CalcRatio(summary.CartCount, summary.ViewCount)
	summary.OrderConversionRate = utils.CalcRatio(summary.OrderCount, summary.CartCount)
	summary.PayConversionRate = utils.CalcRatio(summary.PayCount, summary.ViewCount)
	summary.PayUnitPrice = utils.CalcPerUnit(summary.PayAmount, summary.PayGoodsNum)
}

// appendDayReportSummary 累加商品日报区间汇总。
func (c *GoodsReportCase) appendDayReportSummary(summary *adminApi.GoodsDayReportSummaryResponse, item *adminApi.GoodsDayReportItem) {
	summary.ViewCount += item.ViewCount
	summary.CollectCount += item.CollectCount
	summary.CartCount += item.CartCount
	summary.OrderCount += item.OrderCount
	summary.PayCount += item.PayCount
	summary.PayGoodsNum += item.PayGoodsNum
	summary.PayAmount += item.PayAmount
}

// toGoodsDayReportItem 转换商品日报行数据。
func (c *GoodsReportCase) toGoodsDayReportItem(row *dto.GoodsDayReportRow) *adminApi.GoodsDayReportItem {
	item := c.dayMapper.ToDTO(row)
	item.CartConversionRate = utils.CalcRatio(row.CartCount, row.ViewCount)
	item.OrderConversionRate = utils.CalcRatio(row.OrderCount, row.CartCount)
	item.PayConversionRate = utils.CalcRatio(row.PayCount, row.ViewCount)
	item.PayUnitPrice = utils.CalcPerUnit(row.PayAmount, row.PayGoodsNum)
	return item
}

// fillDayReportDerived 补齐商品日报汇总派生字段。
func (c *GoodsReportCase) fillDayReportDerived(summary *adminApi.GoodsDayReportSummaryResponse) {
	summary.CartConversionRate = utils.CalcRatio(summary.CartCount, summary.ViewCount)
	summary.OrderConversionRate = utils.CalcRatio(summary.OrderCount, summary.CartCount)
	summary.PayConversionRate = utils.CalcRatio(summary.PayCount, summary.ViewCount)
	summary.PayUnitPrice = utils.CalcPerUnit(summary.PayAmount, summary.PayGoodsNum)
}

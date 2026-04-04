package biz

import (
	"context"
	"fmt"
	"time"

	adminApi "shop/api/gen/go/admin"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/service/admin/dto"
	"shop/service/admin/utils"

	"github.com/liujitcn/go-utils/mapper"
)

// OrderReportCase 订单报表业务
type OrderReportCase struct {
	*biz.BaseCase
	*data.OrderStatDayRepo
	mapper *mapper.CopierMapper[adminApi.OrderMonthReportItem, dto.OrderMonthReportRow]
}

// NewOrderReportCase 创建订单报表业务
func NewOrderReportCase(baseCase *biz.BaseCase, orderStatDayRepo *data.OrderStatDayRepo) *OrderReportCase {
	return &OrderReportCase{
		BaseCase:         baseCase,
		OrderStatDayRepo: orderStatDayRepo,
		mapper:           mapper.NewCopierMapper[adminApi.OrderMonthReportItem, dto.OrderMonthReportRow](),
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

	if endMonth.Before(startMonth) {
		return nil, fmt.Errorf("结束月份不能早于开始月份")
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

	if endMonth.Before(startMonth) {
		return nil, fmt.Errorf("结束月份不能早于开始月份")
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

// parseMonth 解析月份字符串并归一化到当月第一天。
func (c *OrderReportCase) parseMonth(month string) (time.Time, error) {
	if month == "" {
		return time.Time{}, fmt.Errorf("月份不能为空")
	}

	location := time.Now().Location()
	parsedTime, err := time.ParseInLocation("2006-01", month, location)
	if err != nil {
		return time.Time{}, fmt.Errorf("月份格式错误：%s", month)
	}
	return time.Date(parsedTime.Year(), parsedTime.Month(), 1, 0, 0, 0, 0, location), nil
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
	if payType > 0 {
		sql += " AND pay_type = ?"
		args = append(args, payType)
	}
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

// appendMonthReportSummary 累加月报区间汇总。
func (c *OrderReportCase) appendMonthReportSummary(summary *adminApi.OrderMonthReportSummaryResponse, item *adminApi.OrderMonthReportItem) {
	summary.PaidOrderCount += item.PaidOrderCount
	summary.PaidOrderAmount += item.PaidOrderAmount
	summary.RefundOrderCount += item.RefundOrderCount
	summary.RefundOrderAmount += item.RefundOrderAmount
	summary.PaidUserCount += item.PaidUserCount
	summary.GoodsCount += item.GoodsCount
}

func (c *OrderReportCase) toOrderMonthReportItem(row *dto.OrderMonthReportRow) *adminApi.OrderMonthReportItem {
	item := c.mapper.ToDTO(row)
	item.NetOrderAmount = row.PaidOrderAmount - row.RefundOrderAmount
	item.CustomerUnitPrice = utils.CalcPerUnit(row.PaidOrderAmount, row.PaidOrderCount)
	return item
}

package utils

import (
	"time"

	_const "shop/pkg/const"

	commonv1 "shop/api/gen/go/common/v1"

	"gorm.io/gen/field"
)

// GetAnalyticsTimeRange 根据时间类型返回当前统计周期的起止时间。
func GetAnalyticsTimeRange(timeType commonv1.AnalyticsTimeType) (time.Time, time.Time) {
	now := time.Now()
	// 按统计维度返回当前周期的起止区间。
	switch timeType {
	case commonv1.AnalyticsTimeType(_const.ANALYTICS_TIME_TYPE_YEAR):
		startAt := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		return startAt, startAt.AddDate(1, 0, 0)
	case commonv1.AnalyticsTimeType(_const.ANALYTICS_TIME_TYPE_MONTH):
		startAt := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		return startAt, startAt.AddDate(0, 1, 0)
	default:
		weekday := int(now.Weekday())
		// Go 的周日返回 0，这里转换成 7 以便统一按周一作为起点。
		if weekday == 0 {
			weekday = 7
		}
		startAt := time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
		return startAt, startAt.AddDate(0, 0, 7)
	}
}

// GetPreviousAnalyticsTimeRange 根据当前周期起点返回上一个对比周期。
func GetPreviousAnalyticsTimeRange(timeType commonv1.AnalyticsTimeType, currentStartAt time.Time) (time.Time, time.Time) {
	// 按当前统计维度回推上一个完整对比周期。
	switch timeType {
	case commonv1.AnalyticsTimeType(_const.ANALYTICS_TIME_TYPE_YEAR):
		startAt := currentStartAt.AddDate(-1, 0, 0)
		return startAt, currentStartAt
	case commonv1.AnalyticsTimeType(_const.ANALYTICS_TIME_TYPE_MONTH):
		startAt := currentStartAt.AddDate(0, -1, 0)
		return startAt, currentStartAt
	default:
		startAt := currentStartAt.AddDate(0, 0, -7)
		return startAt, currentStartAt
	}
}

// FormatAnalyticsAxis 根据时间类型格式化图表横轴标签。
func FormatAnalyticsAxis(timeType commonv1.AnalyticsTimeType, index int, startAt time.Time) string {
	// 按统计维度生成前端图表横轴文案。
	switch timeType {
	case commonv1.AnalyticsTimeType(_const.ANALYTICS_TIME_TYPE_YEAR):
		months := []string{"1月", "2月", "3月", "4月", "5月", "6月", "7月", "8月", "9月", "10月", "11月", "12月"}
		return months[index]
	case commonv1.AnalyticsTimeType(_const.ANALYTICS_TIME_TYPE_MONTH):
		return startAt.AddDate(0, 0, index).Format("01-02")
	default:
		labels := []string{"一", "二", "三", "四", "五", "六", "日"}
		return "周" + labels[index]
	}
}

// GetAnalyticsGroupFieldByColumn 返回指定时间字段的 gorm/gen 聚合表达式和完整横轴数据。
func GetAnalyticsGroupFieldByColumn(timeType commonv1.AnalyticsTimeType, startAt, endAt time.Time, column field.Time) (field.Expr, []string) {
	axis := make([]string, 0)
	// 按统计维度返回 gorm/gen 可组合的 SQL 聚合表达式和完整横轴。
	switch timeType {
	case commonv1.AnalyticsTimeType(_const.ANALYTICS_TIME_TYPE_YEAR):
		for i := 0; i < 12; i++ {
			axis = append(axis, FormatAnalyticsAxis(timeType, i, startAt))
		}
		return column.Month(), axis
	case commonv1.AnalyticsTimeType(_const.ANALYTICS_TIME_TYPE_MONTH):
		monthDays := endAt.AddDate(0, 0, -1).Day()
		for i := 0; i < monthDays; i++ {
			axis = append(axis, FormatAnalyticsAxis(timeType, i, startAt))
		}
		return column.Day(), axis
	default:
		for i := 0; i < 7; i++ {
			axis = append(axis, FormatAnalyticsAxis(timeType, i, startAt))
		}
		return column.DayOfWeek().Add(5).Mod(7).Add(1), axis
	}
}

// AnalyticsGroupAliasField 返回趋势查询分组别名字段。
func AnalyticsGroupAliasField() field.Expr {
	return field.NewField("", "key")
}

// MonthReportGroupField 返回月报使用的日期分组字段。
func MonthReportGroupField(column field.Time) field.Expr {
	return column.DateFormat("%Y-%m")
}

// MonthReportAliasField 返回月报查询分组别名字段。
func MonthReportAliasField() field.Expr {
	return field.NewField("", "month")
}

// DayReportGroupField 返回日报使用的日期分组字段。
func DayReportGroupField(column field.Time) field.Expr {
	return column.DateFormat("%Y-%m-%d")
}

// DayReportAliasField 返回日报查询分组别名字段。
func DayReportAliasField() field.Expr {
	return field.NewField("", "day")
}

// CountAtLeastOccurrences 统计出现次数达到阈值的编号数量。
func CountAtLeastOccurrences(values []int64, threshold int) int64 {
	countMap := make(map[int64]int, len(values))
	for _, value := range values {
		countMap[value]++
	}

	var total int64
	for _, count := range countMap {
		// 仅统计达到指定出现次数的编号。
		if count >= threshold {
			total++
		}
	}
	return total
}

// CalcGrowthRate 计算环比增长百分比。
func CalcGrowthRate(prev, curr int64) int64 {
	// 上期为 0 时，按是否有新增结果返回 0 或 100，避免除零。
	if prev == 0 {
		// 两期都为 0 时，视为没有增长。
		if curr == 0 {
			return 0
		}
		return 100
	}
	return (curr - prev) * 100 / prev
}

// CalcRatio 计算千分比结果，供前端按固定精度展示。
func CalcRatio(numerator, denominator int64) int64 {
	// 分母为 0 时无法计算比例，直接返回 0。
	if denominator == 0 {
		return 0
	}
	return numerator * 1000 / denominator
}

// CalcPerUnit 计算单次平均值。
func CalcPerUnit(total, count int64) int64 {
	// 样本数为 0 时无法计算均值，直接返回 0。
	if count == 0 {
		return 0
	}
	return total / count
}

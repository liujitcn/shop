package utils

import (
	"time"

	commonApi "shop/api/gen/go/common"
)

// GetAnalyticsTimeRange 根据时间类型返回当前统计周期的起止时间。
func GetAnalyticsTimeRange(timeType commonApi.AnalyticsTimeType) (time.Time, time.Time) {
	now := time.Now()
	switch timeType {
	case commonApi.AnalyticsTimeType_YEAR:
		startAt := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		return startAt, startAt.AddDate(1, 0, 0)
	case commonApi.AnalyticsTimeType_MONTH:
		startAt := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		return startAt, startAt.AddDate(0, 1, 0)
	default:
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		startAt := time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
		return startAt, startAt.AddDate(0, 0, 7)
	}
}

// GetPreviousAnalyticsTimeRange 根据当前周期起点返回上一个对比周期。
func GetPreviousAnalyticsTimeRange(timeType commonApi.AnalyticsTimeType, currentStartAt time.Time) (time.Time, time.Time) {
	switch timeType {
	case commonApi.AnalyticsTimeType_YEAR:
		startAt := currentStartAt.AddDate(-1, 0, 0)
		return startAt, currentStartAt
	case commonApi.AnalyticsTimeType_MONTH:
		startAt := currentStartAt.AddDate(0, -1, 0)
		return startAt, currentStartAt
	default:
		startAt := currentStartAt.AddDate(0, 0, -7)
		return startAt, currentStartAt
	}
}

// FormatAnalyticsAxis 根据时间类型格式化图表横轴标签。
func FormatAnalyticsAxis(timeType commonApi.AnalyticsTimeType, index int, startAt time.Time) string {
	switch timeType {
	case commonApi.AnalyticsTimeType_YEAR:
		months := []string{"1月", "2月", "3月", "4月", "5月", "6月", "7月", "8月", "9月", "10月", "11月", "12月"}
		return months[index]
	case commonApi.AnalyticsTimeType_MONTH:
		return startAt.AddDate(0, 0, index).Format("01-02")
	default:
		labels := []string{"一", "二", "三", "四", "五", "六", "日"}
		return "周" + labels[index]
	}
}

// GetAnalyticsGroupExpr 返回聚合表达式和完整横轴数据。
func GetAnalyticsGroupExpr(timeType commonApi.AnalyticsTimeType, startAt, endAt time.Time) (string, []string) {
	axis := make([]string, 0)
	switch timeType {
	case commonApi.AnalyticsTimeType_YEAR:
		for i := 0; i < 12; i++ {
			axis = append(axis, FormatAnalyticsAxis(timeType, i, startAt))
		}
		return "MONTH(created_at)", axis
	case commonApi.AnalyticsTimeType_MONTH:
		monthDays := endAt.AddDate(0, 0, -1).Day()
		for i := 0; i < monthDays; i++ {
			axis = append(axis, FormatAnalyticsAxis(timeType, i, startAt))
		}
		return "DAY(created_at)", axis
	default:
		for i := 0; i < 7; i++ {
			axis = append(axis, FormatAnalyticsAxis(timeType, i, startAt))
		}
		return "WEEKDAY(created_at)+1", axis
	}
}

// CalcGrowthRate 计算环比增长百分比。
func CalcGrowthRate(prev, curr int64) int64 {
	if prev == 0 {
		if curr == 0 {
			return 0
		}
		return 100
	}
	return (curr - prev) * 100 / prev
}

// CalcRatio 计算千分比结果，供前端按固定精度展示。
func CalcRatio(numerator, denominator int64) int64 {
	if denominator == 0 {
		return 0
	}
	return numerator * 1000 / denominator
}

// CalcPerUnit 计算单次平均值。
func CalcPerUnit(total, count int64) int64 {
	if count == 0 {
		return 0
	}
	return total / count
}

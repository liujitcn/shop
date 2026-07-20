package biz

import (
	"fmt"
	"time"

	"shop/pkg/errorsx"

	_time "github.com/liujitcn/go-utils/time"
)

// parseStatDateArg 解析统计日期参数，兼容日期与日期时间格式。
func parseStatDateArg(value string) (time.Time, error) {
	// 未传统计日期时，默认统计昨天数据。
	if value == "" {
		return time.Now().AddDate(0, 0, -1), nil
	}

	statTime := _time.StringDateToTime(&value)
	// 日期格式非法时直接返回错误，避免统计错天。
	if statTime == nil || statTime.Year() <= 1 {
		return time.Time{}, errorsx.InvalidArgument(fmt.Sprintf("statDate 格式错误，支持 %s 或 %s", _time.DateLayout, _time.Layout))
	}
	return *statTime, nil
}

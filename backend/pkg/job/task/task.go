package task

import (
	"fmt"
	"reflect"
	"time"

	"shop/pkg/errorsx"

	_time "github.com/liujitcn/go-utils/time"
)

// NewTaskList 创建定时任务执行器映射。
func NewTaskList(
	tradeBill *TradeBill,
	orderStatDay *OrderStatDay,
	goodsStatDay *GoodsStatDay,
	recommendSync *RecommendSync,
) map[string]TaskExec {
	taskMap := make(map[string]TaskExec, 17)
	registerTask(taskMap, tradeBill)
	registerTask(taskMap, orderStatDay)
	registerTask(taskMap, goodsStatDay)
	registerTask(taskMap, recommendSync)
	return taskMap
}

// TaskExec 定义定时任务执行器接口。
type TaskExec interface {
	Exec(arg map[string]string) ([]string, error)
}

// registerTask 注册单个定时任务执行器。
func registerTask(taskMap map[string]TaskExec, exec TaskExec) {
	t := reflect.TypeOf(exec)
	// 非结构体指针执行器不符合任务注册约定，直接跳过。
	if t == nil || t.Kind() != reflect.Ptr {
		return
	}
	t = t.Elem()
	// 只有结构体指针才允许使用结构体名称作为任务注册名。
	if t.Kind() != reflect.Struct {
		return
	}
	taskMap[t.Name()] = exec
}

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

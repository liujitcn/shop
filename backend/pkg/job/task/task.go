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
	registerTask(taskMap, tradeBill, "申请交易账单")
	registerTask(taskMap, orderStatDay, "订单日汇总")
	registerTask(taskMap, goodsStatDay, "商品日汇总")
	registerTask(taskMap, recommendSync, "推荐系统主数据同步")
	return taskMap
}

// TaskExec 定义定时任务执行器接口。
type TaskExec interface {
	Exec(arg map[string]string) ([]string, error)
}

// registerTask 注册单个定时任务执行器。
func registerTask(taskMap map[string]TaskExec, exec TaskExec, taskDesc string) {
	taskName := getStructName(exec)
	// 任务名为空时，说明执行器类型不符合结构体指针约定。
	if taskName == "" {
		panic(fmt.Sprintf("%s task name is empty", taskDesc))
	}
	// 同名任务已经存在时，直接中断启动避免被错误覆盖。
	if _, ok := taskMap[taskName]; ok {
		panic(fmt.Sprintf("%s task already exists", taskDesc))
	}
	taskMap[taskName] = exec
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

// getStructName 获取结构体指针对应的结构体名称。
func getStructName(ptr interface{}) string {
	// 获取类型信息
	t := reflect.TypeOf(ptr)

	// 非指针执行器不符合任务注册约定，直接返回空名称。
	if t.Kind() != reflect.Ptr {
		return ""
	}

	// 解引用指针，获取指向的结构体类型。
	t = t.Elem()

	// 非结构体指针不作为合法任务执行器。
	if t.Kind() != reflect.Struct {
		return ""
	}

	// 结构体名称作为任务注册名返回。
	return t.Name()
}

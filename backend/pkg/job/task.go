package job

import "reflect"

// TaskExec 定义定时任务执行器接口。
type TaskExec interface {
	Exec(arg map[string]string) ([]string, error)
}

// NewTaskList 创建定时任务执行器映射。
func NewTaskList(tasks ...TaskExec) map[string]TaskExec {
	taskMap := make(map[string]TaskExec, len(tasks))
	for _, task := range tasks {
		registerTask(taskMap, task)
	}
	return taskMap
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

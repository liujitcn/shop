package task

import "reflect"

func NewTaskList(tradeBill *TradeBill) map[string]TaskExec {
	taskMap := make(map[string]TaskExec)
	// 申请交易账单
	tradeBillName := getStructName(tradeBill)
	if _, ok := taskMap[tradeBillName]; ok {
		panic("申请交易账单 task already exists")
	} else {
		taskMap[tradeBillName] = tradeBill
	}
	return taskMap
}

type TaskExec interface {
	Exec(arg map[string]string) ([]string, error)
}

func getStructName(ptr interface{}) string {
	// 获取类型信息
	t := reflect.TypeOf(ptr)

	// 检查是否为指针
	if t.Kind() != reflect.Ptr {
		return ""
	}

	// 解引用指针，获取指向的类型
	t = t.Elem()

	// 检查是否为结构体
	if t.Kind() != reflect.Struct {
		return ""
	}

	// 返回结构体名称
	return t.Name()
}

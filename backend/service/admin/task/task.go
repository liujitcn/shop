package task

import "reflect"

func NewTaskList(
	tradeBill *TradeBill,
	orderStatDay *OrderStatDay,
	goodsStatDay *GoodsStatDay,
	recommendGoodsStatDay *RecommendGoodsStatDay,
	recommendEvalReport *RecommendEvalReport,
	recommendUserPreferenceRebuild *RecommendUserPreferenceRebuild,
	recommendGoodsRelationRebuild *RecommendGoodsRelationRebuild,
) map[string]TaskExec {
	taskMap := make(map[string]TaskExec)
	// 申请交易账单
	tradeBillName := getStructName(tradeBill)
	if _, ok := taskMap[tradeBillName]; ok {
		panic("申请交易账单 task already exists")
	} else {
		taskMap[tradeBillName] = tradeBill
	}
	// 订单日汇总
	orderStatDayName := getStructName(orderStatDay)
	if _, ok := taskMap[orderStatDayName]; ok {
		panic("订单日汇总 task already exists")
	} else {
		taskMap[orderStatDayName] = orderStatDay
	}
	// 商品日汇总
	goodsStatDayName := getStructName(goodsStatDay)
	if _, ok := taskMap[goodsStatDayName]; ok {
		panic("商品日汇总 task already exists")
	} else {
		taskMap[goodsStatDayName] = goodsStatDay
	}
	// 推荐商品日汇总
	recommendGoodsStatDayName := getStructName(recommendGoodsStatDay)
	if _, ok := taskMap[recommendGoodsStatDayName]; ok {
		panic("推荐商品日汇总 task already exists")
	} else {
		taskMap[recommendGoodsStatDayName] = recommendGoodsStatDay
	}
	// 推荐离线评估报告
	recommendEvalReportName := getStructName(recommendEvalReport)
	if _, ok := taskMap[recommendEvalReportName]; ok {
		panic("推荐离线评估报告 task already exists")
	} else {
		taskMap[recommendEvalReportName] = recommendEvalReport
	}
	// 推荐用户偏好重建
	recommendUserPreferenceRebuildName := getStructName(recommendUserPreferenceRebuild)
	if _, ok := taskMap[recommendUserPreferenceRebuildName]; ok {
		panic("推荐用户偏好重建 task already exists")
	} else {
		taskMap[recommendUserPreferenceRebuildName] = recommendUserPreferenceRebuild
	}
	// 推荐商品关联重建
	recommendGoodsRelationRebuildName := getStructName(recommendGoodsRelationRebuild)
	if _, ok := taskMap[recommendGoodsRelationRebuildName]; ok {
		panic("推荐商品关联重建 task already exists")
	} else {
		taskMap[recommendGoodsRelationRebuildName] = recommendGoodsRelationRebuild
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

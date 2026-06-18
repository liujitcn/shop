package flow

import (
	"strings"

	commonv1 "shop/api/gen/go/common/v1"
)

// sliceAnyValue 将任意值安全转换为 []any 切片。
func sliceAnyValue(value any) []any {
	values, ok := value.([]any)
	if !ok {
		return []any{}
	}
	return values
}

// formatAmount 将金额（分）格式化为元展示文本。
func formatAmount(value any) string {
	cents := int64Value(value)
	if cents <= 0 {
		return "0.00"
	}
	yuan := cents / 100
	remainder := cents % 100
	return formatYuan(yuan, remainder)
}

// formatYuan 将整数元和余数分拼接为标准金额文本。
func formatYuan(yuan int64, remainder int64) string {
	if remainder == 0 {
		return int64ToString(yuan) + ".00"
	}
	if remainder < 10 {
		return int64ToString(yuan) + ".0" + int64ToString(remainder)
	}
	return int64ToString(yuan) + "." + int64ToString(remainder)
}

// int64ToString 将 int64 转为字符串，避免引入额外包。
func int64ToString(value int64) string {
	if value == 0 {
		return "0"
	}
	negative := false
	if value < 0 {
		negative = true
		value = -value
	}
	digits := []byte{}
	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}
	if negative {
		return "-" + string(digits)
	}
	return string(digits)
}

// analyticsTimeTypeLabel 将分析时间维度枚举转换为展示文本。
func analyticsTimeTypeLabel(value any) string {
	switch int64Value(value) {
	case int64(commonv1.AnalyticsTimeType_ANALYTICS_TIME_TYPE_WEEK):
		return "周"
	case int64(commonv1.AnalyticsTimeType_ANALYTICS_TIME_TYPE_MONTH):
		return "月"
	case int64(commonv1.AnalyticsTimeType_ANALYTICS_TIME_TYPE_YEAR):
		return "年"
	default:
		return "未知"
	}
}

// orderStatusLabel 将订单状态枚举转换为展示文本。
func orderStatusLabel(value any) string {
	switch int64Value(value) {
	case int64(commonv1.OrderStatus_CREATED):
		return "待支付"
	case int64(commonv1.OrderStatus_PAID):
		return "待发货"
	case int64(commonv1.OrderStatus_SHIPPED):
		return "待收货"
	case int64(commonv1.OrderStatus_WAIT_REVIEW):
		return "待评价"
	case int64(commonv1.OrderStatus_COMPLETED):
		return "已完成"
	case int64(commonv1.OrderStatus_REFUNDING):
		return "已退款"
	case int64(commonv1.OrderStatus_CANCELED):
		return "已取消"
	default:
		return "未知"
	}
}

// commentStatusLabel 将评价审核状态枚举转换为展示文本。
func commentStatusLabel(value any) string {
	switch int64Value(value) {
	case int64(commonv1.CommentStatus_PENDING_REVIEW_CS):
		return "待审核"
	case int64(commonv1.CommentStatus_APPROVED_CS):
		return "审核通过"
	case int64(commonv1.CommentStatus_REJECTED_CS):
		return "审核不通过"
	default:
		return "未知"
	}
}

// goodsStatusLabel 将商品上下架状态枚举转换为展示文本。
func goodsStatusLabel(value any) string {
	switch int64Value(value) {
	case int64(commonv1.GoodsStatus_PUT_ON):
		return "已上架"
	case int64(commonv1.GoodsStatus_PULL_OFF):
		return "已下架"
	default:
		return "未知"
	}
}

// inventoryAlertLabel 将库存预警级别枚举转换为展示文本。
func inventoryAlertLabel(value any) string {
	switch int64Value(value) {
	case int64(commonv1.GoodsInventoryAlert_LOW_STOCK):
		return "低库存"
	case int64(commonv1.GoodsInventoryAlert_ZERO_STOCK):
		return "零库存"
	default:
		return "正常"
	}
}

// payBillStatusLabel 将对账状态枚举转换为展示文本。
func payBillStatusLabel(value any) string {
	switch int64Value(value) {
	case int64(commonv1.PayBillStatus_NO_COMPARE):
		return "未比对"
	case int64(commonv1.PayBillStatus_NO_ERROR):
		return "无误差"
	case int64(commonv1.PayBillStatus_HAS_ERROR):
		return "有误差"
	default:
		return "未知"
	}
}

// storeAuditStatusLabel 将门店审核状态枚举转换为展示文本。
func storeAuditStatusLabel(value any) string {
	switch int64Value(value) {
	case int64(commonv1.UserStoreStatus_PENDING_REVIEW):
		return "待审核"
	case int64(commonv1.UserStoreStatus_FAILED_REVIEW):
		return "审核失败"
	case int64(commonv1.UserStoreStatus_APPROVED):
		return "审核通过"
	default:
		return "未提交"
	}
}

// redactGorseConfig 脱敏 Gorse 推荐配置中的敏感字段。
func redactGorseConfig(output map[string]any) map[string]any {
	if len(output) == 0 {
		return output
	}
	result := make(map[string]any, len(output))
	for key, value := range output {
		lowerKey := strings.ToLower(key)
		if strings.Contains(lowerKey, "secret") || strings.Contains(lowerKey, "password") || strings.Contains(lowerKey, "token") {
			result[key] = "***"
			continue
		}
		result[key] = value
	}
	return result
}

// adminMetricItem 构造管理端指标展示项。
func adminMetricItem(label string, value any, unit string) map[string]any {
	return map[string]any{
		"label": label,
		"value": stringValue(value),
		"unit":  unit,
	}
}

// adminMetricItems 从输出映射中按指定字段构造指标列表。
func adminMetricItems(output map[string]any, fields []adminMetricField) []map[string]any {
	items := make([]map[string]any, 0, len(fields))
	for _, field := range fields {
		value := output[field.key]
		if field.format != nil {
			items = append(items, adminMetricItem(field.label, field.format(value), field.unit))
		} else {
			items = append(items, adminMetricItem(field.label, value, field.unit))
		}
	}
	return items
}

// adminMetricField 表示指标面板字段定义。
type adminMetricField struct {
	label  string
	key    string
	unit   string
	format func(any) string
}

// adminNoticeBlock 构造管理端提示卡片。
func adminNoticeBlock(title string, desc string) map[string]any {
	return map[string]any{
		"type":  "notice",
		"title": title,
		"desc":  desc,
	}
}

// adminSuccessBlock 构造管理端操作成功卡片。
func adminSuccessBlock(title string, desc string) map[string]any {
	return map[string]any{
		"type":  "success",
		"title": title,
		"desc":  desc,
	}
}

// adminErrorBlock 构造管理端操作失败卡片。
func adminErrorBlock(title string, desc string) map[string]any {
	return map[string]any{
		"type":  "error",
		"title": title,
		"desc":  desc,
	}
}

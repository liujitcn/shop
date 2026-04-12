package utils

import "shop/api/gen/go/common"

// PaidOrderStatuses 返回统计支付成功口径认可的订单状态集合。
func PaidOrderStatuses() []int32 {
	return []int32{
		int32(common.OrderStatus_PAID),
		int32(common.OrderStatus_SHIPPED),
		int32(common.OrderStatus_RECEIVED),
		int32(common.OrderStatus_REFUNDING),
	}
}

// IsPaidOrderStatus 判断订单状态是否属于已支付口径。
func IsPaidOrderStatus(status int32) bool {
	// 已付款、已发货、已完成、已退款都视为支付成功订单。
	for _, item := range PaidOrderStatuses() {
		// 命中任一已支付状态时，立即返回 true。
		if item == status {
			return true
		}
	}
	return false
}

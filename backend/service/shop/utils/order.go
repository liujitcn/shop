package utils

import _const "shop/service/shop/consts"

// PaidTradeStatuses 返回已完成支付口径的交易单状态集合。
func PaidTradeStatuses() []int32 {
	return []int32{
		_const.ORDER_TRADE_STATUS_PAID,
		_const.ORDER_TRADE_STATUS_CASH_ON_DELIVERY,
		_const.ORDER_TRADE_STATUS_PARTIAL_REFUND,
		_const.ORDER_TRADE_STATUS_FULL_REFUND,
	}
}

// IsPaidTradeStatus 判断交易单是否已经形成支付事实。
func IsPaidTradeStatus(status int32) bool {
	// 已支付、货到付款、部分退款和全额退款都属于已形成支付事实的交易。
	switch status {
	case _const.ORDER_TRADE_STATUS_PAID,
		_const.ORDER_TRADE_STATUS_CASH_ON_DELIVERY,
		_const.ORDER_TRADE_STATUS_PARTIAL_REFUND,
		_const.ORDER_TRADE_STATUS_FULL_REFUND:
		return true
	default:
		return false
	}
}

import type { OrderInfo, PageOrderInfoRequest } from '@/rpc/shop/app/v1/order_info'
import { OrderInfoStatus, OrderRefundStatus, OrderTradeStatus } from '@/rpc/shop/common/v1/enum'

/** 商城端订单列表支持的后端筛选条件。 */
export type OrderListFilter = Pick<
  PageOrderInfoRequest,
  'status' | 'trade_status' | 'refund_status' | 'has_refund' | 'refundable'
>

const tradeStatusText = new Map<OrderTradeStatus, string>([
  [OrderTradeStatus.PENDING_PAYMENT_OTS, '待支付'],
  [OrderTradeStatus.PAYING_OTS, '支付中'],
  [OrderTradeStatus.PAID_OTS, '已支付'],
  [OrderTradeStatus.CASH_ON_DELIVERY_OTS, '货到付款'],
  [OrderTradeStatus.CLOSED_OTS, '已关闭'],
  [OrderTradeStatus.PARTIAL_REFUND_OTS, '部分退款'],
  [OrderTradeStatus.FULL_REFUND_OTS, '全额退款'],
])

const orderStatusText = new Map<OrderInfoStatus, string>([
  [OrderInfoStatus.NOT_STARTED_OIS, '未进入履约'],
  [OrderInfoStatus.WAIT_SHIPMENT_OIS, '待发货'],
  [OrderInfoStatus.SHIPPED_OIS, '已发货'],
  [OrderInfoStatus.WAIT_REVIEW_OIS, '待评价'],
  [OrderInfoStatus.COMPLETED_OIS, '已完成'],
  [OrderInfoStatus.CANCELED_OIS, '已取消'],
])

const refundStatusText = new Map<OrderRefundStatus, string>([
  [OrderRefundStatus.PROCESSING_ORS, '退款处理中'],
  [OrderRefundStatus.PARTIAL_REFUND_ORS, '部分退款'],
  [OrderRefundStatus.REFUNDED_ORS, '已退款'],
  [OrderRefundStatus.CLOSED_OR_FAILED_ORS, '退款已关闭/失败'],
])

/** getOrderDisplayStatus 返回订单卡片和详情页使用的状态文案。 */
export const getOrderDisplayStatus = (order: OrderInfo) => {
  if (order.is_trade || order.status === OrderInfoStatus.NOT_STARTED_OIS) {
    return tradeStatusText.get(order.trade_status) || '交易处理中'
  }
  const refundStatus = refundStatusText.get(order.refund_status)
  if (refundStatus) {
    return refundStatus
  }
  return orderStatusText.get(order.status) || '订单处理中'
}

/** isPayableTrade 判断交易聚合记录是否允许继续支付或取消。 */
export const isPayableTrade = (order: OrderInfo) => {
  return (
    order.is_trade &&
    (order.trade_status === OrderTradeStatus.PENDING_PAYMENT_OTS ||
      order.trade_status === OrderTradeStatus.PAYING_OTS)
  )
}

/** canRefundOrder 判断门店子订单是否允许用户申请退款。 */
export const canRefundOrder = (order: OrderInfo) => {
  return (
    !order.is_trade &&
    order.status === OrderInfoStatus.WAIT_SHIPMENT_OIS &&
    [
      OrderRefundStatus.NONE_ORS,
      OrderRefundStatus.PARTIAL_REFUND_ORS,
      OrderRefundStatus.CLOSED_OR_FAILED_ORS,
    ].includes(order.refund_status)
  )
}

/** canDeleteOrder 判断交易或门店子订单是否允许从用户订单列表删除。 */
export const canDeleteOrder = (order: OrderInfo) => {
  if (order.is_trade) {
    return order.trade_status === OrderTradeStatus.CLOSED_OTS
  }
  return (
    order.status === OrderInfoStatus.COMPLETED_OIS ||
    order.status === OrderInfoStatus.CANCELED_OIS ||
    order.refund_status === OrderRefundStatus.REFUNDED_ORS
  )
}

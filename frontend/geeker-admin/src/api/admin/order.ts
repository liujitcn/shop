import service from '@/utils/request'
import {
  type Order,
  type OrderRefundResponse,
  type OrderResponse,
  type OrderService,
  type OrderShippedResponse,
  type PageOrderRequest,
  type PageOrderResponse,
  type RefundOrderRequest,
  type ShippedOrderRequest,
} from '@/rpc/admin/order'
import type { Int64Value } from '@/rpc/google/protobuf/wrappers'
import type { Empty } from '@/rpc/google/protobuf/empty'

const ORDER_URL = '/admin/order/info'

/** 订单服务 */
export class OrderServiceImpl implements OrderService {
  /** 查询订单分页列表 */
  PageOrder(request: PageOrderRequest): Promise<PageOrderResponse> {
    return service<PageOrderRequest, PageOrderResponse>({
      url: `${ORDER_URL}`,
      method: 'get',
      params: request,
    })
  }
  /** 查询订单 */
  GetOrder(request: Int64Value): Promise<OrderResponse> {
    return service<Int64Value, OrderResponse>({
      url: `${ORDER_URL}/${request.value}`,
      method: 'get',
    })
  }
  /** 查询订单退款信息 */
  GetOrderRefund(request: Int64Value): Promise<OrderRefundResponse> {
    return service<Int64Value, OrderRefundResponse>({
      url: `${ORDER_URL}/${request.value}/refund`,
      method: 'get',
    })
  }
  /** 订单退款 */
  RefundOrder(request: RefundOrderRequest): Promise<Empty> {
    return service<Int64Value, Order>({
      url: `${ORDER_URL}/${request.orderId}/refund`,
      method: 'put',
      data: request,
    })
  }
  /** 查询订单发货信息 */
  GetOrderShipped(request: Int64Value): Promise<OrderShippedResponse> {
    return service<Int64Value, OrderShippedResponse>({
      url: `${ORDER_URL}/${request.value}/shipped`,
      method: 'get',
    })
  }
  /** 订单发货 */
  ShippedOrder(request: ShippedOrderRequest): Promise<Empty> {
    return service<Int64Value, Order>({
      url: `${ORDER_URL}/${request.orderId}/shipped`,
      method: 'put',
      data: request,
    })
  }
}

export const defOrderService = new OrderServiceImpl()

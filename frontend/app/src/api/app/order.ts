import { http } from '@/utils/http'
import type {
  ConfirmOrderResponse,
  CreateOrderRequest,
  CreateOrderResponse,
  OrderResponse,
  OrderRepurchaseRequest,
  OrderService,
  PageOrderRequest,
  PageOrderResponse,
  CountOrderResponse,
  CancelOrderRequest,
  RefundOrderRequest,
  CreateOrderGoods,
  ReceiveOrderRequest,
} from '@/rpc/app/order'
import type { Int64Value, StringValue } from '@/rpc/google/protobuf/wrappers'
import type { Empty } from '@/rpc/google/protobuf/empty'

const ORDER_URL = '/app/order/info'

/** 订单服务 */
export class OrderServiceImpl implements OrderService {
  /** 预付订单 */
  OrderPre(request: Empty): Promise<ConfirmOrderResponse> {
    return http<ConfirmOrderResponse>({
      url: `${ORDER_URL}/pre`,
      method: 'POST',
      data: request,
    })
  }
  /** 立即购买订单 */
  OrderBuy(request: CreateOrderGoods): Promise<ConfirmOrderResponse> {
    return http<ConfirmOrderResponse>({
      url: `${ORDER_URL}/buy`,
      method: 'POST',
      data: request,
    })
  }
  /** 再次购买订单 */
  OrderRepurchase(request: OrderRepurchaseRequest): Promise<ConfirmOrderResponse> {
    return http<ConfirmOrderResponse>({
      url: `${ORDER_URL}/repurchase`,
      method: 'POST',
      data: request,
    })
  }
  /** 查询订单数量汇总 */
  CountOrder(request: Empty): Promise<CountOrderResponse> {
    return http<CountOrderResponse>({
      url: `${ORDER_URL}/count`,
      method: 'GET',
      data: request,
    })
  }
  /** 查询商品分页列表 */
  PageOrder(request: PageOrderRequest): Promise<PageOrderResponse> {
    return http<PageOrderResponse>({
      url: `${ORDER_URL}`,
      method: 'GET',
      data: request,
    })
  }

  /** 根据订单编号查询订单id */
  GetOrderIdByOrderNo(request: StringValue): Promise<Int64Value> {
    return http<Int64Value>({
      url: `${ORDER_URL}/${request.value}/orderNo`,
      method: 'GET',
    })
  }
  /** 根据订单id查询订单 */
  GetOrderById(request: Int64Value): Promise<OrderResponse> {
    return http<OrderResponse>({
      url: `${ORDER_URL}/${request.value}`,
      method: 'GET',
    })
  }
  /** 创建订单 */
  CreateOrder(request: CreateOrderRequest): Promise<CreateOrderResponse> {
    return http<CreateOrderResponse>({
      url: `${ORDER_URL}`,
      method: 'POST',
      data: request,
    })
  }
  /** 删除订单 */
  DeleteOrder(request: Int64Value): Promise<Empty> {
    return http<Empty>({
      url: `${ORDER_URL}/${request.value}`,
      method: 'DELETE',
    })
  }
  /** 取消订单 */
  CancelOrder(request: CancelOrderRequest): Promise<Empty> {
    return http<Empty>({
      url: `${ORDER_URL}/${request.orderId}/cancel`,
      method: 'PUT',
      data: request,
    })
  }
  /** 订单退款 */
  RefundOrder(request: RefundOrderRequest): Promise<Empty> {
    return http<Empty>({
      url: `${ORDER_URL}/${request.orderId}/refund`,
      method: 'PUT',
      data: request,
    })
  }
  /** 确认收货 */
  ReceiveOrder(request: ReceiveOrderRequest): Promise<Empty> {
    return http<Empty>({
      url: `${ORDER_URL}/${request.orderId}/receive`,
      method: 'PUT',
      data: request,
    })
  }
}

export const defOrderService = new OrderServiceImpl()

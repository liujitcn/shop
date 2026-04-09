import { http } from '@/utils/http'
import type {
  ConfirmOrderInfoResponse,
  CreateOrderInfoRequest,
  CreateOrderInfoResponse,
  OrderInfoResponse,
  OrderRepurchaseInfoRequest,
  OrderInfoService,
  PageOrderInfoRequest,
  PageOrderInfoResponse,
  CountOrderInfoResponse,
  CancelOrderInfoRequest,
  RefundOrderInfoRequest,
  CreateOrderInfoGoods,
  ReceiveOrderInfoRequest,
} from '@/rpc/app/order_info'
import type { Int64Value, StringValue } from '@/rpc/google/protobuf/wrappers'
import type { Empty } from '@/rpc/google/protobuf/empty'

const ORDER_INFO_URL = '/app/order/info'

/** 订单服务 */
export class OrderInfoServiceImpl implements OrderInfoService {
  /** 预付订单 */
  OrderInfoPre(request: Empty): Promise<ConfirmOrderInfoResponse> {
    return http<ConfirmOrderInfoResponse>({
      url: `${ORDER_INFO_URL}/pre`,
      method: 'POST',
      data: request,
    })
  }
  /** 立即购买订单 */
  OrderInfoBuy(request: CreateOrderInfoGoods): Promise<ConfirmOrderInfoResponse> {
    return http<ConfirmOrderInfoResponse>({
      url: `${ORDER_INFO_URL}/buy`,
      method: 'POST',
      data: request,
    })
  }
  /** 再次购买订单 */
  OrderInfoRepurchase(request: OrderRepurchaseInfoRequest): Promise<ConfirmOrderInfoResponse> {
    return http<ConfirmOrderInfoResponse>({
      url: `${ORDER_INFO_URL}/repurchase`,
      method: 'POST',
      data: request,
    })
  }
  /** 查询订单数量汇总 */
  CountOrderInfo(request: Empty): Promise<CountOrderInfoResponse> {
    return http<CountOrderInfoResponse>({
      url: `${ORDER_INFO_URL}/count`,
      method: 'GET',
      data: request,
    })
  }
  /** 查询商品分页列表 */
  PageOrderInfo(request: PageOrderInfoRequest): Promise<PageOrderInfoResponse> {
    return http<PageOrderInfoResponse>({
      url: `${ORDER_INFO_URL}`,
      method: 'GET',
      data: request,
    })
  }

  /** 根据订单编号查询订单id */
  GetOrderInfoIdByOrderNo(request: StringValue): Promise<Int64Value> {
    return http<Int64Value>({
      url: `${ORDER_INFO_URL}/${request.value}/orderNo`,
      method: 'GET',
    })
  }
  /** 根据订单id查询订单 */
  GetOrderInfoById(request: Int64Value): Promise<OrderInfoResponse> {
    return http<OrderInfoResponse>({
      url: `${ORDER_INFO_URL}/${request.value}`,
      method: 'GET',
    })
  }
  /** 创建订单 */
  CreateOrderInfo(request: CreateOrderInfoRequest): Promise<CreateOrderInfoResponse> {
    return http<CreateOrderInfoResponse>({
      url: `${ORDER_INFO_URL}`,
      method: 'POST',
      data: request,
    })
  }
  /** 删除订单 */
  DeleteOrderInfo(request: Int64Value): Promise<Empty> {
    return http<Empty>({
      url: `${ORDER_INFO_URL}/${request.value}`,
      method: 'DELETE',
    })
  }
  /** 取消订单 */
  CancelOrderInfo(request: CancelOrderInfoRequest): Promise<Empty> {
    return http<Empty>({
      url: `${ORDER_INFO_URL}/${request.orderId}/cancel`,
      method: 'PUT',
      data: request,
    })
  }
  /** 订单退款 */
  RefundOrderInfo(request: RefundOrderInfoRequest): Promise<Empty> {
    return http<Empty>({
      url: `${ORDER_INFO_URL}/${request.orderId}/refund`,
      method: 'PUT',
      data: request,
    })
  }
  /** 确认收货 */
  ReceiveOrderInfo(request: ReceiveOrderInfoRequest): Promise<Empty> {
    return http<Empty>({
      url: `${ORDER_INFO_URL}/${request.orderId}/receive`,
      method: 'PUT',
      data: request,
    })
  }
}

export const defOrderService = new OrderInfoServiceImpl()

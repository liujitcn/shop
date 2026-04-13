import { http } from '@/utils/http'
import type {
  BuyNowOrderInfoRequest,
  BuyNowOrderInfoResponse,
  ConfirmOrderInfoResponse,
  CreateOrderInfoRequest,
  CreateOrderInfoResponse,
  OrderInfoResponse,
  RepurchaseOrderInfoRequest,
  RepurchaseOrderInfoResponse,
  OrderInfoService,
  PageOrderInfoRequest,
  PageOrderInfoResponse,
  CountOrderInfoResponse,
  CancelOrderInfoRequest,
  RefundOrderInfoRequest,
  ReceiveOrderInfoRequest,
} from '@/rpc/app/order_info'
import type { Int64Value, StringValue } from '@/rpc/google/protobuf/wrappers'
import type { Empty } from '@/rpc/google/protobuf/empty'

const ORDER_INFO_URL = '/app/order/info'
const ORDER_CONFIRM_URL = '/app/order/confirm'

/** 订单服务 */
export class OrderInfoServiceImpl implements OrderInfoService {
  /** 确认订单信息 */
  ConfirmOrderInfo(request: Empty): Promise<ConfirmOrderInfoResponse> {
    return http<ConfirmOrderInfoResponse>({
      url: `${ORDER_CONFIRM_URL}`,
      method: 'POST',
      data: request,
    })
  }
  /** 立即购买订单 */
  BuyNowOrderInfo(request: BuyNowOrderInfoRequest): Promise<BuyNowOrderInfoResponse> {
    return http<BuyNowOrderInfoResponse>({
      url: `${ORDER_CONFIRM_URL}/buy-now`,
      method: 'POST',
      data: request,
    })
  }
  /** 再次购买订单 */
  RepurchaseOrderInfo(request: RepurchaseOrderInfoRequest): Promise<RepurchaseOrderInfoResponse> {
    return http<RepurchaseOrderInfoResponse>({
      url: `${ORDER_CONFIRM_URL}/repurchase`,
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
      url: `${ORDER_INFO_URL}/no/${request.value}`,
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
      url: `${ORDER_INFO_URL}/${request.orderId}/cancellation`,
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
      url: `${ORDER_INFO_URL}/${request.orderId}/receipt`,
      method: 'PUT',
      data: request,
    })
  }
}

export const defOrderService = new OrderInfoServiceImpl()

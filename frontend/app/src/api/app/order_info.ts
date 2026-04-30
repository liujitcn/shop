import { http } from '@/utils/http'
import type {
  BuyNowOrderInfoRequest,
  BuyNowOrderInfoResponse,
  CancelOrderInfoRequest,
  ConfirmOrderInfoRequest,
  ConfirmOrderInfoResponse,
  CountOrderInfoRequest,
  CountOrderInfoResponse,
  CountOrderInfoResponse_Count,
  CreateOrderInfoRequest,
  CreateOrderInfoResponse,
  DeleteOrderInfoRequest,
  GetOrderInfoByIdRequest,
  GetOrderInfoIdByOrderNoRequest,
  GetOrderInfoIdByOrderNoResponse,
  OrderInfo,
  OrderInfoResponse,
  OrderInfoService,
  PageOrderInfoRequest,
  PageOrderInfoResponse,
  RepurchaseOrderInfoRequest,
  RepurchaseOrderInfoResponse,
  RefundOrderInfoRequest,
  ReceiveOrderInfoRequest,
} from '@/rpc/app/v1/order_info'
import type { Empty } from '@/rpc/google/protobuf/empty'

const ORDER_INFO_URL = '/v1/app/order/info'
const ORDER_CONFIRM_URL = '/v1/app/order/confirm'

type CountOrderInfoResponseCompat = CountOrderInfoResponse & {
  count: CountOrderInfoResponse_Count[]
}

type CountOrderInfoHTTPResponse = Partial<CountOrderInfoResponse> & {
  count?: CountOrderInfoResponse_Count[]
}

type PageOrderInfoResponseCompat = PageOrderInfoResponse & {
  list: OrderInfo[]
}

type PageOrderInfoHTTPResponse = Partial<PageOrderInfoResponse> & {
  list?: OrderInfo[]
}

type GetOrderInfoIdByOrderNoResponseCompat = GetOrderInfoIdByOrderNoResponse & {
  value: number
}

type GetOrderInfoIdByOrderNoHTTPResponse = Partial<GetOrderInfoIdByOrderNoResponse> & {
  value?: number
}

/** 订单服务 */
export class OrderInfoServiceImpl implements OrderInfoService {
  /** 确认订单信息 */
  ConfirmOrderInfo(request: ConfirmOrderInfoRequest): Promise<ConfirmOrderInfoResponse> {
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
  async CountOrderInfo(request: CountOrderInfoRequest): Promise<CountOrderInfoResponseCompat> {
    const response = await http<CountOrderInfoHTTPResponse>({
      url: `${ORDER_INFO_URL}/count`,
      method: 'GET',
      data: request,
    })
    // 兼容未生成前的旧响应 count，同时向新协议的 counts 字段收敛。
    const counts = response.counts ?? response.count ?? []
    return {
      ...response,
      counts,
      count: counts,
    }
  }
  /** 查询商品分页列表 */
  async PageOrderInfo(request: PageOrderInfoRequest): Promise<PageOrderInfoResponseCompat> {
    const response = await http<PageOrderInfoHTTPResponse>({
      url: `${ORDER_INFO_URL}`,
      method: 'GET',
      data: request,
    })
    // 兼容未生成前的旧响应 list，同时向新协议的 orderInfos 字段收敛。
    const orderInfos = response.order_infos ?? response.list ?? []
    return {
      ...response,
      order_infos: orderInfos,
      list: orderInfos,
      total: response.total ?? 0,
    }
  }

  /** 根据订单编号查询订单id */
  async GetOrderInfoIdByOrderNo(
    request: GetOrderInfoIdByOrderNoRequest,
  ): Promise<GetOrderInfoIdByOrderNoResponseCompat> {
    const response = await http<GetOrderInfoIdByOrderNoHTTPResponse>({
      url: `${ORDER_INFO_URL}/no/${request.order_no}`,
      method: 'GET',
    })
    // 兼容未生成前的旧包装响应 value，同时向新协议的 orderId 字段收敛。
    const orderId = response.order_id ?? response.value ?? 0
    return {
      ...response,
      order_id: orderId,
      value: orderId,
    }
  }
  /** 根据订单id查询订单 */
  GetOrderInfoById(request: GetOrderInfoByIdRequest): Promise<OrderInfoResponse> {
    return http<OrderInfoResponse>({
      url: `${ORDER_INFO_URL}/${request.id}`,
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
  DeleteOrderInfo(request: DeleteOrderInfoRequest): Promise<Empty> {
    return http<Empty>({
      url: `${ORDER_INFO_URL}/${request.id}`,
      method: 'DELETE',
    })
  }
  /** 取消订单 */
  CancelOrderInfo(request: CancelOrderInfoRequest): Promise<Empty> {
    return http<Empty>({
      url: `${ORDER_INFO_URL}/${request.order_id}/cancellation`,
      method: 'PUT',
      data: request,
    })
  }
  /** 订单退款 */
  RefundOrderInfo(request: RefundOrderInfoRequest): Promise<Empty> {
    return http<Empty>({
      url: `${ORDER_INFO_URL}/${request.order_id}/refund`,
      method: 'PUT',
      data: request,
    })
  }
  /** 确认收货 */
  ReceiveOrderInfo(request: ReceiveOrderInfoRequest): Promise<Empty> {
    return http<Empty>({
      url: `${ORDER_INFO_URL}/${request.order_id}/receipt`,
      method: 'PUT',
      data: request,
    })
  }
}

export const defOrderService = new OrderInfoServiceImpl()

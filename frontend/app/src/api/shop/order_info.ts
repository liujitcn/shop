import { http } from '@/utils/http'
import type {
  BuyNowOrderInfoRequest,
  BuyNowOrderInfoResponse,
  CancelOrderInfoRequest,
  ConfirmOrderInfoRequest,
  ConfirmOrderInfoResponse,
  CountOrderInfoRequest,
  CountOrderInfoResponse,
  CreateOrderInfoRequest,
  CreateOrderInfoResponse,
  DeleteOrderInfoRequest,
  DeleteOrderTradeRequest,
  GetOrderInfoByIdRequest,
  GetOrderInfoIdByOrderNoRequest,
  GetOrderInfoIdByOrderNoResponse,
  GetOrderTradeByIdRequest,
  OrderInfoResponse,
  OrderInfoService,
  PageOrderInfoRequest,
  PageOrderInfoResponse,
  RepurchaseOrderInfoRequest,
  RepurchaseOrderInfoResponse,
  RefundOrderInfoRequest,
  ReceiveOrderInfoRequest,
} from '@/rpc/shop/app/v1/order_info'
import type { Empty } from '@/rpc/google/protobuf/empty'

const ORDER_INFO_URL = '/v1/app/order/info'
const ORDER_CONFIRM_URL = '/v1/app/order/confirm'
const ORDER_TRADE_URL = '/v1/app/order/trade'

/** 订单服务 */
export class OrderInfoServiceImpl implements OrderInfoService {
  /** 确认订单信息 */
  ConfirmOrderInfo(request: ConfirmOrderInfoRequest): Promise<ConfirmOrderInfoResponse> {
    return http<ConfirmOrderInfoResponse>({
      url: `${ORDER_CONFIRM_URL}`,
      method: 'POST',
      authMode: 'required',
      data: request,
    })
  }
  /** 立即购买订单 */
  BuyNowOrderInfo(request: BuyNowOrderInfoRequest): Promise<BuyNowOrderInfoResponse> {
    return http<BuyNowOrderInfoResponse>({
      url: `${ORDER_CONFIRM_URL}/buy-now`,
      method: 'POST',
      authMode: 'required',
      data: request,
    })
  }
  /** 再次购买订单 */
  RepurchaseOrderInfo(request: RepurchaseOrderInfoRequest): Promise<RepurchaseOrderInfoResponse> {
    return http<RepurchaseOrderInfoResponse>({
      url: `${ORDER_CONFIRM_URL}/repurchase`,
      method: 'POST',
      authMode: 'required',
      data: request,
    })
  }
  /** 查询订单数量汇总 */
  async CountOrderInfo(request: CountOrderInfoRequest): Promise<CountOrderInfoResponse> {
    const response = await http<Partial<CountOrderInfoResponse>>({
      url: `${ORDER_INFO_URL}/count`,
      method: 'GET',
      authMode: 'required',
      data: request,
    })
    return {
      ...response,
      counts: response.counts ?? [],
    }
  }
  /** 查询商品分页列表 */
  async PageOrderInfo(request: PageOrderInfoRequest): Promise<PageOrderInfoResponse> {
    const response = await http<Partial<PageOrderInfoResponse>>({
      url: `${ORDER_INFO_URL}`,
      method: 'GET',
      authMode: 'required',
      data: request,
    })
    return {
      ...response,
      order_infos: response.order_infos ?? [],
      total: response.total ?? 0,
    }
  }

  /** 根据订单编号查询订单id */
  async GetOrderInfoIdByOrderNo(
    request: GetOrderInfoIdByOrderNoRequest,
  ): Promise<GetOrderInfoIdByOrderNoResponse> {
    const response = await http<Partial<GetOrderInfoIdByOrderNoResponse>>({
      url: `${ORDER_INFO_URL}/no/${request.order_no}`,
      method: 'GET',
      authMode: 'required',
    })
    return {
      ...response,
      order_id: response.order_id ?? 0,
    }
  }
  /** 根据订单id查询订单 */
  GetOrderInfoById(request: GetOrderInfoByIdRequest): Promise<OrderInfoResponse> {
    return http<OrderInfoResponse>({
      url: `${ORDER_INFO_URL}/${request.id}`,
      method: 'GET',
      authMode: 'required',
    })
  }
  /** 根据交易单 ID 查询聚合订单 */
  GetOrderTradeById(request: GetOrderTradeByIdRequest): Promise<OrderInfoResponse> {
    return http<OrderInfoResponse>({
      url: `${ORDER_TRADE_URL}/${request.trade_id}`,
      method: 'GET',
      authMode: 'required',
    })
  }
  /** 创建订单 */
  CreateOrderInfo(request: CreateOrderInfoRequest): Promise<CreateOrderInfoResponse> {
    return http<CreateOrderInfoResponse>({
      url: `${ORDER_INFO_URL}`,
      method: 'POST',
      authMode: 'required',
      data: request,
    })
  }
  /** 删除订单 */
  DeleteOrderInfo(request: DeleteOrderInfoRequest): Promise<Empty> {
    return http<Empty>({
      url: `${ORDER_INFO_URL}/${request.id}`,
      method: 'DELETE',
      authMode: 'required',
    })
  }
  /** 删除已关闭交易 */
  DeleteOrderTrade(request: DeleteOrderTradeRequest): Promise<Empty> {
    return http<Empty>({
      url: `${ORDER_TRADE_URL}/${request.trade_id}`,
      method: 'DELETE',
      authMode: 'required',
    })
  }
  /** 取消订单 */
  CancelOrderInfo(request: CancelOrderInfoRequest): Promise<Empty> {
    return http<Empty>({
      url: `${ORDER_TRADE_URL}/${request.trade_id}/cancellation`,
      method: 'PUT',
      authMode: 'required',
      data: request,
    })
  }
  /** 订单退款 */
  RefundOrderInfo(request: RefundOrderInfoRequest): Promise<Empty> {
    return http<Empty>({
      url: `${ORDER_INFO_URL}/${request.order_id}/refund`,
      method: 'PUT',
      authMode: 'required',
      data: request,
    })
  }
  /** 确认收货 */
  ReceiveOrderInfo(request: ReceiveOrderInfoRequest): Promise<Empty> {
    return http<Empty>({
      url: `${ORDER_INFO_URL}/${request.order_id}/receipt`,
      method: 'PUT',
      authMode: 'required',
      data: request,
    })
  }
}

export const defOrderService = new OrderInfoServiceImpl()

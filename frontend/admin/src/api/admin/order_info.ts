import service from "@/utils/request";
import {
  type OrderInfoRefundResponse,
  type OrderInfoResponse,
  type OrderInfoService,
  type OrderInfoShipmentForm,
  type PageOrderInfoRequest,
  type PageOrderInfoResponse,
  type RefundOrderInfoRequest,
  type ShipOrderInfoRequest
} from "@/rpc/admin/order_info";
import type { Int64Value } from "@/rpc/google/protobuf/wrappers";
import type { Empty } from "@/rpc/google/protobuf/empty";

const ORDER_URL = "/admin/order/info";

/** 订单服务 */
export class OrderInfoServiceImpl implements OrderInfoService {
  /** 查询订单分页列表 */
  PageOrderInfo(request: PageOrderInfoRequest): Promise<PageOrderInfoResponse> {
    return service<PageOrderInfoRequest, PageOrderInfoResponse>({
      url: `${ORDER_URL}`,
      method: "get",
      params: request
    });
  }
  /** 查询订单 */
  GetOrderInfo(request: Int64Value): Promise<OrderInfoResponse> {
    return service<Int64Value, OrderInfoResponse>({
      url: `${ORDER_URL}/${request.value}`,
      method: "get"
    });
  }
  /** 查询订单退款信息 */
  GetOrderInfoRefund(request: Int64Value): Promise<OrderInfoRefundResponse> {
    return service<Int64Value, OrderInfoRefundResponse>({
      url: `${ORDER_URL}/${request.value}/refund`,
      method: "get"
    });
  }
  /** 订单退款 */
  RefundOrderInfo(request: RefundOrderInfoRequest): Promise<Empty> {
    return service<RefundOrderInfoRequest, Empty>({
      url: `${ORDER_URL}/${request.orderId}/refund`,
      method: "put",
      data: request
    });
  }
  /** 查询订单发货信息 */
  GetOrderInfoShipment(request: Int64Value): Promise<OrderInfoShipmentForm> {
    return service<Int64Value, OrderInfoShipmentForm>({
      url: `${ORDER_URL}/${request.value}/shipment`,
      method: "get"
    });
  }
  /** 订单发货 */
  ShipOrderInfo(request: ShipOrderInfoRequest): Promise<Empty> {
    return service<ShipOrderInfoRequest, Empty>({
      url: `${ORDER_URL}/${request.orderId}/shipment`,
      method: "put",
      data: request
    });
  }
}

export const defOrderInfoService = new OrderInfoServiceImpl();

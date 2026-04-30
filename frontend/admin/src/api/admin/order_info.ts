import service from "@/utils/request";
import {
  type GetOrderInfoRefundRequest,
  type GetOrderInfoShipmentRequest,
  type GetOrderInfoRequest,
  type OrderInfoRefundResponse,
  type OrderInfoResponse,
  type OrderInfoService,
  type OrderInfoShipmentForm,
  type PageOrderInfosRequest,
  type PageOrderInfosResponse,
  type RefundOrderInfoRequest,
  type ShipOrderInfoRequest
} from "@/rpc/admin/v1/order_info";
import type { Empty } from "@/rpc/google/protobuf/empty";

const ORDER_URL = "/v1/admin/order/info";

/** 订单服务 */
export class OrderInfoServiceImpl implements OrderInfoService {
  /** 查询订单分页列表 */
  PageOrderInfos(request: PageOrderInfosRequest): Promise<PageOrderInfosResponse> {
    return service<PageOrderInfosRequest, PageOrderInfosResponse>({
      url: `${ORDER_URL}`,
      method: "get",
      params: request
    });
  }
  /** 查询订单 */
  GetOrderInfo(request: GetOrderInfoRequest): Promise<OrderInfoResponse> {
    return service<GetOrderInfoRequest, OrderInfoResponse>({
      url: `${ORDER_URL}/${request.id}`,
      method: "get"
    });
  }
  /** 查询订单退款信息 */
  GetOrderInfoRefund(request: GetOrderInfoRefundRequest): Promise<OrderInfoRefundResponse> {
    return service<GetOrderInfoRefundRequest, OrderInfoRefundResponse>({
      url: `${ORDER_URL}/${request.id}/refund`,
      method: "get"
    });
  }
  /** 订单退款 */
  RefundOrderInfo(request: RefundOrderInfoRequest): Promise<Empty> {
    return service<RefundOrderInfoRequest, Empty>({
      url: `${ORDER_URL}/${request.order_id}/refund`,
      method: "put",
      data: request
    });
  }
  /** 查询订单发货信息 */
  GetOrderInfoShipment(request: GetOrderInfoShipmentRequest): Promise<OrderInfoShipmentForm> {
    return service<GetOrderInfoShipmentRequest, OrderInfoShipmentForm>({
      url: `${ORDER_URL}/${request.id}/shipment`,
      method: "get"
    });
  }
  /** 订单发货 */
  ShipOrderInfo(request: ShipOrderInfoRequest): Promise<Empty> {
    return service<ShipOrderInfoRequest, Empty>({
      url: `${ORDER_URL}/${request.order_id}/shipment`,
      method: "put",
      data: request
    });
  }
}

export const defOrderInfoService = new OrderInfoServiceImpl();

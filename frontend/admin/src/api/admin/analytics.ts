import service from "@/utils/request";
import {
  type AnalyticsBarSaleRequest,
  type AnalyticsBarOrderRequest,
  type AnalyticsBarResponse,
  type AnalyticsCountRequest,
  type AnalyticsCountResponse,
  type AnalyticsPieGoodsRequest,
  type AnalyticsPieOrderRequest,
  type AnalyticsPieResponse,
  type AnalyticsService
} from "@/rpc/admin/analytics";

const ADMIN_ANALYTICS = "/admin/analytics";

/** Admin 数据分析服务 */
export class AnalyticsServiceImpl implements AnalyticsService {
  /** 查询汇总数据（用户） */
  AnalyticsCountUser(request: AnalyticsCountRequest): Promise<AnalyticsCountResponse> {
    return service<AnalyticsCountRequest, AnalyticsCountResponse>({
      url: `${ADMIN_ANALYTICS}/count/user`,
      method: "get",
      params: request
    });
  }
  /** 查询汇总数据（商品） */
  AnalyticsCountGoods(request: AnalyticsCountRequest): Promise<AnalyticsCountResponse> {
    return service<AnalyticsCountRequest, AnalyticsCountResponse>({
      url: `${ADMIN_ANALYTICS}/count/goods`,
      method: "get",
      params: request
    });
  }
  /** 查询汇总数据（订单） */
  AnalyticsCountOrder(request: AnalyticsCountRequest): Promise<AnalyticsCountResponse> {
    return service<AnalyticsCountRequest, AnalyticsCountResponse>({
      url: `${ADMIN_ANALYTICS}/count/order`,
      method: "get",
      params: request
    });
  }
  /** 查询汇总数据（销量） */
  AnalyticsCountSale(request: AnalyticsCountRequest): Promise<AnalyticsCountResponse> {
    return service<AnalyticsCountRequest, AnalyticsCountResponse>({
      url: `${ADMIN_ANALYTICS}/count/sale`,
      method: "get",
      params: request
    });
  }
  /** 查询订单销量（柱状图） */
  AnalyticsBarOrder(request: AnalyticsBarOrderRequest): Promise<AnalyticsBarResponse> {
    return service<AnalyticsBarOrderRequest, AnalyticsBarResponse>({
      url: `${ADMIN_ANALYTICS}/bar/order`,
      method: "get",
      params: request
    });
  }
  /** 查询订单销售额（柱状图） */
  AnalyticsBarSale(request: AnalyticsBarSaleRequest): Promise<AnalyticsBarResponse> {
    return service<AnalyticsBarSaleRequest, AnalyticsBarResponse>({
      url: `${ADMIN_ANALYTICS}/bar/sale`,
      method: "get",
      params: request
    });
  }
  /** 查询商品分类（饼状图） */
  AnalyticsPieGoods(request: AnalyticsPieGoodsRequest): Promise<AnalyticsPieResponse> {
    return service<AnalyticsPieGoodsRequest, AnalyticsPieResponse>({
      url: `${ADMIN_ANALYTICS}/pie/goods`,
      method: "get",
      params: request
    });
  }
  /** 查询订单状态（饼状图） */
  AnalyticsPieOrder(request: AnalyticsPieOrderRequest): Promise<AnalyticsPieResponse> {
    return service<AnalyticsPieOrderRequest, AnalyticsPieResponse>({
      url: `${ADMIN_ANALYTICS}/pie/order`,
      method: "get",
      params: request
    });
  }
}

export const defAnalyticsService = new AnalyticsServiceImpl();

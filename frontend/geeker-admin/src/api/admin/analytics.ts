import service from "@/utils/request";
import {
  type AnalyticsBarGoodsRequest,
  type AnalyticsBarOrderRequest,
  type AnalyticsBarResponse,
  type AnalyticsCountRequest,
  type AnalyticsCountResponse,
  type AnalyticsPieGoodsRequest,
  type AnalyticsPieResponse,
  type AnalyticsRadarOrderRequest,
  type AnalyticsRadarResponse,
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
  /** 查询商品销量（柱状图） */
  AnalyticsBarGoods(request: AnalyticsBarGoodsRequest): Promise<AnalyticsBarResponse> {
    return service<AnalyticsBarGoodsRequest, AnalyticsBarResponse>({
      url: `${ADMIN_ANALYTICS}/bar/goods`,
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
  /** 查询商品订单销量状态（雷达图） */
  AnalyticsRadarOrder(request: AnalyticsRadarOrderRequest): Promise<AnalyticsRadarResponse> {
    return service<AnalyticsRadarOrderRequest, AnalyticsRadarResponse>({
      url: `${ADMIN_ANALYTICS}/radar/order`,
      method: "get",
      params: request
    });
  }
}

export const defAnalyticsService = new AnalyticsServiceImpl();

import service from "@/utils/request";
import {
  type DashboardBarGoodsRequest,
  type DashboardBarOrderRequest,
  type DashboardBarResponse,
  type DashboardCountRequest,
  type DashboardCountResponse,
  type DashboardPieGoodsRequest,
  type DashboardPieResponse,
  type DashboardRadarOrderRequest,
  type DashboardRadarResponse,
  type DashboardService,
} from "@/rpc/admin/dashboard";

const GOODS_URL = "/admin/dashboard";

/** Admin首页服务 */
export class DashboardServiceImpl implements DashboardService {
  /** 查询汇总数据（用户） */
  DashboardCountUser(request: DashboardCountRequest): Promise<DashboardCountResponse> {
    return service<DashboardCountRequest, DashboardCountResponse>({
      url: `${GOODS_URL}/count/user`,
      method: "get",
      params: request,
    });
  }
  /** 查询汇总数据（商品） */
  DashboardCountGoods(request: DashboardCountRequest): Promise<DashboardCountResponse> {
    return service<DashboardCountRequest, DashboardCountResponse>({
      url: `${GOODS_URL}/count/goods`,
      method: "get",
      params: request,
    });
  }
  /** 查询汇总数据（订单） */
  DashboardCountOrder(request: DashboardCountRequest): Promise<DashboardCountResponse> {
    return service<DashboardCountRequest, DashboardCountResponse>({
      url: `${GOODS_URL}/count/order`,
      method: "get",
      params: request,
    });
  }
  /** 查询汇总数据（销量） */
  DashboardCountSale(request: DashboardCountRequest): Promise<DashboardCountResponse> {
    return service<DashboardCountRequest, DashboardCountResponse>({
      url: `${GOODS_URL}/count/sale`,
      method: "get",
      params: request,
    });
  }
  /** 查询订单销量（柱状图） */
  DashboardBarOrder(request: DashboardBarOrderRequest): Promise<DashboardBarResponse> {
    return service<DashboardBarOrderRequest, DashboardBarResponse>({
      url: `${GOODS_URL}/bar/order`,
      method: "get",
      params: request,
    });
  }
  /** 查询商品销量（柱状图） */
  DashboardBarGoods(request: DashboardBarGoodsRequest): Promise<DashboardBarResponse> {
    return service<DashboardBarGoodsRequest, DashboardBarResponse>({
      url: `${GOODS_URL}/bar/goods`,
      method: "get",
      params: request,
    });
  }
  /** 查询商品分类（饼状图） */
  DashboardPieGoods(request: DashboardPieGoodsRequest): Promise<DashboardPieResponse> {
    return service<DashboardPieGoodsRequest, DashboardPieResponse>({
      url: `${GOODS_URL}/pie/goods`,
      method: "get",
      params: request,
    });
  }
  /** 查询商品订单销量状态（雷达图） */
  DashboardRadarOrder(request: DashboardRadarOrderRequest): Promise<DashboardRadarResponse> {
    return service<DashboardRadarOrderRequest, DashboardRadarResponse>({
      url: `${GOODS_URL}/radar/order`,
      method: "get",
      params: request,
    });
  }
}

export const defDashboardService = new DashboardServiceImpl();

import service from "@/utils/request";
import type {
  PieOrderAnalyticsRequest,
  SummaryOrderAnalyticsRequest,
  SummaryOrderAnalyticsResponse,
  TrendOrderAnalyticsRequest,
  OrderAnalyticsService
} from "@/rpc/admin/v1/order_analytics";
import type { AnalyticsPieResponse, AnalyticsTrendResponse } from "@/rpc/common/v1/analytics";

const ADMIN_ANALYTICS = "/v1/admin/analytics";

/** Admin 订单分析服务 */
export class OrderAnalyticsServiceImpl implements OrderAnalyticsService {
  /** 查询订单摘要指标 */
  SummaryOrderAnalytics(request: SummaryOrderAnalyticsRequest): Promise<SummaryOrderAnalyticsResponse> {
    return service<SummaryOrderAnalyticsRequest, SummaryOrderAnalyticsResponse>({
      url: `${ADMIN_ANALYTICS}/order/summary`,
      method: "get",
      params: request
    });
  }

  /** 查询订单趋势 */
  TrendOrderAnalytics(request: TrendOrderAnalyticsRequest): Promise<AnalyticsTrendResponse> {
    return service<TrendOrderAnalyticsRequest, AnalyticsTrendResponse>({
      url: `${ADMIN_ANALYTICS}/order/trend`,
      method: "get",
      params: request
    });
  }

  /** 查询订单状态分布 */
  PieOrderAnalytics(request: PieOrderAnalyticsRequest): Promise<AnalyticsPieResponse> {
    return service<PieOrderAnalyticsRequest, AnalyticsPieResponse>({
      url: `${ADMIN_ANALYTICS}/order/pie`,
      method: "get",
      params: request
    });
  }
}

export const defOrderAnalyticsService = new OrderAnalyticsServiceImpl();

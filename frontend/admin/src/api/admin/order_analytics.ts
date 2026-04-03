import service from "@/utils/request";
import type { OrderAnalyticsService } from "@/rpc/admin/order_analytics";
import type { OrderAnalyticsSummaryResponse } from "@/rpc/admin/order_analytics";
import type { AnalyticsPieResponse, AnalyticsTimeRequest, AnalyticsTrendResponse } from "@/rpc/common/analytics";

const ADMIN_ANALYTICS = "/admin/analytics";

/** Admin 订单分析服务 */
export class OrderAnalyticsServiceImpl implements OrderAnalyticsService {
  /** 查询订单摘要指标 */
  GetOrderAnalyticsSummary(request: AnalyticsTimeRequest): Promise<OrderAnalyticsSummaryResponse> {
    return service<AnalyticsTimeRequest, OrderAnalyticsSummaryResponse>({
      url: `${ADMIN_ANALYTICS}/order/summary`,
      method: "get",
      params: request
    });
  }

  /** 查询订单趋势 */
  GetOrderAnalyticsTrend(request: AnalyticsTimeRequest): Promise<AnalyticsTrendResponse> {
    return service<AnalyticsTimeRequest, AnalyticsTrendResponse>({
      url: `${ADMIN_ANALYTICS}/order/trend`,
      method: "get",
      params: request
    });
  }

  /** 查询订单状态分布 */
  GetOrderAnalyticsPie(request: AnalyticsTimeRequest): Promise<AnalyticsPieResponse> {
    return service<AnalyticsTimeRequest, AnalyticsPieResponse>({
      url: `${ADMIN_ANALYTICS}/order/pie`,
      method: "get",
      params: request
    });
  }
}

export const defOrderAnalyticsService = new OrderAnalyticsServiceImpl();

import service from "@/utils/request";
import type { AnalyticsRankResponse, AnalyticsTimeRequest, AnalyticsTrendResponse } from "@/rpc/common/analytics";
import type { UserAnalyticsService } from "@/rpc/admin/user_analytics";
import type { UserAnalyticsSummaryResponse } from "@/rpc/admin/user_analytics";

const ADMIN_ANALYTICS = "/admin/analytics";

/** Admin 用户分析服务 */
export class UserAnalyticsServiceImpl implements UserAnalyticsService {
  /** 查询用户摘要指标 */
  GetUserAnalyticsSummary(request: AnalyticsTimeRequest): Promise<UserAnalyticsSummaryResponse> {
    return service<AnalyticsTimeRequest, UserAnalyticsSummaryResponse>({
      url: `${ADMIN_ANALYTICS}/user/summary`,
      method: "get",
      params: request
    });
  }

  /** 查询用户趋势 */
  GetUserAnalyticsTrend(request: AnalyticsTimeRequest): Promise<AnalyticsTrendResponse> {
    return service<AnalyticsTimeRequest, AnalyticsTrendResponse>({
      url: `${ADMIN_ANALYTICS}/user/trend`,
      method: "get",
      params: request
    });
  }

  /** 查询用户行为覆盖排行 */
  GetUserAnalyticsRank(request: AnalyticsTimeRequest): Promise<AnalyticsRankResponse> {
    return service<AnalyticsTimeRequest, AnalyticsRankResponse>({
      url: `${ADMIN_ANALYTICS}/user/rank`,
      method: "get",
      params: request
    });
  }
}

export const defUserAnalyticsService = new UserAnalyticsServiceImpl();

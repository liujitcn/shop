import service from "@/utils/request";
import type { AnalyticsRankResponse, AnalyticsTrendResponse } from "@/rpc/common/v1/analytics";
import type {
  RankUserAnalyticsRequest,
  SummaryUserAnalyticsRequest,
  SummaryUserAnalyticsResponse,
  TrendUserAnalyticsRequest,
  UserAnalyticsService
} from "@/rpc/admin/v1/user_analytics";

const ADMIN_ANALYTICS = "/v1/admin/analytics";

/** Admin 用户分析服务 */
export class UserAnalyticsServiceImpl implements UserAnalyticsService {
  /** 查询用户摘要指标 */
  SummaryUserAnalytics(request: SummaryUserAnalyticsRequest): Promise<SummaryUserAnalyticsResponse> {
    return service<SummaryUserAnalyticsRequest, SummaryUserAnalyticsResponse>({
      url: `${ADMIN_ANALYTICS}/user/summary`,
      method: "get",
      params: request
    });
  }

  /** 查询用户趋势 */
  TrendUserAnalytics(request: TrendUserAnalyticsRequest): Promise<AnalyticsTrendResponse> {
    return service<TrendUserAnalyticsRequest, AnalyticsTrendResponse>({
      url: `${ADMIN_ANALYTICS}/user/trend`,
      method: "get",
      params: request
    });
  }

  /** 查询用户行为覆盖排行 */
  RankUserAnalytics(request: RankUserAnalyticsRequest): Promise<AnalyticsRankResponse> {
    return service<RankUserAnalyticsRequest, AnalyticsRankResponse>({
      url: `${ADMIN_ANALYTICS}/user/rank`,
      method: "get",
      params: request
    });
  }
}

export const defUserAnalyticsService = new UserAnalyticsServiceImpl();

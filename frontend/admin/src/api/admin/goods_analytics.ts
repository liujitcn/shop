import service from "@/utils/request";
import type { GoodsAnalyticsService } from "@/rpc/admin/goods_analytics";
import type { GoodsAnalyticsSummaryResponse } from "@/rpc/admin/goods_analytics";
import type { AnalyticsPieResponse, AnalyticsTimeRequest, AnalyticsTrendResponse } from "@/rpc/common/analytics";

const ADMIN_ANALYTICS = "/admin/analytics";

/** Admin 商品分析服务 */
export class GoodsAnalyticsServiceImpl implements GoodsAnalyticsService {
  /** 查询商品摘要指标 */
  GetGoodsAnalyticsSummary(request: AnalyticsTimeRequest): Promise<GoodsAnalyticsSummaryResponse> {
    return service<AnalyticsTimeRequest, GoodsAnalyticsSummaryResponse>({
      url: `${ADMIN_ANALYTICS}/goods/summary`,
      method: "get",
      params: request
    });
  }

  /** 查询商品趋势 */
  GetGoodsAnalyticsTrend(request: AnalyticsTimeRequest): Promise<AnalyticsTrendResponse> {
    return service<AnalyticsTimeRequest, AnalyticsTrendResponse>({
      url: `${ADMIN_ANALYTICS}/goods/trend`,
      method: "get",
      params: request
    });
  }

  /** 查询商品分类分布 */
  GetGoodsAnalyticsPie(request: AnalyticsTimeRequest): Promise<AnalyticsPieResponse> {
    return service<AnalyticsTimeRequest, AnalyticsPieResponse>({
      url: `${ADMIN_ANALYTICS}/goods/pie`,
      method: "get",
      params: request
    });
  }
}

export const defGoodsAnalyticsService = new GoodsAnalyticsServiceImpl();

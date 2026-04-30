import service from "@/utils/request";
import type {
  PieGoodsAnalyticsRequest,
  RankGoodsAnalyticsRequest,
  SummaryGoodsAnalyticsRequest,
  SummaryGoodsAnalyticsResponse,
  TrendGoodsAnalyticsRequest,
  GoodsAnalyticsService
} from "@/rpc/admin/v1/goods_analytics";
import type { AnalyticsPieResponse, AnalyticsRankResponse, AnalyticsTrendResponse } from "@/rpc/common/v1/analytics";

const ADMIN_ANALYTICS = "/v1/admin/analytics";

/** Admin 商品分析服务 */
export class GoodsAnalyticsServiceImpl implements GoodsAnalyticsService {
  /** 查询商品摘要指标 */
  SummaryGoodsAnalytics(request: SummaryGoodsAnalyticsRequest): Promise<SummaryGoodsAnalyticsResponse> {
    return service<SummaryGoodsAnalyticsRequest, SummaryGoodsAnalyticsResponse>({
      url: `${ADMIN_ANALYTICS}/goods/summary`,
      method: "get",
      params: request
    });
  }

  /** 查询商品趋势 */
  TrendGoodsAnalytics(request: TrendGoodsAnalyticsRequest): Promise<AnalyticsTrendResponse> {
    return service<TrendGoodsAnalyticsRequest, AnalyticsTrendResponse>({
      url: `${ADMIN_ANALYTICS}/goods/trend`,
      method: "get",
      params: request
    });
  }

  /** 查询商品分类分布 */
  PieGoodsAnalytics(request: PieGoodsAnalyticsRequest): Promise<AnalyticsPieResponse> {
    return service<PieGoodsAnalyticsRequest, AnalyticsPieResponse>({
      url: `${ADMIN_ANALYTICS}/goods/pie`,
      method: "get",
      params: request
    });
  }

  /** 查询商品支付排行 */
  RankGoodsAnalytics(request: RankGoodsAnalyticsRequest): Promise<AnalyticsRankResponse> {
    return service<RankGoodsAnalyticsRequest, AnalyticsRankResponse>({
      url: `${ADMIN_ANALYTICS}/goods/rank`,
      method: "get",
      params: request
    });
  }
}

export const defGoodsAnalyticsService = new GoodsAnalyticsServiceImpl();

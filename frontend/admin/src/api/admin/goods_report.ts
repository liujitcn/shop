import service from "@/utils/request";
import type {
  GoodsDayReportListRequest,
  GoodsDayReportListResponse,
  GoodsDayReportSummaryRequest,
  GoodsDayReportSummaryResponse,
  GoodsMonthReportListRequest,
  GoodsMonthReportListResponse,
  GoodsMonthReportSummaryRequest,
  GoodsMonthReportSummaryResponse
} from "@/rpc/admin/goods_report";

const GOODS_REPORT_URL = "/admin/report/goods";

/** Admin 商品报表服务 */
export class GoodsReportServiceImpl {
  /** 查询商品月报汇总 */
  GoodsMonthReportSummary(request: GoodsMonthReportSummaryRequest): Promise<GoodsMonthReportSummaryResponse> {
    return service<GoodsMonthReportSummaryRequest, GoodsMonthReportSummaryResponse>({
      url: `${GOODS_REPORT_URL}/month/summary`,
      method: "get",
      params: request
    });
  }

  /** 查询商品月报名细 */
  GoodsMonthReportList(request: GoodsMonthReportListRequest): Promise<GoodsMonthReportListResponse> {
    return service<GoodsMonthReportListRequest, GoodsMonthReportListResponse>({
      url: `${GOODS_REPORT_URL}/month/detail`,
      method: "get",
      params: request
    });
  }

  /** 查询商品日报汇总 */
  GoodsDayReportSummary(request: GoodsDayReportSummaryRequest): Promise<GoodsDayReportSummaryResponse> {
    return service<GoodsDayReportSummaryRequest, GoodsDayReportSummaryResponse>({
      url: `${GOODS_REPORT_URL}/day/summary`,
      method: "get",
      params: request
    });
  }

  /** 查询商品日报明细 */
  GoodsDayReportList(request: GoodsDayReportListRequest): Promise<GoodsDayReportListResponse> {
    return service<GoodsDayReportListRequest, GoodsDayReportListResponse>({
      url: `${GOODS_REPORT_URL}/day/detail`,
      method: "get",
      params: request
    });
  }
}

export const defGoodsReportService = new GoodsReportServiceImpl();

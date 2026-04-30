import service from "@/utils/request";
import type {
  SummaryGoodsDayReportRequest,
  SummaryGoodsDayReportResponse,
  SummaryGoodsMonthReportRequest,
  SummaryGoodsMonthReportResponse,
  ListGoodsDayReportsRequest,
  ListGoodsDayReportsResponse,
  ListGoodsMonthReportsRequest,
  ListGoodsMonthReportsResponse
} from "@/rpc/admin/v1/goods_report";

const GOODS_REPORT_URL = "/v1/admin/report/goods";

/** Admin 商品报表服务 */
export class GoodsReportServiceImpl {
  /** 查询商品月报汇总 */
  SummaryGoodsMonthReport(request: SummaryGoodsMonthReportRequest): Promise<SummaryGoodsMonthReportResponse> {
    return service<SummaryGoodsMonthReportRequest, SummaryGoodsMonthReportResponse>({
      url: `${GOODS_REPORT_URL}/month/summary`,
      method: "get",
      params: request
    });
  }

  /** 查询商品月报名细 */
  ListGoodsMonthReports(request: ListGoodsMonthReportsRequest): Promise<ListGoodsMonthReportsResponse> {
    return service<ListGoodsMonthReportsRequest, ListGoodsMonthReportsResponse>({
      url: `${GOODS_REPORT_URL}/month`,
      method: "get",
      params: request
    });
  }

  /** 查询商品日报汇总 */
  SummaryGoodsDayReport(request: SummaryGoodsDayReportRequest): Promise<SummaryGoodsDayReportResponse> {
    return service<SummaryGoodsDayReportRequest, SummaryGoodsDayReportResponse>({
      url: `${GOODS_REPORT_URL}/day/summary`,
      method: "get",
      params: request
    });
  }

  /** 查询商品日报明细 */
  ListGoodsDayReports(request: ListGoodsDayReportsRequest): Promise<ListGoodsDayReportsResponse> {
    return service<ListGoodsDayReportsRequest, ListGoodsDayReportsResponse>({
      url: `${GOODS_REPORT_URL}/day`,
      method: "get",
      params: request
    });
  }
}

export const defGoodsReportService = new GoodsReportServiceImpl();

import service from "@/utils/request";
import type {
  SummaryOrderDayReportRequest,
  SummaryOrderDayReportResponse,
  SummaryOrderMonthReportRequest,
  SummaryOrderMonthReportResponse,
  ListOrderDayReportsRequest,
  ListOrderDayReportsResponse,
  ListOrderMonthReportsRequest,
  ListOrderMonthReportsResponse
} from "@/rpc/admin/v1/order_report";

const ORDER_REPORT_URL = "/v1/admin/report/order";

/** Admin 订单报表服务 */
export class OrderReportServiceImpl {
  /** 查询订单月报汇总 */
  SummaryOrderMonthReport(request: SummaryOrderMonthReportRequest): Promise<SummaryOrderMonthReportResponse> {
    return service<SummaryOrderMonthReportRequest, SummaryOrderMonthReportResponse>({
      url: `${ORDER_REPORT_URL}/month/summary`,
      method: "get",
      params: request
    });
  }

  /** 查询订单月报名细 */
  ListOrderMonthReports(request: ListOrderMonthReportsRequest): Promise<ListOrderMonthReportsResponse> {
    return service<ListOrderMonthReportsRequest, ListOrderMonthReportsResponse>({
      url: `${ORDER_REPORT_URL}/month`,
      method: "get",
      params: request
    });
  }

  /** 查询订单日报汇总 */
  SummaryOrderDayReport(request: SummaryOrderDayReportRequest): Promise<SummaryOrderDayReportResponse> {
    return service<SummaryOrderDayReportRequest, SummaryOrderDayReportResponse>({
      url: `${ORDER_REPORT_URL}/day/summary`,
      method: "get",
      params: request
    });
  }

  /** 查询订单日报明细 */
  ListOrderDayReports(request: ListOrderDayReportsRequest): Promise<ListOrderDayReportsResponse> {
    return service<ListOrderDayReportsRequest, ListOrderDayReportsResponse>({
      url: `${ORDER_REPORT_URL}/day`,
      method: "get",
      params: request
    });
  }
}

export const defOrderReportService = new OrderReportServiceImpl();

import service from "@/utils/request";
import type {
  OrderDayReportListRequest,
  OrderDayReportListResponse,
  OrderDayReportSummaryRequest,
  OrderDayReportSummaryResponse,
  OrderMonthReportListRequest,
  OrderMonthReportListResponse,
  OrderMonthReportSummaryRequest,
  OrderMonthReportSummaryResponse
} from "@/rpc/admin/order_report";

const ORDER_REPORT_URL = "/admin/report/order";

/** Admin 订单报表服务 */
export class OrderReportServiceImpl {
  /** 查询订单月报汇总 */
  OrderMonthReportSummary(request: OrderMonthReportSummaryRequest): Promise<OrderMonthReportSummaryResponse> {
    return service<OrderMonthReportSummaryRequest, OrderMonthReportSummaryResponse>({
      url: `${ORDER_REPORT_URL}/month/summary`,
      method: "get",
      params: request
    });
  }

  /** 查询订单月报名细 */
  OrderMonthReportList(request: OrderMonthReportListRequest): Promise<OrderMonthReportListResponse> {
    return service<OrderMonthReportListRequest, OrderMonthReportListResponse>({
      url: `${ORDER_REPORT_URL}/month/detail`,
      method: "get",
      params: request
    });
  }

  /** 查询订单日报汇总 */
  OrderDayReportSummary(request: OrderDayReportSummaryRequest): Promise<OrderDayReportSummaryResponse> {
    return service<OrderDayReportSummaryRequest, OrderDayReportSummaryResponse>({
      url: `${ORDER_REPORT_URL}/day/summary`,
      method: "get",
      params: request
    });
  }

  /** 查询订单日报明细 */
  OrderDayReportList(request: OrderDayReportListRequest): Promise<OrderDayReportListResponse> {
    return service<OrderDayReportListRequest, OrderDayReportListResponse>({
      url: `${ORDER_REPORT_URL}/day/detail`,
      method: "get",
      params: request
    });
  }
}

export const defOrderReportService = new OrderReportServiceImpl();

import service from "@/utils/request";
import { type PagePayBillsRequest, type PagePayBillsResponse, type PayBillService } from "@/rpc/admin/v1/pay_bill";

const BASE_LOG_URL = "/v1/admin/pay/bill";

/** Admin支付对帐单服务 */
export class PayBillServiceImpl implements PayBillService {
  /** 查询支付对帐单列表 */
  PagePayBills(request: PagePayBillsRequest): Promise<PagePayBillsResponse> {
    return service<PagePayBillsRequest, PagePayBillsResponse>({
      url: `${BASE_LOG_URL}`,
      method: "get",
      params: request
    });
  }
}

export const defPayBillService = new PayBillServiceImpl();

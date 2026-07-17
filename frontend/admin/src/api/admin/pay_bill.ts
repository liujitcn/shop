import service from "@/utils/request";
import { type PagePayBillRequest, type PagePayBillResponse, type PayBillService } from "@/rpc/admin/v1/pay_bill";

const BASE_LOG_URL = "/v1/admin/pay/bill";

/** Admin支付对帐单服务 */
export class PayBillServiceImpl implements PayBillService {
  /** 查询支付对帐单列表 */
  PagePayBill(request: PagePayBillRequest): Promise<PagePayBillResponse> {
    return service<PagePayBillRequest, PagePayBillResponse>({
      url: `${BASE_LOG_URL}`,
      method: "get",
      params: request
    });
  }
}

export const defPayBillService = new PayBillServiceImpl();

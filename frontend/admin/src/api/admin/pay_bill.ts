import service from "@/utils/request";
import {
  type PayBillService,
  type PagePayBillRequest,
  type PagePayBillResponse,
} from "@/rpc/admin/pay_bill";

const BASE_LOG_URL = "/admin/pay/bill";

/** Admin支付对帐单服务 */
export class PayBillServiceImpl implements PayBillService {
  /** 查询支付对帐单分页列表 */
  PagePayBill(request: PagePayBillRequest): Promise<PagePayBillResponse> {
    return service<PagePayBillRequest, PagePayBillResponse>({
      url: `${BASE_LOG_URL}`,
      method: "get",
      params: request,
    });
  }
}

export const defPayBillService = new PayBillServiceImpl();

import { http } from '@/utils/http'
import type {
  H5PayRequest,
  H5PayResponse,
  JsapiPayRequest,
  JsapiPayResponse,
  PayService,
} from '@/rpc/app/pay'
import type { Empty } from '@/rpc/google/protobuf/empty'
const PAY_URL = '/app/pay'

/** 支付服务 */
export class PayServiceImpl implements PayService {
  /** 小程序支付 */
  JsapiPay(request: JsapiPayRequest): Promise<JsapiPayResponse> {
    return http<JsapiPayResponse>({
      url: `${PAY_URL}/${request.orderId}/jsapi`,
      method: 'POST',
      data: request,
    })
  }
  /** H5 支付 */
  H5Pay(request: H5PayRequest): Promise<H5PayResponse> {
    return http<H5PayResponse>({
      url: `${PAY_URL}/${request.orderId}/h5`,
      method: 'POST',
      data: request,
    })
  }
  /** 小程序支付 */
  PayNotify(request: Empty): Promise<Empty> {
    return http<Empty>({
      url: `${PAY_URL}/notify`,
      method: 'POST',
      data: request,
    })
  }
}

export const defPayService = new PayServiceImpl()

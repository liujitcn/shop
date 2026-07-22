import { defPayService } from '@/api/shop/pay'
import { appendOrderPaymentRedirectUrl, redirectToOrderPayment } from '@/utils/navigation'

/** 按当前运行平台发起微信支付，并在支付完成后进入对应交易的结果页。 */
export const startOrderPayment = async (tradeID: number) => {
  // 微信小程序使用 JSAPI 调起原生支付面板。
  // #ifdef MP-WEIXIN
  const jsapiRes = await defPayService.JsapiPay({ trade_id: tradeID })
  uni.requestPayment({
    provider: 'wxpay',
    nonceStr: jsapiRes.nonce_str,
    package: jsapiRes.package,
    paySign: jsapiRes.pay_sign,
    timeStamp: jsapiRes.time_stamp,
    signType: 'RSA',
    complete: () => {
      void redirectToOrderPayment(tradeID)
    },
  })
  // #endif

  // H5 和 App 使用微信 H5 支付链接，H5 端需显式携带商城结果页回跳地址。
  // #ifdef H5 || APP-PLUS
  const h5Res = await defPayService.H5Pay({ trade_id: tradeID })

  // #ifdef H5
  window.location.href = appendOrderPaymentRedirectUrl(h5Res.h5_url, tradeID)
  // #endif

  // #ifdef APP-PLUS
  plus.runtime.openURL(h5Res.h5_url)
  await redirectToOrderPayment(tradeID)
  // #endif
  // #endif
}

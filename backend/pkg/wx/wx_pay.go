package wx

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	nethttp "net/http"
	"strings"
	"time"

	appApi "shop/api/gen/go/app"
	"shop/api/gen/go/conf"
	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/wx/bill"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/google/uuid"
	"github.com/liujitcn/go-utils/trans"
	wxPayCore "github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/auth/verifiers"
	"github.com/wechatpay-apiv3/wechatpay-go/core/downloader"
	"github.com/wechatpay-apiv3/wechatpay-go/core/notify"
	"github.com/wechatpay-apiv3/wechatpay-go/core/option"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/h5"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/jsapi"
	"github.com/wechatpay-apiv3/wechatpay-go/services/refunddomestic"
	"github.com/wechatpay-apiv3/wechatpay-go/utils"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const notifyUrl = "/api/app/pay/notify"

// WxPayCase 微信支付业务实例。
type WxPayCase struct {
	*biz.BaseCase
	wxPay         *conf.WxPay
	mchPrivateKey *rsa.PrivateKey
	ctx           context.Context
	client        *wxPayCore.Client
}

// NewWxPayCase 创建微信支付业务实例。
func NewWxPayCase(baseCase *biz.BaseCase, wxPay *conf.WxPay) (*WxPayCase, error) {
	mchPrivateKey, err := utils.LoadPrivateKeyWithPath(wxPay.GetMchCertPath())
	if err != nil {
		return nil, err
	}

	opts := []wxPayCore.ClientOption{
		option.WithWechatPayAutoAuthCipher(wxPay.GetMchId(), wxPay.GetMchCertSn(), mchPrivateKey, wxPay.GetMchAPIv3Key()),
	}
	ctx := context.Background()

	var client *wxPayCore.Client
	client, err = wxPayCore.NewClient(ctx, opts...)
	if err != nil {
		return nil, err
	}

	return &WxPayCase{
		BaseCase:      baseCase,
		wxPay:         wxPay,
		mchPrivateKey: mchPrivateKey,
		ctx:           ctx,
		client:        client,
	}, nil
}

// JsapiPay 创建 JSAPI 支付预下单信息。
func (c *WxPayCase) JsapiPay(req jsapi.PrepayRequest) (*appApi.JsapiPayResponse, error) {
	// 拼接公共参数。
	req.Appid = trans.String(c.wxPay.GetAppid())
	req.Mchid = trans.String(c.wxPay.GetMchId())
	req.NotifyUrl = trans.String(c.wxPay.GetNotifyUrl() + notifyUrl)

	svc := jsapi.JsapiApiService{Client: c.client}
	resp, result, err := svc.Prepay(c.ctx, req)
	if err != nil {
		log.Errorf("支付失败[%s]", err.Error())
		return nil, err
	}
	// 微信支付返回非成功状态码时，统一按支付失败处理。
	if result.Response.StatusCode != nethttp.StatusOK {
		log.Errorf("支付失败[%s]", result.Response.Status)
		return nil, errorsx.Internal("支付失败")
	}

	nonceStr := strings.ReplaceAll(uuid.New().String(), "-", "")
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	packageStr := fmt.Sprintf("prepay_id=%s", trans.StringValue(resp.PrepayId))
	paySign := c.generatePaySign(timestamp, nonceStr, packageStr)

	return &appApi.JsapiPayResponse{
		AppId:     c.wxPay.GetAppid(),
		TimeStamp: timestamp,
		NonceStr:  nonceStr,
		Package:   packageStr,
		PaySign:   paySign,
	}, err
}

// H5Pay 创建 H5 支付预下单信息。
func (c *WxPayCase) H5Pay(req h5.PrepayRequest) (*appApi.H5PayResponse, error) {
	// 拼接公共参数。
	req.Appid = trans.String(c.wxPay.GetAppid())
	req.Mchid = trans.String(c.wxPay.GetMchId())
	req.NotifyUrl = trans.String(c.wxPay.GetNotifyUrl() + notifyUrl)

	svc := h5.H5ApiService{Client: c.client}
	resp, result, err := svc.Prepay(c.ctx, req)
	if err != nil {
		log.Errorf("支付失败[%s]", err.Error())
		return nil, err
	}
	// 微信支付返回非成功状态码时，统一按支付失败处理。
	if result.Response.StatusCode != nethttp.StatusOK {
		log.Errorf("支付失败[%s]", result.Response.Status)
		return nil, errorsx.Internal("支付失败")
	}

	return &appApi.H5PayResponse{
		H5Url: trans.StringValue(resp.H5Url),
	}, err
}

// TradeBill 申请交易账单。
func (c *WxPayCase) TradeBill(req bill.TradeBillRequest) (*bill.TradeBillResponse, error) {
	svc := bill.BillService{Client: c.client}
	resp, result, err := svc.TradeBill(c.ctx, req)
	if err != nil {
		log.Errorf("申请交易账单失败[%s]", err.Error())
		return nil, errorsx.Internal("申请交易账单失败")
	}
	// 微信账单接口返回非成功状态码时，统一按申请失败处理。
	if result.Response.StatusCode != nethttp.StatusOK {
		log.Errorf("申请交易账单失败[%s]", result.Response.Status)
		return nil, errorsx.Internal("申请交易账单失败")
	}
	return resp, nil
}

// DownloadBill 下载账单。
func (c *WxPayCase) DownloadBill(url string) ([]byte, error) {
	svc := bill.BillService{Client: c.client}
	return svc.DownloadBill(c.ctx, url)
}

// QueryOrderByOutTradeNo 根据商户订单号查询支付订单并转换为项目内支付资源结构。
func (c *WxPayCase) QueryOrderByOutTradeNo(orderNo string) (*appApi.PaymentResource, error) {
	req := jsapi.QueryOrderByOutTradeNoRequest{
		OutTradeNo: wxPayCore.String(orderNo),
		Mchid:      wxPayCore.String(c.wxPay.GetMchId()),
	}
	svc := jsapi.JsapiApiService{Client: c.client}
	resp, result, err := svc.QueryOrderByOutTradeNo(c.ctx, req)
	if err != nil {
		// 命中微信 API 错误结构时，优先识别可恢复的业务场景。
		if apiErr, ok := errors.AsType[*wxPayCore.APIError](err); ok {
			// 订单在微信侧不存在时，按未支付状态返回，避免上层继续报错。
			if apiErr.Code == "ORDER_NOT_EXIST" {
				return &appApi.PaymentResource{
					OutTradeNo:     trans.StringValue(req.OutTradeNo),
					Mchid:          c.wxPay.GetMchId(),
					TradeState:     appApi.PaymentResource_NOTPAY,
					TradeStateDesc: apiErr.Message,
				}, nil
			}
		}
		log.Errorf("查询支付失败[%s]", err.Error())
		return nil, errorsx.Internal("查询支付失败")
	}
	// 微信支付返回非成功状态码时，统一按查询失败处理。
	if result.Response.StatusCode != nethttp.StatusOK {
		log.Errorf("查询支付失败[%s]", result.Response.Status)
		return nil, errorsx.Internal("查询支付失败")
	}

	// 微信支付没有返回交易体时，视为查询失败。
	if resp == nil {
		return nil, errorsx.Internal("查询支付失败")
	}

	paymentResource := &appApi.PaymentResource{
		TransactionId:  trans.StringValue(resp.TransactionId),
		Mchid:          trans.StringValue(resp.Mchid),
		BankType:       trans.StringValue(resp.BankType),
		OutTradeNo:     trans.StringValue(resp.OutTradeNo),
		Appid:          trans.StringValue(resp.Appid),
		TradeStateDesc: trans.StringValue(resp.TradeStateDesc),
		Attach:         trans.StringValue(resp.Attach),
	}

	// 交易状态需要显式映射，避免依赖字符串反序列化带来的未知字段问题。
	switch trans.StringValue(resp.TradeState) {
	case appApi.PaymentResource_SUCCESS.String():
		paymentResource.TradeState = appApi.PaymentResource_SUCCESS
	case appApi.PaymentResource_REFUND.String():
		paymentResource.TradeState = appApi.PaymentResource_REFUND
	case appApi.PaymentResource_NOTPAY.String():
		paymentResource.TradeState = appApi.PaymentResource_NOTPAY
	case appApi.PaymentResource_CLOSED.String():
		paymentResource.TradeState = appApi.PaymentResource_CLOSED
	case appApi.PaymentResource_REVOKED.String():
		paymentResource.TradeState = appApi.PaymentResource_REVOKED
	case appApi.PaymentResource_USERPAYING.String():
		paymentResource.TradeState = appApi.PaymentResource_USERPAYING
	case appApi.PaymentResource_PAYERROR.String():
		paymentResource.TradeState = appApi.PaymentResource_PAYERROR
	default:
		paymentResource.TradeState = appApi.PaymentResource_TRADE_STATE_UNSPECIFIED
	}

	// 交易类型需要显式映射，确保返回结构与项目 proto 定义一致。
	switch trans.StringValue(resp.TradeType) {
	case appApi.PaymentResource_JSAPI.String():
		paymentResource.TradeType = appApi.PaymentResource_JSAPI
	case appApi.PaymentResource_NATIVE.String():
		paymentResource.TradeType = appApi.PaymentResource_NATIVE
	case appApi.PaymentResource_APP.String():
		paymentResource.TradeType = appApi.PaymentResource_APP
	case appApi.PaymentResource_MICROPAY.String():
		paymentResource.TradeType = appApi.PaymentResource_MICROPAY
	case appApi.PaymentResource_MWEB.String():
		paymentResource.TradeType = appApi.PaymentResource_MWEB
	case appApi.PaymentResource_FACEPAY.String():
		paymentResource.TradeType = appApi.PaymentResource_FACEPAY
	default:
		paymentResource.TradeType = appApi.PaymentResource_TRADE_TYPE_UNSPECIFIED
	}

	// 微信支付返回金额信息时，按项目内资源结构回填。
	if resp.Amount != nil {
		paymentResource.Amount = &appApi.PaymentResource_Amount{
			PayerTotal:    trans.Int64Value(resp.Amount.PayerTotal),
			Total:         trans.Int64Value(resp.Amount.Total),
			Currency:      trans.StringValue(resp.Amount.Currency),
			PayerCurrency: trans.StringValue(resp.Amount.PayerCurrency),
		}
	}

	// 微信支付返回支付者信息时，按项目内资源结构回填。
	if resp.Payer != nil {
		paymentResource.Payer = &appApi.PaymentResource_Payer{
			Openid: trans.StringValue(resp.Payer.Openid),
		}
	}

	// 微信支付成功时间存在时，统一转换成 protobuf 时间戳。
	if successTime := trans.StringValue(resp.SuccessTime); successTime != "" {
		// 微信支付返回 RFC3339 时间，这里统一转换为 protobuf 时间戳。
		parsedTime, parseErr := time.Parse(time.RFC3339, successTime)
		// 时间格式非法时，直接返回错误避免写入错误时间。
		if parseErr != nil {
			return nil, parseErr
		}
		paymentResource.SuccessTime = timestamppb.New(parsedTime)
	}

	return paymentResource, nil
}

// Refund 创建退款单。
func (c *WxPayCase) Refund(req refunddomestic.CreateRequest) (*refunddomestic.Refund, error) {
	// 拼接公共参数。
	req.NotifyUrl = trans.String(c.wxPay.GetNotifyUrl() + notifyUrl)

	svc := refunddomestic.RefundsApiService{Client: c.client}
	resp, result, err := svc.Create(c.ctx, req)
	if err != nil {
		log.Errorf("支付失败[%s]", err.Error())
		return nil, err
	}
	// 微信退款接口返回非成功状态码时，统一按退款失败处理。
	if result.Response.StatusCode != nethttp.StatusOK {
		log.Errorf("支付失败[%s]", result.Response.Status)
		return nil, errorsx.Internal("支付失败")
	}

	return resp, err
}

// QueryByOutRefundNo 根据商户退款单号查询退款单并转换为项目内退款资源结构。
func (c *WxPayCase) QueryByOutRefundNo(refundOrderNo string) (*appApi.RefundResource, error) {
	req := refunddomestic.QueryByOutRefundNoRequest{
		OutRefundNo: wxPayCore.String(refundOrderNo),
	}

	svc := refunddomestic.RefundsApiService{Client: c.client}
	resp, result, err := svc.QueryByOutRefundNo(c.ctx, req)
	if err != nil {
		log.Errorf("查询退款失败[%s]", err.Error())
		return nil, errorsx.Internal("查询退款失败")
	}
	// 微信退款查询接口返回非成功状态码时，统一按查询失败处理。
	if result.Response.StatusCode != nethttp.StatusOK {
		log.Errorf("查询退款失败[%s]", result.Response.Status)
		return nil, errorsx.Internal("查询退款失败")
	}

	// 微信退款查询没有返回退款体时，视为查询失败。
	if resp == nil {
		return nil, errorsx.Internal("查询退款失败")
	}

	refundResource := &appApi.RefundResource{
		TransactionId:       trans.StringValue(resp.TransactionId),
		OutTradeNo:          trans.StringValue(resp.OutTradeNo),
		RefundId:            trans.StringValue(resp.RefundId),
		OutRefundNo:         trans.StringValue(resp.OutRefundNo),
		UserReceivedAccount: trans.StringValue(resp.UserReceivedAccount),
	}

	// 退款状态需要显式映射，避免依赖字符串反序列化带来的未知字段问题。
	switch fmt.Sprint(resp.Status) {
	case appApi.RefundResource_SUCCESS.String():
		refundResource.RefundStatus = appApi.RefundResource_SUCCESS
	case appApi.RefundResource_CLOSED.String():
		refundResource.RefundStatus = appApi.RefundResource_CLOSED
	case appApi.RefundResource_PROCESSING.String():
		refundResource.RefundStatus = appApi.RefundResource_PROCESSING
	case appApi.RefundResource_ABNORMAL.String():
		refundResource.RefundStatus = appApi.RefundResource_ABNORMAL
	default:
		refundResource.RefundStatus = appApi.RefundResource_REFUND_STATUS_UNSPECIFIED
	}

	// 微信退款返回金额信息时，按项目内资源结构回填。
	if resp.Amount != nil {
		refundResource.Amount = &appApi.RefundResource_Amount{
			Total:       int32(trans.Int64Value(resp.Amount.Total)),
			Refund:      int32(trans.Int64Value(resp.Amount.Refund)),
			PayerTotal:  int32(trans.Int64Value(resp.Amount.PayerTotal)),
			PayerRefund: int32(trans.Int64Value(resp.Amount.PayerRefund)),
		}
	}

	// 微信退款 SDK 已将成功时间解析为 time.Time，这里统一转换为 protobuf 时间戳。
	if resp.SuccessTime != nil {
		refundResource.SuccessTime = timestamppb.New(*resp.SuccessTime)
	}

	return refundResource, nil
}

// Notify 解析微信支付回调通知。
func (c *WxPayCase) Notify(ctx context.Context) (*notify.Request, error) {
	err := downloader.MgrInstance().RegisterDownloaderWithPrivateKey(ctx, c.mchPrivateKey, c.wxPay.GetMchCertSn(), c.wxPay.GetMchId(), c.wxPay.GetMchAPIv3Key())
	if err != nil {
		return nil, err
	}
	certificateVisitor := downloader.MgrInstance().GetCertificateVisitor(c.wxPay.GetMchId())

	var handler *notify.Handler
	handler, err = notify.NewRSANotifyHandler(c.wxPay.GetMchAPIv3Key(), verifiers.NewSHA256WithRSAVerifier(certificateVisitor))
	if err != nil {
		return nil, err
	}
	var httpReq *nethttp.Request
	// 能从服务端上下文取到传输层信息时，继续尝试提取原始 HTTP 请求。
	if info, ok := transport.FromServerContext(ctx); ok {
		// 当前传输层为 HTTP 时，才能提取微信回调原始请求。
		if htr, htrOk := info.(*http.Transport); htrOk {
			httpReq = htr.Request()
		}
	}
	// 无法从上下文提取原始 HTTP 请求时，无法继续验签通知。
	if httpReq == nil {
		return nil, errorsx.Internal("支付通知请求转换失败")
	}
	var req *notify.Request
	req, err = handler.ParseNotifyRequest(ctx, httpReq, certificateVisitor)
	if err != nil {
		return nil, err
	}
	return req, nil
}

// generatePaySign 生成前端支付签名。
func (c *WxPayCase) generatePaySign(timeStamp, nonceStr, packageStr string) string {
	var signBuilder strings.Builder
	signBuilder.WriteString(c.wxPay.GetAppid() + "\n")
	signBuilder.WriteString(timeStamp + "\n")
	signBuilder.WriteString(nonceStr + "\n")
	signBuilder.WriteString(packageStr + "\n")
	signString := signBuilder.String()

	hashed := sha256.Sum256([]byte(signString))
	signature, err := rsa.SignPKCS1v15(rand.Reader, c.mchPrivateKey, crypto.SHA256, hashed[:])
	if err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(signature)
}

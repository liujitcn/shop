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
	"shop/api/gen/go/app"
	"shop/api/gen/go/conf"
	"shop/pkg/biz"
	"strings"
	"time"

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

// WxPayCase 微信支付业务实例
type WxPayCase struct {
	*biz.BaseCase
	wxPay         *conf.WxPay
	mchPrivateKey *rsa.PrivateKey
	ctx           context.Context
	client        *wxPayCore.Client
}

// NewWxPayCase 创建微信支付业务实例
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

func (c *WxPayCase) JsapiPay(req jsapi.PrepayRequest) (*app.JsapiPayResponse, error) {
	// 拼接公共参数
	req.Appid = trans.String(c.wxPay.GetAppid())
	req.Mchid = trans.String(c.wxPay.GetMchId())
	req.NotifyUrl = trans.String(c.wxPay.GetNotifyUrl() + notifyUrl)

	svc := jsapi.JsapiApiService{Client: c.client}
	resp, result, err := svc.Prepay(c.ctx, req)
	if err != nil {
		log.Errorf("支付失败[%s]", err.Error())
		return nil, errors.New("支付失败")
	}
	if result.Response.StatusCode != nethttp.StatusOK {
		log.Errorf("支付失败[%s]", result.Response.Status)
		return nil, errors.New("支付失败")
	}

	// 1. 生成基础参数
	nonceStr := strings.ReplaceAll(uuid.New().String(), "-", "")
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	packageStr := fmt.Sprintf("prepay_id=%s", trans.StringValue(resp.PrepayId))

	// 计算签名
	paySign := c.generatePaySign(timestamp, nonceStr, packageStr)

	return &app.JsapiPayResponse{
		AppId:     c.wxPay.GetAppid(),
		TimeStamp: timestamp,
		NonceStr:  nonceStr,
		Package:   packageStr,
		PaySign:   paySign,
	}, err
}

func (c *WxPayCase) H5Pay(req h5.PrepayRequest) (*app.H5PayResponse, error) {
	// 拼接公共参数
	req.Appid = trans.String(c.wxPay.GetAppid())
	req.Mchid = trans.String(c.wxPay.GetMchId())
	req.NotifyUrl = trans.String(c.wxPay.GetNotifyUrl() + notifyUrl)

	svc := h5.H5ApiService{Client: c.client}
	resp, result, err := svc.Prepay(c.ctx, req)
	if err != nil {
		log.Errorf("支付失败[%s]", err.Error())
		return nil, errors.New("支付失败")
	}
	if result.Response.StatusCode != nethttp.StatusOK {
		log.Errorf("支付失败[%s]", result.Response.Status)
		return nil, errors.New("支付失败")
	}

	return &app.H5PayResponse{
		H5Url: trans.StringValue(resp.H5Url),
	}, err
}

// QueryOrderByOutTradeNo 根据商户订单号查询支付订单并转换为项目内支付资源结构
func (c *WxPayCase) QueryOrderByOutTradeNo(req jsapi.QueryOrderByOutTradeNoRequest) (*app.PaymentResource, error) {
	req.Mchid = trans.String(c.wxPay.GetMchId())
	svc := jsapi.JsapiApiService{Client: c.client}
	resp, result, err := svc.QueryOrderByOutTradeNo(c.ctx, req)
	if err != nil {
		log.Errorf("查询支付失败[%s]", err.Error())
		return nil, errors.New("查询支付失败")
	}
	if result.Response.StatusCode != nethttp.StatusOK {
		log.Errorf("查询支付失败[%s]", result.Response.Status)
		return nil, errors.New("查询支付失败")
	}

	if resp == nil {
		return nil, errors.New("查询支付失败")
	}

	paymentResource := &app.PaymentResource{
		TransactionId:  trans.StringValue(resp.TransactionId),
		Mchid:          trans.StringValue(resp.Mchid),
		BankType:       trans.StringValue(resp.BankType),
		OutTradeNo:     trans.StringValue(resp.OutTradeNo),
		Appid:          trans.StringValue(resp.Appid),
		TradeStateDesc: trans.StringValue(resp.TradeStateDesc),
		Attach:         trans.StringValue(resp.Attach),
	}

	// 交易状态需要显式映射，避免依赖字符串反序列化带来的未知字段问题
	switch trans.StringValue(resp.TradeState) {
	case app.PaymentResource_SUCCESS.String():
		paymentResource.TradeState = app.PaymentResource_SUCCESS
	case app.PaymentResource_REFUND.String():
		paymentResource.TradeState = app.PaymentResource_REFUND
	case app.PaymentResource_NOTPAY.String():
		paymentResource.TradeState = app.PaymentResource_NOTPAY
	case app.PaymentResource_CLOSED.String():
		paymentResource.TradeState = app.PaymentResource_CLOSED
	case app.PaymentResource_REVOKED.String():
		paymentResource.TradeState = app.PaymentResource_REVOKED
	case app.PaymentResource_USERPAYING.String():
		paymentResource.TradeState = app.PaymentResource_USERPAYING
	case app.PaymentResource_PAYERROR.String():
		paymentResource.TradeState = app.PaymentResource_PAYERROR
	default:
		paymentResource.TradeState = app.PaymentResource_TRADE_STATE_UNSPECIFIED
	}

	// 交易类型需要显式映射，确保返回结构与项目 proto 定义一致
	switch trans.StringValue(resp.TradeType) {
	case app.PaymentResource_JSAPI.String():
		paymentResource.TradeType = app.PaymentResource_JSAPI
	case app.PaymentResource_NATIVE.String():
		paymentResource.TradeType = app.PaymentResource_NATIVE
	case app.PaymentResource_APP.String():
		paymentResource.TradeType = app.PaymentResource_APP
	case app.PaymentResource_MICROPAY.String():
		paymentResource.TradeType = app.PaymentResource_MICROPAY
	case app.PaymentResource_MWEB.String():
		paymentResource.TradeType = app.PaymentResource_MWEB
	case app.PaymentResource_FACEPAY.String():
		paymentResource.TradeType = app.PaymentResource_FACEPAY
	default:
		paymentResource.TradeType = app.PaymentResource_TRADE_TYPE_UNSPECIFIED
	}

	if resp.Amount != nil {
		paymentResource.Amount = &app.PaymentResource_Amount{
			PayerTotal:    trans.Int64Value(resp.Amount.PayerTotal),
			Total:         trans.Int64Value(resp.Amount.Total),
			Currency:      trans.StringValue(resp.Amount.Currency),
			PayerCurrency: trans.StringValue(resp.Amount.PayerCurrency),
		}
	}

	if resp.Payer != nil {
		paymentResource.Payer = &app.PaymentResource_Payer{
			Openid: trans.StringValue(resp.Payer.Openid),
		}
	}

	if successTime := trans.StringValue(resp.SuccessTime); successTime != "" {
		// 微信支付返回 RFC3339 时间，这里统一转换为 protobuf 时间戳
		parsedTime, parseErr := time.Parse(time.RFC3339, successTime)
		if parseErr != nil {
			return nil, parseErr
		}
		paymentResource.SuccessTime = timestamppb.New(parsedTime)
	}

	return paymentResource, nil
}

// Refund 创建退款单
func (c *WxPayCase) Refund(req refunddomestic.CreateRequest) (*refunddomestic.Refund, error) {
	// 拼接公共参数
	req.NotifyUrl = trans.String(c.wxPay.GetNotifyUrl() + notifyUrl)

	svc := refunddomestic.RefundsApiService{Client: c.client}
	resp, result, err := svc.Create(c.ctx, req)
	if err != nil {
		log.Errorf("支付失败[%s]", err.Error())
		return nil, errors.New("支付失败")
	}
	if result.Response.StatusCode != nethttp.StatusOK {
		log.Errorf("支付失败[%s]", result.Response.Status)
		return nil, errors.New("支付失败")
	}

	return resp, err
}

// QueryByOutRefundNo 根据商户退款单号查询退款单并转换为项目内退款资源结构
func (c *WxPayCase) QueryByOutRefundNo(req refunddomestic.QueryByOutRefundNoRequest) (*app.RefundResource, error) {
	// 拼接公共参数
	svc := refunddomestic.RefundsApiService{Client: c.client}
	resp, result, err := svc.QueryByOutRefundNo(c.ctx, req)
	if err != nil {
		log.Errorf("查询退款失败[%s]", err.Error())
		return nil, errors.New("查询退款失败")
	}
	if result.Response.StatusCode != nethttp.StatusOK {
		log.Errorf("查询退款失败[%s]", result.Response.Status)
		return nil, errors.New("查询退款失败")
	}

	if resp == nil {
		return nil, errors.New("查询退款失败")
	}

	refundResource := &app.RefundResource{
		TransactionId:       trans.StringValue(resp.TransactionId),
		OutTradeNo:          trans.StringValue(resp.OutTradeNo),
		RefundId:            trans.StringValue(resp.RefundId),
		OutRefundNo:         trans.StringValue(resp.OutRefundNo),
		UserReceivedAccount: trans.StringValue(resp.UserReceivedAccount),
	}

	// 退款状态需要显式映射，避免依赖字符串反序列化带来的未知字段问题
	switch fmt.Sprint(resp.Status) {
	case app.RefundResource_SUCCESS.String():
		refundResource.RefundStatus = app.RefundResource_SUCCESS
	case app.RefundResource_CLOSED.String():
		refundResource.RefundStatus = app.RefundResource_CLOSED
	case app.RefundResource_PROCESSING.String():
		refundResource.RefundStatus = app.RefundResource_PROCESSING
	case app.RefundResource_ABNORMAL.String():
		refundResource.RefundStatus = app.RefundResource_ABNORMAL
	default:
		refundResource.RefundStatus = app.RefundResource_REFUND_STATUS_UNSPECIFIED
	}

	if resp.Amount != nil {
		refundResource.Amount = &app.RefundResource_Amount{
			Total:       int32(trans.Int64Value(resp.Amount.Total)),
			Refund:      int32(trans.Int64Value(resp.Amount.Refund)),
			PayerTotal:  int32(trans.Int64Value(resp.Amount.PayerTotal)),
			PayerRefund: int32(trans.Int64Value(resp.Amount.PayerRefund)),
		}
	}

	if resp.SuccessTime != nil {
		// 微信支付 SDK 已将退款成功时间解析为 time.Time，这里统一转换为 protobuf 时间戳
		refundResource.SuccessTime = timestamppb.New(*resp.SuccessTime)
	}

	return refundResource, nil
}

// Notify 解析微信支付回调通知
func (c *WxPayCase) Notify(ctx context.Context) (*notify.Request, error) {
	// 1. 使用 `RegisterDownloaderWithPrivateKey` 注册下载器
	err := downloader.MgrInstance().RegisterDownloaderWithPrivateKey(ctx, c.mchPrivateKey, c.wxPay.GetMchCertSn(), c.wxPay.GetMchId(), c.wxPay.GetMchAPIv3Key())
	if err != nil {
		return nil, err
	}
	// 2. 获取商户号对应的微信支付平台证书访问器
	certificateVisitor := downloader.MgrInstance().GetCertificateVisitor(c.wxPay.GetMchId())

	// 3. 使用证书访问器初始化 `notify.Handler`
	var handler *notify.Handler
	handler, err = notify.NewRSANotifyHandler(c.wxPay.GetMchAPIv3Key(), verifiers.NewSHA256WithRSAVerifier(certificateVisitor))
	if err != nil {
		return nil, err
	}
	var httpReq *nethttp.Request
	if info, ok := transport.FromServerContext(ctx); ok {
		if htr, htrOk := info.(*http.Transport); htrOk {
			httpReq = htr.Request()
		}
	}
	if httpReq == nil {
		return nil, errors.New("transport convert nethttp request failed")
	}
	var req *notify.Request
	req, err = handler.ParseNotifyRequest(ctx, httpReq, certificateVisitor)
	if err != nil {
		return nil, err
	}
	return req, nil
}

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

package wx

import (
	"context"
	"crypto/rsa"
	"errors"
	nethttp "net/http"
	"shop/api/gen/go/conf"
	"shop/pkg/biz"
	"shop/service/admin/wx/bill"

	"github.com/go-kratos/kratos/v2/log"
	wxPayCore "github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/option"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/jsapi"
	"github.com/wechatpay-apiv3/wechatpay-go/services/refunddomestic"
	"github.com/wechatpay-apiv3/wechatpay-go/utils"
)

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

// TradeBill 申请交易账单
func (c *WxPayCase) TradeBill(req bill.TradeBillRequest) (*bill.TradeBillResponse, error) {
	svc := bill.BillService{Client: c.client}
	resp, result, err := svc.TradeBill(c.ctx, req)
	if err != nil {
		log.Errorf("申请交易账单失败[%s]", err.Error())
		return nil, errors.New("申请交易账单失败")
	}
	if result.Response.StatusCode != nethttp.StatusOK {
		log.Errorf("申请交易账单失败[%s]", result.Response.Status)
		return nil, errors.New("申请交易账单失败")
	}
	return resp, nil
}

// DownloadBill 下载账单
func (c *WxPayCase) DownloadBill(url string) ([]byte, error) {
	svc := bill.BillService{Client: c.client}
	return svc.DownloadBill(c.ctx, url)
}

// QueryOrderByOutTradeNo 按商户单号查询微信订单
func (c *WxPayCase) QueryOrderByOutTradeNo(req jsapi.QueryOrderByOutTradeNoRequest) (*payments.Transaction, error) {
	req.Mchid = wxPayCore.String(c.wxPay.GetMchId())
	svc := jsapi.JsapiApiService{Client: c.client}
	resp, result, err := svc.QueryOrderByOutTradeNo(c.ctx, req)
	if err != nil {
		log.Errorf("查询微信订单失败[%s]", err.Error())
		return nil, errors.New("查询微信订单失败")
	}
	if result.Response.StatusCode != nethttp.StatusOK {
		log.Errorf("查询微信订单失败[%s]", result.Response.Status)
		return nil, errors.New("查询微信订单失败")
	}
	return resp, nil
}

// Refund 申请微信退款
func (c *WxPayCase) Refund(req refunddomestic.CreateRequest) (*refunddomestic.Refund, error) {
	svc := refunddomestic.RefundsApiService{Client: c.client}
	resp, result, err := svc.Create(c.ctx, req)
	if err != nil {
		log.Errorf("申请退款失败[%s]", err.Error())
		return nil, errors.New("申请退款失败")
	}
	if result.Response.StatusCode != nethttp.StatusOK && result.Response.StatusCode != nethttp.StatusAccepted {
		log.Errorf("申请退款失败[%s]", result.Response.Status)
		return nil, errors.New("申请退款失败")
	}
	return resp, nil
}

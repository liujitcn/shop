package bill

import (
	"context"
	"io"
	nethttp "net/http"
	neturl "net/url"

	"shop/pkg/errorsx"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/auth/validators"
	"github.com/wechatpay-apiv3/wechatpay-go/core/consts"
	"github.com/wechatpay-apiv3/wechatpay-go/services"
)

// BillService 微信账单 API 服务。
type BillService services.Service

// TradeBill 申请交易账单。
func (a *BillService) TradeBill(ctx context.Context, req TradeBillRequest) (resp *TradeBillResponse, result *core.APIResult, err error) {
	// 缺少账单日期时，微信账单接口无法正常调用。
	if req.BillDate == nil || len(*req.BillDate) == 0 {
		return nil, nil, errorsx.InvalidArgument("账单日期不能为空")
	}
	// 缺少账单类型时，微信账单接口无法正常调用。
	if req.BillType == nil || len(*req.BillType) == 0 {
		return nil, nil, errorsx.InvalidArgument("账单类型不能为空")
	}

	requestPath := consts.WechatPayAPIServer + "/v3/bill/tradebill"

	queryParams := neturl.Values{}
	queryParams.Add("bill_date", core.ParameterToString(*req.BillDate, ""))
	queryParams.Add("bill_type", core.ParameterToString(*req.BillType, ""))

	httpContentType := core.SelectHeaderContentType([]string{})

	result, err = a.Client.Request(ctx, nethttp.MethodGet, requestPath, nethttp.Header{}, queryParams, nil, httpContentType)
	if err != nil {
		return nil, result, err
	}

	resp = new(TradeBillResponse)
	err = core.UnMarshalResponse(result.Response, resp)
	if err != nil {
		return nil, result, err
	}
	return resp, result, nil
}

// DownloadBill 下载账单。
func (a *BillService) DownloadBill(ctx context.Context, downloadURL string) ([]byte, error) {
	newClient := core.NewClientWithValidator(a.Client, &validators.NullValidator{})
	httpContentType := core.SelectHeaderContentType([]string{})
	result, err := newClient.Request(ctx, nethttp.MethodGet, downloadURL, nethttp.Header{}, neturl.Values{}, nil, httpContentType)
	if err != nil {
		return nil, err
	}
	httpResp := result.Response

	var body []byte
	body, err = io.ReadAll(httpResp.Body)
	defer func(body io.ReadCloser) {
		// 关闭响应体失败时，仅记录日志，不覆盖主流程错误。
		closeErr := body.Close()
		if closeErr != nil {
			log.Errorf("failed to close body: %v", closeErr)
		}
	}(httpResp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

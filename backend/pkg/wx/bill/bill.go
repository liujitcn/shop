package bill

import (
	"context"
	"fmt"
	"io"
	nethttp "net/http"
	neturl "net/url"

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
	var (
		localVarHTTPMethod   = nethttp.MethodGet
		localVarPostBody     interface{}
		localVarQueryParams  neturl.Values
		localVarHeaderParams = nethttp.Header{}
	)

	// 缺少账单日期时，微信账单接口无法正常调用。
	if req.BillDate == nil || len(*req.BillDate) == 0 {
		return nil, nil, fmt.Errorf("field `BillDate` is required and must be specified in TradeBillRequest")
	}
	// 缺少账单类型时，微信账单接口无法正常调用。
	if req.BillType == nil || len(*req.BillType) == 0 {
		return nil, nil, fmt.Errorf("field `BillType` is required and must be specified in TradeBillRequest")
	}

	localVarPath := consts.WechatPayAPIServer + "/v3/bill/tradebill"

	localVarQueryParams = neturl.Values{}
	localVarQueryParams.Add("bill_date", core.ParameterToString(*req.BillDate, ""))
	localVarQueryParams.Add("bill_type", core.ParameterToString(*req.BillType, ""))

	localVarHTTPContentType := core.SelectHeaderContentType([]string{})

	result, err = a.Client.Request(ctx, localVarHTTPMethod, localVarPath, localVarHeaderParams, localVarQueryParams, localVarPostBody, localVarHTTPContentType)
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
func (a *BillService) DownloadBill(ctx context.Context, url string) ([]byte, error) {
	var (
		localVarHTTPMethod   = nethttp.MethodGet
		localVarPostBody     interface{}
		localVarQueryParams  neturl.Values
		localVarHeaderParams = nethttp.Header{}
	)

	localVarHTTPContentType := core.SelectHeaderContentType([]string{})
	newClient := core.NewClientWithValidator(a.Client, &validators.NullValidator{})
	result, err := newClient.Request(ctx, localVarHTTPMethod, url, localVarHeaderParams, localVarQueryParams, localVarPostBody, localVarHTTPContentType)
	if err != nil {
		return nil, err
	}
	httpResp := result.Response

	body, err := io.ReadAll(httpResp.Body)
	defer func(body io.ReadCloser) {
		closeErr := body.Close()
		// 关闭响应体失败时，仅记录日志，不覆盖主流程错误。
		if closeErr != nil {
			log.Errorf("failed to close body: %v", closeErr)
		}
	}(httpResp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

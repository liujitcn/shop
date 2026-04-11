package biz

import (
	"context"
	"errors"
	"fmt"
	"shop/api/gen/go/app"
	"shop/pkg/configs"
	"shop/pkg/wx"
	"strconv"
	"strings"
	"time"

	"shop/api/gen/go/common"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport"
	kratosHttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/liujitcn/go-utils/ip"
	_string "github.com/liujitcn/go-utils/string"
	_time "github.com/liujitcn/go-utils/time"
	"github.com/liujitcn/go-utils/trans"
	"github.com/liujitcn/gorm-kit/repo"
	wxPayCore "github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/h5"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/jsapi"
	"google.golang.org/protobuf/encoding/protojson"
	"gorm.io/gorm"
)

type PayCase struct {
	*biz.BaseCase
	tx                 data.Transaction
	orderInfoRepo      *data.OrderInfoRepo
	orderGoodsRepo     *data.OrderGoodsRepo
	orderPaymentRepo   *data.OrderPaymentRepo
	orderRefundRepo    *data.OrderRefundRepo
	orderSchedulerCase *OrderSchedulerCase
	wxPayCase          *wx.WxPayCase
}

// NewPayCase 创建支付业务处理对象
func NewPayCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	orderInfoRepo *data.OrderInfoRepo,
	orderGoodsRepo *data.OrderGoodsRepo,
	orderPaymentRepo *data.OrderPaymentRepo,
	orderRefundRepo *data.OrderRefundRepo,
	orderSchedulerCase *OrderSchedulerCase,
	wxPayCase *wx.WxPayCase,
) *PayCase {
	return &PayCase{
		BaseCase:           baseCase,
		tx:                 tx,
		orderInfoRepo:      orderInfoRepo,
		orderGoodsRepo:     orderGoodsRepo,
		orderPaymentRepo:   orderPaymentRepo,
		orderRefundRepo:    orderRefundRepo,
		orderSchedulerCase: orderSchedulerCase,
		wxPayCase:          wxPayCase,
	}
}

// JsapiPay 创建 JSAPI 支付预下单信息
func (c *PayCase) JsapiPay(ctx context.Context, req *app.PayRequest) (*app.JsapiPayResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	var orderInfo *models.OrderInfo
	query := c.orderInfoRepo.Query(ctx).OrderInfo
	orderInfo, err = c.orderInfoRepo.Find(ctx,
		repo.Where(query.ID.Eq(req.GetOrderId())),
		repo.Where(query.UserID.Eq(authInfo.UserId)),
	)
	if err != nil {
		return nil, err
	}
	if orderInfo.Status != int32(common.OrderStatus_CREATED) {
		return nil, fmt.Errorf("订单状态错误：【%s】", common.OrderStatus_name[orderInfo.Status])
	}

	var goodsInfoList []*models.OrderGoods
	orderGoodsQuery := c.orderGoodsRepo.Query(ctx).OrderGoods
	opts := make([]repo.QueryOption, 0, 1)
	opts = append(opts, repo.Where(orderGoodsQuery.OrderID.Eq(orderInfo.ID)))
	goodsInfoList, err = c.orderGoodsRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	goodsDetail := make([]jsapi.GoodsDetail, 0)
	for _, item := range goodsInfoList {
		goodsDetail = append(goodsDetail, jsapi.GoodsDetail{
			MerchantGoodsId: new(fmt.Sprintf("%s_%s", strconv.FormatInt(item.GoodsID, 10), item.SkuCode)),
			GoodsName:       &item.Name,
			Quantity:        &item.Num,
			UnitPrice:       &item.Price,
		})
	}

	payTimeout := configs.ParsePayTimeout()
	var description = "小程序支付"
	if len(goodsDetail) > 0 {
		description = trans.StringValue(goodsDetail[0].GoodsName)
	}

	var jsapiPayResponse *app.JsapiPayResponse
	jsapiPayResponse, err = c.wxPayCase.JsapiPay(jsapi.PrepayRequest{
		Description: &description,
		OutTradeNo:  &orderInfo.OrderNo,
		TimeExpire:  new(orderInfo.CreatedAt.Add(payTimeout)),
		Amount: &jsapi.Amount{
			Total: &orderInfo.PayMoney,
		},
		Payer: &jsapi.Payer{
			Openid: &authInfo.OpenId,
		},
		Detail: &jsapi.Detail{
			GoodsDetail: goodsDetail,
		},
	})
	if err != nil {
		if apiErr, ok := errors.AsType[*wxPayCore.APIError](err); ok {
			// 订单已支付
			if apiErr.Code == "ORDERPAID" {
				// 调用查询订单接口
				var paymentResource *app.PaymentResource
				paymentResource, err = c.wxPayCase.QueryOrderByOutTradeNo(orderInfo.OrderNo)
				if err != nil {
					return nil, err
				}
				err = c.PaySuccess(ctx, orderInfo, paymentResource)
				if err != nil {
					return nil, err
				}
				return nil, errors.New("订单已支付，不能重复支付")
			}
		}
		return nil, err
	}
	return jsapiPayResponse, nil
}

// H5Pay 创建 H5 支付预下单信息
func (c *PayCase) H5Pay(ctx context.Context, req *app.PayRequest) (*app.H5PayResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	var orderInfo *models.OrderInfo
	query := c.orderInfoRepo.Query(ctx).OrderInfo
	orderInfo, err = c.orderInfoRepo.Find(ctx,
		repo.Where(query.ID.Eq(req.GetOrderId())),
		repo.Where(query.UserID.Eq(authInfo.UserId)),
	)
	if err != nil {
		return nil, err
	}
	if orderInfo.Status != int32(common.OrderStatus_CREATED) {
		return nil, fmt.Errorf("订单状态错误：【%s】", common.OrderStatus_name[orderInfo.Status])
	}

	var goodsInfoList []*models.OrderGoods
	orderGoodsQuery := c.orderGoodsRepo.Query(ctx).OrderGoods
	opts := make([]repo.QueryOption, 0, 1)
	opts = append(opts, repo.Where(orderGoodsQuery.OrderID.Eq(orderInfo.ID)))
	goodsInfoList, err = c.orderGoodsRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	goodsDetail := make([]h5.GoodsDetail, 0)
	for _, item := range goodsInfoList {
		goodsDetail = append(goodsDetail, h5.GoodsDetail{
			MerchantGoodsId: new(fmt.Sprintf("%s_%s", strconv.FormatInt(item.GoodsID, 10), item.SkuCode)),
			GoodsName:       &item.Name,
			Quantity:        &item.Num,
			UnitPrice:       &item.Price,
		})
	}
	payTimeout := configs.ParsePayTimeout()
	createdAt := orderInfo.CreatedAt.Add(payTimeout)

	var description = "H5支付"
	if len(goodsDetail) > 0 {
		description = trans.StringValue(goodsDetail[0].GoodsName)
	}
	// 微信 H5 支付要求必须上送发起支付的客户端 IP
	payerClientIp := c.getPayerClientIP(ctx)
	if payerClientIp == "" {
		return nil, errors.New("获取客户端IP失败")
	}

	var h5PayResponse *app.H5PayResponse
	h5PayResponse, err = c.wxPayCase.H5Pay(h5.PrepayRequest{
		Description: trans.String(description),
		OutTradeNo:  trans.String(orderInfo.OrderNo),
		TimeExpire:  trans.Time(createdAt),
		Amount: &h5.Amount{
			Total: &orderInfo.PayMoney,
		},
		SceneInfo: &h5.SceneInfo{
			PayerClientIp: trans.String(payerClientIp),
			DeviceId:      nil,
			StoreInfo:     nil,
			H5Info: &h5.H5Info{
				Type: trans.String("Wap"),
			},
		},
		Detail: &h5.Detail{
			GoodsDetail: goodsDetail,
		},
	})
	if err != nil {
		return nil, err
	}
	return h5PayResponse, nil
}

// PayNotify 处理支付通知
func (c *PayCase) PayNotify(ctx context.Context) error {
	request, err := c.wxPayCase.Notify(ctx)
	if err != nil {
		return err
	}
	resource := request.Resource
	if resource == nil {
		return errors.New("notify resource is nil")
	}

	log.Infof("PayNotify EventType=%s，Plaintext=%s", request.EventType, resource.Plaintext)
	// 判断通知类型
	if strings.HasPrefix(request.EventType, app.ResourceType_TRANSACTION.String()) {
		// 转换
		var paymentResource app.PaymentResource
		err = protojson.Unmarshal([]byte(resource.Plaintext), &paymentResource)
		if err != nil {
			return err
		}
		var orderInfo *models.OrderInfo
		orderInfo, err = c.findByOrderNo(ctx, paymentResource.GetOutTradeNo())
		if err != nil {
			return err
		}
		return c.PaySuccess(ctx, orderInfo, &paymentResource)
	} else if strings.HasPrefix(request.EventType, app.ResourceType_REFUND.String()) {
		// 转换
		var refundResource app.RefundResource
		err = protojson.Unmarshal([]byte(resource.Plaintext), &refundResource)
		if err != nil {
			return err
		}
		var orderInfo *models.OrderInfo
		orderInfo, err = c.findByOrderNo(ctx, refundResource.GetOutTradeNo())
		if err != nil {
			return err
		}
		return c.RefundSuccess(ctx, orderInfo, &refundResource)
	}

	return errors.New("notify event type err")
}

// getPayerClientIP 从当前 HTTP 请求中提取客户端真实 IP
func (c *PayCase) getPayerClientIP(ctx context.Context) string {
	serverTransport, ok := transport.FromServerContext(ctx)
	if !ok {
		return ""
	}

	httpTransport, ok := serverTransport.(*kratosHttp.Transport)
	if !ok {
		return ""
	}

	request := httpTransport.Request()
	if request == nil {
		return ""
	}
	return ip.GetClientRealIP(request)
}

// PaySuccess 支付成功处理
func (c *PayCase) PaySuccess(ctx context.Context, orderInfo *models.OrderInfo, paymentResource *app.PaymentResource) error {
	if orderInfo == nil {
		return errors.New("orderInfo is nil")
	}
	// 查询支付信息
	var orderPayment *models.OrderPayment
	orderPaymentQuery := c.orderPaymentRepo.Query(ctx).OrderPayment
	orderPayment, err := c.orderPaymentRepo.Find(ctx,
		repo.Where(orderPaymentQuery.OrderID.Eq(orderInfo.ID)),
	)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			orderPayment = &models.OrderPayment{}
		} else {
			return err
		}
	}
	successTime := _time.TimestamppbToTime(paymentResource.GetSuccessTime())
	if successTime == nil {
		successTime = trans.Time(time.Now())
	}
	orderPayment.OrderID = orderInfo.ID
	orderPayment.OrderNo = paymentResource.GetOutTradeNo()
	orderPayment.ThirdOrderNo = paymentResource.GetTransactionId()
	orderPayment.TradeType = paymentResource.GetTradeType().String()
	orderPayment.TradeState = paymentResource.GetTradeState().String()
	orderPayment.TradeStateDesc = paymentResource.GetTradeStateDesc()
	orderPayment.BankType = paymentResource.GetBankType()
	orderPayment.SuccessTime = trans.TimeValue(successTime)
	orderPayment.Payer = _string.ConvertAnyToJsonString(paymentResource.GetPayer())
	orderPayment.Amount = _string.ConvertAnyToJsonString(paymentResource.GetAmount())
	orderPayment.SceneInfo = _string.ConvertAnyToJsonString(paymentResource.GetSceneInfo())
	orderPayment.Status = 1

	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		// 添加支付信息
		if orderPayment.ID == 0 {
			err = c.orderPaymentRepo.Create(ctx, orderPayment)
			if err != nil {
				return err
			}
		} else {
			err = c.orderPaymentRepo.UpdateById(ctx, orderPayment)
			if err != nil {
				return err
			}
		}
		// 支付成功，修改订单状态
		if orderPayment.TradeState == app.PaymentResource_SUCCESS.String() {
			err = c.updateOrder(ctx, orderInfo.ID, orderInfo.UserID, common.OrderStatus_PAID)
			if err != nil {
				return err
			}
			// 删除自动取消
			c.orderSchedulerCase.DeleteScheduled(orderInfo.ID)
		}
		return nil
	})
}

// RefundSuccess 退款成功处理
func (c *PayCase) RefundSuccess(ctx context.Context, orderInfo *models.OrderInfo, refundResource *app.RefundResource) error {
	if orderInfo == nil {
		return errors.New("orderInfo is nil")
	}
	// 查询支付信息
	var orderRefund *models.OrderRefund
	orderRefundQuery := c.orderRefundRepo.Query(ctx).OrderRefund
	orderRefund, err := c.orderRefundRepo.Find(ctx,
		repo.Where(orderRefundQuery.OrderID.Eq(orderInfo.ID)),
	)
	successTime := _time.TimestamppbToTime(refundResource.GetSuccessTime())
	if successTime == nil {
		successTime = trans.Time(time.Now())
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			orderRefund = &models.OrderRefund{
				OrderID:    orderInfo.ID,
				RefundNo:   refundResource.GetOutRefundNo(),
				CreateTime: time.Now(),
			}
		} else {
			return err
		}
	}
	orderRefund.OrderNo = refundResource.GetOutTradeNo()
	orderRefund.ThirdOrderNo = refundResource.GetTransactionId()
	orderRefund.ThirdRefundNo = refundResource.GetRefundId()
	orderRefund.UserReceivedAccount = refundResource.GetUserReceivedAccount()
	orderRefund.SuccessTime = trans.TimeValue(successTime)
	orderRefund.RefundState = refundResource.GetRefundStatus().String()
	orderRefund.Amount = _string.ConvertAnyToJsonString(refundResource.GetAmount())
	orderRefund.Status = 1

	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		// 添加退款信息
		if orderRefund.ID == 0 {
			err = c.orderRefundRepo.Create(ctx, orderRefund)
			if err != nil {
				return err
			}
		} else {
			err = c.orderRefundRepo.UpdateById(ctx, orderRefund)
			if err != nil {
				return err
			}
		}
		// 支付成功，修改订单状态
		if orderRefund.RefundState == app.RefundResource_SUCCESS.String() {
			return c.updateOrder(ctx, orderInfo.ID, orderInfo.UserID, common.OrderStatus_REFUNDING)
		}
		return nil
	})
}

// findByOrderNo 根据订单号查询订单
func (c *PayCase) findByOrderNo(ctx context.Context, orderNo string) (*models.OrderInfo, error) {
	// 查询订单
	orderQuery := c.orderInfoRepo.Query(ctx).OrderInfo
	orderInfo, err := c.orderInfoRepo.Find(ctx,
		repo.Where(orderQuery.OrderNo.Eq(orderNo)),
	)
	if err != nil {
		return nil, err
	}
	return orderInfo, nil
}

// updateOrder 更新订单状态
func (c *PayCase) updateOrder(ctx context.Context, orderId, userId int64, status common.OrderStatus) error {
	orderQuery := c.orderInfoRepo.Query(ctx).OrderInfo
	return c.orderInfoRepo.Update(ctx, &models.OrderInfo{Status: int32(status)},
		repo.Where(orderQuery.UserID.Eq(userId)),
		repo.Where(orderQuery.ID.Eq(orderId)),
	)
}

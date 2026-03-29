package biz

import (
	"context"
	"errors"
	"fmt"
	"shop/api/gen/go/app"
	"shop/pkg/configs"
	"shop/service/app/wx"
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
	orderRepo          *data.OrderRepo
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
	orderCase *data.OrderRepo,
	orderGoodsRepo *data.OrderGoodsRepo,
	orderPaymentRepo *data.OrderPaymentRepo,
	orderRefundRepo *data.OrderRefundRepo,
	orderSchedulerCase *OrderSchedulerCase,
	wxPayCase *wx.WxPayCase,
) *PayCase {
	return &PayCase{
		BaseCase:           baseCase,
		tx:                 tx,
		orderRepo:          orderCase,
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

	var order *models.Order
	query := c.orderRepo.Query(ctx).Order
	order, err = c.orderRepo.Find(ctx,
		repo.Where(query.ID.Eq(req.GetOrderId())),
		repo.Where(query.UserID.Eq(authInfo.UserId)),
	)
	if err != nil {
		return nil, err
	}
	if order.Status != int32(common.OrderStatus_CREATED) {
		return nil, fmt.Errorf("订单状态错误：【%s】", common.OrderStatus_name[order.Status])
	}

	var goods []*models.OrderGoods
	orderGoodsQuery := c.orderGoodsRepo.Query(ctx).OrderGoods
	goods, err = c.orderGoodsRepo.List(ctx,
		repo.Where(orderGoodsQuery.OrderID.Eq(order.ID)),
	)
	if err != nil {
		return nil, err
	}

	goodsDetail := make([]jsapi.GoodsDetail, 0)
	for _, item := range goods {
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
		OutTradeNo:  &order.OrderNo,
		TimeExpire:  new(order.CreatedAt.Add(payTimeout)),
		Amount: &jsapi.Amount{
			Total: &order.PayMoney,
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
				err = c.Paid(ctx, order)
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

	var order *models.Order
	query := c.orderRepo.Query(ctx).Order
	order, err = c.orderRepo.Find(ctx,
		repo.Where(query.ID.Eq(req.GetOrderId())),
		repo.Where(query.UserID.Eq(authInfo.UserId)),
	)
	if err != nil {
		return nil, err
	}
	if order.Status != int32(common.OrderStatus_CREATED) {
		return nil, fmt.Errorf("订单状态错误：【%s】", common.OrderStatus_name[order.Status])
	}

	var goods []*models.OrderGoods
	orderGoodsQuery := c.orderGoodsRepo.Query(ctx).OrderGoods
	goods, err = c.orderGoodsRepo.List(ctx,
		repo.Where(orderGoodsQuery.OrderID.Eq(order.ID)),
	)
	if err != nil {
		return nil, err
	}

	goodsDetail := make([]h5.GoodsDetail, 0)
	for _, item := range goods {
		goodsDetail = append(goodsDetail, h5.GoodsDetail{
			MerchantGoodsId: new(fmt.Sprintf("%s_%s", strconv.FormatInt(item.GoodsID, 10), item.SkuCode)),
			GoodsName:       &item.Name,
			Quantity:        &item.Num,
			UnitPrice:       &item.Price,
		})
	}
	payTimeout := configs.ParsePayTimeout()
	createdAt := order.CreatedAt.Add(payTimeout)

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
		OutTradeNo:  trans.String(order.OrderNo),
		TimeExpire:  trans.Time(createdAt),
		Amount: &h5.Amount{
			Total: &order.PayMoney,
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

// Paid 订单已支付查询订单，然后支付通知
func (c *PayCase) Paid(ctx context.Context, order *models.Order) error {
	paymentResource, err := c.wxPayCase.QueryOrderByOutTradeNo(order.OrderNo)
	if err != nil {
		return err
	}
	return c.PaySuccess(ctx, order, paymentResource)
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
		var order *models.Order
		order, err = c.findByOrderNo(ctx, paymentResource.GetOutTradeNo())
		if err != nil {
			return err
		}
		return c.PaySuccess(ctx, order, &paymentResource)
	} else if strings.HasPrefix(request.EventType, app.ResourceType_REFUND.String()) {
		// 转换
		var refundResource app.RefundResource
		err = protojson.Unmarshal([]byte(resource.Plaintext), &refundResource)
		if err != nil {
			return err
		}
		var order *models.Order
		order, err = c.findByOrderNo(ctx, refundResource.GetOutTradeNo())
		if err != nil {
			return err
		}
		return c.RefundSuccess(ctx, order, &refundResource)
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
func (c *PayCase) PaySuccess(ctx context.Context, order *models.Order, paymentResource *app.PaymentResource) error {
	if order == nil {
		return errors.New("order is nil")
	}
	// 查询支付信息
	var orderPayment *models.OrderPayment
	orderPaymentQuery := c.orderPaymentRepo.Query(ctx).OrderPayment
	orderPayment, err := c.orderPaymentRepo.Find(ctx,
		repo.Where(orderPaymentQuery.OrderID.Eq(order.ID)),
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
	orderPayment.OrderID = order.ID
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
			err = c.updateOrder(ctx, order.ID, order.UserID, common.OrderStatus_PAID)
			if err != nil {
				return err
			}
			// 删除自动取消
			c.orderSchedulerCase.DeleteScheduled(order.ID)
		}
		return nil
	})
}

// RefundSuccess 退款成功处理
func (c *PayCase) RefundSuccess(ctx context.Context, order *models.Order, refundResource *app.RefundResource) error {
	if order == nil {
		return errors.New("order is nil")
	}
	// 查询支付信息
	var orderRefund *models.OrderRefund
	orderRefundQuery := c.orderRefundRepo.Query(ctx).OrderRefund
	orderRefund, err := c.orderRefundRepo.Find(ctx,
		repo.Where(orderRefundQuery.OrderID.Eq(order.ID)),
	)
	successTime := _time.TimestamppbToTime(refundResource.GetSuccessTime())
	if successTime == nil {
		successTime = trans.Time(time.Now())
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			orderRefund = &models.OrderRefund{
				OrderID:    order.ID,
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
			return c.updateOrder(ctx, order.ID, order.UserID, common.OrderStatus_REFUNDING)
		}
		return nil
	})
}

// findByOrderNo 根据订单号查询订单
func (c *PayCase) findByOrderNo(ctx context.Context, orderNo string) (*models.Order, error) {
	// 查询订单
	orderQuery := c.orderRepo.Query(ctx).Order
	order, err := c.orderRepo.Find(ctx,
		repo.Where(orderQuery.OrderNo.Eq(orderNo)),
	)
	if err != nil {
		return nil, err
	}
	return order, nil
}

// updateOrder 更新订单状态
func (c *PayCase) updateOrder(ctx context.Context, orderId, userId int64, status common.OrderStatus) error {
	orderQuery := c.orderRepo.Query(ctx).Order
	return c.orderRepo.Update(ctx, &models.Order{Status: int32(status)},
		repo.Where(orderQuery.UserID.Eq(userId)),
		repo.Where(orderQuery.ID.Eq(orderId)),
	)
}

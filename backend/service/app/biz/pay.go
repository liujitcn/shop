package biz

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	appv1 "shop/api/gen/go/app/v1"
	"shop/pkg/config"
	_const "shop/pkg/const"
	"shop/pkg/errorsx"
	"shop/pkg/queue"
	"shop/pkg/recommend/dto"
	"shop/pkg/wx"

	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport"
	kratosHTTP "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/liujitcn/go-utils/ip"
	_string "github.com/liujitcn/go-utils/string"
	_time "github.com/liujitcn/go-utils/time"
	"github.com/liujitcn/go-utils/trans"
	"github.com/liujitcn/gorm-kit/repository"
	wxPayCore "github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/h5"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/jsapi"
	"google.golang.org/protobuf/encoding/protojson"
	"gorm.io/gorm"
)

type PayCase struct {
	*biz.BaseCase
	tx                 data.Transaction
	orderInfoRepo      *data.OrderInfoRepository
	orderGoodsRepo     *data.OrderGoodsRepository
	orderPaymentRepo   *data.OrderPaymentRepository
	orderRefundRepo    *data.OrderRefundRepository
	orderSchedulerCase *OrderSchedulerCase
	wxPayCase          *wx.WxPayCase
}

// NewPayCase 创建支付业务处理对象
func NewPayCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	orderInfoRepo *data.OrderInfoRepository,
	orderGoodsRepo *data.OrderGoodsRepository,
	orderPaymentRepo *data.OrderPaymentRepository,
	orderRefundRepo *data.OrderRefundRepository,
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

// JSAPIPay 创建 JSAPI 支付预下单信息
func (c *PayCase) JSAPIPay(ctx context.Context, req *appv1.JsapiPayRequest) (*appv1.JsapiPayResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	var orderInfo *models.OrderInfo
	query := c.orderInfoRepo.Query(ctx).OrderInfo
	orderOpts := make([]repository.QueryOption, 0, 2)
	orderOpts = append(orderOpts, repository.Where(query.ID.Eq(req.GetOrderId())))
	orderOpts = append(orderOpts, repository.Where(query.UserID.Eq(authInfo.UserId)))
	orderInfo, err = c.orderInfoRepo.Find(ctx, orderOpts...)
	if err != nil {
		return nil, err
	}
	// 仅允许待支付订单进入预下单流程。
	if orderInfo.Status != _const.ORDER_STATUS_CREATED {
		return nil, errorsx.StateConflict(
			fmt.Sprintf("订单状态错误：【%s】", commonv1.OrderStatus_name[orderInfo.Status]),
			"order_info",
			commonv1.OrderStatus(orderInfo.Status).String(),
			commonv1.OrderStatus(_const.ORDER_STATUS_CREATED).String(),
		)
	}

	var orderGoodsList []*models.OrderGoods
	orderGoodsQuery := c.orderGoodsRepo.Query(ctx).OrderGoods
	goodsOpts := make([]repository.QueryOption, 0, 1)
	goodsOpts = append(goodsOpts, repository.Where(orderGoodsQuery.OrderID.Eq(orderInfo.ID)))
	orderGoodsList, err = c.orderGoodsRepo.List(ctx, goodsOpts...)
	if err != nil {
		return nil, err
	}

	goodsDetail := make([]jsapi.GoodsDetail, 0)
	for _, item := range orderGoodsList {
		goodsDetail = append(goodsDetail, jsapi.GoodsDetail{
			MerchantGoodsId: new(fmt.Sprintf("%s_%s", strconv.FormatInt(item.GoodsID, 10), item.SKUCode)),
			GoodsName:       &item.Name,
			Quantity:        &item.Num,
			UnitPrice:       &item.Price,
		})
	}

	payTimeout := config.ParsePayTimeout()
	var description = "小程序支付"
	// 订单存在商品明细时，优先使用首个商品名作为支付描述。
	if len(goodsDetail) > 0 {
		description = trans.StringValue(goodsDetail[0].GoodsName)
	}

	var jsapiPayResponse *appv1.JsapiPayResponse
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
	// 微信预下单失败时，优先识别是否为重复支付通知。
	if err != nil {
		// 命中微信 API 错误类型时，进一步识别可恢复的重复支付场景。
		if apiErr, ok := errors.AsType[*wxPayCore.APIError](err); ok {
			// 订单已支付
			if apiErr.Code == "ORDERPAID" {
				// 调用查询订单接口
				var paymentResource *appv1.PaymentResource
				paymentResource, err = c.wxPayCase.QueryOrderByOutTradeNo(orderInfo.OrderNo)
				if err != nil {
					return nil, err
				}
				err = c.PaySuccess(ctx, orderInfo, paymentResource)
				if err != nil {
					return nil, err
				}
				return nil, errorsx.StateConflict(
					"订单已支付，不能重复支付",
					"order_payment",
					appv1.PaymentResource_TradeState(_const.PAYMENT_RESOURCE_TRADE_STATE_SUCCESS).String(),
					commonv1.OrderStatus(_const.ORDER_STATUS_CREATED).String(),
				)
			}
		}
		return nil, err
	}
	return jsapiPayResponse, nil
}

// H5Pay 创建 H5 支付预下单信息
func (c *PayCase) H5Pay(ctx context.Context, req *appv1.H5PayRequest) (*appv1.H5PayResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	var orderInfo *models.OrderInfo
	query := c.orderInfoRepo.Query(ctx).OrderInfo
	orderOpts := make([]repository.QueryOption, 0, 2)
	orderOpts = append(orderOpts, repository.Where(query.ID.Eq(req.GetOrderId())))
	orderOpts = append(orderOpts, repository.Where(query.UserID.Eq(authInfo.UserId)))
	orderInfo, err = c.orderInfoRepo.Find(ctx, orderOpts...)
	if err != nil {
		return nil, err
	}
	// 仅允许待支付订单进入预下单流程。
	if orderInfo.Status != _const.ORDER_STATUS_CREATED {
		return nil, errorsx.StateConflict(
			fmt.Sprintf("订单状态错误：【%s】", commonv1.OrderStatus_name[orderInfo.Status]),
			"order_info",
			commonv1.OrderStatus(orderInfo.Status).String(),
			commonv1.OrderStatus(_const.ORDER_STATUS_CREATED).String(),
		)
	}

	var orderGoodsList []*models.OrderGoods
	orderGoodsQuery := c.orderGoodsRepo.Query(ctx).OrderGoods
	goodsOpts := make([]repository.QueryOption, 0, 1)
	goodsOpts = append(goodsOpts, repository.Where(orderGoodsQuery.OrderID.Eq(orderInfo.ID)))
	orderGoodsList, err = c.orderGoodsRepo.List(ctx, goodsOpts...)
	if err != nil {
		return nil, err
	}

	goodsDetail := make([]h5.GoodsDetail, 0)
	for _, item := range orderGoodsList {
		goodsDetail = append(goodsDetail, h5.GoodsDetail{
			MerchantGoodsId: new(fmt.Sprintf("%s_%s", strconv.FormatInt(item.GoodsID, 10), item.SKUCode)),
			GoodsName:       &item.Name,
			Quantity:        &item.Num,
			UnitPrice:       &item.Price,
		})
	}
	payTimeout := config.ParsePayTimeout()
	createdAt := orderInfo.CreatedAt.Add(payTimeout)

	var description = "H5支付"
	// 订单存在商品明细时，优先使用首个商品名作为支付描述。
	if len(goodsDetail) > 0 {
		description = trans.StringValue(goodsDetail[0].GoodsName)
	}
	// 微信 H5 支付要求必须上送发起支付的客户端 IP
	serverTransport, hasTransport := transport.FromServerContext(ctx)
	// 非服务端请求上下文时，无法提取客户端请求信息。
	if !hasTransport {
		return nil, errorsx.Internal("获取客户端IP失败")
	}
	httpTransport, isHTTPTransport := serverTransport.(*kratosHTTP.Transport)
	// 当前传输层不是 HTTP 时，不存在客户端 HTTP 请求对象。
	if !isHTTPTransport {
		return nil, errorsx.Internal("获取客户端IP失败")
	}
	request := httpTransport.Request()
	// 底层请求对象为空时，无法继续读取真实 IP。
	if request == nil {
		return nil, errorsx.Internal("获取客户端IP失败")
	}
	payerClientIP := ip.GetClientRealIP(request)
	// 无法识别客户端 IP 时，不满足微信 H5 下单要求。
	if payerClientIP == "" {
		return nil, errorsx.Internal("获取客户端IP失败")
	}

	var h5PayResponse *appv1.H5PayResponse
	h5PayResponse, err = c.wxPayCase.H5Pay(h5.PrepayRequest{
		Description: trans.String(description),
		OutTradeNo:  trans.String(orderInfo.OrderNo),
		TimeExpire:  trans.Time(createdAt),
		Amount: &h5.Amount{
			Total: &orderInfo.PayMoney,
		},
		SceneInfo: &h5.SceneInfo{
			PayerClientIp: trans.String(payerClientIP),
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
	// 回调缺少业务资源体时，无法继续处理通知。
	if resource == nil {
		return errorsx.Internal("支付通知缺少资源体")
	}

	log.Infof("PayNotify EventType=%s，Plaintext=%s", request.EventType, resource.Plaintext)
	// 判断通知类型
	if strings.HasPrefix(request.EventType, commonv1.ResourceType(_const.RESOURCE_TYPE_TRANSACTION).String()) {
		// 转换
		var paymentResource appv1.PaymentResource
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
	} else if strings.HasPrefix(request.EventType, commonv1.ResourceType(_const.RESOURCE_TYPE_REFUND).String()) {
		// 转换
		var refundResource appv1.RefundResource
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

	return errorsx.Internal("支付通知事件类型错误")
}

// PaySuccess 支付成功处理
func (c *PayCase) PaySuccess(ctx context.Context, orderInfo *models.OrderInfo, paymentResource *appv1.PaymentResource) error {
	// 未找到本地订单时，无法回写支付成功状态。
	if orderInfo == nil {
		return errorsx.Internal("支付成功处理失败，订单不存在")
	}
	// 查询支付信息
	var orderPayment *models.OrderPayment
	orderPaymentQuery := c.orderPaymentRepo.Query(ctx).OrderPayment
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(orderPaymentQuery.OrderID.Eq(orderInfo.ID)))
	orderPayment, err := c.orderPaymentRepo.Find(ctx, opts...)
	// 支付记录查询失败时，仅对“记录不存在”做初始化回退。
	if err != nil {
		// 支付记录尚未创建时，初始化空实体供后续写入。
		if errors.Is(err, gorm.ErrRecordNotFound) {
			orderPayment = &models.OrderPayment{}
		} else {
			return err
		}
	}
	successTime := _time.TimestamppbToTime(paymentResource.GetSuccessTime())
	// 微信回调未携带成功时间时，回退到当前时间写入本地记录。
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

	var orderGoodsList []*models.OrderGoods
	var shouldReportOrderPay bool
	err = c.tx.Transaction(ctx, func(ctx context.Context) error {
		// 添加支付信息
		if orderPayment.ID == 0 {
			err = c.orderPaymentRepo.Create(ctx, orderPayment)
			if err != nil {
				return err
			}
		} else {
			err = c.orderPaymentRepo.UpdateByID(ctx, orderPayment)
			if err != nil {
				return err
			}
		}
		// 支付成功，修改订单状态
		if orderPayment.TradeState == appv1.PaymentResource_TradeState(_const.PAYMENT_RESOURCE_TRADE_STATE_SUCCESS).String() {
			// 只有首次从待支付进入已支付，才视为本次通知真正完成支付落账。
			shouldReportOrderPay, err = c.markOrderPaid(ctx, orderInfo.ID, orderInfo.UserID)
			if err != nil {
				return err
			}
			// 首次支付成功时，读取订单商品快照，确保推荐支付行为完全基于后端事实构建。
			if shouldReportOrderPay {
				orderGoodsQuery := c.orderGoodsRepo.Query(ctx).OrderGoods
				orderGoodsOpts := make([]repository.QueryOption, 0, 1)
				orderGoodsOpts = append(orderGoodsOpts, repository.Where(orderGoodsQuery.OrderID.Eq(orderInfo.ID)))
				orderGoodsList, err = c.orderGoodsRepo.List(ctx, orderGoodsOpts...)
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	// 支付成功后无论是否重复通知，都尝试清理超时取消任务，避免历史定时任务继续生效。
	if orderPayment.TradeState == appv1.PaymentResource_TradeState(_const.PAYMENT_RESOURCE_TRADE_STATE_SUCCESS).String() {
		c.orderSchedulerCase.DeleteScheduled(orderInfo.ID)
	}
	// 只有首次支付成功才回写 ORDER_PAY，避免重复通知产生重复推荐事实。
	if shouldReportOrderPay {
		c.dispatchRecommendPayEvent(orderInfo.UserID, orderGoodsList, trans.TimeValue(successTime))
	}
	return nil
}

// RefundSuccess 退款成功处理
func (c *PayCase) RefundSuccess(ctx context.Context, orderInfo *models.OrderInfo, refundResource *appv1.RefundResource) error {
	// 未找到本地订单时，无法回写退款成功状态。
	if orderInfo == nil {
		return errorsx.Internal("退款成功处理失败，订单不存在")
	}
	// 查询支付信息
	var orderRefund *models.OrderRefund
	query := c.orderRefundRepo.Query(ctx).OrderRefund
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.OrderID.Eq(orderInfo.ID)))
	orderRefund, err := c.orderRefundRepo.Find(ctx, opts...)
	successTime := _time.TimestamppbToTime(refundResource.GetSuccessTime())
	// 微信回调未携带成功时间时，回退到当前时间写入本地记录。
	if successTime == nil {
		successTime = trans.Time(time.Now())
	}
	// 退款记录查询失败时，仅对“记录不存在”做初始化回退。
	if err != nil {
		// 退款记录尚未创建时，初始化空实体供后续写入。
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
			err = c.orderRefundRepo.UpdateByID(ctx, orderRefund)
			if err != nil {
				return err
			}
		}
		// 退款成功时，同步把订单标记为售后/已退款状态。
		if orderRefund.RefundState == appv1.RefundResource_RefundStatus(_const.REFUND_RESOURCE_STATUS_SUCCESS).String() {
			orderQuery := c.orderInfoRepo.Query(ctx).OrderInfo
			orderOpts := make([]repository.QueryOption, 0, 2)
			orderOpts = append(orderOpts, repository.Where(orderQuery.UserID.Eq(orderInfo.UserID)))
			orderOpts = append(orderOpts, repository.Where(orderQuery.ID.Eq(orderInfo.ID)))
			return c.orderInfoRepo.Update(ctx, &models.OrderInfo{Status: _const.ORDER_STATUS_REFUNDING}, orderOpts...)
		}
		return nil
	})
}

// findByOrderNo 根据订单号查询订单
func (c *PayCase) findByOrderNo(ctx context.Context, orderNo string) (*models.OrderInfo, error) {
	// 查询订单
	query := c.orderInfoRepo.Query(ctx).OrderInfo
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.OrderNo.Eq(orderNo)))
	orderInfo, err := c.orderInfoRepo.Find(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return orderInfo, nil
}

// markOrderPaid 将待支付订单推进到已支付，并返回是否为首次支付成功。
func (c *PayCase) markOrderPaid(ctx context.Context, orderID, userID int64) (bool, error) {
	query := c.orderInfoRepo.Query(ctx).OrderInfo
	res, err := query.WithContext(ctx).
		Where(
			query.UserID.Eq(userID),
			query.ID.Eq(orderID),
			query.Status.Eq(_const.ORDER_STATUS_CREATED),
		).
		Updates(map[string]interface{}{
			"status": _const.ORDER_STATUS_PAID,
		})
	if err != nil {
		return false, err
	}
	// 已经进入支付成功口径的订单不再重复回写 ORDER_PAY。
	if res.RowsAffected == 0 {
		return false, nil
	}
	return true, res.Error
}

// dispatchRecommendPayEvent 根据已支付订单商品快照回写推荐支付事件。
func (c *PayCase) dispatchRecommendPayEvent(userID int64, goodsList []*models.OrderGoods, eventTime time.Time) {
	// 主体编号非法或订单商品为空时，无法构建可归因的推荐支付事件。
	if userID <= 0 || len(goodsList) == 0 {
		return
	}

	for _, item := range goodsList {
		// 非法商品项直接跳过，避免把脏数据写入推荐链路。
		if item == nil || item.GoodsID <= 0 {
			continue
		}
		payEventReport := &appv1.RecommendEventReportRequest{
			EventType: commonv1.RecommendEventType(_const.RECOMMEND_EVENT_TYPE_ORDER_PAY),
			RecommendContext: &appv1.RecommendEventContext{
				Scene:     commonv1.RecommendScene(item.Scene),
				RequestId: item.RequestID,
			},
			Items: []*appv1.RecommendEventItem{
				{
					GoodsId:  item.GoodsID,
					GoodsNum: item.Num,
					Position: item.Position,
				},
			},
		}

		// 支付事件只在订单真实支付成功后回写，确保推荐链路与后端事实一致。
		queue.DispatchRecommendEvent(&dto.RecommendActor{
			ActorType: commonv1.RecommendActorType(_const.RECOMMEND_ACTOR_TYPE_USER),
			ActorID:   userID,
		}, payEventReport, eventTime)
	}
}

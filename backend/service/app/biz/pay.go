package biz

import (
	"context"
	"encoding/json"
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
	"shop/pkg/workspaceevent"
	"shop/pkg/wx"

	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/go-kratos/kratos/v3/transport"
	kratosHTTP "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/liujitcn/go-utils/ip"
	_string "github.com/liujitcn/go-utils/string"
	_time "github.com/liujitcn/go-utils/time"
	"github.com/liujitcn/go-utils/trans"
	"github.com/liujitcn/gorm-kit/repository"
	kitOauth "github.com/liujitcn/kratos-kit/oauth"
	wxPayCore "github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/h5"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/jsapi"
	"google.golang.org/protobuf/encoding/protojson"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PayCase 处理商城端支付业务。
type PayCase struct {
	*biz.BaseCase
	tx                   data.Transaction
	baseThirdAccountRepo *data.BaseThirdAccountRepository
	orderTradeRepo       *data.OrderTradeRepository
	orderInfoRepo        *data.OrderInfoRepository
	orderGoodsRepo       *data.OrderGoodsRepository
	orderPaymentRepo     *data.OrderPaymentRepository
	orderRefundRepo      *data.OrderRefundRepository
	orderSchedulerCase   *OrderSchedulerCase
	wxPayCase            *wx.WxPayCase
}

// NewPayCase 创建支付业务处理对象
func NewPayCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	baseThirdAccountRepo *data.BaseThirdAccountRepository,
	orderTradeRepo *data.OrderTradeRepository,
	orderInfoRepo *data.OrderInfoRepository,
	orderGoodsRepo *data.OrderGoodsRepository,
	orderPaymentRepo *data.OrderPaymentRepository,
	orderRefundRepo *data.OrderRefundRepository,
	orderSchedulerCase *OrderSchedulerCase,
	wxPayCase *wx.WxPayCase,
) *PayCase {
	return &PayCase{
		BaseCase:             baseCase,
		tx:                   tx,
		baseThirdAccountRepo: baseThirdAccountRepo,
		orderTradeRepo:       orderTradeRepo,
		orderInfoRepo:        orderInfoRepo,
		orderGoodsRepo:       orderGoodsRepo,
		orderPaymentRepo:     orderPaymentRepo,
		orderRefundRepo:      orderRefundRepo,
		orderSchedulerCase:   orderSchedulerCase,
		wxPayCase:            wxPayCase,
	}
}

// JSAPIPay 创建 JSAPI 支付预下单信息
func (c *PayCase) JSAPIPay(ctx context.Context, req *appv1.JsapiPayRequest) (*appv1.JsapiPayResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	var orderTrade *models.OrderTrade
	query := c.orderTradeRepo.Query(ctx).OrderTrade
	tradeOpts := make([]repository.QueryOption, 0, 2)
	tradeOpts = append(tradeOpts, repository.Where(query.ID.Eq(req.GetTradeId())))
	tradeOpts = append(tradeOpts, repository.Where(query.UserID.Eq(authInfo.UserId)))
	orderTrade, err = c.orderTradeRepo.Find(ctx, tradeOpts...)
	if err != nil {
		return nil, err
	}
	// 待支付和支付中的交易允许重复获取预支付参数。
	if orderTrade.Status != _const.ORDER_TRADE_STATUS_PENDING_PAYMENT && orderTrade.Status != _const.ORDER_TRADE_STATUS_PAYING {
		return nil, errorsx.StateConflict(
			fmt.Sprintf("交易状态错误：【%s】", commonv1.OrderTradeStatus_name[orderTrade.Status]),
			"order_trade",
			commonv1.OrderTradeStatus(orderTrade.Status).String(),
			"PENDING_PAYMENT_OTS|PAYING_OTS",
		)
	}

	var orderGoodsList []*models.OrderGoods
	orderGoodsList, err = c.listGoodsByTradeID(ctx, orderTrade.ID)
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

	var description = "小程序支付"
	// 订单存在商品明细时，优先使用首个商品名作为支付描述。
	if len(goodsDetail) > 0 {
		description = trans.StringValue(goodsDetail[0].GoodsName)
	}

	var openID string
	openID, err = c.findWechatMiniOpenID(ctx, authInfo.UserId)
	if err != nil {
		return nil, err
	}
	var jsapiPayResponse *appv1.JsapiPayResponse
	jsapiPayResponse, err = c.wxPayCase.JsapiPay(jsapi.PrepayRequest{
		Description: &description,
		OutTradeNo:  &orderTrade.TradeNo,
		TimeExpire:  trans.Time(orderTrade.CreatedAt.Add(config.ParsePayTimeout())),
		Amount: &jsapi.Amount{
			Total: &orderTrade.PayMoney,
		},
		Payer: &jsapi.Payer{
			Openid: &openID,
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
				paymentResource, err = c.wxPayCase.QueryOrderByOutTradeNo(orderTrade.TradeNo)
				if err != nil {
					return nil, err
				}
				err = c.PaySuccess(ctx, orderTrade, paymentResource)
				if err != nil {
					return nil, err
				}
				return nil, errorsx.StateConflict(
					"订单已支付，不能重复支付",
					"order_payment",
					appv1.PaymentResource_TradeState(_const.PAYMENT_RESOURCE_TRADE_STATE_SUCCESS).String(),
					commonv1.OrderTradeStatus(_const.ORDER_TRADE_STATUS_PENDING_PAYMENT).String(),
				)
			}
		}
		return nil, err
	}
	// 预支付单创建成功后，通过条件更新将交易推进到支付中。
	err = c.markTradePaying(ctx, orderTrade.ID, orderTrade.UserID)
	if err != nil {
		return nil, err
	}
	return jsapiPayResponse, nil
}

// findWechatMiniOpenID 查询当前用户绑定的微信小程序 OpenID。
func (c *PayCase) findWechatMiniOpenID(ctx context.Context, userID int64) (string, error) {
	query := c.baseThirdAccountRepo.Query(ctx).BaseThirdAccount
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.UserID.Eq(userID)))
	opts = append(opts, repository.Where(query.Provider.Eq(string(kitOauth.WechatMini))))
	account, err := c.baseThirdAccountRepo.Find(ctx, opts...)
	if err != nil {
		// 用户未绑定微信小程序时，无法创建 JSAPI 支付预下单。
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", errorsx.PermissionDenied("用户未绑定微信小程序")
		}
		return "", errorsx.Internal("小程序支付失败").WithCause(err)
	}
	if account.Identifier == "" {
		return "", errorsx.Internal("小程序支付失败")
	}
	return account.Identifier, nil
}

// H5Pay 创建 H5 支付预下单信息
func (c *PayCase) H5Pay(ctx context.Context, req *appv1.H5PayRequest) (*appv1.H5PayResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	var orderTrade *models.OrderTrade
	query := c.orderTradeRepo.Query(ctx).OrderTrade
	tradeOpts := make([]repository.QueryOption, 0, 2)
	tradeOpts = append(tradeOpts, repository.Where(query.ID.Eq(req.GetTradeId())))
	tradeOpts = append(tradeOpts, repository.Where(query.UserID.Eq(authInfo.UserId)))
	orderTrade, err = c.orderTradeRepo.Find(ctx, tradeOpts...)
	if err != nil {
		return nil, err
	}
	// 待支付和支付中的交易允许重复获取预支付参数。
	if orderTrade.Status != _const.ORDER_TRADE_STATUS_PENDING_PAYMENT && orderTrade.Status != _const.ORDER_TRADE_STATUS_PAYING {
		return nil, errorsx.StateConflict(
			fmt.Sprintf("交易状态错误：【%s】", commonv1.OrderTradeStatus_name[orderTrade.Status]),
			"order_trade",
			commonv1.OrderTradeStatus(orderTrade.Status).String(),
			"PENDING_PAYMENT_OTS|PAYING_OTS",
		)
	}

	var orderGoodsList []*models.OrderGoods
	orderGoodsList, err = c.listGoodsByTradeID(ctx, orderTrade.ID)
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
	createdAt := orderTrade.CreatedAt.Add(payTimeout)

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
		OutTradeNo:  trans.String(orderTrade.TradeNo),
		TimeExpire:  trans.Time(createdAt),
		Amount: &h5.Amount{
			Total: &orderTrade.PayMoney,
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
	// 微信侧已经支付但本地尚未收到通知时，主动查询并补齐本地交易状态。
	if err != nil {
		if apiErr, ok := errors.AsType[*wxPayCore.APIError](err); ok && apiErr.Code == "ORDERPAID" {
			var paymentResource *appv1.PaymentResource
			paymentResource, err = c.wxPayCase.QueryOrderByOutTradeNo(orderTrade.TradeNo)
			if err != nil {
				return nil, err
			}
			err = c.PaySuccess(ctx, orderTrade, paymentResource)
			if err != nil {
				return nil, err
			}
			return nil, errorsx.StateConflict(
				"订单已支付，不能重复支付",
				"order_payment",
				appv1.PaymentResource_SUCCESS.String(),
				commonv1.OrderTradeStatus(_const.ORDER_TRADE_STATUS_PENDING_PAYMENT).String(),
			)
		}
		return nil, err
	}
	// 预支付单创建成功后，通过条件更新将交易推进到支付中。
	err = c.markTradePaying(ctx, orderTrade.ID, orderTrade.UserID)
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
		return errorsx.InvalidArgument("支付通知缺少资源体")
	}

	log.Info(fmt.Sprintf("PayNotify EventType=%s，Plaintext=%s", request.EventType, resource.Plaintext))
	// 判断通知类型
	if strings.HasPrefix(request.EventType, commonv1.ResourceType(_const.RESOURCE_TYPE_TRANSACTION).String()) {
		// 转换
		var paymentResource appv1.PaymentResource
		err = protojson.Unmarshal([]byte(resource.Plaintext), &paymentResource)
		if err != nil {
			return err
		}
		var orderTrade *models.OrderTrade
		orderTrade, err = c.findTradeByTradeNo(ctx, paymentResource.GetOutTradeNo())
		if err != nil {
			return err
		}
		return c.PaySuccess(ctx, orderTrade, &paymentResource)
	} else if strings.HasPrefix(request.EventType, commonv1.ResourceType(_const.RESOURCE_TYPE_REFUND).String()) {
		// 转换
		var refundResource appv1.RefundResource
		err = protojson.Unmarshal([]byte(resource.Plaintext), &refundResource)
		if err != nil {
			return err
		}
		var orderTrade *models.OrderTrade
		orderTrade, err = c.findTradeByTradeNo(ctx, refundResource.GetOutTradeNo())
		if err != nil {
			return err
		}
		return c.RefundSuccess(ctx, orderTrade, &refundResource)
	}

	return errorsx.Internal("支付通知事件类型错误")
}

// PaySuccess 处理交易支付成功。
func (c *PayCase) PaySuccess(ctx context.Context, orderTrade *models.OrderTrade, paymentResource *appv1.PaymentResource) error {
	// 未找到本地交易时，无法回写支付成功状态。
	if orderTrade == nil {
		return errorsx.Internal("支付成功处理失败，交易不存在")
	}
	// 查询支付信息
	var orderPayment *models.OrderPayment
	orderPaymentQuery := c.orderPaymentRepo.Query(ctx).OrderPayment
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(orderPaymentQuery.TradeID.Eq(orderTrade.ID)))
	var err error
	orderPayment, err = c.orderPaymentRepo.Find(ctx, opts...)
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
	orderPayment.TradeID = orderTrade.ID
	orderPayment.TradeNo = paymentResource.GetOutTradeNo()
	orderPayment.ThirdOrderNo = paymentResource.GetTransactionId()
	orderPayment.TradeType = paymentResource.GetTradeType().String()
	orderPayment.TradeState = paymentResource.GetTradeState().String()
	orderPayment.TradeStateDesc = paymentResource.GetTradeStateDesc()
	orderPayment.BankType = paymentResource.GetBankType()
	orderPayment.SuccessTime = trans.TimeValue(successTime)
	orderPayment.Payer = _string.ConvertAnyToJsonString(paymentResource.GetPayer())
	orderPayment.Amount = _string.ConvertAnyToJsonString(paymentResource.GetAmount())
	orderPayment.SceneInfo = _string.ConvertAnyToJsonString(paymentResource.GetSceneInfo())
	orderPayment.Status = _const.ORDER_BILL_STATUS_NO_CHECK

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
			shouldReportOrderPay, err = c.markTradePaid(ctx, orderTrade.ID, orderTrade.UserID)
			if err != nil {
				return err
			}
			// 首次支付成功时，读取订单商品快照，确保推荐支付行为完全基于后端事实构建。
			if shouldReportOrderPay {
				orderGoodsList, err = c.listGoodsByTradeID(ctx, orderTrade.ID)
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
		c.orderSchedulerCase.DeleteScheduled(orderTrade.ID)
	}
	// 只有首次支付成功才回写 ORDER_PAY，避免重复通知产生重复推荐事实。
	if shouldReportOrderPay {
		c.dispatchRecommendPayEvent(orderTrade.UserID, orderGoodsList, trans.TimeValue(successTime))
		workspaceevent.Publish(ctx, workspaceevent.ReasonOrderChanged, workspaceevent.AreaMetrics, workspaceevent.AreaTodo, workspaceevent.AreaRisk)
	}
	return nil
}

// RefundSuccess 处理门店订单退款结果。
func (c *PayCase) RefundSuccess(ctx context.Context, orderTrade *models.OrderTrade, refundResource *appv1.RefundResource) error {
	// 未找到本地交易时，无法回写退款结果。
	if orderTrade == nil {
		return errorsx.Internal("退款结果处理失败，交易不存在")
	}
	var orderRefund *models.OrderRefund
	query := c.orderRefundRepo.Query(ctx).OrderRefund
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.TradeID.Eq(orderTrade.ID)))
	opts = append(opts, repository.Where(query.RefundNo.Eq(refundResource.GetOutRefundNo())))
	var err error
	orderRefund, err = c.orderRefundRepo.Find(ctx, opts...)
	if err != nil {
		return err
	}
	orderQuery := c.orderInfoRepo.Query(ctx).OrderInfo
	orderOpts := make([]repository.QueryOption, 0, 2)
	orderOpts = append(orderOpts, repository.Where(orderQuery.ID.Eq(orderRefund.OrderID)))
	orderOpts = append(orderOpts, repository.Where(orderQuery.TradeID.Eq(orderTrade.ID)))
	var orderInfo *models.OrderInfo
	orderInfo, err = c.orderInfoRepo.Find(ctx, orderOpts...)
	if err != nil {
		return err
	}
	successTime := _time.TimestamppbToTime(refundResource.GetSuccessTime())
	if successTime == nil {
		successTime = trans.Time(time.Now())
	}
	orderRefund.TradeNo = refundResource.GetOutTradeNo()
	orderRefund.ThirdOrderNo = refundResource.GetTransactionId()
	orderRefund.ThirdRefundNo = refundResource.GetRefundId()
	orderRefund.UserReceivedAccount = refundResource.GetUserReceivedAccount()
	orderRefund.SuccessTime = trans.TimeValue(successTime)
	orderRefund.RefundState = refundResource.GetRefundStatus().String()
	orderRefund.Amount = _string.ConvertAnyToJsonString(refundResource.GetAmount())
	orderRefund.Status = _const.ORDER_BILL_STATUS_NO_CHECK

	err = c.tx.Transaction(ctx, func(ctx context.Context) error {
		err = c.orderRefundRepo.UpdateByID(ctx, orderRefund)
		if err != nil {
			return err
		}
		// 按渠道最终结果更新当前门店订单退款状态。
		switch refundResource.GetRefundStatus() {
		case appv1.RefundResource_SUCCESS:
			return c.refreshRefundStatuses(ctx, orderTrade, orderInfo)
		case appv1.RefundResource_PROCESSING:
			return c.orderInfoRepo.UpdateByID(ctx, &models.OrderInfo{
				ID:           orderInfo.ID,
				RefundStatus: _const.ORDER_REFUND_STATUS_PROCESSING,
			})
		case appv1.RefundResource_CLOSED, appv1.RefundResource_ABNORMAL:
			return c.orderInfoRepo.UpdateByID(ctx, &models.OrderInfo{
				ID:           orderInfo.ID,
				RefundStatus: _const.ORDER_REFUND_STATUS_CLOSED_OR_FAILED,
			})
		}
		return nil
	})
	if err != nil {
		return err
	}
	workspaceevent.Publish(ctx, workspaceevent.ReasonOrderChanged, workspaceevent.AreaMetrics, workspaceevent.AreaTodo)
	return nil
}

// refreshRefundStatuses 汇总成功退款金额并刷新门店订单和交易状态。
func (c *PayCase) refreshRefundStatuses(ctx context.Context, orderTrade *models.OrderTrade, orderInfo *models.OrderInfo) error {
	tradeQuery := c.orderTradeRepo.Query(ctx).OrderTrade
	lockedTrade, err := tradeQuery.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where(tradeQuery.ID.Eq(orderTrade.ID)).
		First()
	if err != nil {
		return err
	}
	query := c.orderRefundRepo.Query(ctx).OrderRefund
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.TradeID.Eq(orderTrade.ID)))
	refunds, err := c.orderRefundRepo.List(ctx, opts...)
	if err != nil {
		return err
	}
	var tradeRefundMoney int64
	var orderRefundMoney int64
	for _, refund := range refunds {
		if refund.RefundState != appv1.RefundResource_SUCCESS.String() {
			continue
		}
		var amount struct {
			Refund int64 `json:"refund"`
		}
		err = json.Unmarshal([]byte(refund.Amount), &amount)
		if err != nil {
			return err
		}
		tradeRefundMoney += amount.Refund
		if refund.OrderID == orderInfo.ID {
			orderRefundMoney += amount.Refund
		}
	}
	orderRefundStatus := _const.ORDER_REFUND_STATUS_PARTIAL_REFUND
	if orderRefundMoney >= orderInfo.PayMoney {
		orderRefundStatus = _const.ORDER_REFUND_STATUS_REFUNDED
	}
	err = c.orderInfoRepo.UpdateByID(ctx, &models.OrderInfo{
		ID:           orderInfo.ID,
		RefundStatus: orderRefundStatus,
	})
	if err != nil {
		return err
	}
	tradeStatus := _const.ORDER_TRADE_STATUS_PARTIAL_REFUND
	if tradeRefundMoney >= lockedTrade.PayMoney {
		tradeStatus = _const.ORDER_TRADE_STATUS_FULL_REFUND
	}
	return c.orderTradeRepo.UpdateByID(ctx, &models.OrderTrade{
		ID:     orderTrade.ID,
		Status: tradeStatus,
	})
}

// findTradeByTradeNo 根据交易单编号查询交易单。
func (c *PayCase) findTradeByTradeNo(ctx context.Context, tradeNo string) (*models.OrderTrade, error) {
	query := c.orderTradeRepo.Query(ctx).OrderTrade
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.TradeNo.Eq(tradeNo)))
	orderTrade, err := c.orderTradeRepo.Find(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return orderTrade, nil
}

// markTradePaying 将待支付交易推进到支付中，避免并发取消后被旧请求覆盖。
func (c *PayCase) markTradePaying(ctx context.Context, tradeID, userID int64) error {
	query := c.orderTradeRepo.Query(ctx).OrderTrade
	res, err := query.WithContext(ctx).
		Where(
			query.UserID.Eq(userID),
			query.ID.Eq(tradeID),
			query.Status.Eq(_const.ORDER_TRADE_STATUS_PENDING_PAYMENT),
		).
		Update(query.Status, _const.ORDER_TRADE_STATUS_PAYING)
	if err != nil {
		return err
	}
	if res.RowsAffected > 0 {
		return nil
	}
	var orderTrade *models.OrderTrade
	orderTrade, err = c.findTradeByUserIDAndID(ctx, userID, tradeID)
	if err != nil {
		return err
	}
	// 并发预支付已经推进到支付中，当前请求可以复用已创建的支付单。
	if orderTrade.Status == _const.ORDER_TRADE_STATUS_PAYING {
		return nil
	}
	return errorsx.StateConflict(
		fmt.Sprintf("交易状态错误：【%s】", commonv1.OrderTradeStatus_name[orderTrade.Status]),
		"order_trade",
		commonv1.OrderTradeStatus(orderTrade.Status).String(),
		commonv1.OrderTradeStatus(_const.ORDER_TRADE_STATUS_PENDING_PAYMENT).String(),
	)
}

// markTradePaid 将待支付交易推进到已支付，并返回是否为首次支付成功。
func (c *PayCase) markTradePaid(ctx context.Context, tradeID, userID int64) (bool, error) {
	query := c.orderTradeRepo.Query(ctx).OrderTrade
	res, err := query.WithContext(ctx).
		Where(
			query.UserID.Eq(userID),
			query.ID.Eq(tradeID),
			query.Status.In(_const.ORDER_TRADE_STATUS_PENDING_PAYMENT, _const.ORDER_TRADE_STATUS_PAYING),
		).
		Update(query.Status, _const.ORDER_TRADE_STATUS_PAID)
	if err != nil {
		return false, err
	}
	// 已经进入支付成功口径的交易不再重复回写 ORDER_PAY。
	if res.RowsAffected == 0 {
		var orderTrade *models.OrderTrade
		orderTrade, err = c.findTradeByUserIDAndID(ctx, userID, tradeID)
		if err != nil {
			return false, err
		}
		// 重复支付通知只更新支付记录，不重复推进履约或上报推荐事件。
		if orderTrade.Status == _const.ORDER_TRADE_STATUS_PAID ||
			orderTrade.Status == _const.ORDER_TRADE_STATUS_PARTIAL_REFUND ||
			orderTrade.Status == _const.ORDER_TRADE_STATUS_FULL_REFUND {
			return false, nil
		}
		return false, errorsx.StateConflict(
			fmt.Sprintf("交易状态错误：【%s】", commonv1.OrderTradeStatus_name[orderTrade.Status]),
			"order_trade",
			commonv1.OrderTradeStatus(orderTrade.Status).String(),
			"PENDING_PAYMENT_OTS|PAYING_OTS",
		)
	}
	orderQuery := c.orderInfoRepo.Query(ctx).OrderInfo
	orderOpts := make([]repository.QueryOption, 0, 2)
	orderOpts = append(orderOpts, repository.Where(orderQuery.TradeID.Eq(tradeID)))
	orderOpts = append(orderOpts, repository.Where(orderQuery.Status.Eq(_const.ORDER_INFO_STATUS_NOT_STARTED)))
	err = c.orderInfoRepo.Update(ctx, &models.OrderInfo{Status: _const.ORDER_INFO_STATUS_WAIT_SHIPMENT}, orderOpts...)
	if err != nil {
		return false, err
	}
	return true, res.Error
}

// findTradeByUserIDAndID 查询指定用户的交易单。
func (c *PayCase) findTradeByUserIDAndID(ctx context.Context, userID, tradeID int64) (*models.OrderTrade, error) {
	query := c.orderTradeRepo.Query(ctx).OrderTrade
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.UserID.Eq(userID)))
	opts = append(opts, repository.Where(query.ID.Eq(tradeID)))
	return c.orderTradeRepo.Find(ctx, opts...)
}

// listGoodsByTradeID 查询交易单下全部门店订单商品。
func (c *PayCase) listGoodsByTradeID(ctx context.Context, tradeID int64) ([]*models.OrderGoods, error) {
	query := c.orderInfoRepo.Query(ctx).OrderInfo
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.TradeID.Eq(tradeID)))
	orderInfos, err := c.orderInfoRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	orderIDs := make([]int64, 0, len(orderInfos))
	for _, orderInfo := range orderInfos {
		orderIDs = append(orderIDs, orderInfo.ID)
	}
	if len(orderIDs) == 0 {
		return nil, nil
	}
	goodsQuery := c.orderGoodsRepo.Query(ctx).OrderGoods
	goodsOpts := make([]repository.QueryOption, 0, 1)
	goodsOpts = append(goodsOpts, repository.Where(goodsQuery.OrderID.In(orderIDs...)))
	return c.orderGoodsRepo.List(ctx, goodsOpts...)
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

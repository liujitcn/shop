package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"shop/pkg/config"
	_const "shop/pkg/const"

	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	appv1 "shop/api/gen/go/app/v1"
	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/queue"
	"shop/pkg/recommend/dto"
	"shop/pkg/workspaceevent"
	"shop/pkg/wx"
	appDto "shop/service/app/dto"
	"shop/service/app/utils"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/liujitcn/go-utils/id"
	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	_time "github.com/liujitcn/go-utils/time"
	"github.com/liujitcn/go-utils/trans"
	"github.com/liujitcn/gorm-kit/repository"
	wxPayCore "github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/refunddomestic"
	"gorm.io/gorm"
)

// ORDER_REFUND_REASON 表示订单退款原因字典编码。
const ORDER_REFUND_REASON string = "order_refund_reason" // 退款原因

// OrderInfoCase 订单业务处理对象
type OrderInfoCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.OrderInfoRepository
	orderTradeCase     *OrderTradeCase
	orderCancelCase    *OrderCancelCase
	orderGoodsCase     *OrderGoodsCase
	orderAddressCase   *OrderAddressCase
	orderLogisticsCase *OrderLogisticsCase
	orderPaymentCase   *OrderPaymentCase
	orderRefundCase    *OrderRefundCase
	goodsInfoCase      *GoodsInfoCase
	goodsSKUCase       *GoodsSKUCase
	tenantStoreCase    *TenantStoreCase
	userAddressCase    *UserAddressCase
	userCartCase       *UserCartCase
	baseDictItemCase   *BaseDictItemCase
	orderSchedulerCase *OrderSchedulerCase
	payCase            *PayCase
	wxPayCase          *wx.WxPayCase
	mapper             *mapper.CopierMapper[appv1.OrderInfo, models.OrderInfo]
}

// NewOrderInfoCase 创建订单业务处理对象
func NewOrderInfoCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	orderInfoRepo *data.OrderInfoRepository,
	orderTradeCase *OrderTradeCase,
	orderCancelCase *OrderCancelCase,
	orderGoodsCase *OrderGoodsCase,
	orderAddressCase *OrderAddressCase,
	orderLogisticsCase *OrderLogisticsCase,
	orderPaymentCase *OrderPaymentCase,
	orderRefundCase *OrderRefundCase,
	goodsInfoCase *GoodsInfoCase,
	goodsSKUCase *GoodsSKUCase,
	tenantStoreCase *TenantStoreCase,
	userAddressCase *UserAddressCase,
	userCartCase *UserCartCase,
	baseDictItemCase *BaseDictItemCase,
	orderSchedulerCase *OrderSchedulerCase,
	payCase *PayCase,
	wxPayCase *wx.WxPayCase,
) (*OrderInfoCase, error) {
	c := &OrderInfoCase{
		BaseCase:            baseCase,
		tx:                  tx,
		OrderInfoRepository: orderInfoRepo,
		orderTradeCase:      orderTradeCase,
		orderCancelCase:     orderCancelCase,
		orderGoodsCase:      orderGoodsCase,
		orderAddressCase:    orderAddressCase,
		orderLogisticsCase:  orderLogisticsCase,
		orderPaymentCase:    orderPaymentCase,
		orderRefundCase:     orderRefundCase,
		goodsInfoCase:       goodsInfoCase,
		goodsSKUCase:        goodsSKUCase,
		tenantStoreCase:     tenantStoreCase,
		userAddressCase:     userAddressCase,
		userCartCase:        userCartCase,
		baseDictItemCase:    baseDictItemCase,
		orderSchedulerCase:  orderSchedulerCase,
		payCase:             payCase,
		wxPayCase:           wxPayCase,
		mapper:              mapper.NewCopierMapper[appv1.OrderInfo, models.OrderInfo](),
	}

	// 服务启动时恢复全部未支付交易的超时取消任务。
	query := c.orderTradeCase.Query(context.Background()).OrderTrade
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.Status.In(
		_const.ORDER_TRADE_STATUS_PENDING_PAYMENT,
		_const.ORDER_TRADE_STATUS_PAYING,
	)))
	list, err := c.orderTradeCase.List(context.Background(), opts...)
	if err != nil {
		return nil, err
	}
	payTimeout := config.ParsePayTimeout()
	for _, item := range list {
		// 计算当前订单距离支付超时还剩余多少秒
		createdAt := item.CreatedAt.Add(payTimeout)
		nowTime := time.Now()
		countdown := createdAt.Sub(nowTime).Seconds()
		// 已超出支付时限的订单立即执行取消。
		if countdown < 0 {
			// 自动取消订单
			err = c.cancelOrder(context.Background(), item.UserID, &appv1.CancelOrderInfoRequest{
				TradeId: item.ID,
			})
			// 自动取消执行失败且交易仍未支付时，注册短周期重试任务。
			if err != nil {
				log.Error(fmt.Sprintf("CancelOrder order %d failed: %v", item.ID, err))
				c.scheduleTradeCancellation(item.ID, item.UserID, time.Minute)
			}
		} else {
			c.scheduleTradeCancellation(item.ID, item.UserID, time.Duration(countdown)*time.Second)
		}
	}

	return c, nil
}

// RefundOrderInfo 申请订单退款
func (c *OrderInfoCase) RefundOrderInfo(ctx context.Context, req *appv1.RefundOrderInfoRequest) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	var orderInfo *models.OrderInfo
	orderInfo, err = c.findByUserIDAndID(ctx, authInfo.UserId, req.GetOrderId())
	if err != nil {
		return err
	}

	// 用户侧退款只允许尚未发货的门店订单发起。
	if orderInfo.Status != _const.ORDER_INFO_STATUS_WAIT_SHIPMENT {
		return errorsx.StateConflict(
			fmt.Sprintf("订单状态错误：【%s】", commonv1.OrderInfoStatus_name[orderInfo.Status]),
			"order_info",
			commonv1.OrderInfoStatus(orderInfo.Status).String(),
			commonv1.OrderInfoStatus(_const.ORDER_INFO_STATUS_WAIT_SHIPMENT).String(),
		)
	}
	if orderInfo.RefundStatus == _const.ORDER_REFUND_STATUS_PROCESSING || orderInfo.RefundStatus == _const.ORDER_REFUND_STATUS_REFUNDED {
		return errorsx.StateConflict(
			"当前订单退款状态不允许重复申请",
			"order_info",
			commonv1.OrderRefundStatus(orderInfo.RefundStatus).String(),
			"NONE_ORS|PARTIAL_REFUND_ORS|CLOSED_OR_FAILED_ORS",
		)
	}
	var orderTrade *models.OrderTrade
	orderTrade, err = c.orderTradeCase.findByUserIDAndID(ctx, authInfo.UserId, orderInfo.TradeID)
	if err != nil {
		return err
	}
	var refundedMoney int64
	refundedMoney, err = c.successfulRefundMoney(ctx, orderInfo.ID)
	if err != nil {
		return err
	}
	refundMoney := orderInfo.PayMoney - refundedMoney
	if refundMoney <= 0 {
		return errorsx.StateConflict(
			"当前门店订单已无可退金额",
			"order_info",
			commonv1.OrderRefundStatus(orderInfo.RefundStatus).String(),
			"NONE_ORS|PARTIAL_REFUND_ORS|CLOSED_OR_FAILED_ORS",
		)
	}

	orderRefund := &models.OrderRefund{
		TradeID:       orderTrade.ID,
		TenantID:      orderInfo.TenantID,
		TenantStoreID: orderInfo.TenantStoreID,
		OrderID:       req.GetOrderId(),
		TradeNo:       orderTrade.TradeNo,
		RefundNo:      strconv.FormatInt(id.GenSnowflakeID(), 10),
		Reason:        int32(req.GetReason()),
		CreateTime:    time.Now(),
		RefundState:   appv1.RefundResource_PROCESSING.String(),
		Amount: _string.ConvertAnyToJsonString(map[string]int64{
			"total":        orderTrade.PayMoney,
			"refund":       refundMoney,
			"payer_total":  orderTrade.PayMoney,
			"payer_refund": refundMoney,
		}),
		Status: _const.ORDER_BILL_STATUS_NO_CHECK,
	}
	err = c.claimOrderRefund(ctx, orderInfo, orderRefund)
	if err != nil {
		return err
	}
	var refundResource *appv1.RefundResource
	// 只有在线支付交易才会走微信退款流程。
	if commonv1.OrderPayType(orderTrade.PayType) == commonv1.OrderPayType(_const.ORDER_PAY_TYPE_ONLINE_PAY) {
		// 先把退款原因值翻译成字典标签，便于退款记录展示
		reason := strconv.Itoa(int(orderRefund.Reason))
		var label string
		label, err = c.baseDictItemCase.findLabelByCodeAndValue(ctx, ORDER_REFUND_REASON, reason)
		// 字典标签查询成功时，使用标签替换原始原因值。
		if err == nil {
			reason = label
		}
		// 仅微信支付订单需要调用微信退款接口
		if commonv1.OrderPayChannel(orderTrade.PayChannel) == commonv1.OrderPayChannel(_const.ORDER_PAY_CHANNEL_WX_PAY) {
			var refund *refunddomestic.Refund
			refund, err = c.wxPayCase.Refund(refunddomestic.CreateRequest{
				OutTradeNo:  trans.String(orderTrade.TradeNo),
				OutRefundNo: trans.String(orderRefund.RefundNo),
				Reason:      trans.String(reason),
				Amount: &refunddomestic.AmountReq{
					Total:    trans.Int64(orderTrade.PayMoney),
					Refund:   trans.Int64(refundMoney),
					Currency: trans.String("CNY"),
				},
			})
			// 微信退款创建失败时，需要识别“已全额退款”的可恢复场景。
			if err != nil {
				// 命中微信 API 错误结构时，再判断是否属于幂等退款场景。
				if apiErr, ok := errors.AsType[*wxPayCore.APIError](err); ok {
					// 微信明确返回“订单已全额退款”时，只能用此前真正创建过的退款号恢复状态。
					if apiErr.Code == "INVALID_REQUEST" && apiErr.Message == "订单已全额退款" {
						err = c.recoverCompletedOrderRefund(ctx, orderTrade, orderInfo, orderRefund)
						if err != nil {
							return err
						}
						return errorsx.StateConflict(
							"订单已退款，不能重复退款",
							"order_refund",
							appv1.RefundResource_RefundStatus(_const.REFUND_RESOURCE_STATUS_SUCCESS).String(),
							"NONE",
						)
					}
				}
				// 网络、超时和微信系统错误无法判断是否受理，必须保留原退款号交给补偿任务查实。
				if wx.IsRefundCreateResultUncertain(err) {
					workspaceevent.Publish(ctx, workspaceevent.ReasonOrderChanged, workspaceevent.AreaTodo, workspaceevent.AreaMetrics)
					return nil
				}
				orderRefund.RefundState = appv1.RefundResource_ABNORMAL.String()
				stateErr := c.tx.Transaction(ctx, func(ctx context.Context) error {
					persistErr := c.orderRefundCase.UpdateByID(ctx, orderRefund)
					if persistErr != nil {
						return persistErr
					}
					return c.OrderInfoRepository.UpdateByID(ctx, &models.OrderInfo{
						ID:           orderInfo.ID,
						RefundStatus: _const.ORDER_REFUND_STATUS_CLOSED_OR_FAILED,
					})
				})
				if stateErr != nil {
					return stateErr
				}
				return err
			}
			refundResource = wx.ConvertRefundResource(refund)
		}
	} else {
		refundResource = &appv1.RefundResource{
			OutTradeNo:   orderTrade.TradeNo,
			OutRefundNo:  orderRefund.RefundNo,
			RefundStatus: appv1.RefundResource_SUCCESS,
			Amount: &appv1.RefundResource_Amount{
				Total:       int32(orderTrade.PayMoney),
				Refund:      int32(refundMoney),
				PayerTotal:  int32(orderTrade.PayMoney),
				PayerRefund: int32(refundMoney),
			},
		}
	}
	// 渠道返回体缺少有效状态时结果仍不确定，保留处理中等待补偿查询。
	if refundResource == nil || refundResource.GetRefundStatus() == appv1.RefundResource_REFUND_STATUS_UNSPECIFIED {
		workspaceevent.Publish(ctx, workspaceevent.ReasonOrderChanged, workspaceevent.AreaTodo, workspaceevent.AreaMetrics)
		return nil
	}
	return c.payCase.RefundSuccess(ctx, orderTrade, refundResource)
}

// recoverCompletedOrderRefund 用此前已创建的退款号恢复微信已全额退款的本地状态。
func (c *OrderInfoCase) recoverCompletedOrderRefund(ctx context.Context, orderTrade *models.OrderTrade, orderInfo *models.OrderInfo, currentRefund *models.OrderRefund) error {
	currentRefund.RefundState = appv1.RefundResource_ABNORMAL.String()
	err := c.orderRefundCase.UpdateByID(ctx, currentRefund)
	if err != nil {
		return err
	}
	query := c.orderRefundCase.Query(ctx).OrderRefund
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.OrderID.Eq(orderInfo.ID)))
	var orderRefunds []*models.OrderRefund
	orderRefunds, err = c.orderRefundCase.List(ctx, opts...)
	if err != nil {
		return err
	}
	var queryErr error
	for _, orderRefund := range orderRefunds {
		if orderRefund.ID == currentRefund.ID || orderRefund.RefundNo == "" {
			continue
		}
		var refundResource *appv1.RefundResource
		refundResource, queryErr = c.wxPayCase.QueryByOutRefundNo(orderRefund.RefundNo)
		if queryErr != nil || refundResource.GetRefundStatus() != appv1.RefundResource_SUCCESS {
			continue
		}
		return c.payCase.RefundSuccess(ctx, orderTrade, refundResource)
	}
	err = c.OrderInfoRepository.UpdateByID(ctx, &models.OrderInfo{
		ID:           orderInfo.ID,
		RefundStatus: _const.ORDER_REFUND_STATUS_CLOSED_OR_FAILED,
	})
	if err != nil {
		return err
	}
	if queryErr != nil {
		return errorsx.Internal("恢复微信退款状态失败").WithCause(queryErr)
	}
	return errorsx.Internal("恢复微信退款状态失败")
}

// claimOrderRefund 抢占门店订单退款权并创建待处理退款记录。
func (c *OrderInfoCase) claimOrderRefund(ctx context.Context, orderInfo *models.OrderInfo, orderRefund *models.OrderRefund) error {
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		query := c.Query(ctx).OrderInfo
		result, err := query.WithContext(ctx).
			Where(
				query.ID.Eq(orderInfo.ID),
				query.UserID.Eq(orderInfo.UserID),
				query.Status.Eq(_const.ORDER_INFO_STATUS_WAIT_SHIPMENT),
				query.RefundStatus.In(
					_const.ORDER_REFUND_STATUS_NONE,
					_const.ORDER_REFUND_STATUS_PARTIAL_REFUND,
					_const.ORDER_REFUND_STATUS_CLOSED_OR_FAILED,
				),
			).
			Update(query.RefundStatus, _const.ORDER_REFUND_STATUS_PROCESSING)
		if err != nil {
			return err
		}
		if result.RowsAffected == 0 {
			return errorsx.StateConflict(
				"当前订单退款状态不允许重复申请",
				"order_info",
				commonv1.OrderRefundStatus(orderInfo.RefundStatus).String(),
				"NONE_ORS|PARTIAL_REFUND_ORS|CLOSED_OR_FAILED_ORS",
			)
		}
		return c.orderRefundCase.Create(ctx, orderRefund)
	})
}

// ConfirmOrderInfo 确认订单
func (c *OrderInfoCase) ConfirmOrderInfo(ctx context.Context) (*appv1.ConfirmOrderInfoResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	member := utils.IsMemberByAuthInfo(authInfo)

	// 查询购物车列表
	query := c.userCartCase.Query(ctx).UserCart
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.UserID.Eq(authInfo.UserId)))
	opts = append(opts, repository.Where(query.IsChecked.Is(true)))
	var userCartList []*models.UserCart
	userCartList, err = c.userCartCase.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	createOrderGoods := make([]*appv1.CreateOrderInfoGoods, 0)
	for _, item := range userCartList {
		createOrderGoods = append(createOrderGoods, &appv1.CreateOrderInfoGoods{
			GoodsId: item.GoodsID,
			SkuCode: item.SKUCode,
			Num:     item.Num,
			RecommendContext: &appv1.RecommendContext{
				Scene:     commonv1.RecommendScene(item.Scene),
				RequestId: item.RequestID,
				Position:  item.Position,
			},
		})
	}
	return c.orderBuy(ctx, member, createOrderGoods)
}

// BuyNowOrderInfo 立即购买订单
func (c *OrderInfoCase) BuyNowOrderInfo(ctx context.Context, req *appv1.BuyNowOrderInfoRequest) (*appv1.BuyNowOrderInfoResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	member := utils.IsMemberByAuthInfo(authInfo)

	// 将单个商品请求封装成统一的下单明细列表
	createOrderGoods := []*appv1.CreateOrderInfoGoods{{
		GoodsId:          req.GetGoodsId(),
		SkuCode:          req.GetSkuCode(),
		Num:              req.GetNum(),
		RecommendContext: req.GetRecommendContext(),
	}}
	var res *appv1.ConfirmOrderInfoResponse
	res, err = c.orderBuy(ctx, member, createOrderGoods)
	if err != nil {
		return nil, err
	}
	return &appv1.BuyNowOrderInfoResponse{
		OrderGoodsStores: res.GetOrderGoodsStores(),
		Summary:          res.GetSummary(),
		ClearCart:        res.GetClearCart(),
	}, nil
}

// RepurchaseOrderInfo 再次购买订单
func (c *OrderInfoCase) RepurchaseOrderInfo(ctx context.Context, req *appv1.RepurchaseOrderInfoRequest) (*appv1.RepurchaseOrderInfoResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	member := utils.IsMemberByAuthInfo(authInfo)
	var orderInfo *models.OrderInfo
	orderInfo, err = c.findByUserIDAndID(ctx, authInfo.UserId, req.GetOrderId())
	if err != nil {
		return nil, err
	}
	// 读取原订单中的商品明细，重新构造成下单请求
	query := c.orderGoodsCase.Query(ctx).OrderGoods
	var oldOrderGoods []*models.OrderGoods
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.OrderID.Eq(orderInfo.ID)))
	oldOrderGoods, err = c.orderGoodsCase.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	createOrderGoods := make([]*appv1.CreateOrderInfoGoods, 0)
	for _, item := range oldOrderGoods {
		createOrderGoods = append(createOrderGoods, &appv1.CreateOrderInfoGoods{
			GoodsId: item.GoodsID,
			SkuCode: item.SKUCode,
			Num:     item.Num,
			RecommendContext: &appv1.RecommendContext{
				Scene:     commonv1.RecommendScene(item.Scene),
				RequestId: item.RequestID,
				Position:  item.Position,
			},
		})
	}
	var res *appv1.ConfirmOrderInfoResponse
	res, err = c.orderBuy(ctx, member, createOrderGoods)
	if err != nil {
		return nil, err
	}
	return &appv1.RepurchaseOrderInfoResponse{
		OrderGoodsStores: res.GetOrderGoodsStores(),
		Summary:          res.GetSummary(),
		ClearCart:        res.GetClearCart(),
	}, nil
}

// CountOrderInfo 查询订单数量汇总
func (c *OrderInfoCase) CountOrderInfo(ctx context.Context) (*appv1.CountOrderInfoResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	tradeQuery := c.orderTradeCase.Query(ctx).OrderTrade
	tradeRows := make([]*appDto.OrderStatusCountRow, 0)
	err = tradeQuery.WithContext(ctx).
		Select(tradeQuery.Status, tradeQuery.ID.Count().As("total")).
		Where(tradeQuery.UserID.Eq(authInfo.UserId), tradeQuery.DeletedAt.IsNull()).
		Group(tradeQuery.Status).
		Scan(&tradeRows)
	if err != nil {
		return nil, err
	}
	query := c.Query(ctx).OrderInfo
	orderRows := make([]*appDto.OrderStatusCountRow, 0)
	err = query.WithContext(ctx).
		Select(query.Status, query.ID.Count().As("total")).
		Where(query.UserID.Eq(authInfo.UserId), query.DeletedAt.IsNull()).
		Group(query.Status).
		Scan(&orderRows)
	if err != nil {
		return nil, err
	}
	refundRows := make([]*appDto.OrderStatusCountRow, 0)
	err = query.WithContext(ctx).
		Select(query.RefundStatus.As("status"), query.ID.Count().As("total")).
		Where(
			query.UserID.Eq(authInfo.UserId),
			query.RefundStatus.Neq(_const.ORDER_REFUND_STATUS_NONE),
			query.DeletedAt.IsNull(),
		).
		Group(query.RefundStatus).
		Scan(&refundRows)
	if err != nil {
		return nil, err
	}
	count := make([]*appv1.CountOrderInfoResponse_Count, 0, len(tradeRows)+len(orderRows)+len(refundRows))
	for _, row := range tradeRows {
		count = append(count, &appv1.CountOrderInfoResponse_Count{
			TradeStatus: commonv1.OrderTradeStatus(row.Status),
			Num:         int32(row.Total),
		})
	}
	for _, row := range orderRows {
		count = append(count, &appv1.CountOrderInfoResponse_Count{
			Status: commonv1.OrderInfoStatus(row.Status),
			Num:    int32(row.Total),
		})
	}
	for _, row := range refundRows {
		count = append(count, &appv1.CountOrderInfoResponse_Count{
			RefundStatus: commonv1.OrderRefundStatus(row.Status),
			Num:          int32(row.Total),
		})
	}
	return &appv1.CountOrderInfoResponse{
		Counts: count,
	}, nil
}

// PageOrderInfo 查询订单分页列表
func (c *OrderInfoCase) PageOrderInfo(ctx context.Context, req *appv1.PageOrderInfoRequest) (*appv1.PageOrderInfoResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	pageNum, pageSize := repository.PageDefault(int64(req.GetPageNum()), int64(req.GetPageSize()))
	// 两类记录各取合并目标页之前的候选，保证结果准确且不随用户全部订单量增长。
	fetchSize := pageNum * pageSize
	tradeStatus := req.GetTradeStatus()
	orderStatus := req.GetStatus()
	refundStatus := req.GetRefundStatus()
	refundableOnly := req.Refundable != nil && req.GetRefundable()
	noStatusFilter := tradeStatus == commonv1.OrderTradeStatus_UNKNOWN_OTS &&
		orderStatus == commonv1.OrderInfoStatus_UNKNOWN_OIS &&
		refundStatus == commonv1.OrderRefundStatus_UNKNOWN_ORS && req.HasRefund == nil && !refundableOnly
	aggregateTradeStatus := tradeStatus == commonv1.OrderTradeStatus_PENDING_PAYMENT_OTS ||
		tradeStatus == commonv1.OrderTradeStatus_PAYING_OTS ||
		tradeStatus == commonv1.OrderTradeStatus_CLOSED_OTS

	var trades []*models.OrderTrade
	var tradeTotal int64
	// 待支付、支付中和已关闭按交易聚合展示，取消履约筛选同样映射到已关闭交易。
	if noStatusFilter || aggregateTradeStatus || orderStatus == commonv1.OrderInfoStatus_CANCELED_OIS {
		query := c.orderTradeCase.Query(ctx).OrderTrade
		opts := make([]repository.QueryOption, 0, 3)
		opts = append(opts, repository.Where(query.UserID.Eq(authInfo.UserId)))
		if aggregateTradeStatus {
			// “待支付”标签同时覆盖尚未发起支付和已经获取预支付参数的交易。
			if tradeStatus == commonv1.OrderTradeStatus_PENDING_PAYMENT_OTS {
				opts = append(opts, repository.Where(query.Status.In(
					_const.ORDER_TRADE_STATUS_PENDING_PAYMENT,
					_const.ORDER_TRADE_STATUS_PAYING,
				)))
			} else {
				opts = append(opts, repository.Where(query.Status.Eq(int32(tradeStatus))))
			}
		} else if orderStatus == commonv1.OrderInfoStatus_CANCELED_OIS {
			opts = append(opts, repository.Where(query.Status.Eq(_const.ORDER_TRADE_STATUS_CLOSED)))
		} else {
			opts = append(opts, repository.Where(query.Status.In(
				_const.ORDER_TRADE_STATUS_PENDING_PAYMENT,
				_const.ORDER_TRADE_STATUS_PAYING,
				_const.ORDER_TRADE_STATUS_CLOSED,
			)))
		}
		// 全部订单页优先展示待支付、支付中和已关闭的交易单，再按创建时间排序。
		if noStatusFilter {
			opts = append(opts, repository.Order(query.Status.Asc(), query.CreatedAt.Desc(), query.ID.Desc()))
		} else {
			opts = append(opts, repository.Order(query.CreatedAt.Desc(), query.ID.Desc()))
		}
		trades, tradeTotal, err = c.orderTradeCase.Page(ctx, 1, fetchSize, opts...)
		if err != nil {
			return nil, err
		}
	}

	childTradeStatus := tradeStatus != commonv1.OrderTradeStatus_UNKNOWN_OTS && !aggregateTradeStatus
	var orderInfos []*models.OrderInfo
	var orderTotal int64
	queryChildOrders := noStatusFilter || childTradeStatus ||
		(orderStatus != commonv1.OrderInfoStatus_UNKNOWN_OIS && orderStatus != commonv1.OrderInfoStatus_CANCELED_OIS) ||
		refundStatus != commonv1.OrderRefundStatus_UNKNOWN_ORS || req.HasRefund != nil || refundableOnly
	if queryChildOrders {
		query := c.Query(ctx).OrderInfo
		opts := make([]repository.QueryOption, 0, 8)
		opts = append(opts, repository.Where(query.UserID.Eq(authInfo.UserId)))
		if childTradeStatus {
			// 交易支付状态属于 order_trade，使用类型化连接直接筛选，避免先加载全部交易编号。
			tradeQuery := c.orderTradeCase.Query(ctx).OrderTrade
			opts = append(opts, repository.Join(tradeQuery, query.TradeID.EqCol(tradeQuery.ID)))
			opts = append(opts, repository.Where(tradeQuery.UserID.Eq(authInfo.UserId)))
			opts = append(opts, repository.Where(tradeQuery.Status.Eq(int32(tradeStatus))))
			opts = append(opts, repository.Where(tradeQuery.DeletedAt.IsNull()))
		}
		if orderStatus != commonv1.OrderInfoStatus_UNKNOWN_OIS {
			opts = append(opts, repository.Where(query.Status.Eq(int32(orderStatus))))
		} else if noStatusFilter {
			opts = append(opts, repository.Where(query.Status.NotIn(
				_const.ORDER_INFO_STATUS_NOT_STARTED,
				_const.ORDER_INFO_STATUS_CANCELED,
			)))
		}
		if refundStatus != commonv1.OrderRefundStatus_UNKNOWN_ORS {
			opts = append(opts, repository.Where(query.RefundStatus.Eq(int32(refundStatus))))
		}
		if req.HasRefund != nil {
			// 售后记录查询需要覆盖处理中、部分退款、已退款和已关闭/失败。
			if req.GetHasRefund() {
				opts = append(opts, repository.Where(query.RefundStatus.Neq(_const.ORDER_REFUND_STATUS_NONE)))
			} else {
				opts = append(opts, repository.Where(query.RefundStatus.Eq(_const.ORDER_REFUND_STATUS_NONE)))
			}
		}
		if refundableOnly {
			// 可申请退款订单必须尚未发货，且当前没有进行中的退款或全额退款。
			opts = append(opts, repository.Where(query.Status.Eq(_const.ORDER_INFO_STATUS_WAIT_SHIPMENT)))
			opts = append(opts, repository.Where(query.RefundStatus.In(
				_const.ORDER_REFUND_STATUS_NONE,
				_const.ORDER_REFUND_STATUS_PARTIAL_REFUND,
				_const.ORDER_REFUND_STATUS_CLOSED_OR_FAILED,
			)))
		}
		opts = append(opts, repository.Order(query.CreatedAt.Desc(), query.ID.Desc()))
		orderInfos, orderTotal, err = c.Page(ctx, 1, fetchSize, opts...)
		if err != nil {
			return nil, err
		}
	}

	tradeIDs := make([]int64, 0, len(trades)+len(orderInfos))
	tradeMap := make(map[int64]*models.OrderTrade, len(trades)+len(orderInfos))
	for _, trade := range trades {
		tradeIDs = append(tradeIDs, trade.ID)
		tradeMap[trade.ID] = trade
	}
	for _, orderInfo := range orderInfos {
		if tradeMap[orderInfo.TradeID] == nil {
			tradeIDs = append(tradeIDs, orderInfo.TradeID)
		}
	}
	if len(tradeIDs) > 0 {
		query := c.orderTradeCase.Query(ctx).OrderTrade
		opts := make([]repository.QueryOption, 0, 1)
		opts = append(opts, repository.Where(query.ID.In(tradeIDs...)))
		var allTrades []*models.OrderTrade
		allTrades, err = c.orderTradeCase.List(ctx, opts...)
		if err != nil {
			return nil, err
		}
		for _, trade := range allTrades {
			tradeMap[trade.ID] = trade
		}
	}

	allOrderInfos := append([]*models.OrderInfo{}, orderInfos...)
	if len(trades) > 0 {
		query := c.Query(ctx).OrderInfo
		opts := make([]repository.QueryOption, 0, 1)
		opts = append(opts, repository.Where(query.TradeID.In(tradeIDs...)))
		var tradeOrderInfos []*models.OrderInfo
		tradeOrderInfos, err = c.List(ctx, opts...)
		if err != nil {
			return nil, err
		}
		allOrderInfos = append(allOrderInfos, tradeOrderInfos...)
	}
	orderIDs := make([]int64, 0, len(allOrderInfos))
	orderInfoMap := make(map[int64]*models.OrderInfo, len(allOrderInfos))
	for _, orderInfo := range allOrderInfos {
		orderIDs = append(orderIDs, orderInfo.ID)
		orderInfoMap[orderInfo.ID] = orderInfo
	}
	var orderGoodsMap map[int64][]*models.OrderGoods
	orderGoodsMap, err = c.orderGoodsCase.mapByOrderIDs(ctx, orderIDs)
	if err != nil {
		return nil, err
	}
	var orderGoodsStoreMap map[int64][]*appv1.OrderGoodsStore
	orderGoodsStoreMap, err = c.buildOrderGoodsStoreMap(ctx, orderGoodsMap)
	if err != nil {
		return nil, err
	}

	items := make([]*appDto.OrderPageItem, 0, len(trades)+len(orderInfos))
	for _, orderInfo := range orderInfos {
		protoOrder := c.convertToProto(orderInfo)
		c.applyTradeToProto(protoOrder, tradeMap[orderInfo.TradeID])
		protoOrder.OrderGoodsStores = orderGoodsStoreMap[orderInfo.ID]
		applyOrderStoreOptions(protoOrder.OrderGoodsStores, []*models.OrderInfo{orderInfo})
		items = append(items, &appDto.OrderPageItem{OrderInfo: protoOrder, CreatedAt: orderInfo.CreatedAt})
	}
	for _, trade := range trades {
		tradeOrders := make([]*models.OrderInfo, 0)
		for _, orderInfo := range orderInfoMap {
			if orderInfo.TradeID == trade.ID {
				tradeOrders = append(tradeOrders, orderInfo)
			}
		}
		var protoOrder *appv1.OrderInfo
		protoOrder, err = c.buildTradeOrderProto(ctx, trade, tradeOrders, orderGoodsMap)
		if err != nil {
			return nil, err
		}
		items = append(items, &appDto.OrderPageItem{OrderInfo: protoOrder, CreatedAt: trade.CreatedAt})
	}
	return &appv1.PageOrderInfoResponse{
		OrderInfos: pageOrderItems(items, pageNum, pageSize, noStatusFilter),
		Total:      int32(tradeTotal + orderTotal),
	}, nil
}

// pageOrderItems 将交易聚合记录和门店订单合并后截取目标页。
func pageOrderItems(items []*appDto.OrderPageItem, pageNum, pageSize int64, prioritizeTrades bool) []*appv1.OrderInfo {
	sort.SliceStable(items, func(i, j int) bool {
		if prioritizeTrades {
			leftTrade := items[i].OrderInfo.GetIsTrade()
			rightTrade := items[j].OrderInfo.GetIsTrade()
			if leftTrade != rightTrade {
				return leftTrade
			}
			if leftTrade && items[i].OrderInfo.GetTradeStatus() != items[j].OrderInfo.GetTradeStatus() {
				return items[i].OrderInfo.GetTradeStatus() < items[j].OrderInfo.GetTradeStatus()
			}
		}
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].OrderInfo.GetId() > items[j].OrderInfo.GetId()
		}
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	start := (pageNum - 1) * pageSize
	if start > int64(len(items)) {
		start = int64(len(items))
	}
	end := min(start+pageSize, int64(len(items)))
	list := make([]*appv1.OrderInfo, 0, end-start)
	for _, item := range items[start:end] {
		list = append(list, item.OrderInfo)
	}
	return list
}

// GetOrderInfoIDByOrderNo 按订单号查询订单编号
func (c *OrderInfoCase) GetOrderInfoIDByOrderNo(ctx context.Context, orderNo string) (int64, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return 0, err
	}

	query := c.Query(ctx).OrderInfo
	var item *models.OrderInfo
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.OrderNo.Eq(orderNo)))
	opts = append(opts, repository.Where(query.UserID.Eq(authInfo.UserId)))
	item, err = c.Find(ctx, opts...)
	if err != nil {
		return 0, err
	}
	return item.ID, nil
}

// GetOrderInfoByID 根据订单编号查询订单
func (c *OrderInfoCase) GetOrderInfoByID(ctx context.Context, id int64) (*appv1.OrderInfoResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	var item *models.OrderInfo
	item, err = c.findByUserIDAndID(ctx, authInfo.UserId, id)
	if err != nil {
		// 管理员查看订单详情时，允许按订单 ID 读取详情。
		if !isAdminRoleCode(authInfo.RoleCode) {
			return nil, err
		}
		item, err = c.FindByID(ctx, id)
		if err != nil {
			return nil, err
		}
	}
	orderInfo := c.convertToProto(item)
	var orderTrade *models.OrderTrade
	orderTrade, err = c.orderTradeCase.findByUserIDAndID(ctx, item.UserID, item.TradeID)
	if err != nil {
		return nil, err
	}
	c.applyTradeToProto(orderInfo, orderTrade)
	var countdown float64
	// 只有未完成支付的交易单才需要返回支付剩余时间。
	if orderTrade.Status == _const.ORDER_TRADE_STATUS_PENDING_PAYMENT || orderTrade.Status == _const.ORDER_TRADE_STATUS_PAYING {
		payTimeout := config.ParsePayTimeout()
		countdown = orderTrade.CreatedAt.Add(payTimeout).Sub(time.Now()).Seconds()
	}

	// 查询订单商品明细，并同时按商品快照中的门店编号生成展示分组。
	var orderGoodsList []*models.OrderGoods
	orderGoodsList, err = c.orderGoodsCase.listByOrderID(ctx, orderInfo.Id)
	if err != nil {
		return nil, err
	}
	var orderGoodsStoreMap map[int64][]*appv1.OrderGoodsStore
	orderGoodsStoreMap, err = c.buildOrderGoodsStoreMap(ctx, map[int64][]*models.OrderGoods{
		orderInfo.Id: orderGoodsList,
	})
	if err != nil {
		return nil, err
	}
	orderInfo.OrderGoodsStores = orderGoodsStoreMap[orderInfo.Id]
	applyOrderStoreOptions(orderInfo.OrderGoodsStores, []*models.OrderInfo{item})

	// 合并支付的门店订单返回同一交易下的其他订单，用于详情页相互跳转。
	query := c.Query(ctx).OrderInfo
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Where(query.TradeID.Eq(item.TradeID)))
	opts = append(opts, repository.Where(query.UserID.Eq(item.UserID)))
	opts = append(opts, repository.Where(query.ID.Neq(item.ID)))
	opts = append(opts, repository.Order(query.ID.Asc()))
	var relatedOrderInfos []*models.OrderInfo
	relatedOrderInfos, err = c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	relatedOrders := make([]*appv1.OrderInfoResponse_RelatedOrder, 0, len(relatedOrderInfos))
	for _, relatedOrderInfo := range relatedOrderInfos {
		relatedOrders = append(relatedOrders, &appv1.OrderInfoResponse_RelatedOrder{
			OrderId: relatedOrderInfo.ID,
			OrderNo: relatedOrderInfo.OrderNo,
		})
	}
	// 查询订单收货地址快照
	var address *appv1.OrderInfoResponse_Address
	address, err = c.orderAddressCase.findByTradeID(ctx, orderInfo.TradeId)
	if err != nil {
		// 地址信息缺失时返回空地址，保持订单详情主体信息可展示。
		if errors.Is(err, gorm.ErrRecordNotFound) {
			address = &appv1.OrderInfoResponse_Address{}
		} else {
			return nil, err
		}
	}

	res := appv1.OrderInfoResponse{
		Order:         orderInfo,
		Address:       address,
		Countdown:     float32(countdown),
		RelatedOrders: relatedOrders,
	}

	// 在线支付交易进入已支付或退款状态后，补充支付完成时间。
	if commonv1.OrderPayType(orderTrade.PayType) == commonv1.OrderPayType(_const.ORDER_PAY_TYPE_ONLINE_PAY) &&
		(orderTrade.Status == _const.ORDER_TRADE_STATUS_PAID ||
			orderTrade.Status == _const.ORDER_TRADE_STATUS_PARTIAL_REFUND ||
			orderTrade.Status == _const.ORDER_TRADE_STATUS_FULL_REFUND) {
		res.Order.PaymentTime, err = c.orderPaymentCase.findPaymentTimeByTradeID(ctx, orderInfo.TradeId)
		if err != nil {
			// 支付记录缺失时，支付时间保持为空。
			if errors.Is(err, gorm.ErrRecordNotFound) {
				res.Order.PaymentTime = ""
			} else {
				return nil, err
			}
		}
	}
	// 已发货、待评价、已完成订单返回当前门店的物流信息。
	if item.Status == _const.ORDER_INFO_STATUS_SHIPPED ||
		item.Status == _const.ORDER_INFO_STATUS_WAIT_REVIEW ||
		item.Status == _const.ORDER_INFO_STATUS_COMPLETED {
		var logistics *appv1.OrderInfoResponse_Logistics
		logistics, err = c.orderLogisticsCase.findByOrderID(ctx, orderInfo.Id)
		if err != nil {
			// 物流信息缺失时返回空物流，保持订单详情主体信息可展示。
			if errors.Is(err, gorm.ErrRecordNotFound) {
				logistics = &appv1.OrderInfoResponse_Logistics{}
			} else {
				return nil, err
			}
		}
		res.Logistics = logistics
	}
	// 交易关闭后返回整笔交易的取消时间。
	if orderTrade.Status == _const.ORDER_TRADE_STATUS_CLOSED {
		res.Order.CancelTime, err = c.orderCancelCase.findCancelTimeByTradeID(ctx, orderInfo.TradeId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				res.Order.CancelTime = ""
			} else {
				return nil, err
			}
		}
	}
	// 当前门店订单存在退款时，返回该子订单的退款时间。
	if item.RefundStatus != _const.ORDER_REFUND_STATUS_NONE {
		res.Order.RefundTime, err = c.orderRefundCase.findRefundTimeByOrderID(ctx, orderInfo.Id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				res.Order.RefundTime = ""
			} else {
				return nil, err
			}
		}
	}
	return &res, nil
}

// GetOrderTradeByID 根据交易单编号查询聚合订单详情。
func (c *OrderInfoCase) GetOrderTradeByID(ctx context.Context, tradeID int64) (*appv1.OrderInfoResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	var orderTrade *models.OrderTrade
	orderTrade, err = c.orderTradeCase.findByUserIDAndID(ctx, authInfo.UserId, tradeID)
	if err != nil {
		return nil, err
	}

	query := c.Query(ctx).OrderInfo
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.TradeID.Eq(tradeID)))
	opts = append(opts, repository.Where(query.UserID.Eq(authInfo.UserId)))
	var orderInfos []*models.OrderInfo
	orderInfos, err = c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	orderIDs := make([]int64, 0, len(orderInfos))
	for _, orderInfo := range orderInfos {
		orderIDs = append(orderIDs, orderInfo.ID)
	}
	var orderGoodsMap map[int64][]*models.OrderGoods
	orderGoodsMap, err = c.orderGoodsCase.mapByOrderIDs(ctx, orderIDs)
	if err != nil {
		return nil, err
	}
	var orderInfo *appv1.OrderInfo
	orderInfo, err = c.buildTradeOrderProto(ctx, orderTrade, orderInfos, orderGoodsMap)
	if err != nil {
		return nil, err
	}

	var countdown float64
	// 只有未完成支付的交易单才需要返回支付剩余时间。
	if orderTrade.Status == _const.ORDER_TRADE_STATUS_PENDING_PAYMENT || orderTrade.Status == _const.ORDER_TRADE_STATUS_PAYING {
		payTimeout := config.ParsePayTimeout()
		countdown = orderTrade.CreatedAt.Add(payTimeout).Sub(time.Now()).Seconds()
	}
	var address *appv1.OrderInfoResponse_Address
	address, err = c.orderAddressCase.findByTradeID(ctx, tradeID)
	if err != nil {
		// 地址信息缺失时返回空地址，保持交易详情主体信息可展示。
		if errors.Is(err, gorm.ErrRecordNotFound) {
			address = &appv1.OrderInfoResponse_Address{}
		} else {
			return nil, err
		}
	}
	res := &appv1.OrderInfoResponse{
		Order:     orderInfo,
		Address:   address,
		Countdown: float32(countdown),
	}
	// 在线支付交易进入已支付或退款状态后，补充支付完成时间。
	if commonv1.OrderPayType(orderTrade.PayType) == commonv1.OrderPayType(_const.ORDER_PAY_TYPE_ONLINE_PAY) &&
		(orderTrade.Status == _const.ORDER_TRADE_STATUS_PAID ||
			orderTrade.Status == _const.ORDER_TRADE_STATUS_PARTIAL_REFUND ||
			orderTrade.Status == _const.ORDER_TRADE_STATUS_FULL_REFUND) {
		res.Order.PaymentTime, err = c.orderPaymentCase.findPaymentTimeByTradeID(ctx, tradeID)
		if err != nil {
			// 支付记录缺失时，支付时间保持为空。
			if errors.Is(err, gorm.ErrRecordNotFound) {
				res.Order.PaymentTime = ""
			} else {
				return nil, err
			}
		}
	}
	// 已关闭交易返回整笔交易的取消时间。
	if orderTrade.Status == _const.ORDER_TRADE_STATUS_CLOSED {
		res.Order.CancelTime, err = c.orderCancelCase.findCancelTimeByTradeID(ctx, tradeID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				res.Order.CancelTime = ""
			} else {
				return nil, err
			}
		}
	}
	return res, nil
}

// CreateOrderInfo 创建订单并发起支付准备
func (c *OrderInfoCase) CreateOrderInfo(ctx context.Context, request *appv1.CreateOrderInfoRequest) (*appv1.CreateOrderInfoResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	if request.PayType != commonv1.OrderPayType_ONLINE_PAY && request.PayType != commonv1.OrderPayType_CASH_ON_DELIVERY {
		return nil, errorsx.InvalidArgument("支付方式无效")
	}
	payChannel := request.PayChannel
	// 货到付款不归属线上支付渠道，避免统计时被错误计入微信或银联。
	if request.PayType == commonv1.OrderPayType_CASH_ON_DELIVERY {
		payChannel = commonv1.OrderPayChannel_UNKNOWN_OPC
	} else if payChannel == commonv1.OrderPayChannel_UNKNOWN_OPC {
		return nil, errorsx.InvalidArgument("支付渠道不能为空")
	} else if payChannel != commonv1.OrderPayChannel_WX_PAY {
		return nil, errorsx.InvalidArgument("当前仅支持微信支付")
	}
	tradeStatus := _const.ORDER_TRADE_STATUS_PENDING_PAYMENT
	orderStatus := _const.ORDER_INFO_STATUS_NOT_STARTED
	// 货到付款交易创建后，门店订单直接进入待发货。
	if request.PayType == commonv1.OrderPayType(_const.ORDER_PAY_TYPE_CASH_ON_DELIVERY) {
		tradeStatus = _const.ORDER_TRADE_STATUS_CASH_ON_DELIVERY
		orderStatus = _const.ORDER_INFO_STATUS_WAIT_SHIPMENT
	}

	storeOptions := make(map[int64]*appv1.OrderStoreOption, len(request.GetOrderStoreOptions()))
	for _, option := range request.GetOrderStoreOptions() {
		storeOptions[option.GetTenantStoreId()] = option
	}
	orderGoodsByStore := make(map[int64][]*models.OrderGoods)
	storeTenantIDs := make(map[int64]int64)
	member := utils.IsMember(ctx)
	for _, item := range request.GetGoods() {
		var orderGoods *models.OrderGoods
		var goodsInfo *models.GoodsInfo
		orderGoods, goodsInfo, err = c.orderGoodsCase.convertToModel(ctx, member, item)
		if err != nil {
			return nil, err
		}
		if goodsInfo.TenantStoreID <= 0 {
			return nil, errorsx.InvalidArgument("商品未绑定门店，无法下单")
		}
		if storeOptions[goodsInfo.TenantStoreID] == nil {
			return nil, errorsx.InvalidArgument("门店配送信息不能为空")
		}
		if storeOptions[goodsInfo.TenantStoreID].GetDeliveryTime() == commonv1.OrderDeliveryTime_UNKNOWN_ODT {
			return nil, errorsx.InvalidArgument("门店配送时间不能为空")
		}
		orderGoods.TenantStoreID = goodsInfo.TenantStoreID
		orderGoodsByStore[goodsInfo.TenantStoreID] = append(orderGoodsByStore[goodsInfo.TenantStoreID], orderGoods)
		storeTenantIDs[goodsInfo.TenantStoreID] = goodsInfo.TenantID
	}
	if len(orderGoodsByStore) == 0 {
		return nil, errorsx.InvalidArgument("订单商品信息不能为空")
	}

	orderTrade := &models.OrderTrade{
		TradeNo:    strconv.FormatInt(id.GenSnowflakeID(), 10),
		UserID:     authInfo.UserId,
		PayType:    int32(request.PayType),
		PayChannel: int32(payChannel),
		Status:     tradeStatus,
	}
	orderInfos := make([]*models.OrderInfo, 0, len(orderGoodsByStore))
	orderGoodsMap := make(map[int64][]*models.OrderGoods, len(orderGoodsByStore))
	for storeID, orderGoodsList := range orderGoodsByStore {
		option := storeOptions[storeID]
		orderInfo := &models.OrderInfo{
			TenantID:      storeTenantIDs[storeID],
			TenantStoreID: storeID,
			OrderNo:       strconv.FormatInt(id.GenSnowflakeID(), 10),
			UserID:        authInfo.UserId,
			DeliveryTime:  int32(option.GetDeliveryTime()),
			Status:        orderStatus,
			RefundStatus:  _const.ORDER_REFUND_STATUS_NONE,
			Remark:        option.GetRemark(),
		}
		for _, orderGoods := range orderGoodsList {
			orderInfo.PayMoney += orderGoods.TotalPayPrice
			orderInfo.TotalMoney += orderGoods.TotalPrice
			orderInfo.GoodsNum += orderGoods.Num
		}
		orderTrade.PayMoney += orderInfo.PayMoney
		orderTrade.TotalMoney += orderInfo.TotalMoney
		orderTrade.GoodsNum += orderInfo.GoodsNum
		orderInfos = append(orderInfos, orderInfo)
		orderGoodsMap[storeID] = orderGoodsList
	}
	orderTrade.OrderNum = int64(len(orderInfos))
	err = c.tx.Transaction(ctx, func(ctx context.Context) error {
		err = c.orderTradeCase.Create(ctx, orderTrade)
		if err != nil {
			return err
		}
		err = c.orderAddressCase.createByTrade(ctx, authInfo.UserId, orderTrade.ID, request.GetAddressId())
		if err != nil {
			return err
		}
		for _, orderInfo := range orderInfos {
			orderInfo.TradeID = orderTrade.ID
			err = c.OrderInfoRepository.Create(ctx, orderInfo)
			if err != nil {
				return err
			}
			orderGoodsList := orderGoodsMap[orderInfo.TenantStoreID]
			err = c.orderGoodsCase.createByOrder(ctx, orderInfo, orderGoodsList)
			if err != nil {
				return err
			}
			for _, orderGoods := range orderGoodsList {
				err = c.goodsInfoCase.addSaleNum(ctx, orderGoods.GoodsID, orderGoods.Num)
				if err != nil {
					return err
				}
				err = c.goodsSKUCase.addSaleNum(ctx, orderGoods.SKUCode, orderGoods.Num)
				if err != nil {
					return err
				}
				if request.GetClearCart() {
					err = c.userCartCase.deleteByUserIDAndGoodsIDAndSKUCode(ctx, authInfo.UserId, orderGoods.GoodsID, orderGoods.SKUCode)
					if err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	c.dispatchRecommendOrderEvent(request.PayType, authInfo.UserId, request.GetGoods(), orderTrade.CreatedAt)
	workspaceevent.Publish(ctx, workspaceevent.ReasonOrderChanged, workspaceevent.AreaMetrics, workspaceevent.AreaTodo, workspaceevent.AreaRisk)
	// 为在线支付交易增加超时自动取消任务。
	if orderTrade.Status == _const.ORDER_TRADE_STATUS_PENDING_PAYMENT {
		// 延迟时间使用支付超时配置。
		payTimeout := config.ParsePayTimeout()
		createdAt := orderTrade.CreatedAt.Add(payTimeout)
		nowTime := time.Now()
		countdown := createdAt.Sub(nowTime).Seconds()
		c.scheduleTradeCancellation(orderTrade.ID, orderTrade.UserID, time.Duration(countdown)*time.Second)
	}
	orderIDs := make([]int64, 0, len(orderInfos))
	for _, orderInfo := range orderInfos {
		orderIDs = append(orderIDs, orderInfo.ID)
	}
	return &appv1.CreateOrderInfoResponse{
		TradeId:  orderTrade.ID,
		TradeNo:  orderTrade.TradeNo,
		OrderIds: orderIDs,
	}, nil
}

// DeleteOrderInfo 删除订单
func (c *OrderInfoCase) DeleteOrderInfo(ctx context.Context, id int64) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}

	var orderInfo *models.OrderInfo
	orderInfo, err = c.findByUserIDAndID(ctx, authInfo.UserId, id)
	if err != nil {
		return err
	}
	// 只有已完成、已取消或已退款订单允许用户侧删除。
	if orderInfo.Status != _const.ORDER_INFO_STATUS_COMPLETED &&
		orderInfo.Status != _const.ORDER_INFO_STATUS_CANCELED &&
		orderInfo.RefundStatus != _const.ORDER_REFUND_STATUS_REFUNDED {
		return errorsx.StateConflict(
			fmt.Sprintf("订单状态错误：【%s】", commonv1.OrderInfoStatus_name[orderInfo.Status]),
			"order_info",
			commonv1.OrderInfoStatus(orderInfo.Status).String(),
			"COMPLETED_OIS|CANCELED_OIS|REFUNDED_ORS",
		)
	}
	return c.DeleteByIDs(ctx, []int64{id})
}

// DeleteOrderTrade 删除已关闭交易及其门店子订单。
func (c *OrderInfoCase) DeleteOrderTrade(ctx context.Context, tradeID int64) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}

	var orderTrade *models.OrderTrade
	orderTrade, err = c.orderTradeCase.findByUserIDAndID(ctx, authInfo.UserId, tradeID)
	if err != nil {
		return err
	}
	// 只有已关闭交易允许用户侧删除。
	if orderTrade.Status != _const.ORDER_TRADE_STATUS_CLOSED {
		return errorsx.StateConflict(
			fmt.Sprintf("交易状态错误：【%s】", commonv1.OrderTradeStatus_name[orderTrade.Status]),
			"order_trade",
			commonv1.OrderTradeStatus(orderTrade.Status).String(),
			commonv1.OrderTradeStatus(_const.ORDER_TRADE_STATUS_CLOSED).String(),
		)
	}

	query := c.Query(ctx).OrderInfo
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.TradeID.Eq(tradeID)))
	opts = append(opts, repository.Where(query.UserID.Eq(authInfo.UserId)))
	var orderInfos []*models.OrderInfo
	orderInfos, err = c.List(ctx, opts...)
	if err != nil {
		return err
	}
	orderIDs := make([]int64, 0, len(orderInfos))
	for _, orderInfo := range orderInfos {
		orderIDs = append(orderIDs, orderInfo.ID)
	}

	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		err = c.DeleteByIDs(ctx, orderIDs)
		if err != nil {
			return err
		}
		return c.orderTradeCase.DeleteByIDs(ctx, []int64{tradeID})
	})
}

// CancelOrderInfo 取消订单并回退库存销量
func (c *OrderInfoCase) CancelOrderInfo(ctx context.Context, req *appv1.CancelOrderInfoRequest) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	err = c.cancelOrder(ctx, authInfo.UserId, req)
	if err != nil {
		return err
	}
	return nil
}

// ReceiveOrderInfo 确认收货
func (c *OrderInfoCase) ReceiveOrderInfo(ctx context.Context, req *appv1.ReceiveOrderInfoRequest) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}

	var orderInfo *models.OrderInfo
	orderInfo, err = c.findByUserIDAndID(ctx, authInfo.UserId, req.GetOrderId())
	if err != nil {
		return err
	}
	// 只有已发货订单才能确认收货。
	if orderInfo.Status != _const.ORDER_INFO_STATUS_SHIPPED {
		return errorsx.StateConflict(
			fmt.Sprintf("订单状态错误：【%s】", commonv1.OrderInfoStatus_name[orderInfo.Status]),
			"order_info",
			commonv1.OrderInfoStatus(orderInfo.Status).String(),
			commonv1.OrderInfoStatus(_const.ORDER_INFO_STATUS_SHIPPED).String(),
		)
	}

	orderIDs := []int64{req.GetOrderId()}
	err = c.updateByIDs(ctx, authInfo.UserId, orderIDs, &models.OrderInfo{
		Status: _const.ORDER_INFO_STATUS_WAIT_REVIEW,
	})
	if err != nil {
		return err
	}
	workspaceevent.Publish(ctx, workspaceevent.ReasonOrderChanged, workspaceevent.AreaTodo)
	return nil
}

// 按订单编号和用户编号查询订单
func (c *OrderInfoCase) findByUserIDAndID(ctx context.Context, userID, orderID int64) (*models.OrderInfo, error) {
	query := c.Query(ctx).OrderInfo
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.ID.Eq(orderID)))
	opts = append(opts, repository.Where(query.UserID.Eq(userID)))
	return c.Find(ctx, opts...)
}

// 按订单编号批量更新当前用户的订单
func (c *OrderInfoCase) updateByIDs(ctx context.Context, userID int64, ids []int64, entity *models.OrderInfo) error {
	// 没有待更新订单时，直接返回避免执行空 SQL。
	if len(ids) == 0 {
		return nil
	}
	query := c.Query(ctx).OrderInfo
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.ID.In(ids...)))
	opts = append(opts, repository.Where(query.UserID.Eq(userID)))
	return c.Update(ctx, entity, opts...)
}

// 将订单模型转换为接口响应
func (c *OrderInfoCase) convertToProto(item *models.OrderInfo) *appv1.OrderInfo {
	res := c.mapper.ToDTO(item)
	res.CreatedAt = _time.TimeToTimeString(item.CreatedAt)
	res.UpdatedAt = _time.TimeToTimeString(item.UpdatedAt)
	return res
}

// applyTradeToProto 将交易单字段补充到门店订单响应。
func (c *OrderInfoCase) applyTradeToProto(orderInfo *appv1.OrderInfo, orderTrade *models.OrderTrade) {
	orderInfo.TradeId = orderTrade.ID
	orderInfo.TradeNo = orderTrade.TradeNo
	orderInfo.PayType = commonv1.OrderPayType(orderTrade.PayType)
	orderInfo.PayChannel = commonv1.OrderPayChannel(orderTrade.PayChannel)
	orderInfo.TradeStatus = commonv1.OrderTradeStatus(orderTrade.Status)
}

// buildTradeOrderProto 将交易单及其门店订单转换为聚合订单响应。
func (c *OrderInfoCase) buildTradeOrderProto(
	ctx context.Context,
	orderTrade *models.OrderTrade,
	orderInfos []*models.OrderInfo,
	orderGoodsMap map[int64][]*models.OrderGoods,
) (*appv1.OrderInfo, error) {
	tradeGoods := make([]*models.OrderGoods, 0)
	for _, orderInfo := range orderInfos {
		tradeGoods = append(tradeGoods, orderGoodsMap[orderInfo.ID]...)
	}
	groups, err := c.buildOrderGoodsStoreMap(ctx, map[int64][]*models.OrderGoods{orderTrade.ID: tradeGoods})
	if err != nil {
		return nil, err
	}
	res := &appv1.OrderInfo{
		Id:               orderTrade.ID,
		OrderNo:          orderTrade.TradeNo,
		TradeId:          orderTrade.ID,
		TradeNo:          orderTrade.TradeNo,
		PayMoney:         orderTrade.PayMoney,
		TotalMoney:       orderTrade.TotalMoney,
		PostFee:          orderTrade.PostFee,
		GoodsNum:         orderTrade.GoodsNum,
		PayType:          commonv1.OrderPayType(orderTrade.PayType),
		PayChannel:       commonv1.OrderPayChannel(orderTrade.PayChannel),
		TradeStatus:      commonv1.OrderTradeStatus(orderTrade.Status),
		IsTrade:          true,
		Status:           commonv1.OrderInfoStatus_NOT_STARTED_OIS,
		CreatedAt:        _time.TimeToTimeString(orderTrade.CreatedAt),
		UpdatedAt:        _time.TimeToTimeString(orderTrade.UpdatedAt),
		OrderGoodsStores: groups[orderTrade.ID],
	}
	// 已关闭交易的聚合履约状态用于兼容统一订单展示模型。
	if orderTrade.Status == _const.ORDER_TRADE_STATUS_CLOSED {
		res.Status = commonv1.OrderInfoStatus_CANCELED_OIS
	}
	applyOrderStoreOptions(res.OrderGoodsStores, orderInfos)
	return res, nil
}

// 汇总下单商品信息并生成确认单
func (c *OrderInfoCase) orderBuy(ctx context.Context, member bool, createOrderGoods []*appv1.CreateOrderInfoGoods) (*appv1.ConfirmOrderInfoResponse, error) {
	newOrderGoods := make([]*models.OrderGoods, 0, len(createOrderGoods))
	for _, item := range createOrderGoods {
		var model *models.OrderGoods
		var goodsInfo *models.GoodsInfo
		var err error
		model, goodsInfo, err = c.orderGoodsCase.convertToModel(ctx, member, item)
		if err != nil {
			return nil, err
		}
		// 商品必须归属明确门店，确认单按商品所属门店分组展示。
		if goodsInfo.TenantStoreID <= 0 {
			return nil, errorsx.InvalidArgument("商品未绑定门店，无法下单")
		}
		model.TenantStoreID = goodsInfo.TenantStoreID
		newOrderGoods = append(newOrderGoods, model)
	}
	orderGoodsStoreMap, err := c.buildOrderGoodsStoreMap(ctx, map[int64][]*models.OrderGoods{
		0: newOrderGoods,
	})
	if err != nil {
		return nil, err
	}
	orderGoodsStores := orderGoodsStoreMap[0]

	var summary appv1.OrderSummary
	for _, orderStore := range orderGoodsStores {
		for _, orderGoods := range orderStore.Goods {
			summary.PayMoney += orderGoods.TotalPayPrice
			summary.TotalMoney += orderGoods.TotalPrice
			summary.GoodsNum += orderGoods.Num
		}
	}
	// 当前版本统一免运费
	summary.PostFee = 0
	return &appv1.ConfirmOrderInfoResponse{
		OrderGoodsStores: orderGoodsStores,
		Summary:          &summary,
	}, nil
}

// buildOrderGoodsStoreMap 按订单和门店构建商品分组映射。
func (c *OrderInfoCase) buildOrderGoodsStoreMap(ctx context.Context, orderGoodsMap map[int64][]*models.OrderGoods) (map[int64][]*appv1.OrderGoodsStore, error) {
	res := make(map[int64][]*appv1.OrderGoodsStore, len(orderGoodsMap))
	storeIDs := make([]int64, 0)
	storeIDSet := make(map[int64]bool)
	for orderID, orderGoodsList := range orderGoodsMap {
		orderStoreMap := make(map[int64]*appv1.OrderGoodsStore)
		orderGoodsStores := make([]*appv1.OrderGoodsStore, 0)
		for _, orderGoods := range orderGoodsList {
			orderStore := orderStoreMap[orderGoods.TenantStoreID]
			// 门店首次命中时，初始化对应的订单商品分组。
			if orderStore == nil {
				orderStore = &appv1.OrderGoodsStore{
					Store:   &appv1.TenantStore{Id: orderGoods.TenantStoreID},
					Goods:   make([]*appv1.OrderGoods, 0),
					Summary: &appv1.OrderSummary{},
				}
				orderStoreMap[orderGoods.TenantStoreID] = orderStore
				orderGoodsStores = append(orderGoodsStores, orderStore)
				if !storeIDSet[orderGoods.TenantStoreID] {
					storeIDSet[orderGoods.TenantStoreID] = true
					storeIDs = append(storeIDs, orderGoods.TenantStoreID)
				}
			}
			orderStore.Goods = append(orderStore.Goods, c.orderGoodsCase.toOrderGoods(orderGoods))
			orderStore.Summary.PayMoney += orderGoods.TotalPayPrice
			orderStore.Summary.TotalMoney += orderGoods.TotalPrice
			orderStore.Summary.GoodsNum += orderGoods.Num
		}
		res[orderID] = orderGoodsStores
	}

	// 批量补齐订单响应的门店展示信息，避免按商品逐条查询门店。
	tenantStoreMap, err := c.tenantStoreCase.GetTenantStoreMapByIDs(ctx, storeIDs)
	if err != nil {
		return nil, err
	}
	for _, orderGoodsStores := range res {
		for _, orderStore := range orderGoodsStores {
			// 门店资料仍存在时，补齐订单商品分组的展示信息。
			tenantStore := tenantStoreMap[orderStore.Store.Id]
			if tenantStore != nil {
				orderStore.Store.Name = tenantStore.Name
				orderStore.Store.Logo = tenantStore.Logo
			}
		}
	}
	return res, nil
}

// applyOrderStoreOptions 将门店子订单信息补充到对应门店分组。
func applyOrderStoreOptions(orderGoodsStores []*appv1.OrderGoodsStore, orderInfos []*models.OrderInfo) {
	orderInfoMap := make(map[int64]*models.OrderInfo, len(orderInfos))
	for _, orderInfo := range orderInfos {
		orderInfoMap[orderInfo.TenantStoreID] = orderInfo
	}
	for _, orderGoodsStore := range orderGoodsStores {
		orderInfo := orderInfoMap[orderGoodsStore.GetStore().GetId()]
		if orderInfo == nil {
			continue
		}
		orderGoodsStore.OrderId = orderInfo.ID
		orderGoodsStore.OrderNo = orderInfo.OrderNo
		orderGoodsStore.DeliveryTime = commonv1.OrderDeliveryTime(orderInfo.DeliveryTime)
		orderGoodsStore.Remark = orderInfo.Remark
	}
}

// dispatchRecommendOrderEvent 根据已落库订单事实回写推荐下单事件。
func (c *OrderInfoCase) dispatchRecommendOrderEvent(payType commonv1.OrderPayType, userID int64, goodsList []*appv1.CreateOrderInfoGoods, eventTime time.Time) {
	// 主体编号非法或订单商品为空时，无法构建可归因的推荐下单事件。
	if userID <= 0 || len(goodsList) == 0 {
		return
	}

	eventTypeList := make([]commonv1.RecommendEventType, 0, 2)
	eventTypeList = append(eventTypeList, commonv1.RecommendEventType(_const.RECOMMEND_EVENT_TYPE_ORDER_CREATE))
	// 货到付款订单在创建时同步回写支付行为。
	if payType == commonv1.OrderPayType(_const.ORDER_PAY_TYPE_CASH_ON_DELIVERY) {
		eventTypeList = append(eventTypeList, commonv1.RecommendEventType(_const.RECOMMEND_EVENT_TYPE_ORDER_PAY))
	}

	for _, eventType := range eventTypeList {
		for _, goodsItem := range goodsList {
			// 非法商品项直接跳过，避免把脏数据写入推荐链路。
			if goodsItem == nil || goodsItem.GetGoodsId() <= 0 {
				continue
			}
			recommendContext := goodsItem.GetRecommendContext()
			// 下单项未携带推荐上下文时，回退到空上下文，避免后续空指针。
			if recommendContext == nil {
				recommendContext = &appv1.RecommendContext{}
			}

			orderEventReport := &appv1.RecommendEventReportRequest{
				EventType: eventType,
				RecommendContext: &appv1.RecommendEventContext{
					Scene:     recommendContext.GetScene(),
					RequestId: recommendContext.GetRequestId(),
				},
				Items: []*appv1.RecommendEventItem{
					{
						GoodsId:  goodsItem.GetGoodsId(),
						GoodsNum: goodsItem.GetNum(),
						Position: recommendContext.GetPosition(),
					},
				},
			}
			// 订单创建事务提交成功后，再按落库事实回写推荐下单事件。
			queue.DispatchRecommendEvent(&dto.RecommendActor{
				ActorType: commonv1.RecommendActorType(_const.RECOMMEND_ACTOR_TYPE_USER),
				ActorID:   userID,
			}, orderEventReport, eventTime)
		}
	}
}

// scheduleTradeCancellation 注册交易超时取消任务，并在临时失败时按未支付状态重试。
func (c *OrderInfoCase) scheduleTradeCancellation(tradeID, userID int64, delay time.Duration) {
	if delay < 0 {
		delay = 0
	}
	c.orderSchedulerCase.AddSchedule(tradeID, delay, func() {
		err := c.cancelOrder(context.Background(), userID, &appv1.CancelOrderInfoRequest{TradeId: tradeID})
		if err == nil {
			return
		}
		log.Error(fmt.Sprintf("CancelOrder trade %d failed: %v", tradeID, err))
		var orderTrade *models.OrderTrade
		orderTrade, err = c.orderTradeCase.findByUserIDAndID(context.Background(), userID, tradeID)
		if err != nil {
			log.Error(fmt.Sprintf("FindOrderTrade trade %d failed: %v", tradeID, err))
			return
		}
		// 交易仍处于未支付状态时，延迟一分钟重试渠道关单和本地取消。
		if orderTrade.Status == _const.ORDER_TRADE_STATUS_PENDING_PAYMENT || orderTrade.Status == _const.ORDER_TRADE_STATUS_PAYING {
			c.scheduleTradeCancellation(tradeID, userID, time.Minute)
		}
	})
}

// successfulRefundMoney 汇总当前门店订单已经成功退款的金额。
func (c *OrderInfoCase) successfulRefundMoney(ctx context.Context, orderID int64) (int64, error) {
	query := c.orderRefundCase.Query(ctx).OrderRefund
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.OrderID.Eq(orderID)))
	orderRefunds, err := c.orderRefundCase.List(ctx, opts...)
	if err != nil {
		return 0, err
	}
	var refundMoney int64
	for _, orderRefund := range orderRefunds {
		if orderRefund.RefundState != appv1.RefundResource_SUCCESS.String() {
			continue
		}
		var amount struct {
			Refund int64 `json:"refund"`
		}
		err = json.Unmarshal([]byte(orderRefund.Amount), &amount)
		if err != nil {
			return 0, err
		}
		refundMoney += amount.Refund
	}
	return refundMoney, nil
}

// cancelOrder 内部执行交易取消并回退全部门店订单库存销量。
func (c *OrderInfoCase) cancelOrder(ctx context.Context, userID int64, req *appv1.CancelOrderInfoRequest) error {
	orderTrade, err := c.orderTradeCase.findByUserIDAndID(ctx, userID, req.GetTradeId())
	if err != nil {
		return err
	}
	// 只有尚未完成支付的交易才能继续取消。
	if orderTrade.Status != _const.ORDER_TRADE_STATUS_PENDING_PAYMENT && orderTrade.Status != _const.ORDER_TRADE_STATUS_PAYING {
		return errorsx.StateConflict(
			fmt.Sprintf("交易状态错误：【%s】", commonv1.OrderTradeStatus_name[orderTrade.Status]),
			"order_trade",
			commonv1.OrderTradeStatus(orderTrade.Status).String(),
			"PENDING_PAYMENT_OTS|PAYING_OTS",
		)
	}

	// 微信在线支付交易在取消前先补查并关闭支付单，避免回调延迟或并发支付时误取消。
	if commonv1.OrderPayType(orderTrade.PayType) == commonv1.OrderPayType(_const.ORDER_PAY_TYPE_ONLINE_PAY) &&
		commonv1.OrderPayChannel(orderTrade.PayChannel) == commonv1.OrderPayChannel(_const.ORDER_PAY_CHANNEL_WX_PAY) {
		var paymentResource *appv1.PaymentResource
		paymentResource, err = c.wxPayCase.QueryOrderByOutTradeNo(orderTrade.TradeNo)
		if err != nil {
			// 尚未成功创建微信预支付单时，远端没有可关闭资源，允许继续取消本地交易。
			if !wx.IsPayOrderNotExist(err) {
				return err
			}
			paymentResource = nil
		}
		if paymentResource != nil {
			// 微信侧已确认支付成功或已经进入退款时，拒绝取消交易。
			if paymentResource.GetTradeState() == appv1.PaymentResource_SUCCESS ||
				paymentResource.GetTradeState() == appv1.PaymentResource_REFUND {
				// 支付成功但本地尚未落账时，先同步支付结果。
				if paymentResource.GetTradeState() == appv1.PaymentResource_SUCCESS {
					err = c.payCase.PaySuccess(ctx, orderTrade, paymentResource)
					if err != nil {
						return err
					}
				}
				return errorsx.StateConflict(
					"交易已支付，无法取消",
					"order_trade",
					paymentResource.GetTradeState().String(),
					appv1.PaymentResource_NOTPAY.String(),
				)
			}
			// 尚未关闭的微信支付单必须先关单，关单成功后才允许回退本地库存。
			if paymentResource.GetTradeState() != appv1.PaymentResource_CLOSED &&
				paymentResource.GetTradeState() != appv1.PaymentResource_REVOKED {
				closeErr := c.wxPayCase.CloseOrderByOutTradeNo(orderTrade.TradeNo)
				if closeErr != nil {
					// 关单与支付并发时重新查询，若已支付则同步本地状态并拒绝取消。
					paymentResource, err = c.wxPayCase.QueryOrderByOutTradeNo(orderTrade.TradeNo)
					if err == nil && paymentResource.GetTradeState() == appv1.PaymentResource_SUCCESS {
						err = c.payCase.PaySuccess(ctx, orderTrade, paymentResource)
						if err != nil {
							return err
						}
						return errorsx.StateConflict(
							"交易已支付，无法取消",
							"order_trade",
							appv1.PaymentResource_SUCCESS.String(),
							appv1.PaymentResource_NOTPAY.String(),
						)
					}
					return closeErr
				}
			}
		}
	}
	query := c.Query(ctx).OrderInfo
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.TradeID.Eq(orderTrade.ID)))
	opts = append(opts, repository.Where(query.UserID.Eq(userID)))
	var orderInfos []*models.OrderInfo
	orderInfos, err = c.List(ctx, opts...)
	if err != nil {
		return err
	}
	orderIDs := make([]int64, 0, len(orderInfos))
	for _, orderInfo := range orderInfos {
		orderIDs = append(orderIDs, orderInfo.ID)
	}
	err = c.tx.Transaction(ctx, func(ctx context.Context) error {
		// 条件更新先抢占交易关闭权，确保支付成功与取消只有一个状态流转能够提交。
		err = c.markTradeClosed(ctx, orderTrade)
		if err != nil {
			return err
		}
		query := c.orderGoodsCase.Query(ctx).OrderGoods
		var orderGoodsList []*models.OrderGoods
		opts := make([]repository.QueryOption, 0, 1)
		opts = append(opts, repository.Where(query.OrderID.In(orderIDs...)))
		orderGoodsList, err = c.orderGoodsCase.List(ctx, opts...)
		if err != nil {
			return err
		}
		for _, orderGoods := range orderGoodsList {
			// 订单取消后恢复库存并回退销量
			err = c.goodsInfoCase.subSaleNum(ctx, orderGoods.GoodsID, orderGoods.Num)
			if err != nil {
				return err
			}
			err = c.goodsSKUCase.subSaleNum(ctx, orderGoods.SKUCode, orderGoods.Num)
			if err != nil {
				return err
			}
		}
		// 保存订单取消记录，便于订单详情展示取消时间
		err = c.orderCancelCase.Create(ctx, &models.OrderCancel{
			TradeID: orderTrade.ID,
			Reason:  int32(req.GetReason()),
		})
		if err != nil {
			return err
		}
		err = c.updateByIDs(ctx, userID, orderIDs, &models.OrderInfo{Status: _const.ORDER_INFO_STATUS_CANCELED})
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	c.orderSchedulerCase.DeleteScheduled(orderTrade.ID)
	workspaceevent.Publish(ctx, workspaceevent.ReasonOrderChanged, workspaceevent.AreaMetrics, workspaceevent.AreaTodo, workspaceevent.AreaRisk)
	return nil
}

// markTradeClosed 通过状态条件更新抢占交易取消权。
func (c *OrderInfoCase) markTradeClosed(ctx context.Context, orderTrade *models.OrderTrade) error {
	query := c.orderTradeCase.Query(ctx).OrderTrade
	result, err := query.WithContext(ctx).
		Where(
			query.ID.Eq(orderTrade.ID),
			query.UserID.Eq(orderTrade.UserID),
			query.Status.In(_const.ORDER_TRADE_STATUS_PENDING_PAYMENT, _const.ORDER_TRADE_STATUS_PAYING),
		).
		Update(query.Status, _const.ORDER_TRADE_STATUS_CLOSED)
	if err != nil {
		return err
	}
	if result.RowsAffected > 0 {
		return nil
	}
	var currentTrade *models.OrderTrade
	currentTrade, err = c.orderTradeCase.findByUserIDAndID(ctx, orderTrade.UserID, orderTrade.ID)
	if err != nil {
		return err
	}
	return errorsx.StateConflict(
		fmt.Sprintf("交易状态错误：【%s】", commonv1.OrderTradeStatus_name[currentTrade.Status]),
		"order_trade",
		commonv1.OrderTradeStatus(currentTrade.Status).String(),
		"PENDING_PAYMENT_OTS|PAYING_OTS",
	)
}

// isAdminRoleCode 判断当前登录角色是否属于后台管理角色。
func isAdminRoleCode(roleCode string) bool {
	return roleCode == _const.BASE_ROLE_CODE_SUPER ||
		roleCode == _const.BASE_ROLE_CODE_ADMIN ||
		roleCode == _const.BASE_ROLE_CODE_TENANT
}

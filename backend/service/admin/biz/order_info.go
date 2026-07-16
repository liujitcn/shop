package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/liujitcn/go-utils/id"
	"github.com/liujitcn/go-utils/mapper"
	_slice "github.com/liujitcn/go-utils/slice"
	_string "github.com/liujitcn/go-utils/string"
	_time "github.com/liujitcn/go-utils/time"
	"github.com/liujitcn/go-utils/trans"
	"github.com/liujitcn/gorm-kit/repository"
	authnEngine "github.com/liujitcn/kratos-kit/auth/authn/engine"
	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"
	wxPayCore "github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/refunddomestic"
	"gorm.io/gorm/clause"

	adminv1 "shop/api/gen/go/admin/v1"
	appv1 "shop/api/gen/go/app/v1"
	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/biz"
	"shop/pkg/config"
	_const "shop/pkg/const"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"shop/pkg/workspaceevent"
	"shop/pkg/wx"
)

// OrderInfoCase 订单业务实例
type OrderInfoCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.OrderInfoRepository
	orderTradeRepo     *data.OrderTradeRepository
	orderAddressCase   *OrderAddressCase
	orderCancelCase    *OrderCancelCase
	orderGoodsCase     *OrderGoodsCase
	orderLogisticsCase *OrderLogisticsCase
	orderPaymentCase   *OrderPaymentCase
	orderRefundCase    *OrderRefundCase
	baseUserCase       *BaseUserCase
	baseDictItemCase   *BaseDictItemCase
	wxPayCase          *wx.WxPayCase
	mapper             *mapper.CopierMapper[adminv1.OrderInfo, models.OrderInfo]
}

// NewOrderInfoCase 创建订单业务实例
func NewOrderInfoCase(baseCase *biz.BaseCase, tx data.Transaction, orderAddressCase *OrderAddressCase, orderInfoRepo *data.OrderInfoRepository, orderTradeRepo *data.OrderTradeRepository, orderCancelCase *OrderCancelCase, orderGoodsCase *OrderGoodsCase, orderLogisticsCase *OrderLogisticsCase, orderPaymentCase *OrderPaymentCase, orderRefundCase *OrderRefundCase, baseUserCase *BaseUserCase, baseDictItemCase *BaseDictItemCase, wxPayCase *wx.WxPayCase) *OrderInfoCase {
	return &OrderInfoCase{
		BaseCase:            baseCase,
		tx:                  tx,
		OrderInfoRepository: orderInfoRepo,
		orderTradeRepo:      orderTradeRepo,
		orderAddressCase:    orderAddressCase,
		orderCancelCase:     orderCancelCase,
		orderGoodsCase:      orderGoodsCase,
		orderLogisticsCase:  orderLogisticsCase,
		orderPaymentCase:    orderPaymentCase,
		orderRefundCase:     orderRefundCase,
		baseUserCase:        baseUserCase,
		baseDictItemCase:    baseDictItemCase,
		wxPayCase:           wxPayCase,
		mapper:              mapper.NewCopierMapper[adminv1.OrderInfo, models.OrderInfo](),
	}
}

// PageOrderInfos 分页查询订单
func (c *OrderInfoCase) PageOrderInfos(ctx context.Context, req *adminv1.PageOrderInfosRequest) (*adminv1.PageOrderInfosResponse, error) {
	query := c.Query(ctx).OrderInfo
	opts := make([]repository.QueryOption, 0, 9)
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	var err error
	// 支付方式、支付渠道和交易支付状态均归属交易单，先收敛可见的交易单编号。
	if req.PayType != nil || req.PayChannel != nil || req.TradeStatus != nil {
		tradeQuery := c.orderTradeRepo.Query(ctx).OrderTrade
		tradeOpts := make([]repository.QueryOption, 0, 3)
		if req.PayType != nil {
			tradeOpts = append(tradeOpts, repository.Where(tradeQuery.PayType.Eq(int32(req.GetPayType()))))
		}
		if req.PayChannel != nil {
			tradeOpts = append(tradeOpts, repository.Where(tradeQuery.PayChannel.Eq(int32(req.GetPayChannel()))))
		}
		if req.TradeStatus != nil {
			tradeOpts = append(tradeOpts, repository.Where(tradeQuery.Status.Eq(int32(req.GetTradeStatus()))))
		}
		var orderTrades []*models.OrderTrade
		orderTrades, err = c.orderTradeRepo.List(ctx, tradeOpts...)
		if err != nil {
			return nil, err
		}
		if len(orderTrades) == 0 {
			return &adminv1.PageOrderInfosResponse{}, nil
		}
		tradeIDs := make([]int64, 0, len(orderTrades))
		for _, orderTrade := range orderTrades {
			tradeIDs = append(tradeIDs, orderTrade.ID)
		}
		opts = append(opts, repository.Where(query.TradeID.In(tradeIDs...)))
	}
	if req.TenantId != nil && req.GetTenantId() > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(req.GetTenantId())))
	}
	if req.TenantStoreId != nil && req.GetTenantStoreId() > 0 {
		opts = append(opts, repository.Where(query.TenantStoreID.Eq(req.GetTenantStoreId())))
	}
	// 传入用户编号时，按用户过滤订单。
	if req.GetUserId() > 0 {
		opts = append(opts, repository.Where(query.UserID.Eq(req.GetUserId())))
	}
	// 传入订单号关键字时，按订单号模糊匹配。
	if req.GetOrderNo() != "" {
		opts = append(opts, repository.Where(query.OrderNo.Like("%"+req.GetOrderNo()+"%")))
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	if req.RefundStatus != nil {
		opts = append(opts, repository.Where(query.RefundStatus.Eq(int32(req.GetRefundStatus()))))
	}
	// 传入时间范围时，按创建时间区间过滤订单。
	if len(req.GetCreatedAt()) == 2 {
		startTime := _time.StringTimeToTime(req.GetCreatedAt()[0])
		endTime := _time.StringTimeToTime(req.GetCreatedAt()[1])
		// 开始时间可解析时，追加起始时间条件。
		if startTime != nil {
			opts = append(opts, repository.Where(query.CreatedAt.Gte(*startTime)))
		}
		// 结束时间可解析时，追加结束时间条件。
		if endTime != nil {
			endAt := endTime.Add(24 * time.Hour)
			opts = append(opts, repository.Where(query.CreatedAt.Lt(endAt)))
		}
	}
	var list []*models.OrderInfo
	var total int64
	list, total, err = c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	var tradeMap map[int64]*models.OrderTrade
	tradeMap, err = c.getOrderTradeMap(ctx, list)
	if err != nil {
		return nil, err
	}

	var userMap map[int64]*models.BaseUser
	userMap, err = c.getOrderUserMap(ctx, list)
	if err != nil {
		return nil, err
	}

	resList := make([]*adminv1.OrderInfo, 0, len(list))
	for _, item := range list {
		orderInfo := c.mapper.ToDTO(item)
		c.applyOrderTrade(orderInfo, tradeMap[item.TradeID])
		// 命中用户信息时，补齐下单用户昵称。
		if user, ok := userMap[item.UserID]; ok {
			orderInfo.NickName = user.NickName
		}
		resList = append(resList, orderInfo)
	}
	return &adminv1.PageOrderInfosResponse{OrderInfos: resList, Total: int32(total)}, nil
}

// GetOrderInfo 获取订单
func (c *OrderInfoCase) GetOrderInfo(ctx context.Context, id int64) (*adminv1.OrderInfoResponse, error) {
	orderInfo, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	var orderTrade *models.OrderTrade
	orderTrade, err = c.findOrderTrade(ctx, orderInfo.TradeID)
	if err != nil {
		return nil, err
	}
	protoOrderInfo := c.mapper.ToDTO(orderInfo)
	c.applyOrderTrade(protoOrderInfo, orderTrade)

	res := &adminv1.OrderInfoResponse{
		Order: protoOrderInfo,
	}
	// 未完成支付的交易单才返回倒计时。
	if orderTrade.Status == _const.ORDER_TRADE_STATUS_PENDING_PAYMENT || orderTrade.Status == _const.ORDER_TRADE_STATUS_PAYING {
		res.Countdown = float32(time.Until(orderTrade.CreatedAt.Add(config.ParsePayTimeout())).Seconds())
	}

	var queryCtx context.Context
	queryCtx, err = c.orderUserQueryContext(ctx)
	if err != nil {
		return nil, err
	}
	var baseUser *models.BaseUser
	baseUser, err = c.baseUserCase.FindByID(queryCtx, orderInfo.UserID)
	// 用户存在时，补齐订单上的下单用户昵称。
	if err == nil {
		res.Order.NickName = baseUser.NickName
	}

	res.Address, err = c.orderAddressCase.FindFromByTradeID(ctx, orderInfo.TradeID)
	if err != nil {
		return nil, err
	}
	res.Goods, err = c.orderGoodsCase.FindFromByOrderID(ctx, orderInfo.ID)
	if err != nil {
		return nil, err
	}
	// 取消、物流、支付、退款记录不是所有订单状态都会生成，详情页按存在情况补齐。
	res.Cancel, err = c.orderCancelCase.FindFromByTradeID(ctx, orderInfo.TradeID)
	if err != nil {
		return nil, err
	}
	res.Logistics, err = c.orderLogisticsCase.FindFromByOrderID(ctx, orderInfo.ID)
	if err != nil {
		return nil, err
	}
	res.Payment, err = c.orderPaymentCase.FindFromByTradeID(ctx, orderInfo.TradeID)
	if err != nil {
		return nil, err
	}
	res.Refund, err = c.orderRefundCase.FindFromByOrderID(ctx, orderInfo.ID)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// GetOrderInfoRefund 获取订单退款信息
func (c *OrderInfoCase) GetOrderInfoRefund(ctx context.Context, id int64) (*adminv1.OrderInfoRefundResponse, error) {
	orderInfo, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	res := &adminv1.OrderInfoRefundResponse{}
	res.Payment, err = c.orderPaymentCase.FindFromByTradeID(ctx, orderInfo.TradeID)
	if err != nil {
		return nil, err
	}
	res.Refund, err = c.orderRefundCase.FindFromByOrderID(ctx, orderInfo.ID)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// RefundOrderInfo 退款订单
func (c *OrderInfoCase) RefundOrderInfo(ctx context.Context, req *adminv1.RefundOrderInfoRequest) error {
	orderInfo, err := c.FindByID(ctx, req.GetOrderId())
	if err != nil {
		return err
	}
	// 只有已进入履约的门店订单才允许后台发起退款。
	if orderInfo.Status != _const.ORDER_INFO_STATUS_WAIT_SHIPMENT &&
		orderInfo.Status != _const.ORDER_INFO_STATUS_SHIPPED &&
		orderInfo.Status != _const.ORDER_INFO_STATUS_WAIT_REVIEW &&
		orderInfo.Status != _const.ORDER_INFO_STATUS_COMPLETED {
		return errorsx.StateConflict(
			fmt.Sprintf("订单履约状态错误：【%s】", commonv1.OrderInfoStatus_name[orderInfo.Status]),
			"order_info",
			commonv1.OrderInfoStatus(orderInfo.Status).String(),
			"WAIT_SHIPMENT_OIS|SHIPPED_OIS|WAIT_REVIEW_OIS|COMPLETED_OIS",
		)
	}
	if orderInfo.RefundStatus == _const.ORDER_REFUND_STATUS_PROCESSING || orderInfo.RefundStatus == _const.ORDER_REFUND_STATUS_REFUNDED {
		return errorsx.StateConflict(
			"当前订单退款状态不允许再次退款",
			"order_info",
			commonv1.OrderRefundStatus(orderInfo.RefundStatus).String(),
			"NONE_ORS|PARTIAL_REFUND_ORS|CLOSED_OR_FAILED_ORS",
		)
	}
	if req.GetRefundMoney() <= 0 {
		return errorsx.InvalidArgument("退款金额必须大于 0")
	}
	var orderTrade *models.OrderTrade
	orderTrade, err = c.findOrderTrade(ctx, orderInfo.TradeID)
	if err != nil {
		return err
	}
	var refundedMoney int64
	refundedMoney, err = c.successfulRefundMoney(ctx, orderInfo.ID)
	if err != nil {
		return err
	}
	if req.GetRefundMoney() > orderInfo.PayMoney-refundedMoney {
		return errorsx.InvalidArgument("退款金额超过当前门店订单可退金额")
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
			"refund":       req.GetRefundMoney(),
			"payer_total":  orderTrade.PayMoney,
			"payer_refund": req.GetRefundMoney(),
		}),
		Status: _const.ORDER_BILL_STATUS_NO_CHECK,
	}
	err = c.claimOrderRefund(ctx, orderInfo, orderRefund)
	if err != nil {
		return err
	}

	// 微信在线支付订单需要先走微信退款单创建流程。
	if commonv1.OrderPayType(orderTrade.PayType) == commonv1.OrderPayType(_const.ORDER_PAY_TYPE_ONLINE_PAY) && commonv1.OrderPayChannel(orderTrade.PayChannel) == commonv1.OrderPayChannel(_const.ORDER_PAY_CHANNEL_WX_PAY) {
		reason := commonv1.OrderRefundReason_name[int32(req.GetReason())]
		var refund *refunddomestic.Refund
		refund, err = c.wxPayCase.Refund(refunddomestic.CreateRequest{
			OutTradeNo:  trans.String(orderTrade.TradeNo),
			OutRefundNo: trans.String(orderRefund.RefundNo),
			Reason:      trans.String(reason),
			Amount: &refunddomestic.AmountReq{
				Total:    trans.Int64(orderTrade.PayMoney),
				Refund:   trans.Int64(req.GetRefundMoney()),
				Currency: trans.String("CNY"),
			},
		})
		if err != nil {
			// 微信已经明确拒绝退款时，释放当前门店订单的退款占用，允许修正后重试。
			if _, ok := errors.AsType[*wxPayCore.APIError](err); ok {
				orderRefund.RefundState = appv1.RefundResource_ABNORMAL.String()
				stateErr := c.tx.Transaction(ctx, func(ctx context.Context) error {
					persistErr := c.orderRefundCase.UpdateByID(ctx, orderRefund)
					if persistErr != nil {
						return persistErr
					}
					return c.UpdateByID(ctx, &models.OrderInfo{
						ID:           orderInfo.ID,
						RefundStatus: _const.ORDER_REFUND_STATUS_CLOSED_OR_FAILED,
					})
				})
				if stateErr != nil {
					return stateErr
				}
			}
			return err
		}
		orderRefund.TradeNo = trans.StringValue(refund.OutTradeNo)
		orderRefund.ThirdOrderNo = trans.StringValue(refund.TransactionId)
		orderRefund.ThirdRefundNo = trans.StringValue(refund.RefundId)
		// 微信返回退款渠道时，补齐退款渠道字段。
		if refund.Channel != nil && refund.Channel.Ptr() != nil {
			orderRefund.Channel = string(*refund.Channel.Ptr())
		}
		orderRefund.UserReceivedAccount = trans.StringValue(refund.UserReceivedAccount)
		orderRefund.CreateTime = trans.TimeValue(refund.CreateTime)
		orderRefund.SuccessTime = trans.TimeValue(refund.SuccessTime)
		// 微信返回退款状态时，补齐退款状态字段。
		if refund.Status != nil && refund.Status.Ptr() != nil {
			orderRefund.RefundState = string(*refund.Status.Ptr())
		}
		// 微信返回资金账户时，补齐资金账户字段。
		if refund.FundsAccount != nil && refund.FundsAccount.Ptr() != nil {
			orderRefund.FundsAccount = string(*refund.FundsAccount.Ptr())
		}
		orderRefund.Amount = _string.ConvertAnyToJsonString(refund.Amount)
	} else {
		now := time.Now()
		orderRefund.CreateTime = now
		orderRefund.SuccessTime = now
		orderRefund.RefundState = appv1.RefundResource_SUCCESS.String()
	}

	err = c.tx.Transaction(ctx, func(ctx context.Context) error {
		err = c.orderRefundCase.UpdateByID(ctx, orderRefund)
		if err != nil {
			return err
		}
		// 渠道已返回最终结果时直接汇总，否则标记当前门店订单正在退款。
		switch orderRefund.RefundState {
		case appv1.RefundResource_SUCCESS.String():
			return c.refreshRefundStatuses(ctx, orderTrade, orderInfo)
		case appv1.RefundResource_CLOSED.String(), appv1.RefundResource_ABNORMAL.String():
			return c.UpdateByID(ctx, &models.OrderInfo{ID: orderInfo.ID, RefundStatus: _const.ORDER_REFUND_STATUS_CLOSED_OR_FAILED})
		default:
			return c.UpdateByID(ctx, &models.OrderInfo{ID: orderInfo.ID, RefundStatus: _const.ORDER_REFUND_STATUS_PROCESSING})
		}
	})
	if err != nil {
		return err
	}
	workspaceevent.Publish(ctx, workspaceevent.ReasonOrderChanged, workspaceevent.AreaTodo, workspaceevent.AreaMetrics)
	return nil
}

// claimOrderRefund 抢占门店订单退款权并创建待处理退款记录。
func (c *OrderInfoCase) claimOrderRefund(ctx context.Context, orderInfo *models.OrderInfo, orderRefund *models.OrderRefund) error {
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		query := c.Query(ctx).OrderInfo
		result, err := query.WithContext(ctx).
			Where(
				query.ID.Eq(orderInfo.ID),
				query.Status.In(
					_const.ORDER_INFO_STATUS_WAIT_SHIPMENT,
					_const.ORDER_INFO_STATUS_SHIPPED,
					_const.ORDER_INFO_STATUS_WAIT_REVIEW,
					_const.ORDER_INFO_STATUS_COMPLETED,
				),
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
				"当前订单退款状态不允许再次退款",
				"order_info",
				commonv1.OrderRefundStatus(orderInfo.RefundStatus).String(),
				"NONE_ORS|PARTIAL_REFUND_ORS|CLOSED_OR_FAILED_ORS",
			)
		}
		return c.orderRefundCase.Create(ctx, orderRefund)
	})
}

// GetOrderInfoShipment 获取订单发货信息
func (c *OrderInfoCase) GetOrderInfoShipment(ctx context.Context, id int64) (*adminv1.OrderInfoShipmentForm, error) {
	orderInfo, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	res := &adminv1.OrderInfoShipmentForm{}
	res.Address, err = c.orderAddressCase.FindFromByTradeID(ctx, orderInfo.TradeID)
	if err != nil {
		return nil, err
	}
	res.Goods, err = c.orderGoodsCase.FindFromByOrderID(ctx, orderInfo.ID)
	if err != nil {
		return nil, err
	}
	// 已发货、待评价或已完成订单需要补充物流信息。
	if orderInfo.Status == _const.ORDER_INFO_STATUS_SHIPPED || orderInfo.Status == _const.ORDER_INFO_STATUS_WAIT_REVIEW || orderInfo.Status == _const.ORDER_INFO_STATUS_COMPLETED {
		res.Logistics, err = c.orderLogisticsCase.FindFromByOrderID(ctx, orderInfo.ID)
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

// ShipOrderInfo 发货订单
func (c *OrderInfoCase) ShipOrderInfo(ctx context.Context, req *adminv1.ShipOrderInfoRequest) error {
	orderInfo, err := c.FindByID(ctx, req.GetOrderId())
	if err != nil {
		return err
	}
	// 只有待发货的门店订单才能继续发货。
	if orderInfo.Status != _const.ORDER_INFO_STATUS_WAIT_SHIPMENT {
		return errorsx.StateConflict(
			fmt.Sprintf("订单履约状态错误：【%s】", commonv1.OrderInfoStatus_name[orderInfo.Status]),
			"order_info",
			commonv1.OrderInfoStatus(orderInfo.Status).String(),
			commonv1.OrderInfoStatus(_const.ORDER_INFO_STATUS_WAIT_SHIPMENT).String(),
		)
	}
	var orderTrade *models.OrderTrade
	orderTrade, err = c.findOrderTrade(ctx, orderInfo.TradeID)
	if err != nil {
		return err
	}
	if orderTrade.Status != _const.ORDER_TRADE_STATUS_PAID &&
		orderTrade.Status != _const.ORDER_TRADE_STATUS_CASH_ON_DELIVERY &&
		orderTrade.Status != _const.ORDER_TRADE_STATUS_PARTIAL_REFUND {
		return errorsx.StateConflict(
			fmt.Sprintf("交易支付状态错误：【%s】", commonv1.OrderTradeStatus_name[orderTrade.Status]),
			"order_trade",
			commonv1.OrderTradeStatus(orderTrade.Status).String(),
			"PAID_OTS|CASH_ON_DELIVERY_OTS|PARTIAL_REFUND_OTS",
		)
	}

	err = c.tx.Transaction(ctx, func(ctx context.Context) error {
		err = c.orderLogisticsCase.Create(ctx, &models.OrderLogistics{
			TenantID:      orderInfo.TenantID,
			TenantStoreID: orderInfo.TenantStoreID,
			OrderID:       orderInfo.ID,
			Name:          req.GetName(),
			No:            req.GetNo(),
			Contact:       req.GetContact(),
			Detail:        "[]",
		})
		if err != nil {
			return err
		}
		return c.UpdateByID(ctx, &models.OrderInfo{
			ID:     orderInfo.ID,
			Status: _const.ORDER_INFO_STATUS_SHIPPED,
		})
	})
	if err != nil {
		return err
	}
	workspaceevent.Publish(ctx, workspaceevent.ReasonOrderChanged, workspaceevent.AreaTodo)
	return nil
}

// getOrderTradeMap 批量查询门店订单所属交易单。
func (c *OrderInfoCase) getOrderTradeMap(ctx context.Context, orderInfos []*models.OrderInfo) (map[int64]*models.OrderTrade, error) {
	tradeIDs := make([]int64, 0, len(orderInfos))
	for _, orderInfo := range orderInfos {
		tradeIDs = append(tradeIDs, orderInfo.TradeID)
	}
	tradeMap := make(map[int64]*models.OrderTrade, len(tradeIDs))
	if len(tradeIDs) == 0 {
		return tradeMap, nil
	}
	tradeIDs = _slice.Unique(tradeIDs)
	query := c.orderTradeRepo.Query(ctx).OrderTrade
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.ID.In(tradeIDs...)))
	orderTrades, err := c.orderTradeRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	for _, orderTrade := range orderTrades {
		tradeMap[orderTrade.ID] = orderTrade
	}
	return tradeMap, nil
}

// findOrderTrade 按编号查询交易单。
func (c *OrderInfoCase) findOrderTrade(ctx context.Context, tradeID int64) (*models.OrderTrade, error) {
	query := c.orderTradeRepo.Query(ctx).OrderTrade
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.ID.Eq(tradeID)))
	return c.orderTradeRepo.Find(ctx, opts...)
}

// applyOrderTrade 将交易单的支付字段补充到门店订单响应。
func (c *OrderInfoCase) applyOrderTrade(orderInfo *adminv1.OrderInfo, orderTrade *models.OrderTrade) {
	if orderTrade == nil {
		return
	}
	orderInfo.TradeId = orderTrade.ID
	orderInfo.TradeNo = orderTrade.TradeNo
	orderInfo.PayType = commonv1.OrderPayType(orderTrade.PayType)
	orderInfo.PayChannel = commonv1.OrderPayChannel(orderTrade.PayChannel)
	orderInfo.TradeStatus = commonv1.OrderTradeStatus(orderTrade.Status)
}

// successfulRefundMoney 汇总当前门店订单已成功退款金额。
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

// refreshRefundStatuses 按成功退款金额刷新门店订单和交易单状态。
func (c *OrderInfoCase) refreshRefundStatuses(ctx context.Context, orderTrade *models.OrderTrade, orderInfo *models.OrderInfo) error {
	tradeQuery := c.orderTradeRepo.Query(ctx).OrderTrade
	lockedTrade, err := tradeQuery.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where(tradeQuery.ID.Eq(orderTrade.ID)).
		First()
	if err != nil {
		return err
	}
	query := c.orderRefundCase.Query(ctx).OrderRefund
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.TradeID.Eq(orderTrade.ID)))
	orderRefunds, err := c.orderRefundCase.List(ctx, opts...)
	if err != nil {
		return err
	}
	var tradeRefundMoney int64
	var orderRefundMoney int64
	for _, orderRefund := range orderRefunds {
		if orderRefund.RefundState != appv1.RefundResource_SUCCESS.String() {
			continue
		}
		var amount struct {
			Refund int64 `json:"refund"`
		}
		err = json.Unmarshal([]byte(orderRefund.Amount), &amount)
		if err != nil {
			return err
		}
		tradeRefundMoney += amount.Refund
		if orderRefund.OrderID == orderInfo.ID {
			orderRefundMoney += amount.Refund
		}
	}
	orderRefundStatus := _const.ORDER_REFUND_STATUS_PARTIAL_REFUND
	if orderRefundMoney >= orderInfo.PayMoney {
		orderRefundStatus = _const.ORDER_REFUND_STATUS_REFUNDED
	}
	err = c.UpdateByID(ctx, &models.OrderInfo{ID: orderInfo.ID, RefundStatus: orderRefundStatus})
	if err != nil {
		return err
	}
	tradeStatus := _const.ORDER_TRADE_STATUS_PARTIAL_REFUND
	if tradeRefundMoney >= lockedTrade.PayMoney {
		tradeStatus = _const.ORDER_TRADE_STATUS_FULL_REFUND
	}
	return c.orderTradeRepo.UpdateByID(ctx, &models.OrderTrade{ID: orderTrade.ID, Status: tradeStatus})
}

// getOrderUserMap 查询订单用户映射
func (c *OrderInfoCase) getOrderUserMap(ctx context.Context, list []*models.OrderInfo) (map[int64]*models.BaseUser, error) {
	userIDs := make([]int64, 0, len(list))
	for _, item := range list {
		userIDs = append(userIDs, item.UserID)
	}
	userMap := make(map[int64]*models.BaseUser)
	// 订单列表为空时，直接返回空用户映射。
	if len(userIDs) == 0 {
		return userMap, nil
	}

	userIDs = _slice.Unique(userIDs)

	// 订单用户属于全员范围，按订单已记录的用户 ID 跨租户回查昵称。
	queryCtx, err := c.orderUserQueryContext(ctx)
	if err != nil {
		return nil, err
	}
	var userList []*models.BaseUser
	userList, err = c.baseUserCase.ListByIDs(queryCtx, userIDs)
	if err != nil {
		return nil, err
	}
	for _, item := range userList {
		userMap[item.ID] = item
	}
	return userMap, nil
}

// orderUserQueryContext 构建保留请求链路的跨租户订单用户查询上下文。
func (c *OrderInfoCase) orderUserQueryContext(ctx context.Context) (context.Context, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	unscopedAuthInfo := *authInfo
	unscopedAuthInfo.TenantCode = databaseGorm.DefaultTenantCode
	return authnEngine.ContextWithAuthClaims(ctx, unscopedAuthInfo.MakeAuthClaims()), nil
}

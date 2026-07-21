package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	shopappv1 "shop/api/gen/go/shop/app/v1"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	_const "shop/service/shop/consts"

	_string "github.com/liujitcn/go-utils/string"
	_time "github.com/liujitcn/go-utils/time"
	"github.com/liujitcn/go-utils/trans"
	"github.com/liujitcn/gorm-kit/repository"
	"gorm.io/gen"
	"gorm.io/gorm/clause"
)

// OrderRefundResultCase 统一处理渠道退款结果及订单状态汇总。
type OrderRefundResultCase struct {
	tx                 data.Transaction
	orderTradeRepo     *data.OrderTradeRepository
	orderInfoRepo      *data.OrderInfoRepository
	orderRefundRepo    *data.OrderRefundRepository
	orderInventoryCase *OrderInventoryCase
}

// NewOrderRefundResultCase 创建退款结果业务处理对象。
func NewOrderRefundResultCase(
	tx data.Transaction,
	orderTradeRepo *data.OrderTradeRepository,
	orderInfoRepo *data.OrderInfoRepository,
	orderRefundRepo *data.OrderRefundRepository,
	orderInventoryCase *OrderInventoryCase,
) *OrderRefundResultCase {
	return &OrderRefundResultCase{
		tx:                 tx,
		orderTradeRepo:     orderTradeRepo,
		orderInfoRepo:      orderInfoRepo,
		orderRefundRepo:    orderRefundRepo,
		orderInventoryCase: orderInventoryCase,
	}
}

// Apply 校验并应用渠道退款结果，返回本次调用是否完成了有效状态迁移。
func (c *OrderRefundResultCase) Apply(ctx context.Context, orderTrade *models.OrderTrade, refundResource *shopappv1.RefundResource) (bool, error) {
	if orderTrade == nil {
		return false, errorsx.Internal("退款结果处理失败，交易不存在")
	}
	query := c.orderRefundRepo.Query(ctx).OrderRefund
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.TradeID.Eq(orderTrade.ID)))
	opts = append(opts, repository.Where(query.RefundNo.Eq(refundResource.GetOutRefundNo())))
	orderRefund, err := c.orderRefundRepo.Find(ctx, opts...)
	if err != nil {
		return false, err
	}
	err = validateOrderRefundResult(orderTrade, orderRefund, refundResource)
	if err != nil {
		return false, err
	}
	if !shouldApplyOrderRefundResult(orderRefund.RefundState, refundResource.GetRefundStatus()) {
		return false, nil
	}

	// 只有退款成功才写成功时间，处理中、关闭和异常状态保持零值。
	if refundResource.GetRefundStatus() == shopappv1.RefundResource_SUCCESS {
		successTime := _time.TimestamppbToTime(refundResource.GetSuccessTime())
		if successTime == nil {
			successTime = trans.Time(time.Now())
		}
		orderRefund.SuccessTime = trans.TimeValue(successTime)
	}
	orderRefund.TradeNo = refundResource.GetOutTradeNo()
	orderRefund.ThirdOrderNo = refundResource.GetTransactionId()
	orderRefund.ThirdRefundNo = refundResource.GetRefundId()
	orderRefund.UserReceivedAccount = refundResource.GetUserReceivedAccount()
	orderRefund.RefundState = refundResource.GetRefundStatus().String()
	orderRefund.Amount = _string.ConvertAnyToJsonString(refundResource.GetAmount())
	orderRefund.Status = _const.ORDER_BILL_STATUS_NO_CHECK
	sourceStates := orderRefundResultSourceStates(refundResource.GetRefundStatus())

	var applied bool
	err = c.tx.Transaction(ctx, func(ctx context.Context) error {
		// 交易行锁保证不同门店退款汇总不会互相覆盖交易退款状态。
		tradeQuery := c.orderTradeRepo.Query(ctx).OrderTrade
		var lockedTrade *models.OrderTrade
		lockedTrade, err = tradeQuery.WithContext(ctx).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where(tradeQuery.ID.Eq(orderTrade.ID)).
			First()
		if err != nil {
			return err
		}

		// 条件更新保证同步响应、异步通知和补偿查询中只有一个结果能够完成迁移。
		refundQuery := c.orderRefundRepo.Query(ctx).OrderRefund
		var result gen.ResultInfo
		result, err = refundQuery.WithContext(ctx).
			Where(refundQuery.ID.Eq(orderRefund.ID), refundQuery.RefundState.In(sourceStates...)).
			Updates(orderRefund)
		if err != nil {
			return err
		}
		if result.RowsAffected == 0 {
			return nil
		}

		// 锁定门店订单，避免晚到的退款成功结果与发货同时推进。
		orderQuery := c.orderInfoRepo.Query(ctx).OrderInfo
		var orderInfo *models.OrderInfo
		orderInfo, err = orderQuery.WithContext(ctx).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where(orderQuery.ID.Eq(orderRefund.OrderID), orderQuery.TradeID.Eq(orderTrade.ID)).
			First()
		if err != nil {
			return err
		}

		applied = true
		switch refundResource.GetRefundStatus() {
		case shopappv1.RefundResource_SUCCESS:
			return c.refreshOrderRefundStatuses(ctx, lockedTrade, orderInfo)
		case shopappv1.RefundResource_PROCESSING:
			return c.orderInfoRepo.UpdateByID(ctx, &models.OrderInfo{
				ID:           orderInfo.ID,
				RefundStatus: _const.ORDER_REFUND_STATUS_PROCESSING,
			})
		case shopappv1.RefundResource_CLOSED, shopappv1.RefundResource_ABNORMAL:
			return c.orderInfoRepo.UpdateByID(ctx, &models.OrderInfo{
				ID:           orderInfo.ID,
				RefundStatus: _const.ORDER_REFUND_STATUS_CLOSED_OR_FAILED,
			})
		default:
			return nil
		}
	})
	if err != nil {
		return false, err
	}
	return applied, nil
}

// FailPending 将渠道明确不存在的待处理退款关闭，并释放门店订单退款占用。
func (c *OrderRefundResultCase) FailPending(ctx context.Context, orderRefund *models.OrderRefund) (bool, error) {
	var applied bool
	err := c.tx.Transaction(ctx, func(ctx context.Context) error {
		query := c.orderRefundRepo.Query(ctx).OrderRefund
		result, updateErr := query.WithContext(ctx).
			Where(query.ID.Eq(orderRefund.ID), query.RefundState.Eq(shopappv1.RefundResource_PROCESSING.String())).
			Update(query.RefundState, shopappv1.RefundResource_ABNORMAL.String())
		if updateErr != nil {
			return updateErr
		}
		if result.RowsAffected == 0 {
			return nil
		}

		orderQuery := c.orderInfoRepo.Query(ctx).OrderInfo
		result, updateErr = orderQuery.WithContext(ctx).
			Where(
				orderQuery.ID.Eq(orderRefund.OrderID),
				orderQuery.TradeID.Eq(orderRefund.TradeID),
				orderQuery.RefundStatus.Eq(_const.ORDER_REFUND_STATUS_PROCESSING),
			).
			Update(orderQuery.RefundStatus, _const.ORDER_REFUND_STATUS_CLOSED_OR_FAILED)
		if updateErr != nil {
			return updateErr
		}
		if result.RowsAffected == 0 {
			return errorsx.Internal(fmt.Sprintf("关闭待处理退款失败，门店订单状态不一致：orderID=%d", orderRefund.OrderID))
		}
		applied = true
		return nil
	})
	if err != nil {
		return false, err
	}
	return applied, nil
}

// refreshOrderRefundStatuses 按成功退款金额刷新门店订单和交易单状态。
func (c *OrderRefundResultCase) refreshOrderRefundStatuses(ctx context.Context, orderTrade *models.OrderTrade, orderInfo *models.OrderInfo) error {
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
		if refund.RefundState != shopappv1.RefundResource_SUCCESS.String() {
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
	isFullRefund := orderRefundMoney >= orderInfo.PayMoney
	if isFullRefund {
		orderRefundStatus = _const.ORDER_REFUND_STATUS_REFUNDED
	}
	// 尚未发货的门店订单首次完成全额退款时，恢复整张订单的库存和销量。
	if orderInfo.RefundStatus != _const.ORDER_REFUND_STATUS_REFUNDED &&
		isFullRefund && orderInfo.Status == _const.ORDER_INFO_STATUS_WAIT_SHIPMENT {
		err = c.orderInventoryCase.RestoreOrder(ctx, orderInfo.ID)
		if err != nil {
			return err
		}
	}
	err = c.orderInfoRepo.UpdateByID(ctx, &models.OrderInfo{
		ID:           orderInfo.ID,
		RefundStatus: orderRefundStatus,
	})
	if err != nil {
		return err
	}

	tradeStatus := _const.ORDER_TRADE_STATUS_PARTIAL_REFUND
	if tradeRefundMoney >= orderTrade.PayMoney {
		tradeStatus = _const.ORDER_TRADE_STATUS_FULL_REFUND
	}
	return c.orderTradeRepo.UpdateByID(ctx, &models.OrderTrade{
		ID:     orderTrade.ID,
		Status: tradeStatus,
	})
}

// validateOrderRefundResult 校验渠道退款结果与本地退款申请一致。
func validateOrderRefundResult(orderTrade *models.OrderTrade, orderRefund *models.OrderRefund, refundResource *shopappv1.RefundResource) error {
	switch refundResource.GetRefundStatus() {
	case shopappv1.RefundResource_SUCCESS, shopappv1.RefundResource_CLOSED, shopappv1.RefundResource_PROCESSING, shopappv1.RefundResource_ABNORMAL:
	default:
		return errorsx.StateConflict(
			"退款结果状态无效",
			"refund_resource",
			refundResource.GetRefundStatus().String(),
			"SUCCESS|CLOSED|PROCESSING|ABNORMAL",
		)
	}
	if refundResource.GetOutTradeNo() != orderTrade.TradeNo || refundResource.GetOutRefundNo() != orderRefund.RefundNo {
		return errorsx.Internal("退款结果编号与本地退款申请不一致")
	}
	if refundResource.GetAmount() == nil {
		return errorsx.Internal("退款结果缺少金额信息")
	}
	var amount map[string]int64
	err := json.Unmarshal([]byte(orderRefund.Amount), &amount)
	if err != nil {
		return errorsx.Internal("本地退款金额解析失败").WithCause(err)
	}
	if int64(refundResource.GetAmount().GetTotal()) != orderTrade.PayMoney ||
		int64(refundResource.GetAmount().GetRefund()) != amount["refund"] {
		return errorsx.Internal("退款结果金额与本地退款申请不一致")
	}
	return nil
}

// shouldApplyOrderRefundResult 判断退款结果是否仍可从当前本地状态向前推进。
func shouldApplyOrderRefundResult(current string, next shopappv1.RefundResource_RefundStatus) bool {
	return slices.Contains(orderRefundResultSourceStates(next), current)
}

// orderRefundResultSourceStates 返回指定渠道结果允许覆盖的本地退款状态。
func orderRefundResultSourceStates(next shopappv1.RefundResource_RefundStatus) []string {
	switch next {
	case shopappv1.RefundResource_SUCCESS:
		return []string{
			shopappv1.RefundResource_PROCESSING.String(),
			shopappv1.RefundResource_CLOSED.String(),
			shopappv1.RefundResource_ABNORMAL.String(),
		}
	case shopappv1.RefundResource_CLOSED, shopappv1.RefundResource_PROCESSING, shopappv1.RefundResource_ABNORMAL:
		return []string{shopappv1.RefundResource_PROCESSING.String()}
	default:
		return nil
	}
}

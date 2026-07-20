package biz

import (
	"context"
	"time"

	shopappv1 "shop/api/gen/go/shop/app/v1"

	_const "shop/pkg/const"
	"shop/pkg/gen/models"
	"shop/service/shop/admin/dto"
	orderutils "shop/service/shop/utils"

	"github.com/liujitcn/gorm-kit/repository"
)

// queryPaidOrderFacts 按支付事实时间查询交易或门店子订单。
func (c *OrderInfoCase) queryPaidOrderFacts(
	ctx context.Context,
	payType, payChannel int32,
	tenantID, tenantStoreID int64,
	startAt, endAt time.Time,
	useGlobalTradeScope bool,
) ([]*dto.OrderPaidFact, error) {
	tradeFactMap := make(map[int64]*dto.OrderPaidFact)
	if payType == 0 || payType == _const.ORDER_PAY_TYPE_ONLINE_PAY {
		paymentQuery := c.orderPaymentCase.Query(ctx).OrderPayment
		paymentOpts := make([]repository.QueryOption, 0, 3)
		paymentOpts = append(paymentOpts, repository.Where(paymentQuery.SuccessTime.Gte(startAt)))
		paymentOpts = append(paymentOpts, repository.Where(paymentQuery.SuccessTime.Lt(endAt)))
		paymentOpts = append(paymentOpts, repository.Where(paymentQuery.TradeState.Eq(shopappv1.PaymentResource_SUCCESS.String())))
		orderPayments, err := c.orderPaymentCase.List(ctx, paymentOpts...)
		if err != nil {
			return nil, err
		}

		paymentTimeMap := make(map[int64]time.Time, len(orderPayments))
		tradeIDs := make([]int64, 0, len(orderPayments))
		for _, orderPayment := range orderPayments {
			paymentTimeMap[orderPayment.TradeID] = orderPayment.SuccessTime
			tradeIDs = append(tradeIDs, orderPayment.TradeID)
		}
		if len(tradeIDs) > 0 {
			tradeQuery := c.orderTradeRepo.Query(ctx).OrderTrade
			tradeOpts := make([]repository.QueryOption, 0, 4)
			tradeOpts = append(tradeOpts, repository.Where(tradeQuery.ID.In(tradeIDs...)))
			tradeOpts = append(tradeOpts, repository.Where(tradeQuery.PayType.Eq(_const.ORDER_PAY_TYPE_ONLINE_PAY)))
			tradeOpts = append(tradeOpts, repository.Where(tradeQuery.Status.In(orderutils.PaidTradeStatuses()...)))
			if payChannel > 0 {
				tradeOpts = append(tradeOpts, repository.Where(tradeQuery.PayChannel.Eq(payChannel)))
			}
			var orderTrades []*models.OrderTrade
			orderTrades, err = c.orderTradeRepo.List(ctx, tradeOpts...)
			if err != nil {
				return nil, err
			}
			for _, orderTrade := range orderTrades {
				tradeFactMap[orderTrade.ID] = &dto.OrderPaidFact{
					TradeID:  orderTrade.ID,
					UserID:   orderTrade.UserID,
					PayMoney: orderTrade.PayMoney,
					PaidAt:   paymentTimeMap[orderTrade.ID],
				}
			}
		}
	}

	if payType == 0 || payType == _const.ORDER_PAY_TYPE_CASH_ON_DELIVERY {
		tradeQuery := c.orderTradeRepo.Query(ctx).OrderTrade
		tradeOpts := make([]repository.QueryOption, 0, 5)
		tradeOpts = append(tradeOpts, repository.Where(tradeQuery.CreatedAt.Gte(startAt)))
		tradeOpts = append(tradeOpts, repository.Where(tradeQuery.CreatedAt.Lt(endAt)))
		tradeOpts = append(tradeOpts, repository.Where(tradeQuery.PayType.Eq(_const.ORDER_PAY_TYPE_CASH_ON_DELIVERY)))
		tradeOpts = append(tradeOpts, repository.Where(tradeQuery.Status.In(orderutils.PaidTradeStatuses()...)))
		if payChannel > 0 {
			tradeOpts = append(tradeOpts, repository.Where(tradeQuery.PayChannel.Eq(payChannel)))
		}
		orderTrades, err := c.orderTradeRepo.List(ctx, tradeOpts...)
		if err != nil {
			return nil, err
		}
		for _, orderTrade := range orderTrades {
			tradeFactMap[orderTrade.ID] = &dto.OrderPaidFact{
				TradeID:  orderTrade.ID,
				UserID:   orderTrade.UserID,
				PayMoney: orderTrade.PayMoney,
				PaidAt:   orderTrade.CreatedAt,
			}
		}
	}

	if useGlobalTradeScope {
		paidFacts := make([]*dto.OrderPaidFact, 0, len(tradeFactMap))
		for _, fact := range tradeFactMap {
			paidFacts = append(paidFacts, fact)
		}
		return paidFacts, nil
	}
	if len(tradeFactMap) == 0 {
		return nil, nil
	}

	tradeIDs := make([]int64, 0, len(tradeFactMap))
	for tradeID := range tradeFactMap {
		tradeIDs = append(tradeIDs, tradeID)
	}
	orderQuery := c.Query(ctx).OrderInfo
	orderOpts := make([]repository.QueryOption, 0, 3)
	orderOpts = append(orderOpts, repository.Where(orderQuery.TradeID.In(tradeIDs...)))
	if tenantID > 0 {
		orderOpts = append(orderOpts, repository.Where(orderQuery.TenantID.Eq(tenantID)))
	}
	if tenantStoreID > 0 {
		orderOpts = append(orderOpts, repository.Where(orderQuery.TenantStoreID.Eq(tenantStoreID)))
	}
	orderInfos, err := c.List(ctx, orderOpts...)
	if err != nil {
		return nil, err
	}

	paidFacts := make([]*dto.OrderPaidFact, 0, len(orderInfos))
	for _, orderInfo := range orderInfos {
		tradeFact := tradeFactMap[orderInfo.TradeID]
		paidFacts = append(paidFacts, &dto.OrderPaidFact{
			TradeID:  orderInfo.TradeID,
			OrderID:  orderInfo.ID,
			UserID:   orderInfo.UserID,
			PayMoney: orderInfo.PayMoney,
			PaidAt:   tradeFact.PaidAt,
		})
	}
	return paidFacts, nil
}

package biz

import (
	"context"
	"time"

	shopappv1 "shop/api/gen/go/shop/app/v1"

	_const "shop/service/shop/consts"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"shop/service/shop/utils"

	"github.com/liujitcn/gorm-kit/repository"
)

// listPaidTradesByFactTime 按支付事实时间查询指定区间内完成支付的交易单。
func listPaidTradesByFactTime(
	ctx context.Context,
	startAt, endAt time.Time,
	orderTradeRepo *data.OrderTradeRepository,
	orderPaymentRepo *data.OrderPaymentRepository,
) (map[int64]*models.OrderTrade, error) {
	tradeMap := make(map[int64]*models.OrderTrade)

	paymentQuery := orderPaymentRepo.Query(ctx).OrderPayment
	paymentOpts := make([]repository.QueryOption, 0, 3)
	paymentOpts = append(paymentOpts, repository.Where(paymentQuery.SuccessTime.Gte(startAt)))
	paymentOpts = append(paymentOpts, repository.Where(paymentQuery.SuccessTime.Lt(endAt)))
	paymentOpts = append(paymentOpts, repository.Where(paymentQuery.TradeState.Eq(shopappv1.PaymentResource_SUCCESS.String())))
	orderPayments, err := orderPaymentRepo.List(ctx, paymentOpts...)
	if err != nil {
		return nil, err
	}

	onlineTradeIDs := make([]int64, 0, len(orderPayments))
	for _, orderPayment := range orderPayments {
		if orderPayment != nil && orderPayment.TradeID > 0 {
			onlineTradeIDs = append(onlineTradeIDs, orderPayment.TradeID)
		}
	}
	if len(onlineTradeIDs) > 0 {
		tradeQuery := orderTradeRepo.Query(ctx).OrderTrade
		tradeOpts := make([]repository.QueryOption, 0, 3)
		tradeOpts = append(tradeOpts, repository.Where(tradeQuery.ID.In(onlineTradeIDs...)))
		tradeOpts = append(tradeOpts, repository.Where(tradeQuery.PayType.Eq(_const.ORDER_PAY_TYPE_ONLINE_PAY)))
		tradeOpts = append(tradeOpts, repository.Where(tradeQuery.Status.In(utils.PaidTradeStatuses()...)))
		var orderTrades []*models.OrderTrade
		orderTrades, err = orderTradeRepo.List(ctx, tradeOpts...)
		if err != nil {
			return nil, err
		}
		for _, orderTrade := range orderTrades {
			tradeMap[orderTrade.ID] = orderTrade
		}
	}

	tradeQuery := orderTradeRepo.Query(ctx).OrderTrade
	tradeOpts := make([]repository.QueryOption, 0, 4)
	tradeOpts = append(tradeOpts, repository.Where(tradeQuery.CreatedAt.Gte(startAt)))
	tradeOpts = append(tradeOpts, repository.Where(tradeQuery.CreatedAt.Lt(endAt)))
	tradeOpts = append(tradeOpts, repository.Where(tradeQuery.PayType.Eq(_const.ORDER_PAY_TYPE_CASH_ON_DELIVERY)))
	tradeOpts = append(tradeOpts, repository.Where(tradeQuery.Status.In(utils.PaidTradeStatuses()...)))
	var cashOnDeliveryTrades []*models.OrderTrade
	cashOnDeliveryTrades, err = orderTradeRepo.List(ctx, tradeOpts...)
	if err != nil {
		return nil, err
	}
	for _, orderTrade := range cashOnDeliveryTrades {
		tradeMap[orderTrade.ID] = orderTrade
	}
	return tradeMap, nil
}

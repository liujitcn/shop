package task

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	appv1 "shop/api/gen/go/app/v1"
	_const "shop/pkg/const"

	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"shop/pkg/utils"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/liujitcn/gorm-kit/repository"
)

// orderStatDayKey 表示订单日统计的租户与支付维度聚合键。
type orderStatDayKey struct {
	tenantID      int64
	tenantStoreID int64
	payType       int32
	payChannel    int32
}

// orderStatDayResult 表示订单日汇总任务执行结果。
type orderStatDayResult struct {
	orderCount int
	statCount  int
}

// OrderStatDay 订单日汇总任务。
type OrderStatDay struct {
	tx               data.Transaction
	orderStatDayRepo *data.OrderStatDayRepository
	orderInfoRepo    *data.OrderInfoRepository
	orderTradeRepo   *data.OrderTradeRepository
	orderRefundRepo  *data.OrderRefundRepository
	ctx              context.Context
}

// NewOrderStatDay 创建订单日汇总任务实例。
func NewOrderStatDay(
	tx data.Transaction,
	orderStatDayRepo *data.OrderStatDayRepository,
	orderInfoRepo *data.OrderInfoRepository,
	orderTradeRepo *data.OrderTradeRepository,
	orderRefundRepo *data.OrderRefundRepository,
) *OrderStatDay {
	return &OrderStatDay{
		tx:               tx,
		orderStatDayRepo: orderStatDayRepo,
		orderInfoRepo:    orderInfoRepo,
		orderTradeRepo:   orderTradeRepo,
		orderRefundRepo:  orderRefundRepo,
		ctx:              context.Background(),
	}
}

// Exec 执行订单日汇总。
func (t *OrderStatDay) Exec(args map[string]string) ([]string, error) {
	log.Info(fmt.Sprintf("Job OrderStatDay Exec %+v", args))

	statTime, err := parseStatDateArg(args["statDate"])
	if err != nil {
		return []string{err.Error()}, err
	}

	statDate := time.Date(statTime.Year(), statTime.Month(), statTime.Day(), 0, 0, 0, 0, statTime.Location())
	startAt := statDate
	endAt := statDate.AddDate(0, 0, 1)

	result := orderStatDayResult{}
	err = t.tx.Transaction(t.ctx, func(ctx context.Context) error {
		query := t.orderStatDayRepo.Query(ctx).OrderStatDay
		// 订单日统计表带软删字段，这里必须物理删除旧数据再回灌。
		_, err = query.WithContext(ctx).Unscoped().Where(query.StatDate.Eq(statDate)).Delete()
		if err != nil {
			return err
		}

		orderQuery := t.orderInfoRepo.Query(ctx).OrderInfo
		opts := make([]repository.QueryOption, 0, 2)
		opts = append(opts, repository.Where(orderQuery.CreatedAt.Gte(startAt)))
		opts = append(opts, repository.Where(orderQuery.CreatedAt.Lt(endAt)))
		var orderInfoList []*models.OrderInfo
		orderInfoList, err = t.orderInfoRepo.List(ctx, opts...)
		if err != nil {
			return err
		}
		result.orderCount = len(orderInfoList)

		refundQuery := t.orderRefundRepo.Query(ctx).OrderRefund
		refundOpts := make([]repository.QueryOption, 0, 3)
		refundOpts = append(refundOpts, repository.Where(refundQuery.SuccessTime.Gte(startAt)))
		refundOpts = append(refundOpts, repository.Where(refundQuery.SuccessTime.Lt(endAt)))
		refundOpts = append(refundOpts, repository.Where(refundQuery.RefundState.Eq(appv1.RefundResource_SUCCESS.String())))
		var orderRefunds []*models.OrderRefund
		orderRefunds, err = t.orderRefundRepo.List(ctx, refundOpts...)
		if err != nil {
			return err
		}

		tradeIDSet := make(map[int64]struct{}, len(orderInfoList)+len(orderRefunds))
		for _, orderInfo := range orderInfoList {
			tradeIDSet[orderInfo.TradeID] = struct{}{}
		}
		for _, orderRefund := range orderRefunds {
			tradeIDSet[orderRefund.TradeID] = struct{}{}
		}
		tradeIDs := make([]int64, 0, len(tradeIDSet))
		for tradeID := range tradeIDSet {
			tradeIDs = append(tradeIDs, tradeID)
		}
		tradeMap := make(map[int64]*models.OrderTrade, len(tradeIDs))
		if len(tradeIDs) > 0 {
			tradeQuery := t.orderTradeRepo.Query(ctx).OrderTrade
			tradeOpts := make([]repository.QueryOption, 0, 1)
			tradeOpts = append(tradeOpts, repository.Where(tradeQuery.ID.In(tradeIDs...)))
			var orderTrades []*models.OrderTrade
			orderTrades, err = t.orderTradeRepo.List(ctx, tradeOpts...)
			if err != nil {
				return err
			}
			for _, orderTrade := range orderTrades {
				tradeMap[orderTrade.ID] = orderTrade
			}
		}

		statMap := make(map[orderStatDayKey]*models.OrderStatDay)
		paidUserMap := make(map[orderStatDayKey]map[int64]struct{})
		refundedOrderMap := make(map[orderStatDayKey]map[int64]struct{})
		ensureStat := func(tenantID, tenantStoreID int64, payType, payChannel int32) *models.OrderStatDay {
			key := orderStatDayKey{tenantID: tenantID, tenantStoreID: tenantStoreID, payType: payType, payChannel: payChannel}
			item, ok := statMap[key]
			// 首次出现的租户支付维度需要先初始化统计对象。
			if !ok {
				item = &models.OrderStatDay{
					TenantID:      tenantID,
					TenantStoreID: tenantStoreID,
					StatDate:      statDate,
					PayType:       payType,
					PayChannel:    payChannel,
				}
				statMap[key] = item
			}
			return item
		}

		for _, item := range orderInfoList {
			// 非法订单不参与统计。
			if item == nil || item.ID <= 0 {
				continue
			}
			orderTrade := tradeMap[item.TradeID]
			// 缺少交易单的子订单无法确定支付维度，不写入日统计。
			if orderTrade == nil {
				continue
			}
			stat := ensureStat(item.TenantID, item.TenantStoreID, orderTrade.PayType, orderTrade.PayChannel)
			// 支付指标以交易单状态为准，门店履约状态不再反推支付事实。
			if utils.IsPaidTradeStatus(orderTrade.Status) {
				stat.PaidOrderCount++
				stat.PaidOrderAmount += item.PayMoney
				stat.GoodsCount += int32(item.GoodsNum)
				key := orderStatDayKey{tenantID: item.TenantID, tenantStoreID: item.TenantStoreID, payType: orderTrade.PayType, payChannel: orderTrade.PayChannel}
				// 当前租户支付维度首次出现用户集合时，先初始化去重容器。
				if _, ok := paidUserMap[key]; !ok {
					paidUserMap[key] = make(map[int64]struct{}, 1)
				}
				// 支付用户数按租户与支付维度做当天去重。
				paidUserMap[key][item.UserID] = struct{}{}
			}
			// 已取消的门店订单按子订单金额累计取消指标。
			if item.Status == _const.ORDER_INFO_STATUS_CANCELED {
				stat.CanceledOrderCount++
				stat.CanceledOrderAmount += item.TotalMoney
			}
		}
		for _, orderRefund := range orderRefunds {
			orderTrade := tradeMap[orderRefund.TradeID]
			// 缺少交易单时无法确定退款的支付维度，不写入日统计。
			if orderTrade == nil {
				continue
			}
			key := orderStatDayKey{
				tenantID:      orderRefund.TenantID,
				tenantStoreID: orderRefund.TenantStoreID,
				payType:       orderTrade.PayType,
				payChannel:    orderTrade.PayChannel,
			}
			stat := ensureStat(key.tenantID, key.tenantStoreID, key.payType, key.payChannel)
			var amount struct {
				Refund int64 `json:"refund"`
			}
			err = json.Unmarshal([]byte(orderRefund.Amount), &amount)
			if err != nil {
				return err
			}
			stat.RefundOrderAmount += amount.Refund
			if refundedOrderMap[key] == nil {
				refundedOrderMap[key] = make(map[int64]struct{})
			}
			// 同一门店订单当天多次部分退款时，订单数只统计一次。
			if _, ok := refundedOrderMap[key][orderRefund.OrderID]; !ok {
				stat.RefundOrderCount++
				refundedOrderMap[key][orderRefund.OrderID] = struct{}{}
			}
		}

		for key, userSet := range paidUserMap {
			statMap[key].PaidUserCount = int32(len(userSet))
		}

		list := make([]*models.OrderStatDay, 0, len(statMap))
		for _, item := range statMap {
			list = append(list, item)
		}
		result.statCount = len(list)
		// 没有统计结果时只保留清理动作，不再写入空数据。
		if len(list) == 0 {
			return nil
		}
		return t.orderStatDayRepo.BatchCreate(ctx, list)
	})
	if err != nil {
		return []string{err.Error()}, err
	}

	return []string{fmt.Sprintf("订单日汇总完成: 查询订单 %d 条，写入统计 %d 条", result.orderCount, result.statCount)}, nil
}

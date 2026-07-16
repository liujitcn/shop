package task

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	appv1 "shop/api/gen/go/app/v1"
	"shop/pkg/gen/data"
	"shop/pkg/wx"
	appBiz "shop/service/app/biz"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/liujitcn/gorm-kit/repository"
	wxPayCore "github.com/wechatpay-apiv3/wechatpay-go/core"
)

const (
	orderRefundRetryDefaultBatchSize   = 100
	orderRefundRetryDefaultDelayMinute = 5
	orderRefundRetryFailureDetailLimit = 10
)

// OrderRefundRetry 补查渠道结果不确定的待处理退款。
type OrderRefundRetry struct {
	payCase         *appBiz.PayCase
	orderTradeRepo  *data.OrderTradeRepository
	orderRefundRepo *data.OrderRefundRepository
	wxPayCase       *wx.WxPayCase
	ctx             context.Context
	mu              sync.Mutex
}

// NewOrderRefundRetry 创建退款状态补偿任务。
func NewOrderRefundRetry(
	payCase *appBiz.PayCase,
	orderTradeRepo *data.OrderTradeRepository,
	orderRefundRepo *data.OrderRefundRepository,
	wxPayCase *wx.WxPayCase,
) *OrderRefundRetry {
	return &OrderRefundRetry{
		payCase:         payCase,
		orderTradeRepo:  orderTradeRepo,
		orderRefundRepo: orderRefundRepo,
		wxPayCase:       wxPayCase,
		ctx:             context.Background(),
	}
}

// Exec 按原退款号补查微信退款状态并推进本地状态机。
func (t *OrderRefundRetry) Exec(args map[string]string) ([]string, error) {
	log.Info(fmt.Sprintf("Job OrderRefundRetry Exec %+v", args))
	if !t.mu.TryLock() {
		return []string{"退款状态补偿任务仍在执行，本轮跳过"}, nil
	}
	defer t.mu.Unlock()

	batchSize, err := parsePositiveJobInt(args, "batchSize", orderRefundRetryDefaultBatchSize)
	if err != nil {
		return []string{err.Error()}, err
	}
	var retryDelayMinutes int
	retryDelayMinutes, err = parsePositiveJobInt(args, "retryDelayMinutes", orderRefundRetryDefaultDelayMinute)
	if err != nil {
		return []string{err.Error()}, err
	}

	query := t.orderRefundRepo.Query(t.ctx).OrderRefund
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Where(query.RefundState.Eq(appv1.RefundResource_PROCESSING.String())))
	opts = append(opts, repository.Where(query.CreateTime.Lte(time.Now().Add(-time.Duration(retryDelayMinutes)*time.Minute))))
	opts = append(opts, repository.Order(query.CreateTime.Asc()))
	opts = append(opts, repository.Limit(batchSize))
	orderRefunds, err := t.orderRefundRepo.List(t.ctx, opts...)
	if err != nil {
		return []string{err.Error()}, err
	}

	var syncedCount int
	var processingCount int
	var closedCount int
	var failedCount int
	failureDetails := make([]string, 0, orderRefundRetryFailureDetailLimit)
	for _, orderRefund := range orderRefunds {
		tradeQuery := t.orderTradeRepo.Query(t.ctx).OrderTrade
		tradeOpts := make([]repository.QueryOption, 0, 1)
		tradeOpts = append(tradeOpts, repository.Where(tradeQuery.ID.Eq(orderRefund.TradeID)))
		orderTrade, queryErr := t.orderTradeRepo.Find(t.ctx, tradeOpts...)
		if queryErr != nil {
			failedCount++
			if len(failureDetails) < orderRefundRetryFailureDetailLimit {
				failureDetails = append(failureDetails, fmt.Sprintf("退款号 %s 查询本地交易失败: %v", orderRefund.RefundNo, queryErr))
			}
			continue
		}

		refundResource, queryErr := t.wxPayCase.QueryByOutRefundNo(orderRefund.RefundNo)
		if queryErr != nil {
			// 沉淀期后渠道仍明确表示退款单不存在，才能释放本地退款占用；网络错误继续保留待处理状态。
			if apiErr, ok := errors.AsType[*wxPayCore.APIError](queryErr); ok && apiErr.Code == "RESOURCE_NOT_EXISTS" {
				queryErr = t.payCase.FailPendingRefund(t.ctx, orderRefund)
				if queryErr == nil {
					closedCount++
					continue
				}
			}
			failedCount++
			if len(failureDetails) < orderRefundRetryFailureDetailLimit {
				failureDetails = append(failureDetails, fmt.Sprintf("退款号 %s 补查失败: %v", orderRefund.RefundNo, queryErr))
			}
			continue
		}

		queryErr = t.payCase.RefundSuccess(t.ctx, orderTrade, refundResource)
		if queryErr != nil {
			failedCount++
			if len(failureDetails) < orderRefundRetryFailureDetailLimit {
				failureDetails = append(failureDetails, fmt.Sprintf("退款号 %s 状态同步失败: %v", orderRefund.RefundNo, queryErr))
			}
			continue
		}
		if refundResource.GetRefundStatus() == appv1.RefundResource_PROCESSING {
			processingCount++
		} else {
			syncedCount++
		}
	}

	output := []string{fmt.Sprintf(
		"退款状态补偿完成: 查询 %d 条，已同步 %d 条，仍处理中 %d 条，渠道不存在关闭 %d 条，失败 %d 条",
		len(orderRefunds), syncedCount, processingCount, closedCount, failedCount,
	)}
	if len(failureDetails) > 0 {
		output = append(output, "失败明细: "+strings.Join(failureDetails, "；"))
	}
	return output, nil
}

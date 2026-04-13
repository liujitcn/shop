package task

import (
	"context"
	"fmt"
	"time"

	"shop/api/gen/go/common"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	pkgUtils "shop/pkg/utils"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/liujitcn/gorm-kit/repo"
)

type orderStatDayKey struct {
	payType    int32
	payChannel int32
}

// OrderStatDay 订单日汇总任务。
type OrderStatDay struct {
	tx               data.Transaction
	orderStatDayRepo *data.OrderStatDayRepo
	orderInfoRepo    *data.OrderInfoRepo
	ctx              context.Context
}

// NewOrderStatDay 创建订单日汇总任务实例。
func NewOrderStatDay(
	tx data.Transaction,
	orderStatDayRepo *data.OrderStatDayRepo,
	orderInfoRepo *data.OrderInfoRepo,
) *OrderStatDay {
	return &OrderStatDay{
		tx:               tx,
		orderStatDayRepo: orderStatDayRepo,
		orderInfoRepo:    orderInfoRepo,
		ctx:              context.Background(),
	}
}

// Exec 执行订单日汇总。
func (t *OrderStatDay) Exec(args map[string]string) ([]string, error) {
	log.Infof("Job OrderStatDay Exec %+v", args)

	statTime, err := parseStatDateArg(args["statDate"])
	if err != nil {
		return []string{err.Error()}, err
	}

	statDate := time.Date(statTime.Year(), statTime.Month(), statTime.Day(), 0, 0, 0, 0, statTime.Location())
	startAt := statDate
	endAt := statDate.AddDate(0, 0, 1)

	err = t.tx.Transaction(t.ctx, func(ctx context.Context) error {
		query := t.orderStatDayRepo.Query(ctx).OrderStatDay
		// 订单日统计表带软删字段，这里必须物理删除旧数据再回灌。
		_, err = query.WithContext(ctx).Unscoped().Where(query.StatDate.Eq(statDate)).Delete()
		if err != nil {
			return err
		}

		orderQuery := t.orderInfoRepo.Query(ctx).OrderInfo
		opts := make([]repo.QueryOption, 0, 2)
		opts = append(opts, repo.Where(orderQuery.CreatedAt.Gte(startAt)))
		opts = append(opts, repo.Where(orderQuery.CreatedAt.Lt(endAt)))
		var orderInfoList []*models.OrderInfo
		orderInfoList, err = t.orderInfoRepo.List(ctx, opts...)
		if err != nil {
			return err
		}

		statMap := make(map[orderStatDayKey]*models.OrderStatDay)
		paidUserMap := make(map[orderStatDayKey]map[int64]struct{})
		ensureStat := func(payType, payChannel int32) *models.OrderStatDay {
			key := orderStatDayKey{payType: payType, payChannel: payChannel}
			item, ok := statMap[key]
			// 首次出现的支付维度需要先初始化统计对象。
			if !ok {
				item = &models.OrderStatDay{
					StatDate:   statDate,
					PayType:    payType,
					PayChannel: payChannel,
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
			stat := ensureStat(item.PayType, item.PayChannel)
			// 已支付口径的订单按主表状态直接累计。
			if pkgUtils.IsPaidOrderStatus(item.Status) {
				stat.PaidOrderCount++
				stat.PaidOrderAmount += item.PayMoney
				stat.GoodsCount += int32(item.GoodsNum)
				key := orderStatDayKey{payType: item.PayType, payChannel: item.PayChannel}
				// 当前支付维度首次出现用户集合时，先初始化去重容器。
				if _, ok := paidUserMap[key]; !ok {
					paidUserMap[key] = make(map[int64]struct{}, 1)
				}
				// 支付用户数按支付维度做当天去重。
				paidUserMap[key][item.UserID] = struct{}{}
			}
			// 已退款状态直接按主表金额累计退款指标。
			if item.Status == int32(common.OrderStatus_REFUNDING) {
				stat.RefundOrderCount++
				stat.RefundOrderAmount += item.PayMoney
			}
			// 已取消状态直接按主表金额累计取消指标。
			if item.Status == int32(common.OrderStatus_CANCELED) {
				stat.CanceledOrderCount++
				stat.CanceledOrderAmount += item.TotalMoney
			}
		}

		for key, userSet := range paidUserMap {
			statMap[key].PaidUserCount = int32(len(userSet))
		}

		list := make([]*models.OrderStatDay, 0, len(statMap))
		for _, item := range statMap {
			list = append(list, item)
		}
		// 没有统计结果时只保留清理动作，不再写入空数据。
		if len(list) == 0 {
			return nil
		}
		return t.orderStatDayRepo.BatchCreate(ctx, list)
	})
	if err != nil {
		return []string{err.Error()}, err
	}

	return []string{fmt.Sprintf("订单日汇总完成: %s", statDate.Format("2006-01-02"))}, nil
}

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

// GoodsStatDay 商品日统计任务。
type GoodsStatDay struct {
	tx                 data.Transaction
	goodsStatDayRepo   *data.GoodsStatDayRepo
	recommendEventRepo *data.RecommendEventRepo
	userCollectRepo    *data.UserCollectRepo
	userCartRepo       *data.UserCartRepo
	orderInfoRepo      *data.OrderInfoRepo
	orderGoodsRepo     *data.OrderGoodsRepo
	ctx                context.Context
}

// NewGoodsStatDay 创建商品日统计任务实例。
func NewGoodsStatDay(
	tx data.Transaction,
	goodsStatDayRepo *data.GoodsStatDayRepo,
	recommendEventRepo *data.RecommendEventRepo,
	userCollectRepo *data.UserCollectRepo,
	userCartRepo *data.UserCartRepo,
	orderInfoRepo *data.OrderInfoRepo,
	orderGoodsRepo *data.OrderGoodsRepo,
) *GoodsStatDay {
	return &GoodsStatDay{
		tx:                 tx,
		goodsStatDayRepo:   goodsStatDayRepo,
		recommendEventRepo: recommendEventRepo,
		userCollectRepo:    userCollectRepo,
		userCartRepo:       userCartRepo,
		orderInfoRepo:      orderInfoRepo,
		orderGoodsRepo:     orderGoodsRepo,
		ctx:                context.Background(),
	}
}

// Exec 执行商品日统计。
func (t *GoodsStatDay) Exec(args map[string]string) ([]string, error) {
	log.Infof("Job GoodsStatDay Exec %+v", args)

	statTime, err := parseStatDateArg(args["statDate"])
	if err != nil {
		return []string{err.Error()}, err
	}

	statDate := time.Date(statTime.Year(), statTime.Month(), statTime.Day(), 0, 0, 0, 0, statTime.Location())
	startAt := statDate
	endAt := statDate.AddDate(0, 0, 1)

	err = t.tx.Transaction(t.ctx, func(ctx context.Context) error {
		statQuery := t.goodsStatDayRepo.Query(ctx).GoodsStatDay
		// 统计任务按天全量重算，先清掉当天旧数据再回写。
		opts := make([]repo.QueryOption, 0, 1)
		opts = append(opts, repo.Where(statQuery.StatDate.Eq(statDate)))
		err = t.goodsStatDayRepo.Delete(ctx, opts...)
		if err != nil {
			return err
		}

		statMap := make(map[int64]*models.GoodsStatDay)
		ensureStat := func(goodsId int64) *models.GoodsStatDay {
			item, ok := statMap[goodsId]
			// 首次出现的商品需要先初始化统计对象。
			if !ok {
				item = &models.GoodsStatDay{
					StatDate: statDate,
					GoodsID:  goodsId,
				}
				statMap[goodsId] = item
			}
			return item
		}

		actionQuery := t.recommendEventRepo.Query(ctx).RecommendEvent
		opts = make([]repo.QueryOption, 0, 3)
		opts = append(opts, repo.Where(actionQuery.EventAt.Gte(startAt)))
		opts = append(opts, repo.Where(actionQuery.EventAt.Lt(endAt)))
		// 浏览数统一从推荐事件表里的 VIEW 事件口径汇总。
		opts = append(opts, repo.Where(actionQuery.EventType.Eq(int32(common.RecommendEventType_VIEW))))
		var viewList []*models.RecommendEvent
		viewList, err = t.recommendEventRepo.List(ctx, opts...)
		if err != nil {
			return err
		}
		for _, item := range viewList {
			// 非法商品不参与统计。
			if item.GoodsID <= 0 {
				continue
			}
			ensureStat(item.GoodsID).ViewCount++
		}

		collectQuery := t.userCollectRepo.Query(ctx).UserCollect
		opts = make([]repo.QueryOption, 0, 2)
		opts = append(opts, repo.Where(collectQuery.CreatedAt.Gte(startAt)))
		opts = append(opts, repo.Where(collectQuery.CreatedAt.Lt(endAt)))
		var collectList []*models.UserCollect
		collectList, err = t.userCollectRepo.List(ctx, opts...)
		if err != nil {
			return err
		}
		for _, item := range collectList {
			// 非法商品不参与统计。
			if item.GoodsID <= 0 {
				continue
			}
			ensureStat(item.GoodsID).CollectCount++
		}

		cartQuery := t.userCartRepo.Query(ctx).UserCart
		opts = make([]repo.QueryOption, 0, 2)
		opts = append(opts, repo.Where(cartQuery.CreatedAt.Gte(startAt)))
		opts = append(opts, repo.Where(cartQuery.CreatedAt.Lt(endAt)))
		var cartList []*models.UserCart
		cartList, err = t.userCartRepo.List(ctx, opts...)
		if err != nil {
			return err
		}
		for _, item := range cartList {
			// 非法商品不参与统计。
			if item.GoodsID <= 0 {
				continue
			}
			ensureStat(item.GoodsID).CartCount += item.Num
		}

		orderQuery := t.orderInfoRepo.Query(ctx).OrderInfo
		opts = make([]repo.QueryOption, 0, 2)
		opts = append(opts, repo.Where(orderQuery.CreatedAt.Gte(startAt)))
		opts = append(opts, repo.Where(orderQuery.CreatedAt.Lt(endAt)))
		var orderInfoList []*models.OrderInfo
		orderInfoList, err = t.orderInfoRepo.List(ctx, opts...)
		if err != nil {
			return err
		}

		orderIds := make([]int64, 0, len(orderInfoList))
		for _, item := range orderInfoList {
			// 非法订单不参与统计。
			if item.ID <= 0 {
				continue
			}
			orderIds = append(orderIds, item.ID)
		}

		// 存在订单数据时，才继续回查订单商品明细。
		orderGoodsByOrderId := make(map[int64][]*models.OrderGoods)
		// 只有命中订单时，才需要继续回查订单商品明细。
		if len(orderIds) > 0 {
			orderGoodsQuery := t.orderGoodsRepo.Query(ctx).OrderGoods
			opts = make([]repo.QueryOption, 0, 1)
			opts = append(opts, repo.Where(orderGoodsQuery.OrderID.In(orderIds...)))
			var orderGoodsList []*models.OrderGoods
			orderGoodsList, err = t.orderGoodsRepo.List(ctx, opts...)
			if err != nil {
				return err
			}
			for _, item := range orderGoodsList {
				// 非法订单或商品不参与统计。
				if item.OrderID <= 0 || item.GoodsID <= 0 {
					continue
				}
				orderGoodsByOrderId[item.OrderID] = append(orderGoodsByOrderId[item.OrderID], item)
			}
		}

		for _, orderInfo := range orderInfoList {
			goodsList := orderGoodsByOrderId[orderInfo.ID]
			seenGoodsIds := make(map[int64]struct{}, len(goodsList))
			for _, item := range goodsList {
				stat := ensureStat(item.GoodsID)
				// 同一订单下同一商品可能有多条明细，这里只按订单去重累计下单次数。
				if _, ok := seenGoodsIds[item.GoodsID]; !ok {
					seenGoodsIds[item.GoodsID] = struct{}{}
					stat.OrderCount++
				}
			}
		}

		for _, orderInfo := range orderInfoList {
			// 只有支付成功口径的订单才累计支付指标。
			if !pkgUtils.IsPaidOrderStatus(orderInfo.Status) {
				continue
			}
			goodsList := orderGoodsByOrderId[orderInfo.ID]
			seenGoodsIds := make(map[int64]struct{}, len(goodsList))
			for _, item := range goodsList {
				stat := ensureStat(item.GoodsID)
				// 同一支付订单下同一商品只记一次支付订单数。
				if _, ok := seenGoodsIds[item.GoodsID]; !ok {
					seenGoodsIds[item.GoodsID] = struct{}{}
					stat.PayCount++
				}
				stat.PayGoodsNum += item.Num
				stat.PayAmount += item.TotalPayPrice
			}
		}

		list := make([]*models.GoodsStatDay, 0, len(statMap))
		for _, item := range statMap {
			list = append(list, item)
		}
		// 没有统计结果时只保留清理动作，不再写入空数据。
		if len(list) == 0 {
			return nil
		}
		return t.goodsStatDayRepo.BatchCreate(ctx, list)
	})
	if err != nil {
		return []string{err.Error()}, err
	}

	return []string{fmt.Sprintf("商品日统计完成: %s", statDate.Format("2006-01-02"))}, nil
}

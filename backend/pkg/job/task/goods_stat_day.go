package task

import (
	"context"
	"fmt"
	"time"

	_const "shop/pkg/const"

	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"shop/pkg/utils"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/liujitcn/gorm-kit/repository"
)

// goodsStatDayResult 表示商品日统计任务执行结果。
type goodsStatDayResult struct {
	viewEventCount  int
	collectCount    int
	cartCount       int
	orderCount      int
	orderGoodsCount int
	statCount       int
}

// goodsStatDayKey 表示商品日统计的租户与商品聚合键。
type goodsStatDayKey struct {
	tenantID int64
	goodsID  int64
}

// GoodsStatDay 商品日统计任务。
type GoodsStatDay struct {
	tx                 data.Transaction
	goodsStatDayRepo   *data.GoodsStatDayRepository
	goodsInfoRepo      *data.GoodsInfoRepository
	recommendEventRepo *data.RecommendEventRepository
	userCollectRepo    *data.UserCollectRepository
	userCartRepo       *data.UserCartRepository
	orderInfoRepo      *data.OrderInfoRepository
	orderGoodsRepo     *data.OrderGoodsRepository
	ctx                context.Context
}

// NewGoodsStatDay 创建商品日统计任务实例。
func NewGoodsStatDay(
	tx data.Transaction,
	goodsStatDayRepo *data.GoodsStatDayRepository,
	goodsInfoRepo *data.GoodsInfoRepository,
	recommendEventRepo *data.RecommendEventRepository,
	userCollectRepo *data.UserCollectRepository,
	userCartRepo *data.UserCartRepository,
	orderInfoRepo *data.OrderInfoRepository,
	orderGoodsRepo *data.OrderGoodsRepository,
) *GoodsStatDay {
	return &GoodsStatDay{
		tx:                 tx,
		goodsStatDayRepo:   goodsStatDayRepo,
		goodsInfoRepo:      goodsInfoRepo,
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
	log.Info(fmt.Sprintf("Job GoodsStatDay Exec %+v", args))

	statTime, err := parseStatDateArg(args["statDate"])
	if err != nil {
		return []string{err.Error()}, err
	}

	statDate := time.Date(statTime.Year(), statTime.Month(), statTime.Day(), 0, 0, 0, 0, statTime.Location())
	startAt := statDate
	endAt := statDate.AddDate(0, 0, 1)

	result := goodsStatDayResult{}
	err = t.tx.Transaction(t.ctx, func(ctx context.Context) error {
		statQuery := t.goodsStatDayRepo.Query(ctx).GoodsStatDay
		// 统计任务按天全量重算，物理清掉当天所有租户的旧数据再回写。
		_, err = statQuery.WithContext(ctx).Unscoped().Where(statQuery.StatDate.Eq(statDate)).Delete()
		if err != nil {
			return err
		}

		actionQuery := t.recommendEventRepo.Query(ctx).RecommendEvent
		opts := make([]repository.QueryOption, 0, 3)
		opts = append(opts, repository.Where(actionQuery.EventAt.Gte(startAt)))
		opts = append(opts, repository.Where(actionQuery.EventAt.Lt(endAt)))
		// 浏览数统一从推荐事件表里的 VIEW 事件口径汇总。
		opts = append(opts, repository.Where(actionQuery.EventType.Eq(_const.RECOMMEND_EVENT_TYPE_VIEW)))
		var viewList []*models.RecommendEvent
		viewList, err = t.recommendEventRepo.List(ctx, opts...)
		if err != nil {
			return err
		}
		result.viewEventCount = len(viewList)

		collectQuery := t.userCollectRepo.Query(ctx).UserCollect
		opts = make([]repository.QueryOption, 0, 2)
		opts = append(opts, repository.Where(collectQuery.CreatedAt.Gte(startAt)))
		opts = append(opts, repository.Where(collectQuery.CreatedAt.Lt(endAt)))
		var collectList []*models.UserCollect
		collectList, err = t.userCollectRepo.List(ctx, opts...)
		if err != nil {
			return err
		}
		result.collectCount = len(collectList)

		cartQuery := t.userCartRepo.Query(ctx).UserCart
		opts = make([]repository.QueryOption, 0, 2)
		opts = append(opts, repository.Where(cartQuery.CreatedAt.Gte(startAt)))
		opts = append(opts, repository.Where(cartQuery.CreatedAt.Lt(endAt)))
		var cartList []*models.UserCart
		cartList, err = t.userCartRepo.List(ctx, opts...)
		if err != nil {
			return err
		}
		result.cartCount = len(cartList)

		goodsIDSet := make(map[int64]struct{}, len(viewList)+len(collectList)+len(cartList))
		for _, item := range viewList {
			// 非法商品不参与租户归属查询。
			if item.GoodsID > 0 {
				goodsIDSet[item.GoodsID] = struct{}{}
			}
		}
		for _, item := range collectList {
			// 非法商品不参与租户归属查询。
			if item.GoodsID > 0 {
				goodsIDSet[item.GoodsID] = struct{}{}
			}
		}
		for _, item := range cartList {
			// 非法商品不参与租户归属查询。
			if item.GoodsID > 0 {
				goodsIDSet[item.GoodsID] = struct{}{}
			}
		}

		goodsTenantIDMap := make(map[int64]int64, len(goodsIDSet))
		// 行为明细不带租户，统一通过商品主表批量解析租户归属。
		if len(goodsIDSet) > 0 {
			goodsIDs := make([]int64, 0, len(goodsIDSet))
			for goodsID := range goodsIDSet {
				goodsIDs = append(goodsIDs, goodsID)
			}
			goodsQuery := t.goodsInfoRepo.Query(ctx).GoodsInfo
			opts = make([]repository.QueryOption, 0, 2)
			opts = append(opts, repository.Where(goodsQuery.ID.In(goodsIDs...)))
			opts = append(opts, repository.Unscoped())
			var goodsInfoList []*models.GoodsInfo
			goodsInfoList, err = t.goodsInfoRepo.List(ctx, opts...)
			if err != nil {
				return err
			}
			for _, item := range goodsInfoList {
				goodsTenantIDMap[item.ID] = item.TenantID
			}
		}

		statMap := make(map[goodsStatDayKey]*models.GoodsStatDay)
		ensureStat := func(tenantID, goodsID int64) *models.GoodsStatDay {
			key := goodsStatDayKey{tenantID: tenantID, goodsID: goodsID}
			item, ok := statMap[key]
			// 首次出现的租户商品需要先初始化统计对象。
			if !ok {
				item = &models.GoodsStatDay{
					TenantID: tenantID,
					StatDate: statDate,
					GoodsID:  goodsID,
				}
				statMap[key] = item
			}
			return item
		}

		for _, item := range viewList {
			tenantID := goodsTenantIDMap[item.GoodsID]
			// 无法确认租户归属的商品行为不进入租户统计。
			if tenantID <= 0 {
				continue
			}
			ensureStat(tenantID, item.GoodsID).ViewCount++
		}
		for _, item := range collectList {
			tenantID := goodsTenantIDMap[item.GoodsID]
			// 无法确认租户归属的商品行为不进入租户统计。
			if tenantID <= 0 {
				continue
			}
			ensureStat(tenantID, item.GoodsID).CollectCount++
		}
		for _, item := range cartList {
			tenantID := goodsTenantIDMap[item.GoodsID]
			// 无法确认租户归属的商品行为不进入租户统计。
			if tenantID <= 0 {
				continue
			}
			ensureStat(tenantID, item.GoodsID).CartCount += item.Num
		}

		orderQuery := t.orderInfoRepo.Query(ctx).OrderInfo
		opts = make([]repository.QueryOption, 0, 2)
		opts = append(opts, repository.Where(orderQuery.CreatedAt.Gte(startAt)))
		opts = append(opts, repository.Where(orderQuery.CreatedAt.Lt(endAt)))
		var orderInfoList []*models.OrderInfo
		orderInfoList, err = t.orderInfoRepo.List(ctx, opts...)
		if err != nil {
			return err
		}
		result.orderCount = len(orderInfoList)

		orderIDs := make([]int64, 0, len(orderInfoList))
		for _, item := range orderInfoList {
			// 非法订单不参与统计。
			if item.ID <= 0 {
				continue
			}
			orderIDs = append(orderIDs, item.ID)
		}

		orderGoodsByOrderID := make(map[int64][]*models.OrderGoods)
		// 只有命中订单时，才需要继续回查订单商品明细。
		if len(orderIDs) > 0 {
			orderGoodsQuery := t.orderGoodsRepo.Query(ctx).OrderGoods
			opts = make([]repository.QueryOption, 0, 1)
			opts = append(opts, repository.Where(orderGoodsQuery.OrderID.In(orderIDs...)))
			var orderGoodsList []*models.OrderGoods
			orderGoodsList, err = t.orderGoodsRepo.List(ctx, opts...)
			if err != nil {
				return err
			}
			result.orderGoodsCount = len(orderGoodsList)
			for _, item := range orderGoodsList {
				// 非法订单或商品不参与统计。
				if item.OrderID <= 0 || item.GoodsID <= 0 {
					continue
				}
				orderGoodsByOrderID[item.OrderID] = append(orderGoodsByOrderID[item.OrderID], item)
			}
		}

		for _, orderInfo := range orderInfoList {
			goodsList := orderGoodsByOrderID[orderInfo.ID]
			seenGoodsIDs := make(map[int64]struct{}, len(goodsList))
			for _, item := range goodsList {
				stat := ensureStat(orderInfo.TenantID, item.GoodsID)
				// 同一订单下同一商品可能有多条明细，这里只按订单去重累计下单次数。
				if _, ok := seenGoodsIDs[item.GoodsID]; !ok {
					seenGoodsIDs[item.GoodsID] = struct{}{}
					stat.OrderCount++
				}
			}
		}

		for _, orderInfo := range orderInfoList {
			// 只有支付成功口径的订单才累计支付指标。
			if !utils.IsPaidOrderStatus(orderInfo.Status) {
				continue
			}
			goodsList := orderGoodsByOrderID[orderInfo.ID]
			seenGoodsIDs := make(map[int64]struct{}, len(goodsList))
			for _, item := range goodsList {
				stat := ensureStat(orderInfo.TenantID, item.GoodsID)
				// 同一支付订单下同一商品只记一次支付订单数。
				if _, ok := seenGoodsIDs[item.GoodsID]; !ok {
					seenGoodsIDs[item.GoodsID] = struct{}{}
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
		result.statCount = len(list)
		// 没有统计结果时只保留清理动作，不再写入空数据。
		if len(list) == 0 {
			return nil
		}
		return t.goodsStatDayRepo.BatchCreate(ctx, list)
	})
	if err != nil {
		return []string{err.Error()}, err
	}

	return []string{fmt.Sprintf(
		"商品日统计完成: 浏览事件 %d 条，收藏记录 %d 条，购物车记录 %d 条，订单 %d 条，订单商品 %d 条，写入统计 %d 条",
		result.viewEventCount,
		result.collectCount,
		result.cartCount,
		result.orderCount,
		result.orderGoodsCount,
		result.statCount,
	)}, nil
}

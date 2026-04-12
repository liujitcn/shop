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
	tx                       data.Transaction
	goodsStatDayRepo         *data.GoodsStatDayRepo
	recommendGoodsActionRepo *data.RecommendGoodsActionRepo
	userCollectRepo          *data.UserCollectRepo
	userCartRepo             *data.UserCartRepo
	orderInfoRepo            *data.OrderInfoRepo
	orderGoodsRepo           *data.OrderGoodsRepo
	ctx                      context.Context
}

// NewGoodsStatDay 创建商品日统计任务实例。
func NewGoodsStatDay(
	tx data.Transaction,
	goodsStatDayRepo *data.GoodsStatDayRepo,
	recommendGoodsActionRepo *data.RecommendGoodsActionRepo,
	userCollectRepo *data.UserCollectRepo,
	userCartRepo *data.UserCartRepo,
	orderInfoRepo *data.OrderInfoRepo,
	orderGoodsRepo *data.OrderGoodsRepo,
) *GoodsStatDay {
	return &GoodsStatDay{
		tx:                       tx,
		goodsStatDayRepo:         goodsStatDayRepo,
		recommendGoodsActionRepo: recommendGoodsActionRepo,
		userCollectRepo:          userCollectRepo,
		userCartRepo:             userCartRepo,
		orderInfoRepo:            orderInfoRepo,
		orderGoodsRepo:           orderGoodsRepo,
		ctx:                      context.Background(),
	}
}

// Exec 执行商品日统计。
func (t *GoodsStatDay) Exec(args map[string]string) ([]string, error) {
	log.Infof("Job GoodsStatDay Exec %+v", args)

	statTime, err := parseStatDateArg(args["statDate"])
	// 统计日期非法时，直接返回错误避免写入错误日期数据。
	if err != nil {
		return []string{err.Error()}, err
	}

	statDate := time.Date(statTime.Year(), statTime.Month(), statTime.Day(), 0, 0, 0, 0, statTime.Location())
	startAt := statDate
	endAt := statDate.AddDate(0, 0, 1)

	err = t.tx.Transaction(t.ctx, func(ctx context.Context) error {
		goodsStatDayQuery := t.goodsStatDayRepo.Query(ctx).GoodsStatDay
		// 统计任务按天全量重算，先清掉当天旧数据再回写。
		err = t.goodsStatDayRepo.Delete(ctx, repo.Where(goodsStatDayQuery.StatDate.Eq(statDate)))
		// 删除旧统计失败时，终止本次重算避免新旧数据并存。
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

		actionQuery := t.recommendGoodsActionRepo.Query(ctx).RecommendGoodsAction
		actionOpts := make([]repo.QueryOption, 0, 3)
		actionOpts = append(actionOpts, repo.Where(actionQuery.CreatedAt.Gte(startAt)))
		actionOpts = append(actionOpts, repo.Where(actionQuery.CreatedAt.Lt(endAt)))
		actionOpts = append(actionOpts, repo.Where(actionQuery.EventType.Eq(int32(common.RecommendGoodsActionType_VIEW))))
		var viewList []*models.RecommendGoodsAction
		viewList, err = t.recommendGoodsActionRepo.List(ctx, actionOpts...)
		// 浏览行为查询失败时，直接返回错误避免统计结果不完整。
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

		userCollectQuery := t.userCollectRepo.Query(ctx).UserCollect
		collectOpts := make([]repo.QueryOption, 0, 2)
		collectOpts = append(collectOpts, repo.Where(userCollectQuery.CreatedAt.Gte(startAt)))
		collectOpts = append(collectOpts, repo.Where(userCollectQuery.CreatedAt.Lt(endAt)))
		var collectList []*models.UserCollect
		collectList, err = t.userCollectRepo.List(ctx, collectOpts...)
		// 收藏记录查询失败时，直接返回错误避免统计结果不完整。
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

		userCartQuery := t.userCartRepo.Query(ctx).UserCart
		cartOpts := make([]repo.QueryOption, 0, 2)
		cartOpts = append(cartOpts, repo.Where(userCartQuery.CreatedAt.Gte(startAt)))
		cartOpts = append(cartOpts, repo.Where(userCartQuery.CreatedAt.Lt(endAt)))
		var cartList []*models.UserCart
		cartList, err = t.userCartRepo.List(ctx, cartOpts...)
		// 购物车记录查询失败时，直接返回错误避免统计结果不完整。
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

		orderInfoQuery := t.orderInfoRepo.Query(ctx).OrderInfo
		orderInfoOpts := make([]repo.QueryOption, 0, 2)
		orderInfoOpts = append(orderInfoOpts, repo.Where(orderInfoQuery.CreatedAt.Gte(startAt)))
		orderInfoOpts = append(orderInfoOpts, repo.Where(orderInfoQuery.CreatedAt.Lt(endAt)))
		var orderInfoList []*models.OrderInfo
		orderInfoList, err = t.orderInfoRepo.List(ctx, orderInfoOpts...)
		// 下单记录查询失败时，直接返回错误避免统计结果不完整。
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
		if len(orderIds) > 0 {
			orderGoodsQuery := t.orderGoodsRepo.Query(ctx).OrderGoods
			var orderGoodsList []*models.OrderGoods
			orderGoodsList, err = t.orderGoodsRepo.List(ctx, repo.Where(orderGoodsQuery.OrderID.In(orderIds...)))
			// 订单商品明细查询失败时，直接返回错误避免统计结果不完整。
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
			item.Score = float64(item.ViewCount)*1.0 +
				float64(item.CollectCount)*3.0 +
				float64(item.CartCount)*4.0 +
				float64(item.OrderCount)*6.0 +
				float64(item.PayCount)*8.0 +
				float64(item.PayGoodsNum)*1.0 +
				float64(item.PayAmount)/10000.0
			list = append(list, item)
		}
		// 没有统计结果时只保留清理动作，不再写入空数据。
		if len(list) == 0 {
			return nil
		}
		return t.goodsStatDayRepo.BatchCreate(ctx, list)
	})
	// 事务执行失败时，直接返回错误交由任务日志记录。
	if err != nil {
		return []string{err.Error()}, err
	}

	return []string{fmt.Sprintf("商品日统计完成: %s", statDate.Format("2006-01-02"))}, nil
}

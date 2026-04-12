package task

import (
	"context"
	"fmt"
	"time"

	"shop/api/gen/go/common"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/liujitcn/gorm-kit/repo"
)

type recommendGoodsStatDayKey struct {
	scene   int32
	goodsId int64
}

type recommendGoodsPayKey struct {
	requestId string
	goodsId   int64
}

// RecommendGoodsStatDay 推荐商品日统计任务。
type RecommendGoodsStatDay struct {
	tx                        data.Transaction
	recommendGoodsStatDayRepo *data.RecommendGoodsStatDayRepo
	recommendRequestRepo      *data.RecommendRequestRepo
	recommendRequestItemRepo  *data.RecommendRequestItemRepo
	recommendExposureRepo     *data.RecommendExposureRepo
	recommendExposureItemRepo *data.RecommendExposureItemRepo
	recommendGoodsActionRepo  *data.RecommendGoodsActionRepo
	orderGoodsRepo            *data.OrderGoodsRepo
	ctx                       context.Context
}

// NewRecommendGoodsStatDay 创建推荐商品日统计任务实例。
func NewRecommendGoodsStatDay(
	tx data.Transaction,
	recommendGoodsStatDayRepo *data.RecommendGoodsStatDayRepo,
	recommendRequestRepo *data.RecommendRequestRepo,
	recommendRequestItemRepo *data.RecommendRequestItemRepo,
	recommendExposureRepo *data.RecommendExposureRepo,
	recommendExposureItemRepo *data.RecommendExposureItemRepo,
	recommendGoodsActionRepo *data.RecommendGoodsActionRepo,
	orderGoodsRepo *data.OrderGoodsRepo,
) *RecommendGoodsStatDay {
	return &RecommendGoodsStatDay{
		tx:                        tx,
		recommendGoodsStatDayRepo: recommendGoodsStatDayRepo,
		recommendRequestRepo:      recommendRequestRepo,
		recommendRequestItemRepo:  recommendRequestItemRepo,
		recommendExposureRepo:     recommendExposureRepo,
		recommendExposureItemRepo: recommendExposureItemRepo,
		recommendGoodsActionRepo:  recommendGoodsActionRepo,
		orderGoodsRepo:            orderGoodsRepo,
		ctx:                       context.Background(),
	}
}

// Exec 执行推荐商品日统计。
func (t *RecommendGoodsStatDay) Exec(args map[string]string) ([]string, error) {
	log.Infof("Job RecommendGoodsStatDay Exec %+v", args)

	statTime, err := parseStatDateArg(args["statDate"])
	// 统计日期非法时，直接返回错误避免写入错误日期数据。
	if err != nil {
		return []string{err.Error()}, err
	}

	statDate := time.Date(statTime.Year(), statTime.Month(), statTime.Day(), 0, 0, 0, 0, statTime.Location())
	startAt := statDate
	endAt := statDate.AddDate(0, 0, 1)

	err = t.tx.Transaction(t.ctx, func(ctx context.Context) error {
		recommendGoodsStatDayQuery := t.recommendGoodsStatDayRepo.Query(ctx).RecommendGoodsStatDay
		// 统计任务按天全量重算，先清掉当天旧数据再回写。
		err := t.recommendGoodsStatDayRepo.Delete(ctx, repo.Where(recommendGoodsStatDayQuery.StatDate.Eq(statDate)))
		// 删除旧统计失败时，终止本次重算避免新旧数据并存。
		if err != nil {
			return err
		}

		statMap := make(map[recommendGoodsStatDayKey]*models.RecommendGoodsStatDay)
		ensureStat := func(scene int32, goodsId int64) *models.RecommendGoodsStatDay {
			key := recommendGoodsStatDayKey{scene: scene, goodsId: goodsId}
			item, ok := statMap[key]
			// 首次出现的场景商品维度需要先初始化统计对象。
			if !ok {
				item = &models.RecommendGoodsStatDay{
					StatDate: statDate,
					Scene:    scene,
					GoodsID:  goodsId,
				}
				statMap[key] = item
			}
			return item
		}

		recommendRequestQuery := t.recommendRequestRepo.Query(ctx).RecommendRequest
		requestOpts := make([]repo.QueryOption, 0, 2)
		requestOpts = append(requestOpts, repo.Where(recommendRequestQuery.CreatedAt.Gte(startAt)))
		requestOpts = append(requestOpts, repo.Where(recommendRequestQuery.CreatedAt.Lt(endAt)))
		requestList, err := t.recommendRequestRepo.List(ctx, requestOpts...)
		// 推荐请求查询失败时，直接返回错误避免统计结果不完整。
		if err != nil {
			return err
		}

		requestSceneMap := make(map[int64]int32, len(requestList))
		requestRecordIds := make([]int64, 0, len(requestList))
		for _, item := range requestList {
			// 非法请求主表记录直接跳过，避免污染 item 明细查询条件。
			if item.ID <= 0 {
				continue
			}
			requestSceneMap[item.ID] = item.Scene
			requestRecordIds = append(requestRecordIds, item.ID)
		}
		// 请求主记录存在时，再读取逐商品明细累计请求次数。
		if len(requestRecordIds) > 0 {
			recommendRequestItemQuery := t.recommendRequestItemRepo.Query(ctx).RecommendRequestItem
			requestItemList, err := t.recommendRequestItemRepo.List(ctx, repo.Where(recommendRequestItemQuery.RecommendRequestID.In(requestRecordIds...)))
			// 推荐请求逐商品明细查询失败时，直接返回错误避免统计结果不完整。
			if err != nil {
				return err
			}
			for _, item := range requestItemList {
				scene, ok := requestSceneMap[item.RecommendRequestID]
				// 逐商品明细无法匹配主表场景或商品非法时，直接跳过。
				if !ok || item.GoodsID <= 0 {
					continue
				}
				ensureStat(scene, item.GoodsID).RequestCount++
			}
		}

		recommendExposureQuery := t.recommendExposureRepo.Query(ctx).RecommendExposure
		exposureOpts := make([]repo.QueryOption, 0, 2)
		exposureOpts = append(exposureOpts, repo.Where(recommendExposureQuery.CreatedAt.Gte(startAt)))
		exposureOpts = append(exposureOpts, repo.Where(recommendExposureQuery.CreatedAt.Lt(endAt)))
		exposureList, err := t.recommendExposureRepo.List(ctx, exposureOpts...)
		// 推荐曝光查询失败时，直接返回错误避免统计结果不完整。
		if err != nil {
			return err
		}

		exposureSceneMap := make(map[int64]int32, len(exposureList))
		exposureIds := make([]int64, 0, len(exposureList))
		for _, item := range exposureList {
			// 非法曝光主表记录直接跳过，避免污染 item 明细查询条件。
			if item.ID <= 0 {
				continue
			}
			exposureSceneMap[item.ID] = item.Scene
			exposureIds = append(exposureIds, item.ID)
		}
		// 曝光主记录存在时，再读取逐商品明细累计曝光次数。
		if len(exposureIds) > 0 {
			recommendExposureItemQuery := t.recommendExposureItemRepo.Query(ctx).RecommendExposureItem
			exposureItemList, err := t.recommendExposureItemRepo.List(ctx, repo.Where(recommendExposureItemQuery.RecommendExposureID.In(exposureIds...)))
			// 推荐曝光逐商品明细查询失败时，直接返回错误避免统计结果不完整。
			if err != nil {
				return err
			}
			for _, item := range exposureItemList {
				scene, ok := exposureSceneMap[item.RecommendExposureID]
				// 逐商品明细无法匹配主表场景或商品非法时，直接跳过。
				if !ok || item.GoodsID <= 0 {
					continue
				}
				ensureStat(scene, item.GoodsID).ExposureCount++
			}
		}

		recommendGoodsActionQuery := t.recommendGoodsActionRepo.Query(ctx).RecommendGoodsAction
		actionOpts := make([]repo.QueryOption, 0, 2)
		actionOpts = append(actionOpts, repo.Where(recommendGoodsActionQuery.CreatedAt.Gte(startAt)))
		actionOpts = append(actionOpts, repo.Where(recommendGoodsActionQuery.CreatedAt.Lt(endAt)))
		actionList, err := t.recommendGoodsActionRepo.List(ctx, actionOpts...)
		// 推荐行为查询失败时，直接返回错误避免统计结果不完整。
		if err != nil {
			return err
		}

		requestIds := make([]string, 0, len(actionList))
		for _, item := range actionList {
			// 只有支付事件需要回查订单商品金额。
			if item.EventType != int32(common.RecommendGoodsActionType_ORDER_PAY) || item.RequestID == "" {
				continue
			}
			requestIds = append(requestIds, item.RequestID)
		}

		payAmountMap := make(map[recommendGoodsPayKey]int64)
		// 存在支付请求时，才继续回查订单商品支付金额。
		if len(requestIds) > 0 {
			orderGoodsQuery := t.orderGoodsRepo.Query(ctx).OrderGoods
			orderGoodsList, err := t.orderGoodsRepo.List(ctx, repo.Where(orderGoodsQuery.RequestID.In(requestIds...)))
			// 订单商品明细查询失败时，直接返回错误避免支付金额统计失真。
			if err != nil {
				return err
			}
			for _, item := range orderGoodsList {
				// 非法请求或商品不参与统计。
				if item.RequestID == "" || item.GoodsID <= 0 {
					continue
				}
				key := recommendGoodsPayKey{requestId: item.RequestID, goodsId: item.GoodsID}
				payAmountMap[key] += item.TotalPayPrice
			}
		}

		for _, item := range actionList {
			// 非法商品不参与统计。
			if item.GoodsID <= 0 {
				continue
			}
			stat := ensureStat(item.Scene, item.GoodsID)
			eventType := common.RecommendGoodsActionType(item.EventType)
			// 按行为事件类型分别累计推荐链路指标。
			switch eventType {
			case common.RecommendGoodsActionType_CLICK:
				// 点击事件只累计点击次数。
				stat.ClickCount++
			case common.RecommendGoodsActionType_VIEW:
				// 浏览事件只累计浏览次数。
				stat.ViewCount++
			case common.RecommendGoodsActionType_COLLECT:
				// 收藏事件只累计收藏次数。
				stat.CollectCount++
			case common.RecommendGoodsActionType_ADD_CART:
				// 加购事件累计商品数量，保持和历史口径一致。
				stat.CartCount += item.GoodsNum
			case common.RecommendGoodsActionType_ORDER_CREATE:
				// 下单事件累计下单次数。
				stat.OrderCount++
			case common.RecommendGoodsActionType_ORDER_PAY:
				// 支付事件累计支付次数、件数和金额。
				stat.PayCount++
				stat.PayGoodsNum += item.GoodsNum
				stat.PayAmount += payAmountMap[recommendGoodsPayKey{requestId: item.RequestID, goodsId: item.GoodsID}]
			default:
				// 其他事件当前不参与推荐统计。
				continue
			}
		}

		list := make([]*models.RecommendGoodsStatDay, 0, len(statMap))
		for _, item := range statMap {
			item.Score = float64(item.ExposureCount)*0.5 +
				float64(item.ClickCount)*2.0 +
				float64(item.ViewCount)*2.0 +
				float64(item.CollectCount)*4.0 +
				float64(item.CartCount)*6.0 +
				float64(item.OrderCount)*8.0 +
				float64(item.PayCount)*10.0 +
				float64(item.PayGoodsNum)*1.0 +
				float64(item.PayAmount)/10000.0
			list = append(list, item)
		}
		// 没有统计结果时只保留清理动作，不再写入空数据。
		if len(list) == 0 {
			return nil
		}
		return t.recommendGoodsStatDayRepo.BatchCreate(ctx, list)
	})
	// 事务执行失败时，直接返回错误交由任务日志记录。
	if err != nil {
		return []string{err.Error()}, err
	}

	return []string{fmt.Sprintf("推荐商品日统计完成: %s", statDate.Format("2006-01-02"))}, nil
}

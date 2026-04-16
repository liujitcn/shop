package aggregate

import (
	"context"
	"time"

	"shop/api/gen/go/common"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	recommendCore "shop/pkg/recommend/core"

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

// BuildRecommendGoodsStatDays 按天聚合推荐请求、曝光和行为事实，生成推荐商品统计日快照。
func BuildRecommendGoodsStatDays(
	ctx context.Context,
	statDate time.Time,
	recommendRequestRepo *data.RecommendRequestRepo,
	recommendRequestItemRepo *data.RecommendRequestItemRepo,
	recommendExposureRepo *data.RecommendExposureRepo,
	recommendExposureItemRepo *data.RecommendExposureItemRepo,
	recommendGoodsActionRepo *data.RecommendGoodsActionRepo,
	orderGoodsRepo *data.OrderGoodsRepo,
) ([]*models.RecommendGoodsStatDay, error) {
	startAt := statDate
	endAt := statDate.AddDate(0, 0, 1)
	var err error

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

	var requestSceneMap map[int64]int32
	var requestRecordIds []int64
	requestSceneMap, requestRecordIds, err = loadRecommendRequestSceneMap(ctx, recommendRequestRepo, startAt, endAt)
	if err != nil {
		return nil, err
	}
	if len(requestRecordIds) > 0 {
		err = accumulateRecommendRequestCounts(ctx, recommendRequestItemRepo, requestSceneMap, requestRecordIds, ensureStat)
		if err != nil {
			return nil, err
		}
	}

	var exposureSceneMap map[int64]int32
	var exposureIds []int64
	exposureSceneMap, exposureIds, err = loadRecommendExposureSceneMap(ctx, recommendExposureRepo, startAt, endAt)
	if err != nil {
		return nil, err
	}
	if len(exposureIds) > 0 {
		err = accumulateRecommendExposureCounts(ctx, recommendExposureItemRepo, exposureSceneMap, exposureIds, ensureStat)
		if err != nil {
			return nil, err
		}
	}

	var actionList []*models.RecommendGoodsAction
	actionList, err = loadRecommendGoodsActions(ctx, recommendGoodsActionRepo, startAt, endAt)
	if err != nil {
		return nil, err
	}
	var payAmountMap map[recommendGoodsPayKey]int64
	payAmountMap, err = loadRecommendPayAmountMap(ctx, orderGoodsRepo, actionList)
	if err != nil {
		return nil, err
	}

	for _, item := range actionList {
		// 非法商品不参与统计。
		if item == nil || item.GoodsID <= 0 {
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
		item.Score = calculateRecommendGoodsStatScore(item)
		list = append(list, item)
	}
	return list, nil
}

// loadRecommendRequestSceneMap 读取当天推荐请求主表并返回请求编号到场景的映射。
func loadRecommendRequestSceneMap(ctx context.Context, recommendRequestRepo *data.RecommendRequestRepo, startAt, endAt time.Time) (map[int64]int32, []int64, error) {
	query := recommendRequestRepo.Query(ctx).RecommendRequest
	opts := make([]repo.QueryOption, 0, 2)
	opts = append(opts, repo.Where(query.CreatedAt.Gte(startAt)))
	opts = append(opts, repo.Where(query.CreatedAt.Lt(endAt)))
	var err error
	var requestList []*models.RecommendRequest
	requestList, err = recommendRequestRepo.List(ctx, opts...)
	if err != nil {
		return nil, nil, err
	}

	requestSceneMap := make(map[int64]int32, len(requestList))
	requestRecordIds := make([]int64, 0, len(requestList))
	for _, item := range requestList {
		// 非法请求主表记录直接跳过，避免污染 item 明细查询条件。
		if item == nil || item.ID <= 0 {
			continue
		}
		requestSceneMap[item.ID] = item.Scene
		requestRecordIds = append(requestRecordIds, item.ID)
	}
	return requestSceneMap, requestRecordIds, nil
}

// accumulateRecommendRequestCounts 累计推荐请求逐商品明细中的请求次数。
func accumulateRecommendRequestCounts(ctx context.Context, recommendRequestItemRepo *data.RecommendRequestItemRepo, requestSceneMap map[int64]int32, requestRecordIds []int64, ensureStat func(scene int32, goodsId int64) *models.RecommendGoodsStatDay) error {
	requestItemQuery := recommendRequestItemRepo.Query(ctx).RecommendRequestItem
	opts := make([]repo.QueryOption, 0, 1)
	opts = append(opts, repo.Where(requestItemQuery.RecommendRequestID.In(requestRecordIds...)))
	var err error
	var requestItemList []*models.RecommendRequestItem
	requestItemList, err = recommendRequestItemRepo.List(ctx, opts...)
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
	return nil
}

// loadRecommendExposureSceneMap 读取当天推荐曝光主表并返回曝光编号到场景的映射。
func loadRecommendExposureSceneMap(ctx context.Context, recommendExposureRepo *data.RecommendExposureRepo, startAt, endAt time.Time) (map[int64]int32, []int64, error) {
	query := recommendExposureRepo.Query(ctx).RecommendExposure
	opts := make([]repo.QueryOption, 0, 2)
	opts = append(opts, repo.Where(query.CreatedAt.Gte(startAt)))
	opts = append(opts, repo.Where(query.CreatedAt.Lt(endAt)))
	var err error
	var exposureList []*models.RecommendExposure
	exposureList, err = recommendExposureRepo.List(ctx, opts...)
	if err != nil {
		return nil, nil, err
	}

	exposureSceneMap := make(map[int64]int32, len(exposureList))
	exposureIds := make([]int64, 0, len(exposureList))
	for _, item := range exposureList {
		// 非法曝光主表记录直接跳过，避免污染 item 明细查询条件。
		if item == nil || item.ID <= 0 {
			continue
		}
		exposureSceneMap[item.ID] = item.Scene
		exposureIds = append(exposureIds, item.ID)
	}
	return exposureSceneMap, exposureIds, nil
}

// accumulateRecommendExposureCounts 累计推荐曝光逐商品明细中的曝光次数。
func accumulateRecommendExposureCounts(ctx context.Context, recommendExposureItemRepo *data.RecommendExposureItemRepo, exposureSceneMap map[int64]int32, exposureIds []int64, ensureStat func(scene int32, goodsId int64) *models.RecommendGoodsStatDay) error {
	exposureItemQuery := recommendExposureItemRepo.Query(ctx).RecommendExposureItem
	opts := make([]repo.QueryOption, 0, 1)
	opts = append(opts, repo.Where(exposureItemQuery.RecommendExposureID.In(exposureIds...)))
	var err error
	var exposureItemList []*models.RecommendExposureItem
	exposureItemList, err = recommendExposureItemRepo.List(ctx, opts...)
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
	return nil
}

// loadRecommendGoodsActions 读取当天推荐商品行为事实。
func loadRecommendGoodsActions(ctx context.Context, recommendGoodsActionRepo *data.RecommendGoodsActionRepo, startAt, endAt time.Time) ([]*models.RecommendGoodsAction, error) {
	query := recommendGoodsActionRepo.Query(ctx).RecommendGoodsAction
	opts := make([]repo.QueryOption, 0, 2)
	opts = append(opts, repo.Where(query.CreatedAt.Gte(startAt)))
	opts = append(opts, repo.Where(query.CreatedAt.Lt(endAt)))
	return recommendGoodsActionRepo.List(ctx, opts...)
}

// loadRecommendPayAmountMap 读取支付行为对应的订单商品支付金额。
func loadRecommendPayAmountMap(ctx context.Context, orderGoodsRepo *data.OrderGoodsRepo, actionList []*models.RecommendGoodsAction) (map[recommendGoodsPayKey]int64, error) {
	requestIds := make([]string, 0, len(actionList))
	for _, item := range actionList {
		// 只有支付事件需要回查订单商品金额。
		if item == nil || item.EventType != int32(common.RecommendGoodsActionType_ORDER_PAY) || item.RequestID == "" {
			continue
		}
		requestIds = append(requestIds, item.RequestID)
	}
	requestIds = recommendCore.DedupeStrings(requestIds)
	if len(requestIds) == 0 {
		return map[recommendGoodsPayKey]int64{}, nil
	}

	orderGoodsQuery := orderGoodsRepo.Query(ctx).OrderGoods
	opts := make([]repo.QueryOption, 0, 1)
	opts = append(opts, repo.Where(orderGoodsQuery.RequestID.In(requestIds...)))
	var err error
	var orderGoodsList []*models.OrderGoods
	orderGoodsList, err = orderGoodsRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	payAmountMap := make(map[recommendGoodsPayKey]int64)
	for _, item := range orderGoodsList {
		// 非法请求或商品不参与统计。
		if item == nil || item.RequestID == "" || item.GoodsID <= 0 {
			continue
		}
		key := recommendGoodsPayKey{requestId: item.RequestID, goodsId: item.GoodsID}
		payAmountMap[key] += item.TotalPayPrice
	}
	return payAmountMap, nil
}

// calculateRecommendGoodsStatScore 按当前固定口径计算推荐商品热度分。
func calculateRecommendGoodsStatScore(item *models.RecommendGoodsStatDay) float64 {
	if item == nil {
		return 0
	}
	return float64(item.ExposureCount)*0.5 +
		float64(item.ClickCount)*2.0 +
		float64(item.ViewCount)*2.0 +
		float64(item.CollectCount)*4.0 +
		float64(item.CartCount)*6.0 +
		float64(item.OrderCount)*8.0 +
		float64(item.PayCount)*10.0 +
		float64(item.PayGoodsNum)*1.0 +
		float64(item.PayAmount)/10000.0
}

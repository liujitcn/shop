package aggregate

import (
	"time"

	"shop/api/gen/go/common"
	"shop/pkg/gen/models"
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
	statDate time.Time,
	requestList []*models.RecommendRequest,
	requestItemList []*models.RecommendRequestItem,
	exposureList []*models.RecommendExposure,
	exposureItemList []*models.RecommendExposureItem,
	actionList []*models.RecommendGoodsAction,
	orderGoodsList []*models.OrderGoods,
) []*models.RecommendGoodsStatDay {
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

	requestSceneMap := buildRecommendRequestSceneMap(requestList)
	for _, item := range requestItemList {
		// 逐商品明细无法匹配主表场景或商品非法时，直接跳过。
		if item == nil || item.GoodsID <= 0 {
			continue
		}
		scene, ok := requestSceneMap[item.RecommendRequestID]
		if !ok {
			continue
		}
		ensureStat(scene, item.GoodsID).RequestCount++
	}

	exposureSceneMap := buildRecommendExposureSceneMap(exposureList)
	for _, item := range exposureItemList {
		// 逐商品明细无法匹配主表场景或商品非法时，直接跳过。
		if item == nil || item.GoodsID <= 0 {
			continue
		}
		scene, ok := exposureSceneMap[item.RecommendExposureID]
		if !ok {
			continue
		}
		ensureStat(scene, item.GoodsID).ExposureCount++
	}

	payAmountMap := buildRecommendPayAmountMap(orderGoodsList)
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
	return list
}

// buildRecommendRequestSceneMap 按推荐请求主表构建请求编号到场景的映射。
func buildRecommendRequestSceneMap(requestList []*models.RecommendRequest) map[int64]int32 {
	requestSceneMap := make(map[int64]int32, len(requestList))
	for _, item := range requestList {
		// 非法请求主表记录直接跳过，避免污染 item 明细查询条件。
		if item == nil || item.ID <= 0 {
			continue
		}
		requestSceneMap[item.ID] = item.Scene
	}
	return requestSceneMap
}

// buildRecommendExposureSceneMap 按推荐曝光主表构建曝光编号到场景的映射。
func buildRecommendExposureSceneMap(exposureList []*models.RecommendExposure) map[int64]int32 {
	exposureSceneMap := make(map[int64]int32, len(exposureList))
	for _, item := range exposureList {
		// 非法曝光主表记录直接跳过，避免污染 item 明细查询条件。
		if item == nil || item.ID <= 0 {
			continue
		}
		exposureSceneMap[item.ID] = item.Scene
	}
	return exposureSceneMap
}

// buildRecommendPayAmountMap 按订单商品列表构建支付金额映射。
func buildRecommendPayAmountMap(orderGoodsList []*models.OrderGoods) map[recommendGoodsPayKey]int64 {
	payAmountMap := make(map[recommendGoodsPayKey]int64)
	for _, item := range orderGoodsList {
		// 非法请求或商品不参与统计。
		if item == nil || item.RequestID == "" || item.GoodsID <= 0 {
			continue
		}
		key := recommendGoodsPayKey{requestId: item.RequestID, goodsId: item.GoodsID}
		payAmountMap[key] += item.TotalPayPrice
	}
	return payAmountMap
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

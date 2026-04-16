package record

import (
	"shop/pkg/gen/models"
	recommendcore "shop/pkg/recommend/core"
)

// BuildRelatedGoodsIds 从推荐请求逐商品明细中提取关联商品编号。
func BuildRelatedGoodsIds(requestItemList []*models.RecommendRequestItem, goodsId int64) []int64 {
	if len(requestItemList) == 0 {
		return []int64{}
	}
	relatedGoodsIds := make([]int64, 0, len(requestItemList))
	for _, item := range requestItemList {
		// 非法商品或当前商品自身都不参与关联商品集合。
		if item == nil || item.GoodsID <= 0 || item.GoodsID == goodsId {
			continue
		}
		relatedGoodsIds = append(relatedGoodsIds, item.GoodsID)
	}
	return recommendcore.DedupeInt64s(relatedGoodsIds)
}

// BuildPositionMap 从推荐请求逐商品明细中构建商品位次映射。
func BuildPositionMap(requestItemList []*models.RecommendRequestItem, goodsIds []int64) map[int64]int32 {
	positionMap := make(map[int64]int32, len(goodsIds))
	if len(requestItemList) == 0 {
		return positionMap
	}
	for _, item := range requestItemList {
		// 非法商品位次明细直接跳过，避免污染曝光位次映射。
		if item == nil || item.GoodsID <= 0 {
			continue
		}
		positionMap[item.GoodsID] = item.Position
	}
	return positionMap
}

package remote

import "strconv"

// buildRecommendGoodsIds 清洗推荐系统返回的原始商品编号列表。
func (r *Recommend) buildRecommendGoodsIds(rawIds []string, limit int64) ([]int64, bool, error) {
	goodsIds := make([]int64, 0, len(rawIds))
	seenGoodsIds := make(map[int64]struct{}, len(rawIds))
	for _, rawId := range rawIds {
		goodsId, err := strconv.ParseInt(rawId, 10, 64)
		// 推荐系统返回了非法商品编号时，直接跳过当前无效值，避免整批结果回退成本地兜底。
		if err != nil {
			continue
		}
		// 返回结果里包含非法商品编号时，直接跳过当前无效值。
		if goodsId <= 0 {
			continue
		}
		// 推荐系统偶发返回重复商品时，仅保留首次命中的结果，避免前端分页出现重复卡片。
		if _, ok := seenGoodsIds[goodsId]; ok {
			continue
		}
		seenGoodsIds[goodsId] = struct{}{}
		goodsIds = append(goodsIds, goodsId)
	}

	hasMore := false
	// 当前结果超过请求上限时，说明远端至少还存在一条后续原始推荐结果。
	if int64(len(goodsIds)) > limit {
		hasMore = true
		goodsIds = goodsIds[:limit]
	}
	return goodsIds, hasMore, nil
}

// buildRecommendPageResult 将推荐系统返回结果转换为项目分页结果。
func (r *Recommend) buildRecommendPageResult(goodsIds []int64, hasMore bool, pageNum, pageSize int64) ([]int64, int64, error) {
	startIndex := (pageNum - 1) * pageSize
	// 当前页起点已经超过已知结果时，仅在仍有后续数据时保留翻页信号。
	if startIndex >= int64(len(goodsIds)) {
		if hasMore {
			return []int64{}, startIndex + 1, nil
		}
		return []int64{}, int64(len(goodsIds)), nil
	}

	endIndex := startIndex + pageSize
	// 已知结果不足一整页时，只截取当前实际存在的数据范围。
	if endIndex > int64(len(goodsIds)) {
		endIndex = int64(len(goodsIds))
	}

	pageGoodsIds := append([]int64(nil), goodsIds[int(startIndex):int(endIndex)]...)
	total := startIndex + int64(len(pageGoodsIds))
	// 当前页后面仍有已知结果，或远端还存在未加载结果时，向前端保留“还有下一页”的信号。
	if int64(len(goodsIds)) > endIndex || hasMore {
		total++
	}
	return pageGoodsIds, total, nil
}

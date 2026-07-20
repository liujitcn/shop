package gorse

import "strconv"

// buildRecommendGoodsIDs 清洗推荐系统返回的原始商品编号列表。
func (r *Recommend) buildRecommendGoodsIDs(rawIDs []string, limit int64) ([]int64, bool, error) {
	goodsIDs := make([]int64, 0, len(rawIDs))
	seenGoodsIDs := make(map[int64]struct{}, len(rawIDs))
	for _, rawID := range rawIDs {
		goodsID, err := strconv.ParseInt(rawID, 10, 64)
		// 推荐系统返回了非法商品编号时，直接跳过当前无效值，避免整批结果回退成本地兜底。
		if err != nil {
			continue
		}
		// 返回结果里包含非法商品编号时，直接跳过当前无效值。
		if goodsID <= 0 {
			continue
		}
		// 推荐系统偶发返回重复商品时，仅保留首次命中的结果，避免前端分页出现重复卡片。
		if _, ok := seenGoodsIDs[goodsID]; ok {
			continue
		}
		seenGoodsIDs[goodsID] = struct{}{}
		goodsIDs = append(goodsIDs, goodsID)
	}

	hasMore := false
	// 当前结果超过请求上限时，说明Gorse至少还存在一条后续原始推荐结果。
	if int64(len(goodsIDs)) > limit {
		hasMore = true
		goodsIDs = goodsIDs[:limit]
	}
	return goodsIDs, hasMore, nil
}

// buildRecommendPageResult 将推荐系统返回结果转换为项目分页结果。
func (r *Recommend) buildRecommendPageResult(goodsIDs []int64, hasMore bool, pageNum, pageSize int64) ([]int64, int64, error) {
	startIndex := (pageNum - 1) * pageSize
	// 当前页起点已经超过已知结果时，仅在仍有后续数据时保留翻页信号。
	if startIndex >= int64(len(goodsIDs)) {
		if hasMore {
			return []int64{}, startIndex + 1, nil
		}
		return []int64{}, int64(len(goodsIDs)), nil
	}

	endIndex := startIndex + pageSize
	// 已知结果不足一整页时，只截取当前实际存在的数据范围。
	if endIndex > int64(len(goodsIDs)) {
		endIndex = int64(len(goodsIDs))
	}

	pageGoodsIDs := append([]int64(nil), goodsIDs[int(startIndex):int(endIndex)]...)
	total := startIndex + int64(len(pageGoodsIDs))
	// 当前页后面仍有已知结果，或Gorse还存在未加载结果时，向前端保留“还有下一页”的信号。
	if int64(len(goodsIDs)) > endIndex || hasMore {
		total++
	}
	return pageGoodsIDs, total, nil
}

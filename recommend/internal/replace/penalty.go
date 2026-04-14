package replace

import "recommend/internal/model"

// ApplyExposurePenalty 应用曝光惩罚。
func ApplyExposurePenalty(candidates []*model.Candidate, penaltyMap map[int64]float64) {
	if len(penaltyMap) == 0 {
		return
	}

	for _, item := range candidates {
		// 空候选或缺失商品实体时，当前候选无法继续参与惩罚计算。
		if item == nil || item.Goods == nil {
			continue
		}
		penalty, ok := penaltyMap[item.Goods.Id]
		if !ok {
			continue
		}
		item.Score.ExposurePenalty += penalty
	}
}

// ApplyRepeatPenalty 应用重复购买惩罚。
func ApplyRepeatPenalty(candidates []*model.Candidate, goodsIds []int64, penalty float64) {
	// 重复购买惩罚列表为空或惩罚值非法时，不做额外处理。
	if len(goodsIds) == 0 || penalty <= 0 {
		return
	}

	goodsIdMap := make(map[int64]struct{}, len(goodsIds))
	for _, goodsId := range goodsIds {
		// 非法商品编号不参与惩罚集合。
		if goodsId <= 0 {
			continue
		}
		goodsIdMap[goodsId] = struct{}{}
	}

	for _, item := range candidates {
		// 空候选或缺失商品实体时，当前候选无法继续参与惩罚计算。
		if item == nil || item.Goods == nil {
			continue
		}
		_, ok := goodsIdMap[item.Goods.Id]
		if ok {
			item.Score.RepeatPenalty += penalty
		}
	}
}

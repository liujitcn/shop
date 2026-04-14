package replace

import "recommend/internal/model"

// FilterUnavailableGoods 过滤下架或无库存商品。
func FilterUnavailableGoods(candidates []*model.Candidate) []*model.Candidate {
	result := make([]*model.Candidate, 0, len(candidates))
	for _, item := range candidates {
		// 空候选或缺失商品实体时，当前候选无法继续参与推荐。
		if item == nil || item.Goods == nil {
			continue
		}
		// 商品未上架时，不能继续参与在线推荐。
		if !item.Goods.OnSale {
			continue
		}
		// 商品无库存时，不能继续参与在线推荐。
		if !item.Goods.InStock {
			continue
		}
		result = append(result, item)
	}
	return result
}

// FilterContextGoods 过滤当前上下文中不应重复推荐的商品。
func FilterContextGoods(request model.Request, candidates []*model.Candidate) []*model.Candidate {
	blockedGoodsIds := make(map[int64]struct{})
	// 商品详情场景需要过滤当前详情商品本身。
	if request.Context.GoodsId > 0 {
		blockedGoodsIds[request.Context.GoodsId] = struct{}{}
	}
	for _, goodsId := range request.Context.CartGoodsIds {
		// 购物车上下文中的商品不应再次重复推荐。
		if goodsId <= 0 {
			continue
		}
		blockedGoodsIds[goodsId] = struct{}{}
	}
	if len(blockedGoodsIds) == 0 {
		return append([]*model.Candidate(nil), candidates...)
	}

	result := make([]*model.Candidate, 0, len(candidates))
	for _, item := range candidates {
		// 空候选或缺失商品实体时，当前候选无法继续参与推荐。
		if item == nil || item.Goods == nil {
			continue
		}
		_, ok := blockedGoodsIds[item.Goods.Id]
		if ok {
			continue
		}
		result = append(result, item)
	}
	return result
}

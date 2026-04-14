package evaluate

import (
	"math"
	"sort"
)

// actionRelevance 返回离线评估使用的正反馈相关性等级。
func actionRelevance(eventType string) (int32, bool) {
	switch eventType {
	// 点击说明推荐结果被用户初步接受。
	case "click":
		return 1, true
	// 收藏与加购说明用户存在更强购买意图。
	case "collect", "add_cart":
		return 2, true
	// 下单说明推荐结果已转化为明确订单意图。
	case "order_create":
		return 3, true
	// 支付是最强成交正反馈。
	case "order_pay":
		return 4, true
	default:
		// 其余行为当前不纳入离线评估正样本。
		return 0, false
	}
}

// dedupeGoodsIds 对推荐结果商品列表按原顺序去重并裁剪 topK。
func dedupeGoodsIds(goodsIds []int64, topK int) []int64 {
	result := make([]int64, 0, len(goodsIds))
	seen := make(map[int64]struct{}, len(goodsIds))
	for _, goodsId := range goodsIds {
		// 非法商品编号不参与排序评估。
		if goodsId <= 0 {
			continue
		}
		if _, ok := seen[goodsId]; ok {
			continue
		}
		seen[goodsId] = struct{}{}
		result = append(result, goodsId)
		// 评估只关心 topK 区间，超过上限的商品无需继续保留。
		if topK > 0 && len(result) >= topK {
			break
		}
	}
	return result
}

// calculateRankingMetrics 计算单个请求的命中数与 NDCG。
func calculateRankingMetrics(goodsIds []int64, positiveGoodsMap map[int64]int32) (int64, float64) {
	hitCount := int64(0)
	dcg := 0.0
	idealRelevanceList := make([]int32, 0, len(positiveGoodsMap))
	for _, relevance := range positiveGoodsMap {
		idealRelevanceList = append(idealRelevanceList, relevance)
	}
	sort.Slice(idealRelevanceList, func(i int, j int) bool {
		return idealRelevanceList[i] > idealRelevanceList[j]
	})

	for index, goodsId := range goodsIds {
		relevance := positiveGoodsMap[goodsId]
		// 当前推荐位没有命中正反馈商品时，不参与 DCG 累计。
		if relevance <= 0 {
			continue
		}
		hitCount++
		dcg += gain(relevance) / math.Log2(float64(index+2))
	}

	idcg := 0.0
	limit := len(goodsIds)
	// 理想排序长度不能超过正反馈商品数量。
	if len(idealRelevanceList) < limit {
		limit = len(idealRelevanceList)
	}
	for index := 0; index < limit; index++ {
		idcg += gain(idealRelevanceList[index]) / math.Log2(float64(index+2))
	}
	// 没有理想增益时，当前请求的 NDCG 直接记为 0。
	if idcg == 0 {
		return hitCount, 0
	}
	return hitCount, dcg / idcg
}

// gain 将相关性等级转换为 NDCG 增益值。
func gain(relevance int32) float64 {
	return math.Pow(2, float64(relevance)) - 1
}

package rank

import (
	"sort"

	app "shop/api/gen/go/app"
	recommendcore "shop/pkg/recommend/core"
)

// ListGoodsIds 提取商品列表中的有效商品编号。
func ListGoodsIds(goodsList []*app.GoodsInfo) []int64 {
	if len(goodsList) == 0 {
		return []int64{}
	}
	result := make([]int64, 0, len(goodsList))
	for _, item := range goodsList {
		// 非法商品不参与结果编号统计。
		if item == nil || item.Id <= 0 {
			continue
		}
		result = append(result, item.Id)
	}
	return recommendcore.DedupeInt64s(result)
}

// BuildPageExplainSnapshot 构建当前页 explain 结果快照。
func BuildPageExplainSnapshot(pageGoods []*app.GoodsInfo, candidates map[int64]*recommendcore.Candidate) PageExplainSnapshot {
	snapshot := PageExplainSnapshot{
		RecallSources:    []string{},
		ScoreDetails:     make([]recommendcore.ScoreDetail, 0, len(pageGoods)),
		ReturnedGoodsIds: ListGoodsIds(pageGoods),
	}
	// 当前页没有商品或 explain 候选为空时，直接返回基础快照。
	if len(pageGoods) == 0 || len(candidates) == 0 {
		return snapshot
	}

	pageRecallSources := make(map[string]struct{}, 8)
	for _, item := range pageGoods {
		// 当前页商品为空或编号非法时，不继续收集 explain 信息。
		if item == nil || item.Id <= 0 {
			continue
		}
		candidate, ok := candidates[item.Id]
		// explain 缺失时，仅跳过当前商品的解释明细，不影响商品结果返回。
		if !ok || candidate == nil || candidate.Goods == nil {
			continue
		}
		recallSources := candidate.RecallSourceList()
		for _, source := range recallSources {
			pageRecallSources[source] = struct{}{}
		}
		snapshot.ScoreDetails = append(snapshot.ScoreDetails, recommendcore.ScoreDetail{
			GoodsId:               candidate.Goods.Id,
			FinalScore:            candidate.FinalScore,
			RelationScore:         candidate.RelationScore,
			UserGoodsScore:        candidate.UserGoodsScore,
			ProfileScore:          candidate.ProfileScore,
			ScenePopularityScore:  candidate.ScenePopularityScore,
			GlobalPopularityScore: candidate.GlobalPopularityScore,
			FreshnessScore:        candidate.FreshnessScore,
			ExposurePenalty:       candidate.ExposurePenalty,
			ActorExposurePenalty:  candidate.ActorExposurePenalty,
			RepeatPenalty:         candidate.RepeatPenalty,
			RecallSources:         recallSources,
		})
	}
	for source := range pageRecallSources {
		snapshot.RecallSources = append(snapshot.RecallSources, source)
	}
	// 当前页召回来源按稳定顺序返回，便于日志和前端比对。
	sort.Strings(snapshot.RecallSources)
	return snapshot
}

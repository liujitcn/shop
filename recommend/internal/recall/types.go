package recall

import (
	"context"
	"recommend"
	"recommend/contract"
	"recommend/internal/model"
	"sort"
	"time"
)

const (
	defaultRecallLimit = 20

	RecallSourceLatest        = "latest"
	RecallSourceSceneHot      = "scene_hot"
	RecallSourceGlobalHot     = "global_hot"
	RecallSourceGoodsRelation = "goods_relation"
	RecallSourceUserGoods     = "user_goods_pref"
	RecallSourceUserCategory  = "user_category_pref"
	RecallSourceSession       = "session_context"
	RecallSourceUserToUser    = "user_to_user"
	RecallSourceCollaborative = "collaborative"
	RecallSourceExternal      = "external"
)

// Request 表示召回器使用的统一请求参数。
type Request struct {
	Scene         model.Scene
	Actor         model.Actor
	Context       model.RequestContext
	Limit         int32
	ReferenceTime time.Time
	Dependencies  recommend.Dependencies
}

// ResolveLimit 返回归一化后的召回数量。
func ResolveLimit(limit int32) int32 {
	// 调用方未显式指定数量时，回退到统一默认值，避免召回范围异常。
	if limit <= 0 {
		return defaultRecallLimit
	}
	return limit
}

// buildWeightedGoodsCandidates 将带分商品结果转换为候选商品。
func buildWeightedGoodsCandidates(
	ctx context.Context,
	goodsSource contract.GoodsSource,
	rows []*contract.WeightedGoods,
	recallSource string,
	assignScore func(candidate *model.Candidate, score float64),
) ([]*model.Candidate, error) {
	goodsIdMap := make(map[int64]float64, len(rows))
	goodsIds := make([]int64, 0, len(rows))

	for _, item := range rows {
		// 非法商品编号不参与候选构建。
		if item == nil || item.GoodsId <= 0 {
			continue
		}
		_, ok := goodsIdMap[item.GoodsId]
		// 相同商品被多次命中时，累计其召回得分。
		if !ok {
			goodsIds = append(goodsIds, item.GoodsId)
		}
		goodsIdMap[item.GoodsId] += item.Score
	}
	if len(goodsIds) == 0 {
		return []*model.Candidate{}, nil
	}

	list, err := goodsSource.ListGoods(ctx, goodsIds)
	if err != nil {
		return nil, err
	}

	candidates := make([]*model.Candidate, 0, len(list))
	for _, item := range list {
		// 商品实体缺失时，不能继续参与推荐候选构建。
		if item == nil || item.Id <= 0 {
			continue
		}
		candidate := model.BuildCandidate(item)
		candidate.AddRecallSource(recallSource)
		assignScore(candidate, goodsIdMap[item.Id])
		candidates = append(candidates, candidate)
	}
	return candidates, nil
}

// buildCategoryCandidates 根据类目偏好结果构建候选商品。
func buildCategoryCandidates(
	ctx context.Context,
	goodsSource contract.GoodsSource,
	rows []*contract.WeightedCategory,
	limit int32,
) ([]*model.Candidate, error) {
	categoryScoreMap := make(map[int64]float64, len(rows))
	categoryIds := make([]int64, 0, len(rows))

	for _, item := range rows {
		// 非法类目编号不参与候选构建。
		if item == nil || item.CategoryId <= 0 {
			continue
		}
		_, ok := categoryScoreMap[item.CategoryId]
		// 相同类目被多次命中时，累计其偏好得分。
		if !ok {
			categoryIds = append(categoryIds, item.CategoryId)
		}
		categoryScoreMap[item.CategoryId] += item.Score
	}
	if len(categoryIds) == 0 {
		return []*model.Candidate{}, nil
	}

	list, err := goodsSource.ListGoodsByCategoryIds(ctx, categoryIds, limit)
	if err != nil {
		return nil, err
	}

	candidates := make([]*model.Candidate, 0, len(list))
	for _, item := range list {
		// 商品实体缺失或类目不匹配时，不继续参与构建。
		if item == nil || item.Id <= 0 {
			continue
		}
		score, ok := categoryScoreMap[item.CategoryId]
		if !ok {
			continue
		}
		candidate := model.BuildCandidate(item)
		candidate.AddRecallSource(RecallSourceUserCategory)
		candidate.Score.CategoryScore = score
		candidates = append(candidates, candidate)
	}
	return candidates, nil
}

// buildSessionWeightedGoods 将会话行为转换为按商品聚合的关联得分。
func buildSessionWeightedGoods(
	ctx context.Context,
	behaviorSource contract.BehaviorSource,
	recommendSource contract.RecommendSource,
	actor model.Actor,
	limit int32,
) ([]*contract.WeightedGoods, error) {
	events, err := behaviorSource.ListSessionEvents(ctx, int32(actor.Type), actor.Id, limit)
	if err != nil {
		return nil, err
	}

	scoreMap := make(map[int64]float64)
	for index, item := range events {
		// 非法会话事件不参与聚合。
		if item == nil || item.GoodsId <= 0 {
			continue
		}
		rows, err := recommendSource.ListRelatedGoods(ctx, item.GoodsId, limit)
		if err != nil {
			return nil, err
		}

		weight := sessionEventWeight(index)
		for _, row := range rows {
			// 非法关联商品不参与会话召回聚合。
			if row == nil || row.GoodsId <= 0 {
				continue
			}
			scoreMap[row.GoodsId] += row.Score * weight
		}
	}

	rows := make([]*contract.WeightedGoods, 0, len(scoreMap))
	for goodsId, score := range scoreMap {
		rows = append(rows, &contract.WeightedGoods{
			GoodsId: goodsId,
			Score:   score,
		})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		// 会话聚合结果优先按聚合得分倒序排序。
		if rows[i].Score != rows[j].Score {
			return rows[i].Score > rows[j].Score
		}
		return rows[i].GoodsId < rows[j].GoodsId
	})

	normalizedLimit := int(ResolveLimit(limit))
	// 聚合结果超过上限时，只保留前 N 个商品，避免后续读商品详情过大。
	if len(rows) > normalizedLimit {
		rows = rows[:normalizedLimit]
	}
	return rows, nil
}

// sessionEventWeight 返回会话事件的衰减权重。
func sessionEventWeight(index int) float64 {
	// 最近事件优先级最高，随着事件位置增大逐步衰减。
	switch {
	case index <= 0:
		return 1
	case index == 1:
		return 0.8
	case index == 2:
		return 0.6
	default:
		return 0.4
	}
}

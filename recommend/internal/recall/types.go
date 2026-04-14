package recall

import (
	"context"
	recommendv1 "recommend/api/gen/go/recommend/v1"
	"recommend/contract"
	cachex "recommend/internal/cache"
	"recommend/internal/core"
	"recommend/internal/model"
	"time"
)

const (
	defaultRecallLimit = 20

	// RecallSourceLatest 表示最新商品召回来源。
	RecallSourceLatest = "latest"
	// RecallSourceSceneHot 表示场景热销召回来源。
	RecallSourceSceneHot = "scene_hot"
	// RecallSourceGlobalHot 表示全站热销召回来源。
	RecallSourceGlobalHot = "global_hot"
	// RecallSourceGoodsRelation 表示商品关联召回来源。
	RecallSourceGoodsRelation = "goods_relation"
	// RecallSourceUserGoods 表示用户商品偏好召回来源。
	RecallSourceUserGoods = "user_goods_pref"
	// RecallSourceUserCategory 表示用户类目偏好召回来源。
	RecallSourceUserCategory = "user_category_pref"
	// RecallSourceSession 表示会话上下文召回来源。
	RecallSourceSession = "session_context"
	// RecallSourceUserToUser 表示相似用户召回来源。
	RecallSourceUserToUser = "user_to_user"
	// RecallSourceCollaborative 表示协同过滤召回来源。
	RecallSourceCollaborative = "collaborative"
	// RecallSourceExternal 表示外部推荐池召回来源。
	RecallSourceExternal = "external"
	// RecallSourceVector 表示向量召回来源。
	RecallSourceVector = "vector"
)

// Request 表示召回器使用的统一请求参数。
type Request struct {
	Scene         model.Scene
	Actor         model.Actor
	Context       model.RequestContext
	Limit         int32
	ReferenceTime time.Time
	Dependencies  core.Dependencies
	PoolStore     *cachex.PoolStore
	RuntimeStore  *cachex.RuntimeStore
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

// buildPoolGoodsCandidates 将缓存池中的候选商品项转换为候选商品。
func buildPoolGoodsCandidates(
	ctx context.Context,
	goodsSource contract.GoodsSource,
	items []*recommendv1.RecommendCandidateItem,
	recallSource string,
	assignScore func(candidate *model.Candidate, score float64),
) ([]*model.Candidate, error) {
	scoreByGoodsId := make(map[int64]float64, len(items))
	goodsIds := make([]int64, 0, len(items))

	for _, item := range items {
		// 非法商品编号不参与缓存候选构建。
		if item == nil || item.GetGoodsId() <= 0 {
			continue
		}
		score, ok := resolvePoolSourceScore(item, recallSource)
		// 需要来源分值时，只有可恢复出指定来源原始得分的候选才允许参与在线召回。
		if assignScore != nil && !ok {
			continue
		}
		// 不需要来源分值时，只要池项标记了该召回来源即可继续使用。
		if assignScore == nil && !poolItemContainsRecallSource(item, recallSource) {
			continue
		}
		if _, ok := scoreByGoodsId[item.GetGoodsId()]; !ok {
			goodsIds = append(goodsIds, item.GetGoodsId())
		}
		scoreByGoodsId[item.GetGoodsId()] += score
	}
	if len(goodsIds) == 0 {
		return []*model.Candidate{}, nil
	}

	list, err := goodsSource.ListGoods(ctx, goodsIds)
	if err != nil {
		return nil, err
	}

	candidates := make([]*model.Candidate, 0, len(list))
	for _, goods := range list {
		// 商品实体缺失时，不能继续参与在线候选构建。
		if goods == nil || goods.Id <= 0 {
			continue
		}
		candidate := model.BuildCandidate(goods)
		candidate.AddRecallSource(recallSource)
		if assignScore != nil {
			assignScore(candidate, scoreByGoodsId[goods.Id])
		}
		candidates = append(candidates, candidate)
	}
	return candidates, nil
}

// poolItemContainsRecallSource 判断缓存池商品项是否包含指定召回来源。
func poolItemContainsRecallSource(item *recommendv1.RecommendCandidateItem, recallSource string) bool {
	if item == nil || recallSource == "" {
		return false
	}
	for _, source := range item.GetRecallSources() {
		// 候选池已明确记录该来源时，可直接判定当前来源命中。
		if source == recallSource {
			return true
		}
	}
	return false
}

// resolvePoolSourceScore 解析缓存池商品项中指定来源的原始得分。
func resolvePoolSourceScore(item *recommendv1.RecommendCandidateItem, recallSource string) (float64, bool) {
	if item == nil || recallSource == "" {
		return 0, false
	}
	score, ok := item.GetSourceScores()[recallSource]
	if ok {
		return score, true
	}
	// 旧版本单路池只保存总分时，允许把总分回退为该来源原始得分。
	if poolItemContainsRecallSource(item, recallSource) && len(item.GetRecallSources()) == 1 {
		return item.GetScore(), true
	}
	return 0, false
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
	// 会话行为源或推荐事实源缺失时，当前请求无法继续构造会话召回。
	if behaviorSource == nil || recommendSource == nil {
		return nil, nil
	}

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

	return finalizeSessionWeightedGoods(scoreMap, limit), nil
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

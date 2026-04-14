package recall

import (
	"context"
	"errors"
	recommendv1 "recommend/api/gen/go/recommend/v1"
	"recommend/contract"
	"recommend/internal/cache"
	"recommend/internal/model"
	"sort"

	goleveldb "github.com/syndtr/goleveldb/leveldb"
)

const (
	// sessionCartWeight 表示加购行为在会话召回中的权重。
	sessionCartWeight = 1.2
	// sessionClickWeight 表示点击行为在会话召回中的权重。
	sessionClickWeight = 1.0
	// sessionViewWeight 表示浏览行为在会话召回中的权重。
	sessionViewWeight = 0.8
)

// RecallSessionContext 基于当前主体最近会话行为召回上下文商品。
func RecallSessionContext(ctx context.Context, request Request) ([]*model.Candidate, error) {
	rows, err := loadRuntimeSessionWeightedGoods(ctx, request)
	if err != nil {
		return nil, err
	}
	// 运行态未命中会话数据时，再回退到行为事实源，兼容旧接入方式。
	if len(rows) == 0 {
		rows, err = buildSessionWeightedGoods(ctx, request.Dependencies.Behavior, request.Dependencies.Recommend, request.Actor, ResolveLimit(request.Limit))
		if err != nil {
			return nil, err
		}
	}
	return buildWeightedGoodsCandidates(ctx, request.Dependencies.Goods, rows, RecallSourceSession, func(candidate *model.Candidate, score float64) {
		candidate.Score.SessionScore = score
	})
}

// loadRuntimeSessionWeightedGoods 从运行态会话缓存恢复 session_context 召回输入。
func loadRuntimeSessionWeightedGoods(ctx context.Context, request Request) ([]*contract.WeightedGoods, error) {
	state, err := loadRuntimeSessionState(request.RuntimeStore, request.Actor)
	if err != nil {
		return nil, err
	}
	// 未配置或未命中运行态会话缓存时，交由行为事实源继续兜底。
	if state == nil || !hasSessionStateGoods(state) {
		return nil, nil
	}
	return buildSessionWeightedGoodsFromState(ctx, request.Dependencies.Recommend, state, ResolveLimit(request.Limit))
}

// loadRuntimeSessionState 优先读取具体会话槽位，再回退到主体级共享会话态。
func loadRuntimeSessionState(runtimeStore *cache.RuntimeStore, actor model.Actor) (*recommendv1.RecommendSessionState, error) {
	if runtimeStore == nil {
		return nil, nil
	}
	// 主体编号和会话编号都缺失时，无法构造有效运行态键。
	if actor.Id <= 0 && actor.SessionId == "" {
		return nil, nil
	}

	var state *recommendv1.RecommendSessionState
	var err error
	if actor.SessionId != "" {
		state, err = runtimeStore.GetSessionState(int32(actor.Type), actor.Id, actor.SessionId)
		if err == nil && hasSessionStateGoods(state) {
			return state, nil
		}
		if err != nil && !errors.Is(err, goleveldb.ErrNotFound) {
			return nil, err
		}
	}

	state, err = runtimeStore.GetSessionState(int32(actor.Type), actor.Id, "")
	if err != nil {
		// 共享会话态不存在时，按未命中处理，由行为事实源兜底。
		if errors.Is(err, goleveldb.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if !hasSessionStateGoods(state) {
		return nil, nil
	}
	return state, nil
}

// hasSessionStateGoods 判断运行态会话缓存中是否存在可用于召回的行为序列。
func hasSessionStateGoods(state *recommendv1.RecommendSessionState) bool {
	if state == nil {
		return false
	}
	return len(state.GetRecentCartGoodsIds()) > 0 ||
		len(state.GetRecentClickGoodsIds()) > 0 ||
		len(state.GetRecentViewGoodsIds()) > 0
}

// buildSessionWeightedGoodsFromState 将运行态会话序列转换为加权商品结果。
func buildSessionWeightedGoodsFromState(
	ctx context.Context,
	recommendSource contract.RecommendSource,
	state *recommendv1.RecommendSessionState,
	limit int32,
) ([]*contract.WeightedGoods, error) {
	if recommendSource == nil || state == nil {
		return nil, nil
	}

	scoreMap := make(map[int64]float64)
	err := mergeSessionStateWeightedGoods(ctx, recommendSource, state.GetRecentCartGoodsIds(), limit, sessionCartWeight, scoreMap)
	if err != nil {
		return nil, err
	}
	err = mergeSessionStateWeightedGoods(ctx, recommendSource, state.GetRecentClickGoodsIds(), limit, sessionClickWeight, scoreMap)
	if err != nil {
		return nil, err
	}
	err = mergeSessionStateWeightedGoods(ctx, recommendSource, state.GetRecentViewGoodsIds(), limit, sessionViewWeight, scoreMap)
	if err != nil {
		return nil, err
	}
	return finalizeSessionWeightedGoods(scoreMap, limit), nil
}

// mergeSessionStateWeightedGoods 将一组会话商品映射为关联商品得分并合并到总分表。
func mergeSessionStateWeightedGoods(
	ctx context.Context,
	recommendSource contract.RecommendSource,
	goodsIds []int64,
	limit int32,
	sourceWeight float64,
	scoreMap map[int64]float64,
) error {
	if recommendSource == nil || len(goodsIds) == 0 || sourceWeight <= 0 {
		return nil
	}

	for index, goodsId := range goodsIds {
		// 非法商品编号不参与运行态会话召回计算。
		if goodsId <= 0 {
			continue
		}
		rows, err := recommendSource.ListRelatedGoods(ctx, goodsId, limit)
		if err != nil {
			return err
		}

		weight := sessionEventWeight(index) * sourceWeight
		for _, row := range rows {
			// 非法关联商品不参与运行态会话召回聚合。
			if row == nil || row.GoodsId <= 0 {
				continue
			}
			scoreMap[row.GoodsId] += row.Score * weight
		}
	}
	return nil
}

// finalizeSessionWeightedGoods 将会话聚合分值转为稳定有序的商品列表。
func finalizeSessionWeightedGoods(scoreMap map[int64]float64, limit int32) []*contract.WeightedGoods {
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
	return rows
}

package recall

import (
	"context"
	"errors"
	"recommend/contract"
	"recommend/internal/core"
	"recommend/internal/model"
	"sort"

	goleveldb "github.com/syndtr/goleveldb/leveldb"
)

const (
	// vectorTargetTypeActor 表示使用主体画像做向量查询。
	vectorTargetTypeActor = "actor"
	// vectorTargetTypeGoods 表示使用商品锚点做向量查询。
	vectorTargetTypeGoods = "goods"
)

// RecallVector 召回向量检索结果。
func RecallVector(ctx context.Context, request Request, config core.VectorConfig) ([]*model.Candidate, error) {
	// 未启用向量召回时，直接返回空结果，避免额外查询外部向量服务。
	if !config.Enabled {
		return []*model.Candidate{}, nil
	}
	targetType, targetIds := resolveVectorTargets(request)
	if targetType == "" || len(targetIds) == 0 {
		return []*model.Candidate{}, nil
	}

	if request.PoolStore != nil {
		list, hasPool, err := recallVectorFromPools(ctx, request, targetType, targetIds)
		if err != nil {
			return nil, err
		}
		// 只要离线池已经构建出可用结果，就优先消费缓存结果，避免每次实时打向量服务。
		if hasPool && len(list) > 0 {
			return list, nil
		}
	}

	// 未提供向量数据源时，向量召回只保留离线池能力。
	if request.Dependencies.Vector == nil {
		return []*model.Candidate{}, nil
	}
	rows, err := request.Dependencies.Vector.ListVectorGoods(ctx, buildVectorRecallRequest(request, config, targetIds))
	if err != nil {
		return nil, err
	}
	return buildWeightedGoodsCandidates(ctx, request.Dependencies.Goods, rows, RecallSourceVector, func(candidate *model.Candidate, score float64) {
		candidate.Score.VectorScore = score
	})
}

// buildVectorRecallRequest 构建向量数据源使用的查询条件。
func buildVectorRecallRequest(request Request, config core.VectorConfig, targetIds []int64) contract.VectorRecallRequest {
	return contract.VectorRecallRequest{
		Scene:          request.Scene.String(),
		ActorType:      int32(request.Actor.Type),
		ActorId:        request.Actor.Id,
		SessionId:      request.Actor.SessionId,
		SourceGoodsIds: append([]int64(nil), targetIds...),
		Limit:          resolveVectorLimit(config, request.Limit),
		Attributes:     cloneAttributes(request.Context.Attributes),
	}
}

// resolveVectorLimit 解析向量召回使用的候选上限。
func resolveVectorLimit(config core.VectorConfig, requestLimit int32) int32 {
	if config.RecallLimit > 0 {
		return config.RecallLimit
	}
	return ResolveLimit(requestLimit)
}

// resolveVectorTargets 解析当前请求的向量查询目标。
func resolveVectorTargets(request Request) (string, []int64) {
	if request.Context.GoodsId > 0 {
		return vectorTargetTypeGoods, []int64{request.Context.GoodsId}
	}
	if len(request.Context.CartGoodsIds) > 0 {
		return vectorTargetTypeGoods, dedupePositiveIds(request.Context.CartGoodsIds)
	}
	if request.Actor.Id > 0 {
		return vectorTargetTypeActor, []int64{request.Actor.Id}
	}
	return "", nil
}

// recallVectorFromPools 从离线向量池中恢复候选结果。
func recallVectorFromPools(ctx context.Context, request Request, targetType string, targetIds []int64) ([]*model.Candidate, bool, error) {
	candidateMap := make(map[int64]*model.Candidate)
	foundAnyPool := false

	for _, targetId := range targetIds {
		pool, err := request.PoolStore.GetVectorPool(request.Scene.String(), targetType, targetId)
		if err != nil {
			if errors.Is(err, goleveldb.ErrNotFound) {
				continue
			}
			return nil, false, err
		}
		foundAnyPool = true

		list, err := buildPoolGoodsCandidates(ctx, request.Dependencies.Goods, pool.GetItems(), RecallSourceVector, func(candidate *model.Candidate, score float64) {
			candidate.Score.VectorScore = score
		})
		if err != nil {
			return nil, false, err
		}
		mergeVectorCandidates(candidateMap, list)
	}
	return buildSortedVectorCandidates(candidateMap), foundAnyPool, nil
}

// mergeVectorCandidates 合并多个向量池返回的商品结果。
func mergeVectorCandidates(target map[int64]*model.Candidate, list []*model.Candidate) {
	for _, item := range list {
		// 空候选或缺失商品编号时，不参与向量召回聚合。
		if item == nil || item.Goods == nil || item.Goods.Id <= 0 {
			continue
		}
		existing, ok := target[item.Goods.Id]
		if !ok {
			target[item.Goods.Id] = item
			continue
		}
		existing.Score.VectorScore += item.Score.VectorScore
		existing.AddRecallSource(RecallSourceVector)
	}
}

// buildSortedVectorCandidates 将聚合后的向量候选 map 转成稳定列表。
func buildSortedVectorCandidates(candidateMap map[int64]*model.Candidate) []*model.Candidate {
	result := make([]*model.Candidate, 0, len(candidateMap))
	for _, item := range candidateMap {
		if item == nil {
			continue
		}
		result = append(result, item)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Score.VectorScore != result[j].Score.VectorScore {
			return result[i].Score.VectorScore > result[j].Score.VectorScore
		}
		return result[i].GoodsId() < result[j].GoodsId()
	})
	return result
}

// cloneAttributes 复制扩展属性，避免 provider 修改原始请求上下文。
func cloneAttributes(attributes map[string]string) map[string]string {
	if len(attributes) == 0 {
		return nil
	}
	result := make(map[string]string, len(attributes))
	for key, value := range attributes {
		result[key] = value
	}
	return result
}

// dedupePositiveIds 对正整数编号做去重。
func dedupePositiveIds(ids []int64) []int64 {
	if len(ids) == 0 {
		return nil
	}
	result := make([]int64, 0, len(ids))
	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

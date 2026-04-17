package cache

import (
	"context"
	"strconv"
)

// LoadInt64ScoreMap 按商品编号列表加载排序型缓存中的分数映射。
func LoadInt64ScoreMap(ctx context.Context, store Store, collection string, subset string, ids []int64) (map[int64]float64, bool, error) {
	scoreMap := make(map[int64]float64, len(ids))
	// 存储未初始化、集合为空、子集合为空或目标编号为空时，不继续读取缓存。
	if store == nil || collection == "" || subset == "" || len(ids) == 0 {
		return scoreMap, false, nil
	}

	collectionKey := CollectionKey(collection)
	_, err := store.SearchScores(ctx, collectionKey, subset, 0, 1)
	if err != nil {
		// 子集合不存在时，直接回退为未命中，不视为异常。
		if err == ErrObjectNotExist {
			return scoreMap, false, nil
		}
		return nil, false, err
	}

	scoreHashKey := scoreHashKey(collectionKey, subset)
	normalizedIds := dedupeInt64s(ids)
	for _, id := range normalizedIds {
		// 非法编号不参与缓存读取。
		if id <= 0 {
			continue
		}
		payload, getErr := store.HGet(scoreHashKey, strconv.FormatInt(id, 10))
		if getErr != nil {
			// 单个条目缺失时，继续读取其它商品分数。
			if getErr == ErrObjectNotExist {
				continue
			}
			return nil, true, getErr
		}
		document, unmarshalErr := unmarshalScoreDocument(payload)
		if unmarshalErr != nil {
			return nil, true, unmarshalErr
		}
		// 隐藏条目不向读取侧返回，避免污染排序阶段。
		if document.IsHidden {
			continue
		}
		documentId, parseErr := strconv.ParseInt(document.Id, 10, 64)
		if parseErr != nil || documentId <= 0 {
			continue
		}
		scoreMap[documentId] = document.Score
	}
	return scoreMap, true, nil
}

// dedupeInt64s 对编号列表做稳定去重。
func dedupeInt64s(ids []int64) []int64 {
	if len(ids) == 0 {
		return []int64{}
	}
	result := make([]int64, 0, len(ids))
	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

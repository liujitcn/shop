package cache

import (
	"context"
	"strconv"
	"time"

	recommendDomain "shop/pkg/recommend/domain"

	"github.com/go-kratos/kratos/v2/log"
)

// ReadInt64Ids 读取指定缓存子集合中的编号列表。
func ReadInt64Ids(
	ctx context.Context,
	store Store,
	collection string,
	subset string,
	hitSource string,
	version string,
	versionPublishedAt time.Time,
	limit int64,
	excludeIds []int64,
) (*recommendDomain.CacheReadResult, error) {
	result := NewReadResult(hitSource, collection, subset, version, versionPublishedAt, limit, len(excludeIds))
	// 限制数量非法时，不需要继续读取缓存。
	if limit <= 0 {
		result.ReadContext["skipped"] = true
		result.ReadContext["skipReason"] = "invalid_limit"
		return result, nil
	}
	// 存储未初始化时，直接返回跳过结果，避免主链路空指针。
	if store == nil {
		result.ReadContext["skipped"] = true
		result.ReadContext["skipReason"] = "store_not_configured"
		return result, nil
	}

	collectionKey := CollectionKey(collection)
	searchEnd := int(limit) * 3
	// 排除列表较大时，扩大读取窗口，避免命中过滤后结果不足。
	if searchEnd < int(limit)+len(excludeIds) {
		searchEnd = int(limit) + len(excludeIds)
	}
	result.ReadContext["searchWindow"] = searchEnd
	documents, err := store.SearchScores(ctx, collectionKey, subset, 0, searchEnd)
	if err != nil {
		// 缓存对象不存在时直接回退查库，不把未命中当作异常。
		if err == ErrObjectNotExist {
			result.ReadContext["missReason"] = "object_not_exist"
			return result, nil
		}
		return nil, err
	}
	result.ReadContext["subsetExists"] = true
	result.ReadContext["scannedCount"] = len(documents)
	fillCacheReadMeta(store, result, collection, subset)

	excludeIdMap := make(map[int64]struct{}, len(excludeIds))
	for _, itemId := range excludeIds {
		excludeIdMap[itemId] = struct{}{}
	}

	int64Ids := make([]int64, 0, limit)
	for _, item := range documents {
		itemId, parseErr := strconv.ParseInt(item.Id, 10, 64)
		// 缓存条目编号非法时，直接跳过异常条目。
		if parseErr != nil || itemId <= 0 {
			continue
		}
		_, exists := excludeIdMap[itemId]
		// 已在排除集合中的编号不再重复返回。
		if exists {
			continue
		}
		int64Ids = append(int64Ids, itemId)
		// 已满足读取数量时，直接结束缓存扫描。
		if int64(len(int64Ids)) >= limit {
			break
		}
	}
	result.Ids = int64Ids
	result.ReadContext["returnedCount"] = len(int64Ids)
	// 当前读取到了有效缓存结果时，补一条命中日志用于后续排查。
	if len(int64Ids) > 0 {
		result.ReadContext["hit"] = true
		log.Infof("recommend cache hit source=%s subset=%s count=%d", hitSource, subset, len(int64Ids))
	}
	return result, nil
}

// ReadScoreMap 读取指定缓存子集合中的分数映射。
func ReadScoreMap(
	ctx context.Context,
	store Store,
	collection string,
	subset string,
	hitSource string,
	version string,
	versionPublishedAt time.Time,
	candidateGoodsIds []int64,
) (*recommendDomain.CacheScoreReadResult, error) {
	result := NewScoreReadResult(hitSource, collection, subset, version, versionPublishedAt, int64(len(candidateGoodsIds)))
	// 候选商品为空时，不需要继续读取缓存分数。
	if len(candidateGoodsIds) == 0 {
		result.ReadContext["skipped"] = true
		result.ReadContext["skipReason"] = "empty_candidate_goods"
		return result, nil
	}
	if subset == "" {
		result.ReadContext["skipped"] = true
		result.ReadContext["skipReason"] = "empty_subset"
		return result, nil
	}
	// 存储未初始化时，直接返回跳过结果，避免主链路空指针。
	if store == nil {
		result.ReadContext["skipped"] = true
		result.ReadContext["skipReason"] = "store_not_configured"
		return result, nil
	}

	scoreMap, subsetExists, err := LoadInt64ScoreMap(ctx, store, collection, subset, candidateGoodsIds)
	if err != nil {
		return nil, err
	}
	result.ReadContext["subsetExists"] = subsetExists
	if !subsetExists {
		result.ReadContext["missReason"] = "object_not_exist"
		return result, nil
	}
	fillCacheScoreReadMeta(store, result, collection, subset)
	result.Scores = scoreMap
	result.ReadContext["returnedCount"] = len(scoreMap)
	// 当前读取到了有效缓存结果时，补一条命中日志用于后续排查。
	if len(scoreMap) > 0 {
		result.ReadContext["hit"] = true
		log.Infof("recommend score cache hit source=%s subset=%s count=%d", hitSource, subset, len(scoreMap))
	}
	return result, nil
}

// fillCacheReadMeta 补齐缓存编号读取的元信息。
func fillCacheReadMeta(store Store, result *recommendDomain.CacheReadResult, collection string, subset string) {
	// 结果对象或子集合为空时，不继续读取元信息。
	if store == nil || result == nil || subset == "" {
		return
	}

	documentCount, err := loadCacheDocumentCount(store, collection, subset)
	if err == nil && documentCount >= 0 {
		result.ReadContext["documentCount"] = documentCount
	} else if err != nil {
		// 元信息读取失败时只打日志，不影响主推荐链路。
		log.Errorf("fillCacheReadMeta document_count %v", err)
	}

	cacheUpdatedAt, updateErr := loadCacheUpdateTime(store, collection, subset)
	if updateErr == nil && !cacheUpdatedAt.IsZero() {
		result.ReadContext["cacheUpdatedAt"] = cacheUpdatedAt.Format(time.RFC3339Nano)
	} else if updateErr != nil {
		// 元信息读取失败时只打日志，不影响主推荐链路。
		log.Errorf("fillCacheReadMeta update_time %v", updateErr)
	}

	digest, digestErr := loadCacheDigest(store, collection, subset)
	if digestErr == nil && digest != "" {
		result.ReadContext["digest"] = digest
	} else if digestErr != nil {
		// 元信息读取失败时只打日志，不影响主推荐链路。
		log.Errorf("fillCacheReadMeta digest %v", digestErr)
	}
}

// fillCacheScoreReadMeta 补齐缓存分数读取的元信息。
func fillCacheScoreReadMeta(store Store, result *recommendDomain.CacheScoreReadResult, collection string, subset string) {
	// 结果对象或子集合为空时，不继续读取元信息。
	if store == nil || result == nil || subset == "" {
		return
	}

	documentCount, err := loadCacheDocumentCount(store, collection, subset)
	if err == nil && documentCount >= 0 {
		result.ReadContext["documentCount"] = documentCount
	} else if err != nil {
		// 元信息读取失败时只打日志，不影响主推荐链路。
		log.Errorf("fillCacheScoreReadMeta document_count %v", err)
	}

	cacheUpdatedAt, updateErr := loadCacheUpdateTime(store, collection, subset)
	if updateErr == nil && !cacheUpdatedAt.IsZero() {
		result.ReadContext["cacheUpdatedAt"] = cacheUpdatedAt.Format(time.RFC3339Nano)
	} else if updateErr != nil {
		// 元信息读取失败时只打日志，不影响主推荐链路。
		log.Errorf("fillCacheScoreReadMeta update_time %v", updateErr)
	}

	digest, digestErr := loadCacheDigest(store, collection, subset)
	if digestErr == nil && digest != "" {
		result.ReadContext["digest"] = digest
	} else if digestErr != nil {
		// 元信息读取失败时只打日志，不影响主推荐链路。
		log.Errorf("fillCacheScoreReadMeta digest %v", digestErr)
	}
}

// loadCacheDocumentCount 读取缓存子集合的文档数量。
func loadCacheDocumentCount(store Store, collection string, subset string) (int, error) {
	value, err := store.Get(DocumentCountKey(collection, subset))
	if err != nil {
		// 元信息不存在时，统一回退到未知数量。
		if err == ErrObjectNotExist {
			return -1, nil
		}
		return 0, err
	}
	return strconv.Atoi(value)
}

// loadCacheUpdateTime 读取缓存子集合的发布时间。
func loadCacheUpdateTime(store Store, collection string, subset string) (time.Time, error) {
	value, err := store.Get(UpdateTimeKey(collection, subset))
	if err != nil {
		// 元信息不存在时，统一回退到零时间。
		if err == ErrObjectNotExist {
			return time.Time{}, nil
		}
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339Nano, value)
}

// loadCacheDigest 读取缓存子集合的内容摘要。
func loadCacheDigest(store Store, collection string, subset string) (string, error) {
	value, err := store.Get(DigestKey(collection, subset))
	if err != nil {
		// 元信息不存在时，统一回退到空摘要。
		if err == ErrObjectNotExist {
			return "", nil
		}
		return "", err
	}
	return value, nil
}

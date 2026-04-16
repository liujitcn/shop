package biz

import (
	"context"
	"strconv"
	"time"

	"shop/api/gen/go/common"
	recommendCache "shop/pkg/recommend/cache"

	"github.com/go-kratos/kratos/v2/log"
)

const (
	// recommendCacheHitGoodsDetail 表示商品详情相似商品缓存命中。
	recommendCacheHitGoodsDetail = "goods_detail_cache"
	// recommendCacheHitSceneHot 表示场景热门榜缓存命中。
	recommendCacheHitSceneHot = "scene_hot_cache"
	// recommendCacheHitLatest 表示最新榜缓存命中。
	recommendCacheHitLatest = "latest_cache"
)

// recommendSceneVersionInfo 表示当前场景启用的缓存版本信息。
type recommendSceneVersionInfo struct {
	version     string
	publishedAt time.Time
}

// recommendCacheReadResult 表示一次缓存读取的结果与调试上下文。
type recommendCacheReadResult struct {
	ids         []int64
	readContext map[string]any
}

// loadRecommendSceneVersionInfo 查询当前场景启用的推荐缓存版本信息。
func (c *RecommendRequestCase) loadRecommendSceneVersionInfo(ctx context.Context, scene int32) (*recommendSceneVersionInfo, error) {
	entity, err := c.loadRecommendSceneVersionEntity(ctx, scene)
	if err != nil {
		return nil, err
	}
	// 当前场景没有启用版本时，统一回退到默认缓存版本。
	if entity == nil {
		return &recommendSceneVersionInfo{
			version: recommendCache.DefaultVersion,
		}, nil
	}
	return &recommendSceneVersionInfo{
		version:     recommendCache.NormalizeVersion(entity.Version),
		publishedAt: entity.CreatedAt,
	}, nil
}

// listCachedSceneHotGoodsIds 读取场景热门榜缓存商品。
func (c *RecommendRequestCase) listCachedSceneHotGoodsIds(ctx context.Context, scene int32, limit int64, excludeGoodsIds []int64) (*recommendCacheReadResult, error) {
	versionInfo, err := c.loadRecommendSceneVersionInfo(ctx, scene)
	if err != nil {
		return nil, err
	}
	return c.listCachedGoodsIds(
		ctx,
		recommendCache.NonPersonalized,
		recommendCache.SceneHotSubset(scene, versionInfo.version),
		recommendCacheHitSceneHot,
		versionInfo.version,
		versionInfo.publishedAt,
		limit,
		excludeGoodsIds,
	)
}

// listCachedLatestGoodsIds 读取场景最新榜缓存商品。
func (c *RecommendRequestCase) listCachedLatestGoodsIds(ctx context.Context, scene int32, limit int64, excludeGoodsIds []int64) (*recommendCacheReadResult, error) {
	versionInfo, err := c.loadRecommendSceneVersionInfo(ctx, scene)
	if err != nil {
		return nil, err
	}
	return c.listCachedGoodsIds(
		ctx,
		recommendCache.NonPersonalized,
		recommendCache.SceneLatestSubset(scene, versionInfo.version),
		recommendCacheHitLatest,
		versionInfo.version,
		versionInfo.publishedAt,
		limit,
		excludeGoodsIds,
	)
}

// listCachedSimilarItemGoodsIds 读取相似商品缓存。
func (c *RecommendRequestCase) listCachedSimilarItemGoodsIds(ctx context.Context, goodsId int64, limit int64, excludeGoodsIds []int64) (*recommendCacheReadResult, error) {
	// 商品编号非法时，不需要继续读取相似商品缓存。
	if goodsId <= 0 {
		return newRecommendCacheReadResult(
			recommendCacheHitGoodsDetail,
			recommendCache.ItemToItem,
			"",
			recommendCache.DefaultVersion,
			time.Time{},
			limit,
			len(excludeGoodsIds),
		), nil
	}

	versionInfo, err := c.loadRecommendSceneVersionInfo(ctx, int32(common.RecommendScene_GOODS_DETAIL))
	if err != nil {
		return nil, err
	}
	return c.listCachedGoodsIds(
		ctx,
		recommendCache.ItemToItem,
		recommendCache.SimilarItemSubset(goodsId, versionInfo.version),
		recommendCacheHitGoodsDetail,
		versionInfo.version,
		versionInfo.publishedAt,
		limit,
		excludeGoodsIds,
	)
}

// newRecommendCacheReadResult 创建缓存读取结果对象。
func newRecommendCacheReadResult(hitSource string, collection string, subset string, version string, versionPublishedAt time.Time, requestedCount int64, excludeCount int) *recommendCacheReadResult {
	readContext := map[string]any{
		"source":         hitSource,
		"collection":     collection,
		"subset":         subset,
		"version":        recommendCache.NormalizeVersion(version),
		"requestedCount": requestedCount,
		"excludeCount":   excludeCount,
		"hit":            false,
		"subsetExists":   false,
		"returnedCount":  0,
	}
	// 版本发布时间存在时，再补充到缓存调试上下文。
	if !versionPublishedAt.IsZero() {
		readContext["versionPublishedAt"] = versionPublishedAt.Format(time.RFC3339Nano)
	}
	return &recommendCacheReadResult{
		ids:         []int64{},
		readContext: readContext,
	}
}

// appendRecommendCacheReadContext 合并缓存读取调试上下文。
func appendRecommendCacheReadContext(sourceContext map[string]any, cacheReadContext map[string]any) map[string]any {
	// 当前请求没有缓存读取上下文时，直接返回原来源上下文。
	if len(cacheReadContext) == 0 {
		return sourceContext
	}
	if sourceContext == nil {
		sourceContext = make(map[string]any, 1)
	}
	sourceContext["cacheReadContext"] = cacheReadContext
	return sourceContext
}

// mergeRecommendCacheReadResult 合并单次缓存读取结果。
func mergeRecommendCacheReadResult(cacheReadContext map[string]any, result *recommendCacheReadResult) map[string]any {
	// 当前没有读取结果时，不继续合并调试上下文。
	if result == nil || len(result.readContext) == 0 {
		return cacheReadContext
	}
	source, ok := result.readContext["source"].(string)
	// 命中来源为空时，不写入调试上下文，避免产生匿名键。
	if !ok || source == "" {
		return cacheReadContext
	}
	if cacheReadContext == nil {
		cacheReadContext = make(map[string]any, 1)
	}
	cacheReadContext[source] = result.readContext
	return cacheReadContext
}

// fillRecommendCacheReadMeta 补齐缓存读取的元信息。
func (c *RecommendRequestCase) fillRecommendCacheReadMeta(result *recommendCacheReadResult, collection string, subset string) {
	// 结果对象或子集合为空时，不继续读取元信息。
	if result == nil || subset == "" {
		return
	}

	documentCount, err := c.loadRecommendCacheDocumentCount(collection, subset)
	if err == nil && documentCount >= 0 {
		result.readContext["documentCount"] = documentCount
	} else if err != nil {
		// 元信息读取失败时只打日志，不影响主推荐链路。
		log.Errorf("fillRecommendCacheReadMeta document_count %v", err)
	}

	cacheUpdatedAt, updateErr := c.loadRecommendCacheUpdateTime(collection, subset)
	if updateErr == nil && !cacheUpdatedAt.IsZero() {
		result.readContext["cacheUpdatedAt"] = cacheUpdatedAt.Format(time.RFC3339Nano)
	} else if updateErr != nil {
		// 元信息读取失败时只打日志，不影响主推荐链路。
		log.Errorf("fillRecommendCacheReadMeta update_time %v", updateErr)
	}

	digest, digestErr := c.loadRecommendCacheDigest(collection, subset)
	if digestErr == nil && digest != "" {
		result.readContext["digest"] = digest
	} else if digestErr != nil {
		// 元信息读取失败时只打日志，不影响主推荐链路。
		log.Errorf("fillRecommendCacheReadMeta digest %v", digestErr)
	}
}

// loadRecommendCacheDocumentCount 读取缓存子集合的文档数量。
func (c *RecommendRequestCase) loadRecommendCacheDocumentCount(collection string, subset string) (int, error) {
	value, err := c.recommendCacheStore.Get(recommendCache.DocumentCountKey(collection, subset))
	if err != nil {
		// 元信息不存在时，统一回退到未知数量。
		if err == recommendCache.ErrObjectNotExist {
			return -1, nil
		}
		return 0, err
	}
	return strconv.Atoi(value)
}

// loadRecommendCacheUpdateTime 读取缓存子集合的发布时间。
func (c *RecommendRequestCase) loadRecommendCacheUpdateTime(collection string, subset string) (time.Time, error) {
	value, err := c.recommendCacheStore.Get(recommendCache.UpdateTimeKey(collection, subset))
	if err != nil {
		// 元信息不存在时，统一回退到零时间。
		if err == recommendCache.ErrObjectNotExist {
			return time.Time{}, nil
		}
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339Nano, value)
}

// loadRecommendCacheDigest 读取缓存子集合的内容摘要。
func (c *RecommendRequestCase) loadRecommendCacheDigest(collection string, subset string) (string, error) {
	value, err := c.recommendCacheStore.Get(recommendCache.DigestKey(collection, subset))
	if err != nil {
		// 元信息不存在时，统一回退到空摘要。
		if err == recommendCache.ErrObjectNotExist {
			return "", nil
		}
		return "", err
	}
	return value, nil
}

// listCachedInt64Ids 读取指定缓存子集合中的编号列表。
func (c *RecommendRequestCase) listCachedInt64Ids(
	ctx context.Context,
	collection string,
	subset string,
	hitSource string,
	version string,
	versionPublishedAt time.Time,
	limit int64,
	excludeIds []int64,
) (*recommendCacheReadResult, error) {
	result := newRecommendCacheReadResult(hitSource, collection, subset, version, versionPublishedAt, limit, len(excludeIds))
	// 限制数量非法时，不需要继续读取缓存。
	if limit <= 0 {
		result.readContext["skipped"] = true
		result.readContext["skipReason"] = "invalid_limit"
		return result, nil
	}

	collectionKey := recommendCache.CollectionKey(collection)
	searchEnd := int(limit) * 3
	// 排除列表较大时，扩大读取窗口，避免命中过滤后结果不足。
	if searchEnd < int(limit)+len(excludeIds) {
		searchEnd = int(limit) + len(excludeIds)
	}
	result.readContext["searchWindow"] = searchEnd
	documents, err := c.recommendCacheStore.SearchScores(ctx, collectionKey, subset, 0, searchEnd)
	if err != nil {
		// 缓存对象不存在时直接回退查库，不把未命中当作异常。
		if err == recommendCache.ErrObjectNotExist {
			result.readContext["missReason"] = "object_not_exist"
			return result, nil
		}
		return nil, err
	}
	result.readContext["subsetExists"] = true
	result.readContext["scannedCount"] = len(documents)
	c.fillRecommendCacheReadMeta(result, collection, subset)

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
	result.ids = int64Ids
	result.readContext["returnedCount"] = len(int64Ids)
	// 当前读取到了有效缓存结果时，补一条命中日志用于后续排查。
	if len(int64Ids) > 0 {
		result.readContext["hit"] = true
		log.Infof("recommend cache hit source=%s subset=%s count=%d", hitSource, subset, len(int64Ids))
	}
	return result, nil
}

// listCachedGoodsIds 读取指定缓存子集合中的商品编号列表。
func (c *RecommendRequestCase) listCachedGoodsIds(
	ctx context.Context,
	collection string,
	subset string,
	hitSource string,
	version string,
	versionPublishedAt time.Time,
	limit int64,
	excludeGoodsIds []int64,
) (*recommendCacheReadResult, error) {
	return c.listCachedInt64Ids(ctx, collection, subset, hitSource, version, versionPublishedAt, limit, excludeGoodsIds)
}

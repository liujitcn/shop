package biz

import (
	"context"
	"strconv"

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

// loadRecommendSceneVersion 查询当前场景启用的推荐缓存版本。
func (c *RecommendRequestCase) loadRecommendSceneVersion(ctx context.Context, scene int32) (string, error) {
	entity, err := c.loadRecommendSceneVersionEntity(ctx, scene)
	if err != nil {
		return "", err
	}
	// 当前场景没有启用版本时，统一回退到默认缓存版本。
	if entity == nil {
		return recommendCache.DefaultVersion, nil
	}
	return recommendCache.NormalizeVersion(entity.Version), nil
}

// listCachedSceneHotGoodsIds 读取场景热门榜缓存商品。
func (c *RecommendRequestCase) listCachedSceneHotGoodsIds(ctx context.Context, scene int32, limit int64, excludeGoodsIds []int64) ([]int64, error) {
	version, err := c.loadRecommendSceneVersion(ctx, scene)
	if err != nil {
		return nil, err
	}
	return c.listCachedGoodsIds(
		ctx,
		recommendCache.NonPersonalized,
		recommendCache.SceneHotSubset(scene, version),
		recommendCacheHitSceneHot,
		limit,
		excludeGoodsIds,
	)
}

// listCachedLatestGoodsIds 读取场景最新榜缓存商品。
func (c *RecommendRequestCase) listCachedLatestGoodsIds(ctx context.Context, scene int32, limit int64, excludeGoodsIds []int64) ([]int64, error) {
	version, err := c.loadRecommendSceneVersion(ctx, scene)
	if err != nil {
		return nil, err
	}
	return c.listCachedGoodsIds(
		ctx,
		recommendCache.NonPersonalized,
		recommendCache.SceneLatestSubset(scene, version),
		recommendCacheHitLatest,
		limit,
		excludeGoodsIds,
	)
}

// listCachedSimilarItemGoodsIds 读取相似商品缓存。
func (c *RecommendRequestCase) listCachedSimilarItemGoodsIds(ctx context.Context, goodsId int64, limit int64, excludeGoodsIds []int64) ([]int64, error) {
	// 商品编号非法时，不需要继续读取相似商品缓存。
	if goodsId <= 0 {
		return []int64{}, nil
	}

	version, err := c.loadRecommendSceneVersion(ctx, int32(common.RecommendScene_GOODS_DETAIL))
	if err != nil {
		return nil, err
	}
	return c.listCachedGoodsIds(
		ctx,
		recommendCache.ItemToItem,
		recommendCache.SimilarItemSubset(goodsId, version),
		recommendCacheHitGoodsDetail,
		limit,
		excludeGoodsIds,
	)
}

// listCachedInt64Ids 读取指定缓存子集合中的编号列表。
func (c *RecommendRequestCase) listCachedInt64Ids(
	ctx context.Context,
	collection string,
	subset string,
	hitSource string,
	limit int64,
	excludeIds []int64,
) ([]int64, error) {
	// 限制数量非法时，不需要继续读取缓存。
	if limit <= 0 {
		return []int64{}, nil
	}

	collectionKey := recommendCache.CollectionKey(collection)
	searchEnd := int(limit) * 3
	// 排除列表较大时，扩大读取窗口，避免命中过滤后结果不足。
	if searchEnd < int(limit)+len(excludeIds) {
		searchEnd = int(limit) + len(excludeIds)
	}
	documents, err := c.recommendCacheStore.SearchScores(ctx, collectionKey, subset, 0, searchEnd)
	if err != nil {
		// 缓存对象不存在时直接回退查库，不把未命中当作异常。
		if err == recommendCache.ErrObjectNotExist {
			return []int64{}, nil
		}
		return nil, err
	}

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
	// 当前读取到了有效缓存结果时，补一条命中日志用于后续排查。
	if len(int64Ids) > 0 {
		log.Infof("recommend cache hit source=%s subset=%s count=%d", hitSource, subset, len(int64Ids))
	}
	return int64Ids, nil
}

// listCachedGoodsIds 读取指定缓存子集合中的商品编号列表。
func (c *RecommendRequestCase) listCachedGoodsIds(
	ctx context.Context,
	collection string,
	subset string,
	hitSource string,
	limit int64,
	excludeGoodsIds []int64,
) ([]int64, error) {
	return c.listCachedInt64Ids(ctx, collection, subset, hitSource, limit, excludeGoodsIds)
}

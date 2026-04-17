package rerank

import (
	"context"
	"time"

	recommendCache "shop/pkg/recommend/cache"
	recommendMaterialize "shop/pkg/recommend/offline/materialize"
)

// WriteBackScores 将在线 LLM 重排结果回写到推荐缓存。
func WriteBackScores(
	ctx context.Context,
	store recommendCache.Store,
	scene int32,
	actorType int32,
	actorId int64,
	requestHash string,
	version string,
	documents []recommendCache.Score,
	ttl time.Duration,
) error {
	// 当前没有缓存存储、没有文档或 TTL 非法时，不继续执行回写。
	if store == nil || len(documents) == 0 || ttl <= 0 {
		return nil
	}
	materializer := recommendMaterialize.NewMaterializer(store)
	err := materializer.MaterializeLlmRerank(ctx, scene, actorType, actorId, requestHash, version, documents)
	if err != nil {
		return err
	}
	return recommendCache.ExpireScoreSubset(
		store,
		recommendCache.LlmRerank,
		recommendCache.LlmRerankSubset(scene, actorType, actorId, requestHash, version),
		ttl,
	)
}

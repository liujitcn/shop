package task

import (
	"context"
	"fmt"
	"strings"
	"time"

	"shop/pkg/errorsx"
	recommendCache "shop/pkg/recommend/cache"
	"shop/pkg/recommend/offline/materialize"

	"github.com/go-kratos/kratos/v2/log"
)

// RecommendLlmRerankMaterialize LLM 二次重排分数写缓存任务。
type RecommendLlmRerankMaterialize struct {
	store        recommendCache.Store
	materializer *materialize.Materializer
	ctx          context.Context
}

// NewRecommendLlmRerankMaterialize 创建 LLM 二次重排分数写缓存任务实例。
func NewRecommendLlmRerankMaterialize(
	store recommendCache.Store,
	materializer *materialize.Materializer,
) *RecommendLlmRerankMaterialize {
	return &RecommendLlmRerankMaterialize{
		store:        store,
		materializer: materializer,
		ctx:          context.Background(),
	}
}

// Exec 执行 LLM 二次重排分数写缓存任务。
func (t *RecommendLlmRerankMaterialize) Exec(args map[string]string) ([]string, error) {
	log.Infof("Job RecommendLlmRerankMaterialize Exec %+v", args)

	limit, err := parseRecommendMaterializeLimitArg(args["limit"])
	if err != nil {
		return []string{err.Error()}, err
	}
	version, err := parseRecommendMaterializeRequiredVersionArg(args["version"])
	if err != nil {
		return []string{err.Error()}, err
	}
	clearStale, err := parseRecommendMaterializeBoolArg(args["clearStale"], false)
	if err != nil {
		return []string{err.Error()}, err
	}
	stats := newRecommendMaterializeStats("RecommendLlmRerankMaterialize", limit)
	stats.AddVersion(version)

	stats.SetStage("load_llm_rerank_snapshot")
	entryList, err := loadRecommendStageScoreEntryList(args["path"])
	if err != nil {
		return returnRecommendMaterializeFailure(stats, err)
	}
	stats.AddInputCount("entry_count", len(entryList))
	stats.AddInputCount("document_count", countRecommendStageScoreDocuments(entryList))

	publishedAt := time.Now()
	currentSubsetMap := make(map[string]struct{}, len(entryList))
	result := make([]string, 0, len(entryList)+2)
	for _, entry := range entryList {
		// 空快照条目不参与缓存发布。
		if entry == nil {
			continue
		}
		// LLM 二次重排快照必须带有效场景，避免写出无法读取的缓存键。
		if entry.Scene <= 0 {
			return returnRecommendMaterializeFailure(stats, errorsx.InvalidArgument("llm_rerank 快照条目的 scene 必须大于 0"))
		}
		trimmedRequestHash := strings.TrimSpace(entry.RequestHash)
		// LLM 二次重排缓存按请求哈希分桶，请求哈希缺失时拒绝发布。
		if trimmedRequestHash == "" {
			return returnRecommendMaterializeFailure(stats, errorsx.InvalidArgument("llm_rerank 快照条目的 requestHash 不能为空"))
		}

		documentList := normalizeRecommendStageDocuments(entry.Documents, limit, publishedAt)
		subset := recommendCache.LlmRerankSubset(entry.Scene, entry.ActorType, entry.ActorId, trimmedRequestHash, version)
		currentSubsetMap[subset] = struct{}{}
		stats.SetStage(fmt.Sprintf(
			"publish_llm_rerank_scene_%d_actor_type_%d_actor_%d_request_%s",
			entry.Scene,
			entry.ActorType,
			entry.ActorId,
			recommendCache.NormalizeRequestHash(trimmedRequestHash),
		))
		err = t.materializer.MaterializeLlmRerank(t.ctx, entry.Scene, entry.ActorType, entry.ActorId, trimmedRequestHash, version, documentList)
		if err != nil {
			return returnRecommendMaterializeFailure(stats, err)
		}
		stats.AddPublishedSubset(len(documentList))
		result = append(result, fmt.Sprintf(
			"scene=%d actor_type=%d actor_id=%d request_hash=%s version=%s count=%d",
			entry.Scene,
			entry.ActorType,
			entry.ActorId,
			recommendCache.NormalizeRequestHash(trimmedRequestHash),
			version,
			len(documentList),
		))
	}

	// 显式要求全量快照发布时，再清理当前版本下没有出现在快照里的旧子集合。
	if clearStale {
		stats.SetStage(fmt.Sprintf("clear_llm_rerank_version_%s", version))
		clearedSubsetCount, clearErr := clearStaleVersionedSubsets(t.ctx, t.store, recommendCache.LlmRerank, version, currentSubsetMap)
		if clearErr != nil {
			return returnRecommendMaterializeFailure(stats, clearErr)
		}
		stats.AddClearedSubsets(clearedSubsetCount)
		result = append(result, fmt.Sprintf("version=%s cleared_subsets=%d", version, clearedSubsetCount))
	}

	result = append(result, stats.BuildSummary())
	stats.LogSummary()
	return result, nil
}

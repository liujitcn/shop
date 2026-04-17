package cache

import (
	"time"

	recommendDomain "shop/pkg/recommend/domain"
)

// NewReadResult 创建缓存编号读取结果对象。
func NewReadResult(hitSource string, collection string, subset string, version string, versionPublishedAt time.Time, requestedCount int64, excludeCount int) *recommendDomain.CacheReadResult {
	readContext := map[string]any{
		"source":         hitSource,
		"collection":     collection,
		"subset":         subset,
		"version":        NormalizeVersion(version),
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
	return &recommendDomain.CacheReadResult{
		Ids:         []int64{},
		ReadContext: readContext,
	}
}

// NewScoreReadResult 创建缓存分数读取结果对象。
func NewScoreReadResult(hitSource string, collection string, subset string, version string, versionPublishedAt time.Time, requestedCount int64) *recommendDomain.CacheScoreReadResult {
	readContext := map[string]any{
		"source":         hitSource,
		"collection":     collection,
		"subset":         subset,
		"version":        NormalizeVersion(version),
		"requestedCount": requestedCount,
		"hit":            false,
		"subsetExists":   false,
		"returnedCount":  0,
	}
	// 版本发布时间存在时，再补充到缓存调试上下文。
	if !versionPublishedAt.IsZero() {
		readContext["versionPublishedAt"] = versionPublishedAt.Format(time.RFC3339Nano)
	}
	return &recommendDomain.CacheScoreReadResult{
		Scores:      map[int64]float64{},
		ReadContext: readContext,
	}
}

// MergeReadContext 合并单次缓存编号读取结果。
func MergeReadContext(cacheReadContext map[string]any, result *recommendDomain.CacheReadResult) map[string]any {
	// 当前没有读取结果时，不继续合并调试上下文。
	if result == nil || len(result.ReadContext) == 0 {
		return cacheReadContext
	}
	source, ok := result.ReadContext["source"].(string)
	// 命中来源为空时，不写入调试上下文，避免产生匿名键。
	if !ok || source == "" {
		return cacheReadContext
	}
	if cacheReadContext == nil {
		cacheReadContext = make(map[string]any, 1)
	}
	cacheReadContext[source] = result.ReadContext
	return cacheReadContext
}

// MergeScoreReadContext 合并单次缓存分数读取结果。
func MergeScoreReadContext(cacheReadContext map[string]any, result *recommendDomain.CacheScoreReadResult) map[string]any {
	// 当前没有读取结果时，不继续合并调试上下文。
	if result == nil || len(result.ReadContext) == 0 {
		return cacheReadContext
	}
	source, ok := result.ReadContext["source"].(string)
	// 命中来源为空时，不写入调试上下文，避免产生匿名键。
	if !ok || source == "" {
		return cacheReadContext
	}
	if cacheReadContext == nil {
		cacheReadContext = make(map[string]any, 1)
	}
	cacheReadContext[source] = result.ReadContext
	return cacheReadContext
}

package comment

import (
	"sync"

	einoStructured "shop/pkg/agent/eino/structured"
)

var (
	// reviewResultSchemaOnce 确保审核结果 Schema 只生成一次，避免高频审核时重复反射类型。
	reviewResultSchemaOnce sync.Once
	reviewResultSchema     *einoStructured.Schema
	reviewResultSchemaErr  error

	// summaryResultSchemaOnce 确保摘要结果 Schema 只生成一次，减少定时摘要刷新时的固定开销。
	summaryResultSchemaOnce sync.Once
	summaryResultSchema     *einoStructured.Schema
	summaryResultSchemaErr  error
)

// cachedReviewResultSchema 返回缓存后的评论审核结构化输出 Schema。
func cachedReviewResultSchema() (*einoStructured.Schema, error) {
	reviewResultSchemaOnce.Do(func() {
		reviewResultSchema, reviewResultSchemaErr = einoStructured.SchemaFor[ReviewResult]()
	})
	return reviewResultSchema, reviewResultSchemaErr
}

// cachedSummaryResultSchema 返回缓存后的评价摘要结构化输出 Schema。
func cachedSummaryResultSchema() (*einoStructured.Schema, error) {
	summaryResultSchemaOnce.Do(func() {
		summaryResultSchema, summaryResultSchemaErr = einoStructured.SchemaFor[SummaryResult]()
	})
	return summaryResultSchema, summaryResultSchemaErr
}

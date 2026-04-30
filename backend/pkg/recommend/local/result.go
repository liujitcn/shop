package local

// ProviderName 表示本地推荐 provider 标识。
type ProviderName string

const (
	// CONTEXT_CATEGORY_7D 表示 7 天窗口下的上下文类目推荐。
	CONTEXT_CATEGORY_7D ProviderName = "context.category_7d"
	// CONTEXT_CATEGORY_30D 表示 30 天窗口下的上下文类目推荐。
	CONTEXT_CATEGORY_30D ProviderName = "context.category_30d"
	// NON_PERSONALIZED_HOT_7D 表示 7 天窗口下的非个性化全站热度推荐。
	NON_PERSONALIZED_HOT_7D ProviderName = "hot.7d"
	// NON_PERSONALIZED_HOT_30D 表示 30 天窗口下的非个性化全站热度推荐。
	NON_PERSONALIZED_HOT_30D ProviderName = "hot.30d"
	// EXPLORE_ALL_GOODS 表示全量商品探索推荐。
	EXPLORE_ALL_GOODS ProviderName = "explore"
)

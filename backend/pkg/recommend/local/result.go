package local

// LocalProviderName 表示本地推荐 provider 标识。
type LocalProviderName string

const (
	// ContextCategory7d 表示 7 天窗口下的上下文类目推荐。
	ContextCategory7d LocalProviderName = "context.category_7d"
	// ContextCategory30d 表示 30 天窗口下的上下文类目推荐。
	ContextCategory30d LocalProviderName = "context.category_30d"
	// NonPersonalizedHot7d 表示 7 天窗口下的非个性化全站热度推荐。
	NonPersonalizedHot7d LocalProviderName = "hot.7d"
	// NonPersonalizedHot30d 表示 30 天窗口下的非个性化全站热度推荐。
	NonPersonalizedHot30d LocalProviderName = "hot.30d"
	// ExploreAllGoods 表示全量商品探索推荐。
	ExploreAllGoods LocalProviderName = "explore"
)

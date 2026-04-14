package contract

import "context"

// LevelDbLayout 表示当前模块使用的三个固定 LevelDB 文件位置。
type LevelDbLayout struct {
	// PoolPath 表示 `pool.db` 文件路径。
	PoolPath string
	// RuntimePath 表示 `runtime.db` 文件路径。
	RuntimePath string
	// TracePath 表示 `trace.db` 文件路径。
	TracePath string
}

// CacheSource 定义推荐工具所需的缓存文件位置。
type CacheSource interface {
	RecommendLevelDb(context.Context) (LevelDbLayout, error)
}

package domain

// CacheReadResult 表示一次缓存读取的结果与调试上下文。
type CacheReadResult struct {
	Ids         []int64        // 当前缓存读取返回的编号列表。
	ReadContext map[string]any // 当前缓存读取过程的调试上下文。
}

// CacheScoreReadResult 表示一次缓存分数读取的结果与调试上下文。
type CacheScoreReadResult struct {
	Scores      map[int64]float64 // 当前缓存读取返回的分数映射。
	ReadContext map[string]any    // 当前缓存读取过程的调试上下文。
}

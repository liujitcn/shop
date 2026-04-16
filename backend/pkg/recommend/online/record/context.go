package record

// BuildPersistedSourceContext 构建推荐请求主表需要持久化的来源上下文。
func BuildPersistedSourceContext(sourceContext map[string]any) map[string]any {
	// 主表只保留排查请求所需的精简上下文，大体量 explain 明细下沉到 item 表。
	persistedSourceContext := make(map[string]any, len(sourceContext))
	for key, value := range sourceContext {
		// 逐商品 explain 明细已经落到 item 表，这里不再重复保存。
		if key == "returnedScoreDetails" {
			continue
		}
		// 主体信息已经有独立列，不再在上下文里重复冗余。
		if key == "actorType" || key == "actorId" {
			continue
		}
		persistedSourceContext[key] = value
	}
	return compactOnlineDebugContext(persistedSourceContext)
}

// compactOnlineDebugContext 收口推荐链路的在线排障上下文。
func compactOnlineDebugContext(sourceContext map[string]any) map[string]any {
	// 来源上下文为空时，不需要继续收口。
	if len(sourceContext) == 0 {
		return sourceContext
	}

	onlineDebugContext := make(map[string]any, 4)
	mergeOnlineDebugField(onlineDebugContext, "cacheHitSources", sourceContext)
	mergeOnlineDebugField(onlineDebugContext, "cacheReadContext", sourceContext)
	mergeOnlineDebugField(onlineDebugContext, "recallProbeContext", sourceContext)
	mergeOnlineDebugField(onlineDebugContext, "observedRecallSources", sourceContext)
	mergeOnlineDebugField(onlineDebugContext, "joinRecallContext", sourceContext)
	mergeOnlineDebugField(onlineDebugContext, "similarUserObservationContext", sourceContext)
	// 这些拉平字段已经被对应子上下文覆盖，不再保留顶层重复定义。
	removeOnlineDebugField(sourceContext, "joinedRecallSources")
	removeOnlineDebugField(sourceContext, "effectiveJoinRecallSources")
	removeOnlineDebugField(sourceContext, "returnedJoinRecallSources")
	// 顶层只有这一层排障结构时，才写回统一的在线调试上下文。
	if len(onlineDebugContext) == 0 {
		return sourceContext
	}
	sourceContext["onlineDebugContext"] = onlineDebugContext
	return sourceContext
}

// mergeOnlineDebugField 将指定排障字段收口到统一上下文。
func mergeOnlineDebugField(target map[string]any, key string, sourceContext map[string]any) {
	value, ok := sourceContext[key]
	// 当前字段不存在时，不需要写入统一上下文。
	if !ok {
		return
	}
	target[key] = value
	delete(sourceContext, key)
}

// removeOnlineDebugField 删除已经被统一上下文覆盖的顶层字段。
func removeOnlineDebugField(sourceContext map[string]any, key string) {
	if len(sourceContext) == 0 || key == "" {
		return
	}
	delete(sourceContext, key)
}

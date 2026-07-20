package comment

// cleanStringList 清理字符串列表并去重。
func cleanStringList(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		// 这里处理的是模型输入或模型输出标签，空白不是业务内容，去掉后再参与去重。
		cleanValue := value
		// 清理后为空或已经出现过的值不再保留。
		if cleanValue == "" {
			continue
		}
		// 已经保留过的值不再重复追加。
		if _, ok := seen[cleanValue]; ok {
			continue
		}
		seen[cleanValue] = struct{}{}
		result = append(result, cleanValue)
	}
	return result
}

// limitStringList 清理字符串列表并限制数量和单项长度。
func limitStringList(values []string, limit int, runeLimit int) []string {
	// 限制小于等于 0 时，直接返回空列表。
	if limit <= 0 {
		return []string{}
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		cleanValue := trimStringByRunes(value, runeLimit)
		// 清理后为空的值不进入最终列表。
		if cleanValue == "" {
			continue
		}
		result = append(result, cleanValue)
		// 已达到数量上限时，停止继续追加。
		if len(result) >= limit {
			break
		}
	}
	return result
}

// trimStringByRunes 按字符数清理并截断字符串。
func trimStringByRunes(value string, limit int) string {
	// 摘要输入按 rune 截断，避免中文被 byte 截断造成无效 UTF-8。
	cleanValue := value
	if limit <= 0 {
		return ""
	}
	runes := []rune(cleanValue)
	if len(runes) <= limit {
		return cleanValue
	}
	return string(runes[:limit])
}

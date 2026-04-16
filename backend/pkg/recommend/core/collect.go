package core

import mapset "github.com/deckarep/golang-set/v2"

// DedupeInt64s 对整型切片做稳定去重。
func DedupeInt64s(values []int64) []int64 {
	result := make([]int64, 0, len(values))
	seen := mapset.NewSet[int64]()
	for _, value := range values {
		// 零值编号无业务意义，直接跳过。
		if value == 0 {
			continue
		}
		// 已收录过的编号不再重复写入结果。
		if seen.Contains(value) {
			continue
		}
		seen.Add(value)
		result = append(result, value)
	}
	return result
}

// DedupeStrings 对字符串切片做稳定去重。
func DedupeStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := mapset.NewSet[string]()
	for _, value := range values {
		// 空字符串没有业务意义，直接跳过。
		if value == "" {
			continue
		}
		// 已收录过的字符串不再重复写入结果。
		if seen.Contains(value) {
			continue
		}
		seen.Add(value)
		result = append(result, value)
	}
	return result
}

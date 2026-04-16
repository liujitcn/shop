package recall

import (
	"sort"

	recommendcore "shop/pkg/recommend/core"
)

// AppendProbeContext 将召回探针上下文合并到来源上下文。
func AppendProbeContext(sourceContext map[string]any, probeContext map[string]any) map[string]any {
	if len(probeContext) == 0 {
		return sourceContext
	}
	if sourceContext == nil {
		sourceContext = make(map[string]any, 1)
	}
	sourceContext["recallProbeContext"] = probeContext
	observedSources, ok := probeContext["observedSources"].([]string)
	if ok && len(observedSources) > 0 {
		sourceContext["observedRecallSources"] = recommendcore.DedupeStrings(observedSources)
	}
	return sourceContext
}

// AppendJoinContext 将灰度召回入池信息合并到来源上下文。
func AppendJoinContext(sourceContext map[string]any, joinRecallGoodsIds map[string][]int64, candidateGoodsIds []int64, returnedGoodsIds []int64) map[string]any {
	if len(joinRecallGoodsIds) == 0 {
		return sourceContext
	}
	if sourceContext == nil {
		sourceContext = make(map[string]any, 1)
	}

	normalizedJoinRecallGoodsIds := normalizeRecallGoodsIdsMap(joinRecallGoodsIds)
	effectiveJoinRecallGoodsIds := filterRecallGoodsIdsMap(normalizedJoinRecallGoodsIds, candidateGoodsIds)
	returnedJoinRecallGoodsIds := filterRecallGoodsIdsMap(normalizedJoinRecallGoodsIds, returnedGoodsIds)
	joinContext := map[string]any{
		"joinedRecallSources":         listRecallSourceNames(normalizedJoinRecallGoodsIds),
		"joinedRecallGoodsIds":        normalizedJoinRecallGoodsIds,
		"effectiveJoinRecallSources":  listRecallSourceNames(effectiveJoinRecallGoodsIds),
		"effectiveJoinRecallGoodsIds": effectiveJoinRecallGoodsIds,
		"returnedJoinRecallSources":   listRecallSourceNames(returnedJoinRecallGoodsIds),
		"returnedJoinRecallGoodsIds":  returnedJoinRecallGoodsIds,
		"candidateGoodsIdsSnapshot":   recommendcore.DedupeInt64s(candidateGoodsIds),
		"returnedGoodsIdsSnapshot":    recommendcore.DedupeInt64s(returnedGoodsIds),
	}
	sourceContext["joinRecallContext"] = joinContext
	sourceContext["joinedRecallSources"] = joinContext["joinedRecallSources"]
	sourceContext["effectiveJoinRecallSources"] = joinContext["effectiveJoinRecallSources"]
	sourceContext["returnedJoinRecallSources"] = joinContext["returnedJoinRecallSources"]
	return sourceContext
}

// AppendSimilarUserObservationContext 将相似用户观测结果合并到来源上下文。
func AppendSimilarUserObservationContext(sourceContext map[string]any, similarUserIds []int64, observedGoodsIds []int64, joinRecallGoodsIds map[string][]int64, candidateGoodsIds []int64, returnedGoodsIds []int64) map[string]any {
	if len(similarUserIds) == 0 {
		return sourceContext
	}
	if sourceContext == nil {
		sourceContext = make(map[string]any, 1)
	}

	normalizedSimilarUserIds := recommendcore.DedupeInt64s(similarUserIds)
	normalizedObservedGoodsIds := recommendcore.DedupeInt64s(observedGoodsIds)
	normalizedJoinRecallGoodsIds := normalizeRecallGoodsIdsMap(joinRecallGoodsIds)
	candidateOverlapGoodsIds := filterRecallGoodsIds(normalizedObservedGoodsIds, candidateGoodsIds)
	returnedOverlapGoodsIds := filterRecallGoodsIds(normalizedObservedGoodsIds, returnedGoodsIds)
	effectiveJoinRecallGoodsIds := filterRecallGoodsIdsMap(normalizedJoinRecallGoodsIds, candidateGoodsIds)
	returnedJoinRecallGoodsIds := filterRecallGoodsIdsMap(normalizedJoinRecallGoodsIds, returnedGoodsIds)
	observeContext := map[string]any{
		"similarUserIds":           normalizedSimilarUserIds,
		"observedGoodsIds":         normalizedObservedGoodsIds,
		"candidateOverlapGoodsIds": candidateOverlapGoodsIds,
		"returnedOverlapGoodsIds":  returnedOverlapGoodsIds,
		"observedGoodsCount":       len(normalizedObservedGoodsIds),
		"candidateGoodsCount":      len(recommendcore.DedupeInt64s(candidateGoodsIds)),
		"returnedGoodsCount":       len(recommendcore.DedupeInt64s(returnedGoodsIds)),
		"candidateOverlapCount":    len(candidateOverlapGoodsIds),
		"returnedOverlapCount":     len(returnedOverlapGoodsIds),
		"candidateOverlapRate":     divideObservedCount(len(candidateOverlapGoodsIds), len(normalizedObservedGoodsIds)),
		"returnedOverlapRate":      divideObservedCount(len(returnedOverlapGoodsIds), len(normalizedObservedGoodsIds)),
		"candidateCoverageRate":    divideObservedCount(len(candidateOverlapGoodsIds), len(recommendcore.DedupeInt64s(candidateGoodsIds))),
		"returnedCoverageRate":     divideObservedCount(len(returnedOverlapGoodsIds), len(recommendcore.DedupeInt64s(returnedGoodsIds))),
		"joinedRecallOverlap":      buildObservedJoinOverlap(normalizedObservedGoodsIds, normalizedJoinRecallGoodsIds),
		"effectiveRecallOverlap":   buildObservedJoinOverlap(normalizedObservedGoodsIds, effectiveJoinRecallGoodsIds),
		"returnedRecallOverlap":    buildObservedJoinOverlap(normalizedObservedGoodsIds, returnedJoinRecallGoodsIds),
	}
	sourceContext["similarUserObservationContext"] = observeContext
	return sourceContext
}

// ListContentBasedJoinCandidateGoodsIds 返回允许并入候选池的内容相似商品编号。
func ListContentBasedJoinCandidateGoodsIds(probeContext map[string]any) []int64 {
	return listProbeJoinGoodsIds(probeContext, "contentBased")
}

// ListCollaborativeFilteringJoinCandidateGoodsIds 返回允许并入候选池的协同过滤商品编号。
func ListCollaborativeFilteringJoinCandidateGoodsIds(probeContext map[string]any) []int64 {
	return listProbeJoinGoodsIds(probeContext, "collaborativeFiltering")
}

// ListSimilarUserProbeUserIds 返回相似用户探针命中的用户编号。
func ListSimilarUserProbeUserIds(probeContext map[string]any) []int64 {
	probeItem := loadProbeItem(probeContext, "similarUser")
	userIds, ok := probeItem["userIds"].([]int64)
	if !ok {
		return []int64{}
	}
	return recommendcore.DedupeInt64s(userIds)
}

func divideObservedCount(numerator int, denominator int) float64 {
	if denominator <= 0 || numerator <= 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

func buildObservedJoinOverlap(observedGoodsIds []int64, joinRecallGoodsIds map[string][]int64) map[string]any {
	if len(observedGoodsIds) == 0 || len(joinRecallGoodsIds) == 0 {
		return map[string]any{}
	}
	result := make(map[string]any, len(joinRecallGoodsIds))
	for source, goodsIds := range joinRecallGoodsIds {
		detail := buildObservedOverlapDetail(observedGoodsIds, goodsIds)
		if len(detail) == 0 {
			continue
		}
		result[source] = detail
	}
	return result
}

func buildObservedOverlapDetail(observedGoodsIds []int64, targetGoodsIds []int64) map[string]any {
	normalizedObservedGoodsIds := recommendcore.DedupeInt64s(observedGoodsIds)
	normalizedTargetGoodsIds := recommendcore.DedupeInt64s(targetGoodsIds)
	if len(normalizedObservedGoodsIds) == 0 || len(normalizedTargetGoodsIds) == 0 {
		return map[string]any{}
	}
	overlapGoodsIds := filterRecallGoodsIds(normalizedObservedGoodsIds, normalizedTargetGoodsIds)
	return map[string]any{
		"targetGoodsIds":    normalizedTargetGoodsIds,
		"overlapGoodsIds":   overlapGoodsIds,
		"targetGoodsCount":  len(normalizedTargetGoodsIds),
		"overlapGoodsCount": len(overlapGoodsIds),
		"overlapRate":       divideObservedCount(len(overlapGoodsIds), len(normalizedObservedGoodsIds)),
		"coverageRate":      divideObservedCount(len(overlapGoodsIds), len(normalizedTargetGoodsIds)),
	}
}

func normalizeRecallGoodsIdsMap(joinRecallGoodsIds map[string][]int64) map[string][]int64 {
	if len(joinRecallGoodsIds) == 0 {
		return map[string][]int64{}
	}
	result := make(map[string][]int64, len(joinRecallGoodsIds))
	for source, goodsIds := range joinRecallGoodsIds {
		dedupedGoodsIds := recommendcore.DedupeInt64s(goodsIds)
		if len(dedupedGoodsIds) == 0 {
			continue
		}
		result[source] = dedupedGoodsIds
	}
	return result
}

func filterRecallGoodsIdsMap(joinRecallGoodsIds map[string][]int64, targetGoodsIds []int64) map[string][]int64 {
	if len(joinRecallGoodsIds) == 0 || len(targetGoodsIds) == 0 {
		return map[string][]int64{}
	}
	result := make(map[string][]int64, len(joinRecallGoodsIds))
	for source, goodsIds := range joinRecallGoodsIds {
		matchedGoodsIds := filterRecallGoodsIds(goodsIds, targetGoodsIds)
		if len(matchedGoodsIds) == 0 {
			continue
		}
		result[source] = matchedGoodsIds
	}
	return result
}

func filterRecallGoodsIds(goodsIds []int64, targetGoodsIds []int64) []int64 {
	if len(goodsIds) == 0 || len(targetGoodsIds) == 0 {
		return []int64{}
	}
	targetGoodsIdSet := make(map[int64]struct{}, len(targetGoodsIds))
	for _, goodsId := range targetGoodsIds {
		if goodsId <= 0 {
			continue
		}
		targetGoodsIdSet[goodsId] = struct{}{}
	}
	result := make([]int64, 0, len(goodsIds))
	for _, goodsId := range goodsIds {
		if _, ok := targetGoodsIdSet[goodsId]; !ok {
			continue
		}
		result = append(result, goodsId)
	}
	return recommendcore.DedupeInt64s(result)
}

func listRecallSourceNames(joinRecallGoodsIds map[string][]int64) []string {
	if len(joinRecallGoodsIds) == 0 {
		return []string{}
	}
	result := make([]string, 0, len(joinRecallGoodsIds))
	for source, goodsIds := range joinRecallGoodsIds {
		if len(goodsIds) == 0 {
			continue
		}
		result = append(result, source)
	}
	sort.Strings(result)
	return result
}

func listProbeJoinGoodsIds(probeContext map[string]any, key string) []int64 {
	probeItem := loadProbeItem(probeContext, key)
	if !shouldJoinProbeCandidate(probeItem) {
		return []int64{}
	}
	goodsIds, ok := probeItem["goodsIds"].([]int64)
	if !ok {
		return []int64{}
	}
	return recommendcore.DedupeInt64s(goodsIds)
}

func shouldJoinProbeCandidate(probeItem map[string]any) bool {
	joinCandidate, ok := probeItem["joinCandidate"].(bool)
	if !ok || !joinCandidate {
		return false
	}
	enabled, ok := probeItem["enabled"].(bool)
	return ok && enabled
}

func loadProbeItem(probeContext map[string]any, key string) map[string]any {
	if len(probeContext) == 0 || key == "" {
		return map[string]any{}
	}
	probeItem, ok := probeContext[key].(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return probeItem
}

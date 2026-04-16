package biz

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"shop/api/gen/go/common"
	"shop/pkg/gen/models"
	recommendCache "shop/pkg/recommend/cache"
	recommendcore "shop/pkg/recommend/core"
	recommendDomain "shop/pkg/recommend/domain"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/liujitcn/gorm-kit/repo"
)

const (
	// recommendRecallProbeSimilarUser 表示相似用户召回探针。
	recommendRecallProbeSimilarUser = "similar_user_probe"
	// recommendRecallProbeCollaborativeFiltering 表示协同过滤召回探针。
	recommendRecallProbeCollaborativeFiltering = "collaborative_filtering_probe"
	// recommendRecallProbeContentBased 表示内容相似召回探针。
	recommendRecallProbeContentBased = "content_based_probe"
)

// loadRecommendSceneVersionEntity 查询当前场景启用的推荐版本记录。
func (c *RecommendRequestCase) loadRecommendSceneVersionEntity(ctx context.Context, scene int32) (*models.RecommendModelVersion, error) {
	query := c.recommendModelVersionRepo.Query(ctx).RecommendModelVersion
	opts := make([]repo.QueryOption, 0, 3)
	opts = append(opts, repo.Where(query.Scene.Eq(scene)))
	opts = append(opts, repo.Where(query.Status.Eq(int32(common.Status_ENABLE))))
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))
	list, _, err := c.recommendModelVersionRepo.Page(ctx, 1, 1, opts...)
	if err != nil {
		return nil, err
	}
	// 当前场景没有启用版本时，直接回退为空记录。
	if len(list) == 0 || list[0] == nil {
		return nil, nil
	}
	return list[0], nil
}

// loadRecommendRecallProbeConfig 查询当前场景启用的召回探针配置。
func (c *RecommendRequestCase) loadRecommendRecallProbeConfig(ctx context.Context, scene int32) (string, *recommendDomain.RecallProbeStrategy, error) {
	version := recommendCache.DefaultVersion
	entity, err := c.loadRecommendSceneVersionEntity(ctx, scene)
	if err != nil {
		return "", nil, err
	}
	// 当前场景没有启用版本时，直接回退到默认版本和空探针配置。
	if entity == nil {
		return version, &recommendDomain.RecallProbeStrategy{}, nil
	}

	version = recommendCache.NormalizeVersion(entity.Version)
	config := &recommendDomain.StrategyVersionConfig{}
	// 当前版本没有扩展配置时，直接返回空探针配置。
	if strings.TrimSpace(entity.ConfigJSON) == "" {
		return version, &recommendDomain.RecallProbeStrategy{}, nil
	}

	err = json.Unmarshal([]byte(entity.ConfigJSON), config)
	if err != nil {
		// 阶段 4 的召回探针是增量能力，配置解析失败时不影响主推荐链路。
		log.Errorf("loadRecommendRecallProbeConfig %v", err)
		return version, &recommendDomain.RecallProbeStrategy{}, nil
	}
	if config.RecallProbe == nil {
		return version, &recommendDomain.RecallProbeStrategy{}, nil
	}
	return version, config.RecallProbe, nil
}

// listCachedSimilarUserIds 读取相似用户召回探针缓存。
func (c *RecommendRequestCase) listCachedSimilarUserIds(ctx context.Context, userId int64, version string, limit int64) ([]int64, error) {
	// 登录用户编号非法时，不需要继续读取相似用户缓存。
	if userId <= 0 {
		return []int64{}, nil
	}
	return c.listCachedInt64Ids(
		ctx,
		recommendCache.UserToUser,
		recommendCache.SimilarUserSubset(userId, version),
		recommendRecallProbeSimilarUser,
		limit,
		nil,
	)
}

// listCachedCollaborativeFilteringGoodsIds 读取协同过滤召回探针缓存。
func (c *RecommendRequestCase) listCachedCollaborativeFilteringGoodsIds(ctx context.Context, userId int64, version string, limit int64, excludeGoodsIds []int64) ([]int64, error) {
	// 登录用户编号非法时，不需要继续读取协同过滤缓存。
	if userId <= 0 {
		return []int64{}, nil
	}
	return c.listCachedInt64Ids(
		ctx,
		recommendCache.CollaborativeFiltering,
		recommendCache.CollaborativeFilteringSubset(userId, version),
		recommendRecallProbeCollaborativeFiltering,
		limit,
		excludeGoodsIds,
	)
}

// listCachedContentBasedGoodsIds 读取内容相似召回探针缓存。
func (c *RecommendRequestCase) listCachedContentBasedGoodsIds(ctx context.Context, goodsId int64, version string, limit int64, excludeGoodsIds []int64) ([]int64, error) {
	// 商品编号非法时，不需要继续读取内容相似缓存。
	if goodsId <= 0 {
		return []int64{}, nil
	}
	return c.listCachedInt64Ids(
		ctx,
		recommendCache.ContentBased,
		recommendCache.ContentBasedSubset(goodsId, version),
		recommendRecallProbeContentBased,
		limit,
		excludeGoodsIds,
	)
}

// buildRecommendRecallProbeContext 构建当前请求的召回探针上下文。
func (c *RecommendRequestCase) buildRecommendRecallProbeContext(
	ctx context.Context,
	scene int32,
	userId int64,
	goodsId int64,
	defaultLimit int64,
	excludeGoodsIds []int64,
) (map[string]any, error) {
	version, probeConfig, err := c.loadRecommendRecallProbeConfig(ctx, scene)
	if err != nil {
		return nil, err
	}
	// 当前版本没有启用探针时，不需要额外记录上下文。
	if !probeConfig.HasEnabledProbe() {
		return map[string]any{}, nil
	}

	probeContext := map[string]any{
		"sceneVersion": version,
	}
	observedSources := make([]string, 0, 3)
	if probeConfig.IsSimilarUserEnabled() && userId > 0 {
		limit := probeConfig.SimilarUser.ResolveLimit(defaultLimit)
		similarUserIds, listErr := c.listCachedSimilarUserIds(ctx, userId, version, limit)
		if listErr != nil {
			return nil, listErr
		}
		probeContext["similarUser"] = map[string]any{
			"enabled":       true,
			"joinCandidate": probeConfig.SimilarUser.ShouldJoinCandidate(),
			"limit":         limit,
			"userIds":       similarUserIds,
		}
		// 读取到了有效相似用户时，记录探针命中来源。
		if len(similarUserIds) > 0 {
			observedSources = append(observedSources, recommendRecallProbeSimilarUser)
		}
	}
	if probeConfig.IsCollaborativeFilteringEnabled() && userId > 0 {
		limit := probeConfig.CollaborativeFiltering.ResolveLimit(defaultLimit)
		goodsIds, listErr := c.listCachedCollaborativeFilteringGoodsIds(ctx, userId, version, limit, excludeGoodsIds)
		if listErr != nil {
			return nil, listErr
		}
		probeContext["collaborativeFiltering"] = map[string]any{
			"enabled":       true,
			"joinCandidate": probeConfig.CollaborativeFiltering.ShouldJoinCandidate(),
			"limit":         limit,
			"goodsIds":      goodsIds,
		}
		// 读取到了有效协同过滤商品时，记录探针命中来源。
		if len(goodsIds) > 0 {
			observedSources = append(observedSources, recommendRecallProbeCollaborativeFiltering)
		}
	}
	if probeConfig.IsContentBasedEnabled() && goodsId > 0 {
		limit := probeConfig.ContentBased.ResolveLimit(defaultLimit)
		goodsIds, listErr := c.listCachedContentBasedGoodsIds(ctx, goodsId, version, limit, excludeGoodsIds)
		if listErr != nil {
			return nil, listErr
		}
		probeContext["contentBased"] = map[string]any{
			"enabled":       true,
			"joinCandidate": probeConfig.ContentBased.ShouldJoinCandidate(),
			"limit":         limit,
			"goodsIds":      goodsIds,
		}
		// 读取到了有效内容相似商品时，记录探针命中来源。
		if len(goodsIds) > 0 {
			observedSources = append(observedSources, recommendRecallProbeContentBased)
		}
	}
	probeContext["observedSources"] = recommendcore.DedupeStrings(observedSources)
	return probeContext, nil
}

// appendRecommendRecallProbeContext 将召回探针上下文合并到来源上下文。
func appendRecommendRecallProbeContext(sourceContext map[string]any, probeContext map[string]any) map[string]any {
	// 没有探针上下文时，直接返回原来源上下文。
	if len(probeContext) == 0 {
		return sourceContext
	}
	if sourceContext == nil {
		sourceContext = make(map[string]any, 1)
	}
	sourceContext["recallProbeContext"] = probeContext
	observedSources, ok := probeContext["observedSources"].([]string)
	// 存在已观测探针来源时，再额外拉平一份字段方便排查。
	if ok && len(observedSources) > 0 {
		sourceContext["observedRecallSources"] = recommendcore.DedupeStrings(observedSources)
	}
	return sourceContext
}

// appendRecommendRecallJoinContext 将灰度召回入池信息合并到来源上下文。
func appendRecommendRecallJoinContext(sourceContext map[string]any, joinRecallGoodsIds map[string][]int64, candidateGoodsIds []int64, returnedGoodsIds []int64) map[string]any {
	// 当前请求没有灰度入池召回时，不额外写入上下文。
	if len(joinRecallGoodsIds) == 0 {
		return sourceContext
	}
	if sourceContext == nil {
		sourceContext = make(map[string]any, 1)
	}

	normalizedJoinRecallGoodsIds := normalizeRecommendRecallGoodsIdsMap(joinRecallGoodsIds)
	effectiveJoinRecallGoodsIds := filterRecommendRecallGoodsIdsMap(normalizedJoinRecallGoodsIds, candidateGoodsIds)
	returnedJoinRecallGoodsIds := filterRecommendRecallGoodsIdsMap(normalizedJoinRecallGoodsIds, returnedGoodsIds)
	joinContext := map[string]any{
		"joinedRecallSources":         listRecommendRecallSourceNames(normalizedJoinRecallGoodsIds),
		"joinedRecallGoodsIds":        normalizedJoinRecallGoodsIds,
		"effectiveJoinRecallSources":  listRecommendRecallSourceNames(effectiveJoinRecallGoodsIds),
		"effectiveJoinRecallGoodsIds": effectiveJoinRecallGoodsIds,
		"returnedJoinRecallSources":   listRecommendRecallSourceNames(returnedJoinRecallGoodsIds),
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

// appendRecommendSimilarUserObservationContext 将相似用户观测结果合并到来源上下文。
func appendRecommendSimilarUserObservationContext(sourceContext map[string]any, similarUserIds []int64, observedGoodsIds []int64, joinRecallGoodsIds map[string][]int64, candidateGoodsIds []int64, returnedGoodsIds []int64) map[string]any {
	// 没有相似用户探针命中时，不额外写入观测上下文。
	if len(similarUserIds) == 0 {
		return sourceContext
	}
	if sourceContext == nil {
		sourceContext = make(map[string]any, 1)
	}

	normalizedSimilarUserIds := recommendcore.DedupeInt64s(similarUserIds)
	normalizedObservedGoodsIds := recommendcore.DedupeInt64s(observedGoodsIds)
	normalizedJoinRecallGoodsIds := normalizeRecommendRecallGoodsIdsMap(joinRecallGoodsIds)
	candidateOverlapGoodsIds := filterRecommendRecallGoodsIds(normalizedObservedGoodsIds, candidateGoodsIds)
	returnedOverlapGoodsIds := filterRecommendRecallGoodsIds(normalizedObservedGoodsIds, returnedGoodsIds)
	effectiveJoinRecallGoodsIds := filterRecommendRecallGoodsIdsMap(normalizedJoinRecallGoodsIds, candidateGoodsIds)
	returnedJoinRecallGoodsIds := filterRecommendRecallGoodsIdsMap(normalizedJoinRecallGoodsIds, returnedGoodsIds)
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
		"candidateOverlapRate":     divideRecommendObservedCount(len(candidateOverlapGoodsIds), len(normalizedObservedGoodsIds)),
		"returnedOverlapRate":      divideRecommendObservedCount(len(returnedOverlapGoodsIds), len(normalizedObservedGoodsIds)),
		"candidateCoverageRate":    divideRecommendObservedCount(len(candidateOverlapGoodsIds), len(recommendcore.DedupeInt64s(candidateGoodsIds))),
		"returnedCoverageRate":     divideRecommendObservedCount(len(returnedOverlapGoodsIds), len(recommendcore.DedupeInt64s(returnedGoodsIds))),
		"joinedRecallOverlap":      buildRecommendObservedJoinOverlap(normalizedObservedGoodsIds, normalizedJoinRecallGoodsIds),
		"effectiveRecallOverlap":   buildRecommendObservedJoinOverlap(normalizedObservedGoodsIds, effectiveJoinRecallGoodsIds),
		"returnedRecallOverlap":    buildRecommendObservedJoinOverlap(normalizedObservedGoodsIds, returnedJoinRecallGoodsIds),
	}
	sourceContext["similarUserObservationContext"] = observeContext
	return sourceContext
}

// compactRecommendOnlineDebugContext 收口推荐链路的在线排障上下文。
func compactRecommendOnlineDebugContext(sourceContext map[string]any) map[string]any {
	// 来源上下文为空时，不需要继续收口。
	if len(sourceContext) == 0 {
		return sourceContext
	}

	onlineDebugContext := make(map[string]any, 4)
	mergeRecommendOnlineDebugField(onlineDebugContext, "cacheHitSources", sourceContext)
	mergeRecommendOnlineDebugField(onlineDebugContext, "recallProbeContext", sourceContext)
	mergeRecommendOnlineDebugField(onlineDebugContext, "observedRecallSources", sourceContext)
	mergeRecommendOnlineDebugField(onlineDebugContext, "joinRecallContext", sourceContext)
	mergeRecommendOnlineDebugField(onlineDebugContext, "similarUserObservationContext", sourceContext)
	// 这些拉平字段已经被对应子上下文覆盖，不再保留顶层重复定义。
	removeRecommendOnlineDebugField(sourceContext, "joinedRecallSources")
	removeRecommendOnlineDebugField(sourceContext, "effectiveJoinRecallSources")
	removeRecommendOnlineDebugField(sourceContext, "returnedJoinRecallSources")
	// 顶层只有这一层排障结构时，才写回统一的在线调试上下文。
	if len(onlineDebugContext) == 0 {
		return sourceContext
	}
	sourceContext["onlineDebugContext"] = onlineDebugContext
	return sourceContext
}

// mergeRecommendOnlineDebugField 将指定排障字段收口到统一上下文。
func mergeRecommendOnlineDebugField(target map[string]any, key string, sourceContext map[string]any) {
	value, ok := sourceContext[key]
	// 当前字段不存在时，不需要写入统一上下文。
	if !ok {
		return
	}
	target[key] = value
	delete(sourceContext, key)
}

// removeRecommendOnlineDebugField 删除已经被统一上下文覆盖的顶层字段。
func removeRecommendOnlineDebugField(sourceContext map[string]any, key string) {
	if len(sourceContext) == 0 || key == "" {
		return
	}
	delete(sourceContext, key)
}

// divideRecommendObservedCount 安全计算观测命中占比。
func divideRecommendObservedCount(numerator int, denominator int) float64 {
	// 分母为空时，统一返回 0，避免出现无意义的 NaN。
	if denominator <= 0 || numerator <= 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

// buildRecommendObservedJoinOverlap 构建相似用户观测结果与灰度召回结果的逐来源重合信息。
func buildRecommendObservedJoinOverlap(observedGoodsIds []int64, joinRecallGoodsIds map[string][]int64) map[string]any {
	if len(observedGoodsIds) == 0 || len(joinRecallGoodsIds) == 0 {
		return map[string]any{}
	}
	result := make(map[string]any, len(joinRecallGoodsIds))
	for source, goodsIds := range joinRecallGoodsIds {
		detail := buildRecommendObservedOverlapDetail(observedGoodsIds, goodsIds)
		// 当前来源没有任何可对比商品时，不再写入观测结果。
		if len(detail) == 0 {
			continue
		}
		result[source] = detail
	}
	return result
}

// buildRecommendObservedOverlapDetail 构建观测商品和目标商品集合的重合明细。
func buildRecommendObservedOverlapDetail(observedGoodsIds []int64, targetGoodsIds []int64) map[string]any {
	normalizedObservedGoodsIds := recommendcore.DedupeInt64s(observedGoodsIds)
	normalizedTargetGoodsIds := recommendcore.DedupeInt64s(targetGoodsIds)
	// 观测集合或目标集合为空时，不返回重合明细。
	if len(normalizedObservedGoodsIds) == 0 || len(normalizedTargetGoodsIds) == 0 {
		return map[string]any{}
	}
	overlapGoodsIds := filterRecommendRecallGoodsIds(normalizedObservedGoodsIds, normalizedTargetGoodsIds)
	return map[string]any{
		"targetGoodsIds":    normalizedTargetGoodsIds,
		"overlapGoodsIds":   overlapGoodsIds,
		"targetGoodsCount":  len(normalizedTargetGoodsIds),
		"overlapGoodsCount": len(overlapGoodsIds),
		"overlapRate":       divideRecommendObservedCount(len(overlapGoodsIds), len(normalizedObservedGoodsIds)),
		"coverageRate":      divideRecommendObservedCount(len(overlapGoodsIds), len(normalizedTargetGoodsIds)),
	}
}

// normalizeRecommendRecallGoodsIdsMap 对灰度召回商品编号映射做稳定去重。
func normalizeRecommendRecallGoodsIdsMap(joinRecallGoodsIds map[string][]int64) map[string][]int64 {
	if len(joinRecallGoodsIds) == 0 {
		return map[string][]int64{}
	}
	result := make(map[string][]int64, len(joinRecallGoodsIds))
	for source, goodsIds := range joinRecallGoodsIds {
		dedupedGoodsIds := recommendcore.DedupeInt64s(goodsIds)
		// 去重后为空的来源不再写入上下文，避免噪音字段。
		if len(dedupedGoodsIds) == 0 {
			continue
		}
		result[source] = dedupedGoodsIds
	}
	return result
}

// filterRecommendRecallGoodsIdsMap 过滤出真正进入目标集合的灰度召回商品编号。
func filterRecommendRecallGoodsIdsMap(joinRecallGoodsIds map[string][]int64, targetGoodsIds []int64) map[string][]int64 {
	if len(joinRecallGoodsIds) == 0 || len(targetGoodsIds) == 0 {
		return map[string][]int64{}
	}
	result := make(map[string][]int64, len(joinRecallGoodsIds))
	for source, goodsIds := range joinRecallGoodsIds {
		matchedGoodsIds := filterRecommendRecallGoodsIds(goodsIds, targetGoodsIds)
		// 当前来源没有命中目标集合时，不再写入结果。
		if len(matchedGoodsIds) == 0 {
			continue
		}
		result[source] = matchedGoodsIds
	}
	return result
}

// filterRecommendRecallGoodsIds 按目标集合过滤灰度召回商品编号，并保持原顺序。
func filterRecommendRecallGoodsIds(goodsIds []int64, targetGoodsIds []int64) []int64 {
	if len(goodsIds) == 0 || len(targetGoodsIds) == 0 {
		return []int64{}
	}
	targetGoodsIdSet := make(map[int64]struct{}, len(targetGoodsIds))
	for _, goodsId := range targetGoodsIds {
		// 非法商品编号不参与命中判定。
		if goodsId <= 0 {
			continue
		}
		targetGoodsIdSet[goodsId] = struct{}{}
	}
	result := make([]int64, 0, len(goodsIds))
	for _, goodsId := range goodsIds {
		_, ok := targetGoodsIdSet[goodsId]
		// 没有进入目标集合的商品，不计入当前层级命中结果。
		if !ok {
			continue
		}
		result = append(result, goodsId)
	}
	return recommendcore.DedupeInt64s(result)
}

// listRecommendRecallSourceNames 返回稳定排序后的灰度召回来源列表。
func listRecommendRecallSourceNames(joinRecallGoodsIds map[string][]int64) []string {
	if len(joinRecallGoodsIds) == 0 {
		return []string{}
	}
	result := make([]string, 0, len(joinRecallGoodsIds))
	for source, goodsIds := range joinRecallGoodsIds {
		// 没有商品编号的来源不计入返回值。
		if len(goodsIds) == 0 {
			continue
		}
		result = append(result, source)
	}
	sort.Strings(result)
	return result
}

// listContentBasedJoinCandidateGoodsIds 返回允许并入候选池的内容相似商品编号。
func listContentBasedJoinCandidateGoodsIds(probeContext map[string]any) []int64 {
	return listRecommendRecallProbeJoinGoodsIds(probeContext, "contentBased")
}

// listSimilarUserProbeUserIds 返回相似用户探针命中的用户编号。
func listSimilarUserProbeUserIds(probeContext map[string]any) []int64 {
	probeItem := loadRecommendRecallProbeItem(probeContext, "similarUser")
	userIds, ok := probeItem["userIds"].([]int64)
	if !ok {
		return []int64{}
	}
	return recommendcore.DedupeInt64s(userIds)
}

// listCollaborativeFilteringJoinCandidateGoodsIds 返回允许并入候选池的协同过滤商品编号。
func listCollaborativeFilteringJoinCandidateGoodsIds(probeContext map[string]any) []int64 {
	return listRecommendRecallProbeJoinGoodsIds(probeContext, "collaborativeFiltering")
}

// listRecommendRecallProbeJoinGoodsIds 返回指定探针中允许并入候选池的商品编号。
func listRecommendRecallProbeJoinGoodsIds(probeContext map[string]any, key string) []int64 {
	probeItem := loadRecommendRecallProbeItem(probeContext, key)
	// 探针未启用候选融合时，不返回任何商品编号。
	if !shouldJoinRecommendRecallProbeCandidate(probeItem) {
		return []int64{}
	}
	goodsIds, ok := probeItem["goodsIds"].([]int64)
	if !ok {
		return []int64{}
	}
	return recommendcore.DedupeInt64s(goodsIds)
}

// shouldJoinRecommendRecallProbeCandidate 判断探针结果是否允许并入候选池。
func shouldJoinRecommendRecallProbeCandidate(probeItem map[string]any) bool {
	joinCandidate, ok := probeItem["joinCandidate"].(bool)
	if !ok || !joinCandidate {
		return false
	}
	enabled, ok := probeItem["enabled"].(bool)
	return ok && enabled
}

// loadRecommendRecallProbeItem 读取指定探针的上下文字段。
func loadRecommendRecallProbeItem(probeContext map[string]any, key string) map[string]any {
	if len(probeContext) == 0 || key == "" {
		return map[string]any{}
	}
	probeItem, ok := probeContext[key].(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return probeItem
}

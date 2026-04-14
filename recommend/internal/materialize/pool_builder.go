package materialize

import (
	"context"
	"errors"
	"math"
	recommendv1 "recommend/api/gen/go/recommend/v1"
	"recommend/contract"
	cachex "recommend/internal/cache"
	cacheleveldb "recommend/internal/cache/leveldb"
	"recommend/internal/core"
	"sort"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	// defaultBuildLimit 表示离线构建默认保留的候选数量上限。
	defaultBuildLimit = 50
	// defaultNeighborLimit 表示相似用户离线构建默认保留的邻居数量上限。
	defaultNeighborLimit = 20
)

var (
	// defaultScenes 表示离线构建默认覆盖的全部场景。
	defaultScenes = []core.Scene{
		core.SceneHome,
		core.SceneGoodsDetail,
		core.SceneCart,
		core.SceneProfile,
		core.SceneOrderDetail,
		core.SceneOrderPaid,
	}
	// defaultUserCandidateScenes 表示用户候选池默认覆盖的场景集合。
	defaultUserCandidateScenes = []core.Scene{
		core.SceneHome,
		core.SceneProfile,
	}
	// defaultRelationScenes 表示商品关联池默认覆盖的场景集合。
	defaultRelationScenes = []core.Scene{
		core.SceneGoodsDetail,
		core.SceneCart,
		core.SceneOrderDetail,
		core.SceneOrderPaid,
	}
)

// BuildNonPersonalized 构建最新商品、场景热销和全站热销候选池。
func BuildNonPersonalized(ctx context.Context, dependencies core.Dependencies, config core.ServiceConfig, request core.BuildNonPersonalizedRequest) (*core.BuildResult, error) {
	err := validateBuildDependencies(dependencies, true)
	if err != nil {
		return nil, err
	}

	manager, err := cacheleveldb.OpenManager(ctx, dependencies.Cache)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = manager.Close()
	}()

	store := &cachex.PoolStore{Driver: manager}
	scenes := normalizeScenes(request.Scenes, defaultScenes)
	limit := normalizeBuildLimit(request.Limit)
	queryTime := resolveBuildTime(request.StatDate)
	updatedAt := time.Now()
	keyCount := int64(0)

	for _, scene := range scenes {
		items, err := buildNonPersonalizedItems(ctx, dependencies, config, scene, limit, queryTime)
		if err != nil {
			return nil, err
		}
		pool := &recommendv1.RecommendCandidatePool{
			Meta:  buildPoolMeta(string(scene), int32(core.ActorTypeAnonymous), 0, updatedAt),
			Items: items,
		}
		err = store.SaveCandidatePool(string(scene), int32(core.ActorTypeAnonymous), 0, pool)
		if err != nil {
			return nil, err
		}
		keyCount++
	}

	return buildResult("non_personalized", keyCount, updatedAt), nil
}

// BuildUserCandidate 构建用户商品偏好和类目偏好候选池。
func BuildUserCandidate(ctx context.Context, dependencies core.Dependencies, request core.BuildUserCandidateRequest) (*core.BuildResult, error) {
	err := validateBuildDependencies(dependencies, true)
	if err != nil {
		return nil, err
	}

	manager, err := cacheleveldb.OpenManager(ctx, dependencies.Cache)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = manager.Close()
	}()

	store := &cachex.PoolStore{Driver: manager}
	userIds := normalizeIds(request.UserIds)
	limit := normalizeBuildLimit(request.Limit)
	updatedAt := time.Now()
	keyCount := int64(0)

	for _, userId := range userIds {
		items, err := buildUserCandidateItems(ctx, dependencies, userId, limit)
		if err != nil {
			return nil, err
		}
		for _, scene := range defaultUserCandidateScenes {
			pool := &recommendv1.RecommendUserCandidatePool{
				Meta:   buildPoolMeta(string(scene), int32(core.ActorTypeUser), userId, updatedAt),
				UserId: userId,
				Items:  items,
			}
			err = store.SaveUserCandidatePool(string(scene), userId, pool)
			if err != nil {
				return nil, err
			}
			keyCount++
		}
	}

	return buildResult("user_candidate", keyCount, updatedAt), nil
}

// BuildGoodsRelation 构建商品关联候选池。
func BuildGoodsRelation(ctx context.Context, dependencies core.Dependencies, request core.BuildGoodsRelationRequest) (*core.BuildResult, error) {
	err := validateBuildDependencies(dependencies, true)
	if err != nil {
		return nil, err
	}

	manager, err := cacheleveldb.OpenManager(ctx, dependencies.Cache)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = manager.Close()
	}()

	store := &cachex.PoolStore{Driver: manager}
	goodsIds := normalizeIds(request.GoodsIds)
	limit := normalizeBuildLimit(request.Limit)
	updatedAt := time.Now()
	keyCount := int64(0)

	for _, goodsId := range goodsIds {
		rows, err := dependencies.Recommend.ListRelatedGoods(ctx, goodsId, limit)
		if err != nil {
			return nil, err
		}
		items, err := buildWeightedGoodsItems(ctx, dependencies.Goods, rows, "goods_relation")
		if err != nil {
			return nil, err
		}
		for _, scene := range defaultRelationScenes {
			pool := &recommendv1.RecommendRelatedGoodsPool{
				Meta:          buildPoolMeta(string(scene), 0, 0, updatedAt),
				SourceGoodsId: goodsId,
				Items:         items,
			}
			err = store.SaveRelatedGoodsPool(string(scene), goodsId, pool)
			if err != nil {
				return nil, err
			}
			keyCount++
		}
	}

	return buildResult("goods_relation", keyCount, updatedAt), nil
}

// BuildUserToUser 构建相似用户召回所需的邻居用户池和商品候选池。
func BuildUserToUser(ctx context.Context, dependencies core.Dependencies, request core.BuildUserToUserRequest) (*core.BuildResult, error) {
	err := validateBuildDependencies(dependencies, true)
	if err != nil {
		return nil, err
	}

	manager, err := cacheleveldb.OpenManager(ctx, dependencies.Cache)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = manager.Close()
	}()

	store := &cachex.PoolStore{Driver: manager}
	userIds := normalizeIds(request.UserIds)
	neighborLimit := normalizeNeighborLimit(request.NeighborLimit)
	limit := normalizeBuildLimit(request.Limit)
	updatedAt := time.Now()
	keyCount := int64(0)

	for _, userId := range userIds {
		neighborRows, err := dependencies.Recommend.ListNeighborUsers(ctx, userId, neighborLimit)
		if err != nil {
			return nil, err
		}
		itemRows, err := dependencies.Recommend.ListUserToUserGoods(ctx, userId, limit)
		if err != nil {
			return nil, err
		}
		items, err := buildWeightedGoodsItems(ctx, dependencies.Goods, itemRows, "user_to_user")
		if err != nil {
			return nil, err
		}
		pool := &recommendv1.RecommendUserNeighborPool{
			Meta:            buildPoolMeta("", int32(core.ActorTypeUser), userId, updatedAt),
			UserId:          userId,
			NeighborUserIds: buildNeighborUserIds(neighborRows, int(neighborLimit)),
			Items:           items,
		}
		err = store.SaveUserNeighborPool(userId, pool)
		if err != nil {
			return nil, err
		}
		keyCount++
	}

	return buildResult("user_to_user", keyCount, updatedAt), nil
}

// BuildCollaborative 构建协同过滤候选池。
func BuildCollaborative(ctx context.Context, dependencies core.Dependencies, request core.BuildCollaborativeRequest) (*core.BuildResult, error) {
	err := validateBuildDependencies(dependencies, true)
	if err != nil {
		return nil, err
	}

	manager, err := cacheleveldb.OpenManager(ctx, dependencies.Cache)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = manager.Close()
	}()

	store := &cachex.PoolStore{Driver: manager}
	userIds := normalizeIds(request.UserIds)
	limit := normalizeBuildLimit(request.Limit)
	updatedAt := time.Now()
	keyCount := int64(0)

	for _, userId := range userIds {
		rows, err := dependencies.Recommend.ListCollaborativeGoods(ctx, userId, limit)
		if err != nil {
			return nil, err
		}
		items, err := buildWeightedGoodsItems(ctx, dependencies.Goods, rows, "collaborative")
		if err != nil {
			return nil, err
		}
		for _, scene := range defaultScenes {
			pool := &recommendv1.RecommendCollaborativePool{
				Meta:   buildPoolMeta(string(scene), int32(core.ActorTypeUser), userId, updatedAt),
				UserId: userId,
				Items:  items,
			}
			err = store.SaveCollaborativePool(string(scene), userId, pool)
			if err != nil {
				return nil, err
			}
			keyCount++
		}
	}

	return buildResult("collaborative", keyCount, updatedAt), nil
}

// BuildExternal 构建活动池、营销池、人工池等外部推荐池。
func BuildExternal(ctx context.Context, dependencies core.Dependencies, request core.BuildExternalRequest) (*core.BuildResult, error) {
	err := validateBuildDependencies(dependencies, true)
	if err != nil {
		return nil, err
	}

	manager, err := cacheleveldb.OpenManager(ctx, dependencies.Cache)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = manager.Close()
	}()

	store := &cachex.PoolStore{Driver: manager}
	scenes := normalizeScenes(request.Scenes, defaultScenes)
	strategies := normalizeStrategies(request.Strategies)
	actorIds := normalizeActorIds(request.ActorIds)
	limit := normalizeBuildLimit(request.Limit)
	updatedAt := time.Now()
	keyCount := int64(0)

	for _, scene := range scenes {
		for _, strategy := range strategies {
			for _, actorId := range actorIds {
				rows, err := dependencies.Recommend.ListExternalGoods(ctx, string(scene), strategy, int32(request.ActorType), actorId, limit)
				if err != nil {
					return nil, err
				}
				items, err := buildWeightedGoodsItems(ctx, dependencies.Goods, rows, "external")
				if err != nil {
					return nil, err
				}
				pool := &recommendv1.RecommendExternalPool{
					Meta:     buildPoolMeta(string(scene), int32(request.ActorType), actorId, updatedAt),
					Strategy: strategy,
					Items:    items,
				}
				err = store.SaveExternalPool(string(scene), strategy, int32(request.ActorType), actorId, pool)
				if err != nil {
					return nil, err
				}
				keyCount++
			}
		}
	}

	return buildResult("external", keyCount, updatedAt), nil
}

// validateBuildDependencies 校验离线构建依赖。
func validateBuildDependencies(dependencies core.Dependencies, requireRecommend bool) error {
	// 离线构建需要缓存落库能力，否则无法产出任何可消费结果。
	if dependencies.Cache == nil {
		return errors.New("recommend: 缓存数据源未配置")
	}
	// 商品信息缺失时，无法给候选池补齐分类和上下架属性。
	if dependencies.Goods == nil {
		return errors.New("recommend: 商品数据源未配置")
	}
	// 依赖推荐事实表的构建动作必须提供推荐数据源。
	if requireRecommend && dependencies.Recommend == nil {
		return errors.New("recommend: 推荐数据源未配置")
	}
	return nil
}

// validateCacheAndRecommendDependencies 校验仅依赖缓存与推荐事实的构建依赖。
func validateCacheAndRecommendDependencies(dependencies core.Dependencies) error {
	// 相似用户池不需要商品详情，但仍然需要缓存和推荐事实来源。
	if dependencies.Cache == nil {
		return errors.New("recommend: 缓存数据源未配置")
	}
	if dependencies.Recommend == nil {
		return errors.New("recommend: 推荐数据源未配置")
	}
	return nil
}

// buildNonPersonalizedItems 构建单个场景的非个性化候选列表。
func buildNonPersonalizedItems(
	ctx context.Context,
	dependencies core.Dependencies,
	config core.ServiceConfig,
	scene core.Scene,
	limit int32,
	queryTime time.Time,
) ([]*recommendv1.RecommendCandidateItem, error) {
	itemMap := make(map[int64]*recommendv1.RecommendCandidateItem)
	for _, sourceName := range resolveNonPersonalizedSources(config.Strategy.NonPersonalizedSources) {
		items, err := buildNonPersonalizedSourceItems(ctx, dependencies, scene, limit, queryTime, sourceName)
		if err != nil {
			return nil, err
		}
		mergeCandidateItems(itemMap, items)
	}

	return finalizeCandidateItems(itemMap, int(limit)), nil
}

// resolveNonPersonalizedSources 解析非个性化池使用的来源顺序。
func resolveNonPersonalizedSources(sourceNames []string) []string {
	if len(sourceNames) == 0 {
		return []string{"latest", "scene_hot", "global_hot"}
	}
	result := make([]string, 0, len(sourceNames))
	seen := make(map[string]struct{}, len(sourceNames))
	for _, sourceName := range sourceNames {
		if sourceName == "" {
			continue
		}
		if _, ok := seen[sourceName]; ok {
			continue
		}
		seen[sourceName] = struct{}{}
		result = append(result, sourceName)
	}
	if len(result) == 0 {
		return []string{"latest", "scene_hot", "global_hot"}
	}
	return result
}

// buildNonPersonalizedSourceItems 构建单一路非个性化来源的候选商品项。
func buildNonPersonalizedSourceItems(
	ctx context.Context,
	dependencies core.Dependencies,
	scene core.Scene,
	limit int32,
	queryTime time.Time,
	sourceName string,
) ([]*recommendv1.RecommendCandidateItem, error) {
	switch sourceName {
	// 最新商品直接来自商品主表，不依赖推荐统计事实。
	case "latest":
		latestGoods, err := dependencies.Goods.ListLatestGoods(ctx, limit)
		if err != nil {
			return nil, err
		}
		return buildLatestItems(latestGoods), nil
	// 场景热销依赖推荐事实聚合结果。
	case "scene_hot":
		rows, err := dependencies.Recommend.ListSceneHotGoods(ctx, string(scene), queryTime, limit)
		if err != nil {
			return nil, err
		}
		return buildWeightedGoodsItems(ctx, dependencies.Goods, rows, "scene_hot")
	// 全站热销依赖推荐事实聚合结果。
	case "global_hot":
		rows, err := dependencies.Recommend.ListGlobalHotGoods(ctx, queryTime, limit)
		if err != nil {
			return nil, err
		}
		return buildWeightedGoodsItems(ctx, dependencies.Goods, rows, "global_hot")
	default:
		return nil, nil
	}
}

// buildUserCandidateItems 构建单个用户的用户候选池。
func buildUserCandidateItems(
	ctx context.Context,
	dependencies core.Dependencies,
	userId int64,
	limit int32,
) ([]*recommendv1.RecommendCandidateItem, error) {
	itemMap := make(map[int64]*recommendv1.RecommendCandidateItem)

	userGoodsRows, err := dependencies.Recommend.ListUserGoodsPreference(ctx, userId, limit)
	if err != nil {
		return nil, err
	}
	userGoodsItems, err := buildWeightedGoodsItems(ctx, dependencies.Goods, userGoodsRows, "user_goods_pref")
	if err != nil {
		return nil, err
	}
	mergeCandidateItems(itemMap, userGoodsItems)

	categoryRows, err := dependencies.Recommend.ListUserCategoryPreference(ctx, userId, limit)
	if err != nil {
		return nil, err
	}
	categoryItems, err := buildCategoryItems(ctx, dependencies.Goods, categoryRows, limit)
	if err != nil {
		return nil, err
	}
	mergeCandidateItems(itemMap, categoryItems)

	return finalizeCandidateItems(itemMap, int(limit)), nil
}

// buildWeightedGoodsItems 将带分商品事实转换为候选商品项。
func buildWeightedGoodsItems(
	ctx context.Context,
	goodsSource contract.GoodsSource,
	rows []*contract.WeightedGoods,
	recallSource string,
) ([]*recommendv1.RecommendCandidateItem, error) {
	scoreByGoodsId := make(map[int64]float64, len(rows))
	goodsIds := make([]int64, 0, len(rows))

	for _, item := range rows {
		// 非法商品编号不参与缓存构建。
		if item == nil || item.GoodsId <= 0 {
			continue
		}
		if _, ok := scoreByGoodsId[item.GoodsId]; !ok {
			goodsIds = append(goodsIds, item.GoodsId)
		}
		scoreByGoodsId[item.GoodsId] += item.Score
	}
	if len(goodsIds) == 0 {
		return nil, nil
	}

	list, err := goodsSource.ListGoods(ctx, goodsIds)
	if err != nil {
		return nil, err
	}

	items := make([]*recommendv1.RecommendCandidateItem, 0, len(list))
	for _, goods := range list {
		// 商品实体缺失时，无法补齐类目等缓存字段。
		if goods == nil || goods.Id <= 0 {
			continue
		}
		score := scoreByGoodsId[goods.Id]
		items = append(items, &recommendv1.RecommendCandidateItem{
			GoodsId:       goods.Id,
			Score:         score,
			RecallSources: []string{recallSource},
			CategoryId:    goods.CategoryId,
			Trace:         recallSource,
			SourceScores: map[string]float64{
				recallSource: score,
			},
		})
	}
	return items, nil
}

// buildCategoryItems 将类目偏好事实转换为候选商品项。
func buildCategoryItems(
	ctx context.Context,
	goodsSource contract.GoodsSource,
	rows []*contract.WeightedCategory,
	limit int32,
) ([]*recommendv1.RecommendCandidateItem, error) {
	categoryScoreMap := make(map[int64]float64, len(rows))
	categoryIds := make([]int64, 0, len(rows))

	for _, item := range rows {
		// 非法类目编号不参与缓存构建。
		if item == nil || item.CategoryId <= 0 {
			continue
		}
		if _, ok := categoryScoreMap[item.CategoryId]; !ok {
			categoryIds = append(categoryIds, item.CategoryId)
		}
		categoryScoreMap[item.CategoryId] += item.Score
	}
	if len(categoryIds) == 0 {
		return nil, nil
	}

	list, err := goodsSource.ListGoodsByCategoryIds(ctx, categoryIds, limit)
	if err != nil {
		return nil, err
	}

	items := make([]*recommendv1.RecommendCandidateItem, 0, len(list))
	for _, goods := range list {
		// 商品实体缺失或类目偏好未命中时，不写入候选池。
		if goods == nil || goods.Id <= 0 {
			continue
		}
		score, ok := categoryScoreMap[goods.CategoryId]
		if !ok {
			continue
		}
		items = append(items, &recommendv1.RecommendCandidateItem{
			GoodsId:       goods.Id,
			Score:         score,
			RecallSources: []string{"user_category_pref"},
			CategoryId:    goods.CategoryId,
			Trace:         "user_category_pref",
			SourceScores: map[string]float64{
				"user_category_pref": score,
			},
		})
	}
	return items, nil
}

// buildLatestItems 将最新商品列表转换为候选商品项。
func buildLatestItems(list []*contract.Goods) []*recommendv1.RecommendCandidateItem {
	result := make([]*recommendv1.RecommendCandidateItem, 0, len(list))
	total := len(list)
	for index, goods := range list {
		// 商品实体缺失时，不进入最新商品候选池。
		if goods == nil || goods.Id <= 0 {
			continue
		}
		score := 1.0
		// 列表存在多条商品时，按位次给一个递减分值，便于和其他来源合并排序。
		if total > 1 {
			score = float64(total-index) / float64(total)
		}
		result = append(result, &recommendv1.RecommendCandidateItem{
			GoodsId:       goods.Id,
			Score:         score,
			RecallSources: []string{"latest"},
			CategoryId:    goods.CategoryId,
			Trace:         "latest",
			SourceScores: map[string]float64{
				"latest": score,
			},
		})
	}
	return result
}

// mergeCandidateItems 合并多路候选商品项。
func mergeCandidateItems(target map[int64]*recommendv1.RecommendCandidateItem, items []*recommendv1.RecommendCandidateItem) {
	for _, item := range items {
		// 非法商品项不参与池内合并。
		if item == nil || item.GetGoodsId() <= 0 {
			continue
		}
		existing, ok := target[item.GetGoodsId()]
		if !ok {
			target[item.GetGoodsId()] = cloneCandidateItem(item)
			continue
		}
		existing.Score += item.GetScore()
		existing.RecallSources = mergeRecallSources(existing.GetRecallSources(), item.GetRecallSources())
		existing.SourceScores = mergeSourceScores(existing.GetSourceScores(), item.GetSourceScores())
	}
}

// finalizeCandidateItems 将候选商品 map 转成稳定有序的列表。
func finalizeCandidateItems(itemMap map[int64]*recommendv1.RecommendCandidateItem, limit int) []*recommendv1.RecommendCandidateItem {
	items := make([]*recommendv1.RecommendCandidateItem, 0, len(itemMap))
	for _, item := range itemMap {
		// 空商品项理论上不会出现，但这里仍做一次兜底过滤。
		if item == nil {
			continue
		}
		item.RecallSources = mergeRecallSources(item.GetRecallSources())
		items = append(items, item)
	}
	sort.SliceStable(items, func(i int, j int) bool {
		// 候选池默认按综合得分倒序，分值相同再按商品编号稳定排序。
		if items[i].GetScore() != items[j].GetScore() {
			return items[i].GetScore() > items[j].GetScore()
		}
		return items[i].GetGoodsId() < items[j].GetGoodsId()
	})
	// 调用方指定了上限时，只保留前 N 个候选，避免缓存无限膨胀。
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items
}

// cloneCandidateItem 复制候选商品项。
func cloneCandidateItem(item *recommendv1.RecommendCandidateItem) *recommendv1.RecommendCandidateItem {
	return &recommendv1.RecommendCandidateItem{
		GoodsId:       item.GetGoodsId(),
		Score:         item.GetScore(),
		RecallSources: append([]string(nil), item.GetRecallSources()...),
		CategoryId:    item.GetCategoryId(),
		Trace:         item.GetTrace(),
		SourceScores:  mergeSourceScores(item.GetSourceScores()),
	}
}

// mergeRecallSources 合并并去重召回来源。
func mergeRecallSources(sourceGroups ...[]string) []string {
	if len(sourceGroups) == 0 {
		return nil
	}

	sourceSet := make(map[string]struct{})
	for _, group := range sourceGroups {
		for _, source := range group {
			// 空召回来源没有业务意义，不写入缓存。
			if source == "" {
				continue
			}
			sourceSet[source] = struct{}{}
		}
	}

	result := make([]string, 0, len(sourceSet))
	for source := range sourceSet {
		result = append(result, source)
	}
	sort.Strings(result)
	return result
}

// mergeSourceScores 合并并累计各召回来源原始得分。
func mergeSourceScores(sourceMaps ...map[string]float64) map[string]float64 {
	if len(sourceMaps) == 0 {
		return nil
	}

	result := make(map[string]float64)
	for _, sourceMap := range sourceMaps {
		for source, score := range sourceMap {
			// 空召回来源或非法分值不需要写入缓存。
			if source == "" || score == 0 {
				continue
			}
			result[source] += score
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// buildNeighborUserIds 构建稳定排序后的相似用户列表。
func buildNeighborUserIds(rows []*contract.WeightedUser, limit int) []int64 {
	list := make([]*contract.WeightedUser, 0, len(rows))
	for _, item := range rows {
		// 非法用户编号不进入相似用户池。
		if item == nil || item.UserId <= 0 {
			continue
		}
		list = append(list, item)
	}
	sort.SliceStable(list, func(i int, j int) bool {
		// 相似用户优先按相似度倒序，再按用户编号稳定排序。
		if list[i].Score != list[j].Score {
			return list[i].Score > list[j].Score
		}
		return list[i].UserId < list[j].UserId
	})

	result := make([]int64, 0, len(list))
	seen := make(map[int64]struct{}, len(list))
	for _, item := range list {
		if _, ok := seen[item.UserId]; ok {
			continue
		}
		seen[item.UserId] = struct{}{}
		result = append(result, item.UserId)
		// 超过邻居上限后，后续用户不再继续写入。
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result
}

// buildPoolMeta 构建候选池缓存元信息。
func buildPoolMeta(scene string, actorType int32, actorId int64, updatedAt time.Time) *recommendv1.CacheMeta {
	if updatedAt.IsZero() {
		updatedAt = time.Now()
	}
	return &recommendv1.CacheMeta{
		SchemaVersion: "v1",
		Scene:         scene,
		ActorType:     actorType,
		ActorId:       actorId,
		UpdatedAt:     timestamppb.New(updatedAt),
	}
}

// buildResult 构建统一的离线构建结果。
func buildResult(scope string, keyCount int64, updatedAt time.Time) *core.BuildResult {
	return &core.BuildResult{
		Scope:     scope,
		KeyCount:  keyCount,
		UpdatedAt: updatedAt,
	}
}

// normalizeScenes 归一化场景列表。
func normalizeScenes(input []core.Scene, defaults []core.Scene) []core.Scene {
	list := input
	if len(list) == 0 {
		list = defaults
	}

	result := make([]core.Scene, 0, len(list))
	seen := make(map[core.Scene]struct{}, len(list))
	for _, scene := range list {
		// 空场景不参与离线构建，避免写出无效键。
		if scene == "" {
			continue
		}
		if _, ok := seen[scene]; ok {
			continue
		}
		seen[scene] = struct{}{}
		result = append(result, scene)
	}
	return result
}

// normalizeIds 归一化编号列表。
func normalizeIds(input []int64) []int64 {
	result := make([]int64, 0, len(input))
	seen := make(map[int64]struct{}, len(input))
	for _, id := range input {
		// 非法编号不参与离线构建，避免生成错误缓存键。
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

// normalizeActorIds 归一化外部池使用的主体编号。
func normalizeActorIds(input []int64) []int64 {
	result := normalizeIds(input)
	// 外部池允许构建无主体兜底池，因此空列表回退到 0。
	if len(result) == 0 {
		return []int64{0}
	}
	return result
}

// normalizeStrategies 归一化外部策略列表。
func normalizeStrategies(input []string) []string {
	result := make([]string, 0, len(input))
	seen := make(map[string]struct{}, len(input))
	for _, strategy := range input {
		// 空策略名无法形成稳定缓存键，不参与构建。
		if strategy == "" {
			continue
		}
		if _, ok := seen[strategy]; ok {
			continue
		}
		seen[strategy] = struct{}{}
		result = append(result, strategy)
	}
	return result
}

// normalizeBuildLimit 归一化离线构建的数量上限。
func normalizeBuildLimit(limit int32) int32 {
	if limit <= 0 {
		return defaultBuildLimit
	}
	return int32(math.Max(1, float64(limit)))
}

// normalizeNeighborLimit 归一化相似用户数量上限。
func normalizeNeighborLimit(limit int32) int32 {
	if limit <= 0 {
		return defaultNeighborLimit
	}
	return limit
}

// resolveBuildTime 归一化离线构建使用的时间。
func resolveBuildTime(statDate time.Time) time.Time {
	if statDate.IsZero() {
		return time.Now()
	}
	return statDate
}

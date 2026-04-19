package task

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	app "shop/api/gen/go/app"
	"shop/api/gen/go/common"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	recommendCache "shop/pkg/recommend/cache"
	recommendEvent "shop/pkg/recommend/event"
	"shop/pkg/recommend/offline/materialize"
	appBiz "shop/service/app/biz"
	appDto "shop/service/app/dto"

	"github.com/go-kratos/kratos/v2/log"
)

// RecommendResultMaterialize 首页登录态最终推荐结果写缓存任务。
type RecommendResultMaterialize struct {
	recommendRequestCase     *appBiz.RecommendRequestCase
	recommendRequestRepo     *data.RecommendRequestRepo
	recommendGoodsActionRepo *data.RecommendGoodsActionRepo
	store                    recommendCache.Store
	materializer             *materialize.Materializer
	ctx                      context.Context
}

// recommendActiveUserRow 表示活跃用户及其最近活跃时间。
type recommendActiveUserRow struct {
	ActorId  int64     `gorm:"column:actor_id"`
	RecentAt time.Time `gorm:"column:recent_at"`
}

// NewRecommendResultMaterialize 创建首页登录态最终推荐结果写缓存任务实例。
func NewRecommendResultMaterialize(
	recommendRequestCase *appBiz.RecommendRequestCase,
	recommendRequestRepo *data.RecommendRequestRepo,
	recommendGoodsActionRepo *data.RecommendGoodsActionRepo,
	store recommendCache.Store,
	materializer *materialize.Materializer,
) *RecommendResultMaterialize {
	return &RecommendResultMaterialize{
		recommendRequestCase:     recommendRequestCase,
		recommendRequestRepo:     recommendRequestRepo,
		recommendGoodsActionRepo: recommendGoodsActionRepo,
		store:                    store,
		materializer:             materializer,
		ctx:                      context.Background(),
	}
}

// Exec 执行首页登录态最终推荐结果写缓存任务。
func (t *RecommendResultMaterialize) Exec(args map[string]string) ([]string, error) {
	log.Infof("Job RecommendResultMaterialize Exec %+v", args)

	limit, err := parseRecommendMaterializeLimitArg(args["limit"])
	if err != nil {
		return []string{err.Error()}, err
	}
	lookbackDays, err := parseRecommendTrainLookbackDaysArg(args["lookbackDays"], 30)
	if err != nil {
		return []string{err.Error()}, err
	}
	userLimit, err := parseRecommendMaterializeUserLimitArg(args["userLimit"], 500)
	if err != nil {
		return []string{err.Error()}, err
	}
	stats := newRecommendMaterializeStats("RecommendResultMaterialize", limit)

	stats.SetStage("load_active_users")
	activeUserIds, err := loadRecommendActiveUserIds(t.ctx, t.recommendRequestRepo, t.recommendGoodsActionRepo, lookbackDays, userLimit)
	if err != nil {
		return returnRecommendMaterializeFailure(stats, err)
	}
	stats.AddInputCount("active_user_count", len(activeUserIds))

	currentSubsetMap := make(map[string]struct{}, len(activeUserIds))
	result := make([]string, 0, len(activeUserIds)+2)
	for _, userId := range activeUserIds {
		actor := &appDto.RecommendActor{
			ActorType: recommendEvent.ActorTypeUser,
			ActorId:   userId,
		}
		req := &app.RecommendGoodsRequest{
			Scene:    common.RecommendScene_HOME,
			PageNum:  1,
			PageSize: limit,
		}
		stats.SetStage(fmt.Sprintf("build_home_recommend_user_%d", userId))
		pageResult, buildErr := t.recommendRequestCase.BuildOnlineRecommendPageResult(t.ctx, actor, req)
		if buildErr != nil {
			return returnRecommendMaterializeFailure(stats, buildErr)
		}
		version := extractRecommendResultVersion(pageResult.SourceContext)
		stats.AddVersion(version)
		documentList := buildRecommendResultDocuments(pageResult.List)
		subset := recommendCache.RecommendSubset(int32(common.RecommendScene_HOME), recommendEvent.ActorTypeUser, userId, version)
		// 当前用户确实生成了可读结果时，才把子集合计入本轮保留清单。
		if len(documentList) > 0 {
			currentSubsetMap[subset] = struct{}{}
		}
		stats.SetStage(fmt.Sprintf("publish_home_recommend_user_%d", userId))
		err = t.materializer.MaterializeRecommend(
			t.ctx,
			int32(common.RecommendScene_HOME),
			recommendEvent.ActorTypeUser,
			userId,
			version,
			documentList,
		)
		if err != nil {
			return returnRecommendMaterializeFailure(stats, err)
		}
		stats.AddPublishedSubset(len(documentList))
		result = append(result, fmt.Sprintf("user=%d version=%s count=%d", userId, version, len(documentList)))
	}

	stats.SetStage("clear_stale_home_recommend")
	clearedSubsetCount, err := clearStaleRecommendResultSubsets(
		t.ctx,
		t.store,
		int32(common.RecommendScene_HOME),
		recommendEvent.ActorTypeUser,
		currentSubsetMap,
	)
	if err != nil {
		return returnRecommendMaterializeFailure(stats, err)
	}
	stats.AddClearedSubsets(clearedSubsetCount)
	result = append(result, fmt.Sprintf(
		"scene=%d user_limit=%d active_users=%d cleared_subsets=%d",
		common.RecommendScene_HOME,
		userLimit,
		len(activeUserIds),
		clearedSubsetCount,
	))
	result = append(result, stats.BuildSummary())
	stats.LogSummary()
	return result, nil
}

// parseRecommendMaterializeUserLimitArg 解析最终推荐缓存任务的活跃用户上限参数。
func parseRecommendMaterializeUserLimitArg(value string, defaultValue int) (int, error) {
	trimmedValue := strings.TrimSpace(value)
	// 未显式传用户上限时，回退到任务默认值。
	if trimmedValue == "" {
		return defaultValue, nil
	}
	userLimit, err := strconv.Atoi(trimmedValue)
	if err != nil {
		return 0, errorsx.InvalidArgument("userLimit 格式错误")
	}
	// 活跃用户上限必须大于零，避免生成空任务窗口。
	if userLimit <= 0 {
		return 0, errorsx.InvalidArgument("userLimit 必须大于 0")
	}
	return userLimit, nil
}

// loadRecommendActiveUserIds 加载本轮需要预生成首页推荐结果的活跃用户集合。
func loadRecommendActiveUserIds(
	ctx context.Context,
	recommendRequestRepo *data.RecommendRequestRepo,
	recommendGoodsActionRepo *data.RecommendGoodsActionRepo,
	lookbackDays int,
	userLimit int,
) ([]int64, error) {
	// 活跃用户上限非法时，不继续查询活跃用户集合。
	if userLimit <= 0 {
		return []int64{}, nil
	}

	startAt := time.Now().AddDate(0, 0, -lookbackDays)
	requestRows, err := loadRecommendActiveRequestUsers(ctx, recommendRequestRepo, startAt, userLimit)
	if err != nil {
		return nil, err
	}
	actionRows, err := loadRecommendActiveActionUsers(ctx, recommendGoodsActionRepo, startAt, userLimit)
	if err != nil {
		return nil, err
	}
	return mergeRecommendActiveUsers(requestRows, actionRows, userLimit), nil
}

// loadRecommendActiveRequestUsers 查询最近一段时间内有推荐请求的登录用户。
func loadRecommendActiveRequestUsers(
	ctx context.Context,
	recommendRequestRepo *data.RecommendRequestRepo,
	startAt time.Time,
	userLimit int,
) ([]recommendActiveUserRow, error) {
	// 仓储未注入或用户上限非法时，不继续查询请求活跃用户。
	if recommendRequestRepo == nil || userLimit <= 0 {
		return []recommendActiveUserRow{}, nil
	}

	rows := make([]recommendActiveUserRow, 0, userLimit)
	err := recommendRequestRepo.Query(ctx).RecommendRequest.WithContext(ctx).UnderlyingDB().
		Model(&models.RecommendRequest{}).
		Select("actor_id, MAX(created_at) AS recent_at").
		Where("actor_type = ?", recommendEvent.ActorTypeUser).
		Where("created_at >= ?", startAt).
		Group("actor_id").
		Order("recent_at DESC").
		Limit(userLimit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// loadRecommendActiveActionUsers 查询最近一段时间内有推荐行为的登录用户。
func loadRecommendActiveActionUsers(
	ctx context.Context,
	recommendGoodsActionRepo *data.RecommendGoodsActionRepo,
	startAt time.Time,
	userLimit int,
) ([]recommendActiveUserRow, error) {
	// 仓储未注入或用户上限非法时，不继续查询行为活跃用户。
	if recommendGoodsActionRepo == nil || userLimit <= 0 {
		return []recommendActiveUserRow{}, nil
	}

	rows := make([]recommendActiveUserRow, 0, userLimit)
	err := recommendGoodsActionRepo.Query(ctx).RecommendGoodsAction.WithContext(ctx).UnderlyingDB().
		Model(&models.RecommendGoodsAction{}).
		Select("actor_id, MAX(created_at) AS recent_at").
		Where("actor_type = ?", recommendEvent.ActorTypeUser).
		Where("created_at >= ?", startAt).
		Group("actor_id").
		Order("recent_at DESC").
		Limit(userLimit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// mergeRecommendActiveUsers 合并请求和行为两路活跃用户，并按最近活跃时间排序截断。
func mergeRecommendActiveUsers(requestRows []recommendActiveUserRow, actionRows []recommendActiveUserRow, userLimit int) []int64 {
	activeUserMap := make(map[int64]time.Time, len(requestRows)+len(actionRows))
	for _, row := range requestRows {
		// 非法用户编号不进入活跃用户集合。
		if row.ActorId <= 0 {
			continue
		}
		activeUserMap[row.ActorId] = maxRecommendTime(activeUserMap[row.ActorId], row.RecentAt)
	}
	for _, row := range actionRows {
		// 非法用户编号不进入活跃用户集合。
		if row.ActorId <= 0 {
			continue
		}
		activeUserMap[row.ActorId] = maxRecommendTime(activeUserMap[row.ActorId], row.RecentAt)
	}

	activeUserRows := make([]recommendActiveUserRow, 0, len(activeUserMap))
	for actorId, recentAt := range activeUserMap {
		activeUserRows = append(activeUserRows, recommendActiveUserRow{
			ActorId:  actorId,
			RecentAt: recentAt,
		})
	}
	sort.SliceStable(activeUserRows, func(i int, j int) bool {
		// 活跃时间优先倒序，时间相同时按用户编号升序稳定打平。
		if activeUserRows[i].RecentAt.Equal(activeUserRows[j].RecentAt) {
			return activeUserRows[i].ActorId < activeUserRows[j].ActorId
		}
		return activeUserRows[i].RecentAt.After(activeUserRows[j].RecentAt)
	})
	// 活跃用户数量超过限制时，只保留最近活跃的 TopN 用户。
	if userLimit > 0 && len(activeUserRows) > userLimit {
		activeUserRows = activeUserRows[:userLimit]
	}

	result := make([]int64, 0, len(activeUserRows))
	for _, row := range activeUserRows {
		result = append(result, row.ActorId)
	}
	return result
}

// buildRecommendResultDocuments 将在线推荐页结果转换为最终推荐缓存文档。
func buildRecommendResultDocuments(goodsList []*app.GoodsInfo) []recommendCache.Score {
	documentList := make([]recommendCache.Score, 0, len(goodsList))
	goodsIdSet := make(map[int64]struct{}, len(goodsList))
	for index, item := range goodsList {
		// 非法商品结果不发布到最终推荐缓存。
		if item == nil || item.Id <= 0 {
			continue
		}
		_, exists := goodsIdSet[item.Id]
		// 当前商品已经写入过缓存文档时，不再重复发布。
		if exists {
			continue
		}
		goodsIdSet[item.Id] = struct{}{}
		documentList = append(documentList, recommendCache.Score{
			Id:    strconv.FormatInt(item.Id, 10),
			Score: float64(len(goodsList) - index),
		})
	}
	return documentList
}

// extractRecommendResultVersion 从在线推荐结果上下文里提取当前用户实际命中的缓存版本。
func extractRecommendResultVersion(sourceContext map[string]any) string {
	version := extractRecommendPublishContextVersion(sourceContext)
	// 顶层上下文没有发布信息时，再尝试从收口后的在线调试上下文里解析。
	if version == "" {
		onlineDebugContext, ok := sourceContext["onlineDebugContext"].(map[string]any)
		if ok {
			version = extractRecommendPublishContextVersion(onlineDebugContext)
		}
	}
	// 上下文缺少版本字段时，统一回退到默认缓存版本。
	if version == "" {
		return recommendCache.DefaultVersion
	}
	return recommendCache.NormalizeVersion(version)
}

// extractRecommendPublishContextVersion 从发布上下文里解析有效版本号。
func extractRecommendPublishContextVersion(sourceContext map[string]any) string {
	// 来源上下文为空时，无法继续提取发布信息。
	if len(sourceContext) == 0 {
		return ""
	}
	publishContext, ok := sourceContext["publishContext"].(map[string]any)
	// 发布上下文不存在时，不继续提取有效版本。
	if !ok || len(publishContext) == 0 {
		return ""
	}
	effectiveVersion, ok := publishContext["effectiveVersion"].(string)
	// 当前发布上下文明确给出了有效版本时，优先返回该值。
	if ok && strings.TrimSpace(effectiveVersion) != "" {
		return recommendCache.NormalizeVersion(effectiveVersion)
	}
	sceneVersion, ok := publishContext["sceneVersion"].(string)
	// 没有有效版本时，回退到当前场景启用版本。
	if ok && strings.TrimSpace(sceneVersion) != "" {
		return recommendCache.NormalizeVersion(sceneVersion)
	}
	return ""
}

// clearStaleRecommendResultSubsets 清理首页登录态最终推荐缓存中已经失效的用户子集合。
func clearStaleRecommendResultSubsets(
	ctx context.Context,
	store recommendCache.Store,
	scene int32,
	actorType int32,
	currentSubsetMap map[string]struct{},
) (int, error) {
	// 存储未注入时，跳过旧子集合清理。
	if store == nil {
		return 0, nil
	}

	collectionKey := recommendCache.CollectionKey(recommendCache.Recommend)
	subsetIndexMap, err := store.HGetAll(recommendCache.ScoreSubsetIndexKey(collectionKey))
	if err != nil {
		// 当前集合尚未建立索引时，说明没有旧缓存需要清理。
		if err == recommendCache.ErrObjectNotExist {
			return 0, nil
		}
		return 0, err
	}

	subsetPrefix := fmt.Sprintf("scene/%d/actor_type/%d/", scene, actorType)
	clearedSubsetCount := 0
	for subset := range subsetIndexMap {
		// 只清理首页登录态最终推荐缓存空间下的子集合。
		if !strings.HasPrefix(subset, subsetPrefix) {
			continue
		}
		_, exists := currentSubsetMap[subset]
		// 本轮仍然保留的子集合不做清理。
		if exists {
			continue
		}
		err = store.DeleteScores(ctx, []string{collectionKey}, recommendCache.ScoreCondition{Subset: &subset})
		if err != nil {
			return 0, err
		}
		err = store.Del(recommendCache.DigestKey(recommendCache.Recommend, subset))
		if err != nil {
			return 0, err
		}
		err = store.Del(recommendCache.DocumentCountKey(recommendCache.Recommend, subset))
		if err != nil {
			return 0, err
		}
		err = store.Del(recommendCache.UpdateTimeKey(recommendCache.Recommend, subset))
		if err != nil {
			return 0, err
		}
		clearedSubsetCount++
	}
	return clearedSubsetCount, nil
}

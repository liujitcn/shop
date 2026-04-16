package biz

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"shop/api/gen/go/common"
	"shop/pkg/gen/models"
	recommendCache "shop/pkg/recommend/cache"
	recommendcore "shop/pkg/recommend/core"
	recommendDomain "shop/pkg/recommend/domain"
	recommendOnlineRecall "shop/pkg/recommend/online/recall"

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
func (c *RecommendRequestCase) loadRecommendRecallProbeConfig(ctx context.Context, scene int32) (string, time.Time, *recommendDomain.RecallProbeStrategy, error) {
	version := recommendCache.DefaultVersion
	versionPublishedAt := time.Time{}
	entity, err := c.loadRecommendSceneVersionEntity(ctx, scene)
	if err != nil {
		return "", time.Time{}, nil, err
	}
	// 当前场景没有启用版本时，直接回退到默认版本和空探针配置。
	if entity == nil {
		return version, versionPublishedAt, &recommendDomain.RecallProbeStrategy{}, nil
	}

	version = recommendCache.NormalizeVersion(entity.Version)
	versionPublishedAt = entity.CreatedAt
	config := &recommendDomain.StrategyVersionConfig{}
	// 当前版本没有扩展配置时，直接返回空探针配置。
	if strings.TrimSpace(entity.ConfigJSON) == "" {
		return version, versionPublishedAt, &recommendDomain.RecallProbeStrategy{}, nil
	}

	err = json.Unmarshal([]byte(entity.ConfigJSON), config)
	if err != nil {
		// 阶段 4 的召回探针是增量能力，配置解析失败时不影响主推荐链路。
		log.Errorf("loadRecommendRecallProbeConfig %v", err)
		return version, versionPublishedAt, &recommendDomain.RecallProbeStrategy{}, nil
	}
	if config.RecallProbe == nil {
		return version, versionPublishedAt, &recommendDomain.RecallProbeStrategy{}, nil
	}
	return version, versionPublishedAt, config.RecallProbe, nil
}

// listCachedSimilarUserIds 读取相似用户召回探针缓存。
func (c *RecommendRequestCase) listCachedSimilarUserIds(ctx context.Context, userId int64, version string, versionPublishedAt time.Time, limit int64) (*recommendCacheReadResult, error) {
	// 登录用户编号非法时，不需要继续读取相似用户缓存。
	if userId <= 0 {
		return newRecommendCacheReadResult(
			recommendRecallProbeSimilarUser,
			recommendCache.UserToUser,
			"",
			version,
			versionPublishedAt,
			limit,
			0,
		), nil
	}
	return c.listCachedInt64Ids(
		ctx,
		recommendCache.UserToUser,
		recommendCache.SimilarUserSubset(userId, version),
		recommendRecallProbeSimilarUser,
		version,
		versionPublishedAt,
		limit,
		nil,
	)
}

// listCachedCollaborativeFilteringGoodsIds 读取协同过滤召回探针缓存。
func (c *RecommendRequestCase) listCachedCollaborativeFilteringGoodsIds(ctx context.Context, userId int64, version string, versionPublishedAt time.Time, limit int64, excludeGoodsIds []int64) (*recommendCacheReadResult, error) {
	// 登录用户编号非法时，不需要继续读取协同过滤缓存。
	if userId <= 0 {
		return newRecommendCacheReadResult(
			recommendRecallProbeCollaborativeFiltering,
			recommendCache.CollaborativeFiltering,
			"",
			version,
			versionPublishedAt,
			limit,
			len(excludeGoodsIds),
		), nil
	}
	return c.listCachedInt64Ids(
		ctx,
		recommendCache.CollaborativeFiltering,
		recommendCache.CollaborativeFilteringSubset(userId, version),
		recommendRecallProbeCollaborativeFiltering,
		version,
		versionPublishedAt,
		limit,
		excludeGoodsIds,
	)
}

// listCachedContentBasedGoodsIds 读取内容相似召回探针缓存。
func (c *RecommendRequestCase) listCachedContentBasedGoodsIds(ctx context.Context, goodsId int64, version string, versionPublishedAt time.Time, limit int64, excludeGoodsIds []int64) (*recommendCacheReadResult, error) {
	// 商品编号非法时，不需要继续读取内容相似缓存。
	if goodsId <= 0 {
		return newRecommendCacheReadResult(
			recommendRecallProbeContentBased,
			recommendCache.ContentBased,
			"",
			version,
			versionPublishedAt,
			limit,
			len(excludeGoodsIds),
		), nil
	}
	return c.listCachedInt64Ids(
		ctx,
		recommendCache.ContentBased,
		recommendCache.ContentBasedSubset(goodsId, version),
		recommendRecallProbeContentBased,
		version,
		versionPublishedAt,
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
	version, versionPublishedAt, probeConfig, err := c.loadRecommendRecallProbeConfig(ctx, scene)
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
	// 当前场景存在启用版本时，再补充版本发布时间。
	if !versionPublishedAt.IsZero() {
		probeContext["sceneVersionPublishedAt"] = versionPublishedAt.Format(time.RFC3339Nano)
	}
	observedSources := make([]string, 0, 3)
	if probeConfig.IsSimilarUserEnabled() && userId > 0 {
		limit := probeConfig.SimilarUser.ResolveLimit(defaultLimit)
		similarUserResult, listErr := c.listCachedSimilarUserIds(ctx, userId, version, versionPublishedAt, limit)
		if listErr != nil {
			return nil, listErr
		}
		similarUserIds := similarUserResult.Ids
		probeContext["similarUser"] = map[string]any{
			"enabled":          true,
			"joinCandidate":    probeConfig.SimilarUser.ShouldJoinCandidate(),
			"limit":            limit,
			"userIds":          similarUserIds,
			"cacheReadContext": similarUserResult.ReadContext,
		}
		// 读取到了有效相似用户时，记录探针命中来源。
		if len(similarUserIds) > 0 {
			observedSources = append(observedSources, recommendRecallProbeSimilarUser)
		}
	}
	if probeConfig.IsCollaborativeFilteringEnabled() && userId > 0 {
		limit := probeConfig.CollaborativeFiltering.ResolveLimit(defaultLimit)
		collaborativeFilteringResult, listErr := c.listCachedCollaborativeFilteringGoodsIds(ctx, userId, version, versionPublishedAt, limit, excludeGoodsIds)
		if listErr != nil {
			return nil, listErr
		}
		goodsIds := collaborativeFilteringResult.Ids
		probeContext["collaborativeFiltering"] = map[string]any{
			"enabled":          true,
			"joinCandidate":    probeConfig.CollaborativeFiltering.ShouldJoinCandidate(),
			"limit":            limit,
			"goodsIds":         goodsIds,
			"cacheReadContext": collaborativeFilteringResult.ReadContext,
		}
		// 读取到了有效协同过滤商品时，记录探针命中来源。
		if len(goodsIds) > 0 {
			observedSources = append(observedSources, recommendRecallProbeCollaborativeFiltering)
		}
	}
	if probeConfig.IsContentBasedEnabled() && goodsId > 0 {
		limit := probeConfig.ContentBased.ResolveLimit(defaultLimit)
		contentBasedResult, listErr := c.listCachedContentBasedGoodsIds(ctx, goodsId, version, versionPublishedAt, limit, excludeGoodsIds)
		if listErr != nil {
			return nil, listErr
		}
		goodsIds := contentBasedResult.Ids
		probeContext["contentBased"] = map[string]any{
			"enabled":          true,
			"joinCandidate":    probeConfig.ContentBased.ShouldJoinCandidate(),
			"limit":            limit,
			"goodsIds":         goodsIds,
			"cacheReadContext": contentBasedResult.ReadContext,
		}
		// 读取到了有效内容相似商品时，记录探针命中来源。
		if len(goodsIds) > 0 {
			observedSources = append(observedSources, recommendRecallProbeContentBased)
		}
	}
	probeContext["observedSources"] = recommendcore.DedupeStrings(observedSources)
	return probeContext, nil
}

var (
	appendRecommendRecallProbeContext            = recommendOnlineRecall.AppendProbeContext
	appendRecommendRecallJoinContext             = recommendOnlineRecall.AppendJoinContext
	appendRecommendSimilarUserObservationContext = recommendOnlineRecall.AppendSimilarUserObservationContext
)

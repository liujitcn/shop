package task

import (
	"context"
	"fmt"

	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	recommendCache "shop/pkg/recommend/cache"
	"shop/pkg/recommend/offline/materialize"

	"github.com/go-kratos/kratos/v2/log"
)

// RecommendSimilarUserMaterialize 相似用户写缓存任务。
type RecommendSimilarUserMaterialize struct {
	recommendUserPreferenceRepo      *data.RecommendUserPreferenceRepo
	recommendUserGoodsPreferenceRepo *data.RecommendUserGoodsPreferenceRepo
	recommendModelVersionRepo        *data.RecommendModelVersionRepo
	store                            recommendCache.Store
	materializer                     *materialize.Materializer
	ctx                              context.Context
}

// NewRecommendSimilarUserMaterialize 创建相似用户写缓存任务实例。
func NewRecommendSimilarUserMaterialize(
	recommendUserPreferenceRepo *data.RecommendUserPreferenceRepo,
	recommendUserGoodsPreferenceRepo *data.RecommendUserGoodsPreferenceRepo,
	recommendModelVersionRepo *data.RecommendModelVersionRepo,
	store recommendCache.Store,
	materializer *materialize.Materializer,
) *RecommendSimilarUserMaterialize {
	return &RecommendSimilarUserMaterialize{
		recommendUserPreferenceRepo:      recommendUserPreferenceRepo,
		recommendUserGoodsPreferenceRepo: recommendUserGoodsPreferenceRepo,
		recommendModelVersionRepo:        recommendModelVersionRepo,
		store:                            store,
		materializer:                     materializer,
		ctx:                              context.Background(),
	}
}

// Exec 执行相似用户写缓存任务。
func (t *RecommendSimilarUserMaterialize) Exec(args map[string]string) ([]string, error) {
	log.Infof("Job RecommendSimilarUserMaterialize Exec %+v", args)

	limit, err := parseRecommendMaterializeLimitArg(args["limit"])
	if err != nil {
		return []string{err.Error()}, err
	}
	stats := newRecommendMaterializeStats("RecommendSimilarUserMaterialize", limit)

	stats.SetStage("load_enabled_versions")
	var versionList []string
	versionList, err = loadEnabledRecommendVersions(t.ctx, t.recommendModelVersionRepo)
	if err != nil {
		return returnRecommendMaterializeFailure(stats, err)
	}
	// 当前没有启用版本时，不需要继续发布相似用户缓存。
	if len(versionList) == 0 {
		stats.SetStage("skip_no_enabled_versions")
		result := []string{"no enabled recommend version", stats.BuildSummary()}
		stats.LogSummary()
		return result, nil
	}

	stats.SetStage("load_category_preference")
	var categoryPreferenceList []*models.RecommendUserPreference
	categoryPreferenceList, err = loadRecommendCategoryPreferenceList(t.ctx, t.recommendUserPreferenceRepo)
	if err != nil {
		return returnRecommendMaterializeFailure(stats, err)
	}
	stats.AddInputCount("category_preference_count", len(categoryPreferenceList))
	stats.SetStage("load_goods_preference")
	var userGoodsPreferenceList []*models.RecommendUserGoodsPreference
	userGoodsPreferenceList, err = loadRecommendUserGoodsPreferenceList(t.ctx, t.recommendUserGoodsPreferenceRepo)
	if err != nil {
		return returnRecommendMaterializeFailure(stats, err)
	}
	stats.AddInputCount("goods_preference_count", len(userGoodsPreferenceList))

	stats.SetStage("build_similar_user_map")
	scoreMap := buildSimilarUserScoreMap(categoryPreferenceList, userGoodsPreferenceList)
	stats.AddInputCount("neighbor_user_count", len(scoreMap))
	documentMap := buildSimilarUserMaterializeMap(scoreMap, limit)
	stats.AddInputCount("published_user_count", len(documentMap))

	result := make([]string, 0, len(versionList))
	for _, version := range versionList {
		stats.AddVersion(version)
		currentSubsetMap := make(map[string]struct{}, len(documentMap))
		for userId, documentList := range documentMap {
			subset := recommendCache.SimilarUserSubset(userId, version)
			currentSubsetMap[subset] = struct{}{}
			stats.SetStage(fmt.Sprintf("publish_similar_user_version_%s_user_%d", version, userId))
			err = t.materializer.MaterializeSimilarUsers(t.ctx, userId, version, documentList)
			if err != nil {
				return returnRecommendMaterializeFailure(stats, err)
			}
			stats.AddPublishedSubset(len(documentList))
		}
		stats.SetStage(fmt.Sprintf("clear_similar_user_version_%s", version))
		clearedSubsetCount, clearErr := clearStaleVersionedSubsets(t.ctx, t.store, recommendCache.UserToUser, version, currentSubsetMap)
		if clearErr != nil {
			return returnRecommendMaterializeFailure(stats, clearErr)
		}
		stats.AddClearedSubsets(clearedSubsetCount)
		result = append(result, fmt.Sprintf("version=%s total_users=%d cleared_subsets=%d", version, len(documentMap), clearedSubsetCount))
	}
	result = append(result, stats.BuildSummary())
	stats.LogSummary()
	return result, nil
}

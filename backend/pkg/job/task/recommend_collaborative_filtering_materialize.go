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

// RecommendCollaborativeFilteringMaterialize 协同过滤写缓存任务。
type RecommendCollaborativeFilteringMaterialize struct {
	recommendUserPreferenceRepo      *data.RecommendUserPreferenceRepo
	recommendUserGoodsPreferenceRepo *data.RecommendUserGoodsPreferenceRepo
	recommendModelVersionRepo        *data.RecommendModelVersionRepo
	goodsInfoRepo                    *data.GoodsInfoRepo
	store                            recommendCache.Store
	materializer                     *materialize.Materializer
	ctx                              context.Context
}

// NewRecommendCollaborativeFilteringMaterialize 创建协同过滤写缓存任务实例。
func NewRecommendCollaborativeFilteringMaterialize(
	recommendUserPreferenceRepo *data.RecommendUserPreferenceRepo,
	recommendUserGoodsPreferenceRepo *data.RecommendUserGoodsPreferenceRepo,
	recommendModelVersionRepo *data.RecommendModelVersionRepo,
	goodsInfoRepo *data.GoodsInfoRepo,
	store recommendCache.Store,
	materializer *materialize.Materializer,
) *RecommendCollaborativeFilteringMaterialize {
	return &RecommendCollaborativeFilteringMaterialize{
		recommendUserPreferenceRepo:      recommendUserPreferenceRepo,
		recommendUserGoodsPreferenceRepo: recommendUserGoodsPreferenceRepo,
		recommendModelVersionRepo:        recommendModelVersionRepo,
		goodsInfoRepo:                    goodsInfoRepo,
		store:                            store,
		materializer:                     materializer,
		ctx:                              context.Background(),
	}
}

// Exec 执行协同过滤写缓存任务。
func (t *RecommendCollaborativeFilteringMaterialize) Exec(args map[string]string) ([]string, error) {
	log.Infof("Job RecommendCollaborativeFilteringMaterialize Exec %+v", args)

	limit, err := parseRecommendMaterializeLimitArg(args["limit"])
	if err != nil {
		return []string{err.Error()}, err
	}
	stats := newRecommendMaterializeStats("RecommendCollaborativeFilteringMaterialize", limit)

	stats.SetStage("load_enabled_versions")
	var versionList []string
	versionList, err = loadEnabledRecommendVersions(t.ctx, t.recommendModelVersionRepo)
	if err != nil {
		return returnRecommendMaterializeFailure(stats, err)
	}
	// 当前没有启用版本时，不需要继续发布协同过滤缓存。
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
	similarUserScoreMap := buildSimilarUserScoreMap(categoryPreferenceList, userGoodsPreferenceList)
	similarUserDocumentMap := buildSimilarUserMaterializeMap(similarUserScoreMap, limit)
	stats.AddInputCount("similar_user_count", len(similarUserDocumentMap))
	stats.SetStage("build_collaborative_filtering_map")
	collaborativeFilteringMap := make(map[int64][]recommendCache.Score)
	collaborativeFilteringMap, err = buildCollaborativeFilteringMaterializeMap(
		t.ctx,
		similarUserDocumentMap,
		userGoodsPreferenceList,
		t.goodsInfoRepo,
		limit,
	)
	if err != nil {
		return returnRecommendMaterializeFailure(stats, err)
	}
	stats.AddInputCount("candidate_user_count", len(collaborativeFilteringMap))

	result := make([]string, 0, len(versionList))
	for _, version := range versionList {
		stats.AddVersion(version)
		currentSubsetMap := make(map[string]struct{}, len(collaborativeFilteringMap))
		for userId, documentList := range collaborativeFilteringMap {
			subset := recommendCache.CollaborativeFilteringSubset(userId, version)
			currentSubsetMap[subset] = struct{}{}
			stats.SetStage(fmt.Sprintf("publish_collaborative_filtering_version_%s_user_%d", version, userId))
			err = t.materializer.MaterializeCollaborativeFiltering(t.ctx, userId, version, documentList)
			if err != nil {
				return returnRecommendMaterializeFailure(stats, err)
			}
			stats.AddPublishedSubset(len(documentList))
		}
		stats.SetStage(fmt.Sprintf("clear_collaborative_filtering_version_%s", version))
		clearedSubsetCount, clearErr := clearStaleVersionedSubsets(t.ctx, t.store, recommendCache.CollaborativeFiltering, version, currentSubsetMap)
		if clearErr != nil {
			return returnRecommendMaterializeFailure(stats, clearErr)
		}
		stats.AddClearedSubsets(clearedSubsetCount)
		result = append(result, fmt.Sprintf("version=%s total_users=%d cleared_subsets=%d", version, len(collaborativeFilteringMap), clearedSubsetCount))
	}
	result = append(result, stats.BuildSummary())
	stats.LogSummary()
	return result, nil
}

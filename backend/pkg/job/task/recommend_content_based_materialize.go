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

// RecommendContentBasedMaterialize 内容相似写缓存任务。
type RecommendContentBasedMaterialize struct {
	recommendModelVersionRepo *data.RecommendModelVersionRepo
	goodsInfoRepo             *data.GoodsInfoRepo
	store                     recommendCache.Store
	materializer              *materialize.Materializer
	ctx                       context.Context
}

// NewRecommendContentBasedMaterialize 创建内容相似写缓存任务实例。
func NewRecommendContentBasedMaterialize(
	recommendModelVersionRepo *data.RecommendModelVersionRepo,
	goodsInfoRepo *data.GoodsInfoRepo,
	store recommendCache.Store,
	materializer *materialize.Materializer,
) *RecommendContentBasedMaterialize {
	return &RecommendContentBasedMaterialize{
		recommendModelVersionRepo: recommendModelVersionRepo,
		goodsInfoRepo:             goodsInfoRepo,
		store:                     store,
		materializer:              materializer,
		ctx:                       context.Background(),
	}
}

// Exec 执行内容相似写缓存任务。
func (t *RecommendContentBasedMaterialize) Exec(args map[string]string) ([]string, error) {
	log.Infof("Job RecommendContentBasedMaterialize Exec %+v", args)

	limit, err := parseRecommendMaterializeLimitArg(args["limit"])
	if err != nil {
		return []string{err.Error()}, err
	}
	stats := newRecommendMaterializeStats("RecommendContentBasedMaterialize", limit)

	stats.SetStage("load_enabled_versions")
	var versionList []string
	versionList, err = loadEnabledRecommendVersions(t.ctx, t.recommendModelVersionRepo)
	if err != nil {
		return returnRecommendMaterializeFailure(stats, err)
	}
	// 当前没有启用版本时，不需要继续发布内容相似缓存。
	if len(versionList) == 0 {
		stats.SetStage("skip_no_enabled_versions")
		result := []string{"no enabled recommend version", stats.BuildSummary()}
		stats.LogSummary()
		return result, nil
	}

	stats.SetStage("load_put_on_goods")
	var goodsList []*models.GoodsInfo
	goodsList, err = loadPutOnGoodsList(t.ctx, t.goodsInfoRepo)
	if err != nil {
		return returnRecommendMaterializeFailure(stats, err)
	}
	stats.AddInputCount("put_on_goods_count", len(goodsList))
	stats.AddInputCount("category_count", countRecommendGoodsCategory(goodsList))
	stats.SetStage("build_content_based_map")
	contentBasedMap := buildContentBasedMaterializeMap(goodsList, limit)
	stats.AddInputCount("candidate_goods_count", len(contentBasedMap))

	result := make([]string, 0, len(versionList))
	for _, version := range versionList {
		stats.AddVersion(version)
		currentSubsetMap := make(map[string]struct{}, len(contentBasedMap))
		for goodsId, documentList := range contentBasedMap {
			subset := recommendCache.ContentBasedSubset(goodsId, version)
			currentSubsetMap[subset] = struct{}{}
			stats.SetStage(fmt.Sprintf("publish_content_based_version_%s_goods_%d", version, goodsId))
			err = t.materializer.MaterializeContentBased(t.ctx, goodsId, version, documentList)
			if err != nil {
				return returnRecommendMaterializeFailure(stats, err)
			}
			stats.AddPublishedSubset(len(documentList))
		}
		stats.SetStage(fmt.Sprintf("clear_content_based_version_%s", version))
		clearedSubsetCount, clearErr := clearStaleVersionedSubsets(t.ctx, t.store, recommendCache.ContentBased, version, currentSubsetMap)
		if clearErr != nil {
			return returnRecommendMaterializeFailure(stats, clearErr)
		}
		stats.AddClearedSubsets(clearedSubsetCount)
		result = append(result, fmt.Sprintf("version=%s total_goods=%d cleared_subsets=%d", version, len(contentBasedMap), clearedSubsetCount))
	}
	result = append(result, stats.BuildSummary())
	stats.LogSummary()
	return result, nil
}

package task

import (
	"context"
	"fmt"

	"shop/api/gen/go/common"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	recommendCache "shop/pkg/recommend/cache"
	"shop/pkg/recommend/offline/materialize"

	"github.com/go-kratos/kratos/v2/log"
)

// RecommendSimilarItemMaterialize 相似商品写缓存任务。
type RecommendSimilarItemMaterialize struct {
	recommendGoodsRelationRepo *data.RecommendGoodsRelationRepo
	recommendModelVersionRepo  *data.RecommendModelVersionRepo
	goodsInfoRepo              *data.GoodsInfoRepo
	store                      recommendCache.Store
	materializer               *materialize.Materializer
	ctx                        context.Context
}

// NewRecommendSimilarItemMaterialize 创建相似商品写缓存任务实例。
func NewRecommendSimilarItemMaterialize(
	recommendGoodsRelationRepo *data.RecommendGoodsRelationRepo,
	recommendModelVersionRepo *data.RecommendModelVersionRepo,
	goodsInfoRepo *data.GoodsInfoRepo,
	store recommendCache.Store,
	materializer *materialize.Materializer,
) *RecommendSimilarItemMaterialize {
	return &RecommendSimilarItemMaterialize{
		recommendGoodsRelationRepo: recommendGoodsRelationRepo,
		recommendModelVersionRepo:  recommendModelVersionRepo,
		goodsInfoRepo:              goodsInfoRepo,
		store:                      store,
		materializer:               materializer,
		ctx:                        context.Background(),
	}
}

// Exec 执行相似商品写缓存任务。
func (t *RecommendSimilarItemMaterialize) Exec(args map[string]string) ([]string, error) {
	log.Infof("Job RecommendSimilarItemMaterialize Exec %+v", args)

	limit, err := parseRecommendMaterializeLimitArg(args["limit"])
	if err != nil {
		return []string{err.Error()}, err
	}
	stats := newRecommendMaterializeStats("RecommendSimilarItemMaterialize", limit)

	sceneList := []int32{int32(common.RecommendScene_GOODS_DETAIL)}
	stats.SetStage("load_enabled_version")
	versionMap := make(map[int32]string)
	versionMap, err = loadEnabledRecommendVersionMap(t.ctx, t.recommendModelVersionRepo, sceneList)
	if err != nil {
		return returnRecommendMaterializeFailure(stats, err)
	}
	// 相似商品当前仅服务商品详情场景，因此复用该场景的启用版本。
	version := resolveRecommendSceneVersion(versionMap, int32(common.RecommendScene_GOODS_DETAIL))
	stats.AddVersion(version)

	relationMap := make(map[int64][]*models.RecommendGoodsRelation)
	stats.SetStage("load_similar_item_map")
	relationMap, err = loadSimilarItemMaterializeMap(t.ctx, t.recommendGoodsRelationRepo, t.goodsInfoRepo, limit)
	if err != nil {
		return returnRecommendMaterializeFailure(stats, err)
	}

	currentSubsetMap := make(map[string]struct{}, len(relationMap))
	result := make([]string, 0, len(relationMap)+1)
	for goodsId, list := range relationMap {
		subset := recommendCache.SimilarItemSubset(goodsId, version)
		currentSubsetMap[subset] = struct{}{}
		stats.SetStage(fmt.Sprintf("publish_similar_item_goods_%d", goodsId))
		err = t.materializer.MaterializeSimilarItems(t.ctx, goodsId, version, list)
		if err != nil {
			return returnRecommendMaterializeFailure(stats, err)
		}
		stats.AddPublishedSubset(len(list))
		result = append(result, fmt.Sprintf("goodsId=%d version=%s count=%d", goodsId, version, len(list)))
	}
	stats.SetStage("clear_stale_similar_item_subsets")
	clearedSubsetCount, clearErr := clearStaleSimilarItemSubsets(t.ctx, t.store, version, currentSubsetMap)
	if clearErr != nil {
		return returnRecommendMaterializeFailure(stats, clearErr)
	}
	stats.AddClearedSubsets(clearedSubsetCount)
	result = append(result, fmt.Sprintf("version=%s total_goods=%d cleared_subsets=%d", version, len(relationMap), clearedSubsetCount))
	result = append(result, stats.BuildSummary())
	stats.LogSummary()
	return result, nil
}

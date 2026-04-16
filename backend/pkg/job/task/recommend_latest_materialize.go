package task

import (
	"context"
	"fmt"

	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"shop/pkg/recommend/offline/materialize"

	"github.com/go-kratos/kratos/v2/log"
)

// RecommendLatestMaterialize 推荐最新榜写缓存任务。
type RecommendLatestMaterialize struct {
	recommendModelVersionRepo *data.RecommendModelVersionRepo
	goodsInfoRepo             *data.GoodsInfoRepo
	materializer              *materialize.Materializer
	ctx                       context.Context
}

// NewRecommendLatestMaterialize 创建推荐最新榜写缓存任务实例。
func NewRecommendLatestMaterialize(
	recommendModelVersionRepo *data.RecommendModelVersionRepo,
	goodsInfoRepo *data.GoodsInfoRepo,
	materializer *materialize.Materializer,
) *RecommendLatestMaterialize {
	return &RecommendLatestMaterialize{
		recommendModelVersionRepo: recommendModelVersionRepo,
		goodsInfoRepo:             goodsInfoRepo,
		materializer:              materializer,
		ctx:                       context.Background(),
	}
}

// Exec 执行推荐最新榜写缓存任务。
func (t *RecommendLatestMaterialize) Exec(args map[string]string) ([]string, error) {
	log.Infof("Job RecommendLatestMaterialize Exec %+v", args)

	limit, err := parseRecommendMaterializeLimitArg(args["limit"])
	if err != nil {
		return []string{err.Error()}, err
	}
	stats := newRecommendMaterializeStats("RecommendLatestMaterialize", limit)

	sceneList := listRecommendMaterializeScenes()
	stats.SetStage("load_enabled_version")
	versionMap := make(map[int32]string)
	versionMap, err = loadEnabledRecommendVersionMap(t.ctx, t.recommendModelVersionRepo, sceneList)
	if err != nil {
		return returnRecommendMaterializeFailure(stats, err)
	}

	goodsList := make([]*models.GoodsInfo, 0)
	stats.SetStage("load_latest_goods_list")
	goodsList, err = loadLatestGoodsMaterializeList(t.ctx, t.goodsInfoRepo, limit)
	if err != nil {
		return returnRecommendMaterializeFailure(stats, err)
	}

	result := make([]string, 0, len(sceneList))
	for _, scene := range sceneList {
		version := resolveRecommendSceneVersion(versionMap, scene)
		stats.AddVersion(version)
		stats.SetStage(fmt.Sprintf("publish_scene_latest_scene_%d", scene))
		err = t.materializer.MaterializeSceneLatest(t.ctx, scene, version, goodsList)
		if err != nil {
			return returnRecommendMaterializeFailure(stats, err)
		}
		stats.AddPublishedSubset(len(goodsList))
		result = append(result, fmt.Sprintf("scene=%d version=%s count=%d", scene, version, len(goodsList)))
	}
	result = append(result, stats.BuildSummary())
	stats.LogSummary()
	return result, nil
}

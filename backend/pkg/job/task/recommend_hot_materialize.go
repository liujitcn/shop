package task

import (
	"context"
	"fmt"
	"time"

	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	recommendCandidate "shop/pkg/recommend/candidate"
	"shop/pkg/recommend/offline/materialize"

	"github.com/go-kratos/kratos/v2/log"
)

// RecommendHotMaterialize 推荐热门榜写缓存任务。
type RecommendHotMaterialize struct {
	recommendGoodsStatDayRepo *data.RecommendGoodsStatDayRepo
	recommendModelVersionRepo *data.RecommendModelVersionRepo
	goodsInfoRepo             *data.GoodsInfoRepo
	materializer              *materialize.Materializer
	ctx                       context.Context
}

// NewRecommendHotMaterialize 创建推荐热门榜写缓存任务实例。
func NewRecommendHotMaterialize(
	recommendGoodsStatDayRepo *data.RecommendGoodsStatDayRepo,
	recommendModelVersionRepo *data.RecommendModelVersionRepo,
	goodsInfoRepo *data.GoodsInfoRepo,
	materializer *materialize.Materializer,
) *RecommendHotMaterialize {
	return &RecommendHotMaterialize{
		recommendGoodsStatDayRepo: recommendGoodsStatDayRepo,
		recommendModelVersionRepo: recommendModelVersionRepo,
		goodsInfoRepo:             goodsInfoRepo,
		materializer:              materializer,
		ctx:                       context.Background(),
	}
}

// Exec 执行推荐热门榜写缓存任务。
func (t *RecommendHotMaterialize) Exec(args map[string]string) ([]string, error) {
	log.Infof("Job RecommendHotMaterialize Exec %+v", args)

	limit, err := parseRecommendMaterializeLimitArg(args["limit"])
	if err != nil {
		return []string{err.Error()}, err
	}
	stats := newRecommendMaterializeStats("RecommendHotMaterialize", limit)

	sceneList := listRecommendMaterializeScenes()
	stats.SetStage("load_enabled_version")
	versionMap := make(map[int32]string)
	versionMap, err = loadEnabledRecommendVersionMap(t.ctx, t.recommendModelVersionRepo, sceneList)
	if err != nil {
		return returnRecommendMaterializeFailure(stats, err)
	}

	startDate := time.Now().AddDate(0, 0, -recommendCandidate.StatLookbackDays)
	result := make([]string, 0, len(sceneList))
	for _, scene := range sceneList {
		version := resolveRecommendSceneVersion(versionMap, scene)
		stats.AddVersion(version)
		stats.SetStage(fmt.Sprintf("load_scene_hot_list_scene_%d", scene))
		list := make([]*models.RecommendGoodsStatDay, 0)
		list, err = loadSceneHotMaterializeList(t.ctx, t.recommendGoodsStatDayRepo, t.goodsInfoRepo, scene, startDate, limit)
		if err != nil {
			return returnRecommendMaterializeFailure(stats, err)
		}
		stats.SetStage(fmt.Sprintf("publish_scene_hot_scene_%d", scene))
		err = t.materializer.MaterializeSceneHot(t.ctx, scene, version, list)
		if err != nil {
			return returnRecommendMaterializeFailure(stats, err)
		}
		stats.AddPublishedSubset(len(list))
		result = append(result, fmt.Sprintf("scene=%d version=%s count=%d", scene, version, len(list)))
	}
	result = append(result, stats.BuildSummary())
	stats.LogSummary()
	return result, nil
}

package task

import (
	"context"
	"fmt"
	"strings"
	"time"

	"shop/api/gen/go/common"
	"shop/pkg/gen/data"
	recommendCache "shop/pkg/recommend/cache"
	"shop/pkg/recommend/offline/materialize"
	recommendCf "shop/pkg/recommend/offline/train/cf"
	recommendTune "shop/pkg/recommend/offline/train/tune"

	"github.com/go-kratos/kratos/v2/log"
)

// RecommendCollaborativeFilteringMaterialize 协同过滤写缓存任务。
type RecommendCollaborativeFilteringMaterialize struct {
	recommendGoodsActionRepo  *data.RecommendGoodsActionRepo
	recommendModelVersionRepo *data.RecommendModelVersionRepo
	goodsInfoRepo             *data.GoodsInfoRepo
	store                     recommendCache.Store
	materializer              *materialize.Materializer
	ctx                       context.Context
}

// NewRecommendCollaborativeFilteringMaterialize 创建协同过滤写缓存任务实例。
func NewRecommendCollaborativeFilteringMaterialize(
	recommendGoodsActionRepo *data.RecommendGoodsActionRepo,
	recommendModelVersionRepo *data.RecommendModelVersionRepo,
	goodsInfoRepo *data.GoodsInfoRepo,
	store recommendCache.Store,
	materializer *materialize.Materializer,
) *RecommendCollaborativeFilteringMaterialize {
	return &RecommendCollaborativeFilteringMaterialize{
		recommendGoodsActionRepo:  recommendGoodsActionRepo,
		recommendModelVersionRepo: recommendModelVersionRepo,
		goodsInfoRepo:             goodsInfoRepo,
		store:                     store,
		materializer:              materializer,
		ctx:                       context.Background(),
	}
}

// Exec 执行协同过滤写缓存任务。
func (t *RecommendCollaborativeFilteringMaterialize) Exec(args map[string]string) ([]string, error) {
	log.Infof("Job RecommendCollaborativeFilteringMaterialize Exec %+v", args)

	limit, err := parseRecommendMaterializeLimitArg(args["limit"])
	if err != nil {
		return []string{err.Error()}, err
	}
	lookbackDays, err := parseRecommendTrainLookbackDaysArg(args["lookbackDays"], 90)
	if err != nil {
		return []string{err.Error()}, err
	}
	epochCount, err := parseRecommendTrainEpochArg(args["epochs"], 120)
	if err != nil {
		return []string{err.Error()}, err
	}
	testRatio, err := ParseRecommendTrainTestRatioArg(args["testRatio"], 0.2)
	if err != nil {
		return []string{err.Error()}, err
	}
	trialCount, err := ParseRecommendTrainTrialCountArg(args["trialCount"], 1)
	if err != nil {
		return []string{err.Error()}, err
	}
	backend, err := parseRecommendTrainBackendArg(args["backend"], recommendCf.BackendGoMLX)
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

	stats.SetStage("load_goods_action")
	startAt := time.Now().AddDate(0, 0, -lookbackDays)
	actionList, err := loadRecommendGoodsActionSince(t.ctx, t.recommendGoodsActionRepo, startAt)
	if err != nil {
		return returnRecommendMaterializeFailure(stats, err)
	}
	stats.AddInputCount("goods_action_count", len(actionList))
	stats.SetStage("build_bpr_interactions")
	interactionList := buildRecommendCollaborativeInteractions(actionList)
	stats.AddInputCount("interaction_count", len(interactionList))
	stats.SetStage("load_put_on_goods")
	putOnGoodsMap, err := loadRecommendPutOnGoodsMap(t.ctx, t.goodsInfoRepo)
	if err != nil {
		return returnRecommendMaterializeFailure(stats, err)
	}
	stats.AddInputCount("put_on_goods_count", len(putOnGoodsMap))
	// 训练样本或可推荐商品不足时，不继续执行真实协同过滤训练。
	if len(interactionList) == 0 || len(putOnGoodsMap) == 0 {
		stats.SetStage("skip_no_training_data")
		result := []string{"no collaborative filtering training data", stats.BuildSummary()}
		stats.LogSummary()
		return result, nil
	}

	stats.SetStage("build_bpr_dataset")
	dataset := recommendCf.BuildDataset(interactionList)
	trainSet, testSet := dataset.Split(testRatio, int64(lookbackDays*1000+epochCount)+int64(limit))
	stats.AddInputCount("dataset_user_count", dataset.CountUsers())
	stats.AddInputCount("dataset_item_count", dataset.CountItems())
	stats.AddInputCount("train_count", trainSet.Count())
	stats.AddInputCount("test_count", testSet.Count())

	baseConfig := recommendCf.Config{
		Backend:   backend,
		BatchSize: 512,
		Factors:   32,
		Epochs:    epochCount,
		Learning:  0.03,
		Reg:       0.01,
		Optimizer: "sgd",
		Seed:      int64(common.RecommendScene_HOME),
	}
	targetMetric := strings.TrimSpace(strings.ToLower(args["targetMetric"]))
	// 没有显式指定调参指标时，默认按 NDCG 选最优参数。
	if targetMetric == "" {
		targetMetric = "ndcg"
	}
	stats.SetStage("tune_bpr_model")
	tuneResult, err := recommendTune.TuneBPR(t.ctx, trainSet, testSet, baseConfig, recommendTune.BPROptions{
		TrialCount:   trialCount,
		TargetMetric: targetMetric,
		TopK:         int(limit),
		Candidates:   max(int(limit)*20, 200),
	})
	if err != nil {
		return returnRecommendMaterializeFailure(stats, err)
	}

	stats.SetStage("fit_bpr_model")
	model := recommendCf.Fit(t.ctx, dataset.Interactions(), tuneResult.Config)
	collaborativeFilteringMap := buildRecommendCollaborativeDocuments(model, putOnGoodsMap, int(limit))
	stats.AddInputCount("candidate_user_count", len(collaborativeFilteringMap))
	modelSnapshot, err := model.ExportSnapshot()
	if err != nil {
		return returnRecommendMaterializeFailure(stats, err)
	}
	stats.SetStage("write_bpr_artifact")
	artifactCreatedAt := time.Now()
	artifactDir, err := writeRecommendTrainArtifacts(
		"collaborative_filtering",
		"",
		artifactCreatedAt,
		&recommendTrainArtifactManifest{
			ModelType:    "bpr",
			Versions:     append([]string{}, versionList...),
			Backend:      backend,
			TargetMetric: targetMetric,
			BestValue:    tuneResult.Value,
			Score: map[string]float64{
				"ndcg":      float64(tuneResult.Score.NDCG),
				"precision": float64(tuneResult.Score.Precision),
				"recall":    float64(tuneResult.Score.Recall),
			},
			Counts: map[string]int{
				"goodsActionCount":   len(actionList),
				"interactionCount":   len(interactionList),
				"datasetUserCount":   dataset.CountUsers(),
				"datasetItemCount":   dataset.CountItems(),
				"trainCount":         trainSet.Count(),
				"testCount":          testSet.Count(),
				"candidateUserCount": len(collaborativeFilteringMap),
			},
		},
		modelSnapshot,
		buildRecommendCollaborativeFilteringArtifactSnapshot(collaborativeFilteringMap),
	)
	if err != nil {
		return returnRecommendMaterializeFailure(stats, err)
	}
	stats.SetStage("write_bpr_tune_summary")
	tuneConfigUpdatedCount, tuneConfigUpdateErr := writeRecommendTuneLatestForEnabledVersions(
		t.ctx,
		t.recommendModelVersionRepo,
		versionList,
		buildRecommendTuneLatestSummary(
			"collaborative_filtering",
			"bpr",
			backend,
			artifactDir,
			artifactCreatedAt,
			"",
			versionList,
			tuneResult.Value,
			map[string]float64{
				"ndcg":      float64(tuneResult.Score.NDCG),
				"precision": float64(tuneResult.Score.Precision),
				"recall":    float64(tuneResult.Score.Recall),
			},
		),
	)
	// 训练摘要回写失败不阻断缓存发布，只记录告警方便后续排查。
	if tuneConfigUpdateErr != nil {
		log.Errorf("writeCollaborativeFilteringTuneLatest %v", tuneConfigUpdateErr)
	}

	result := make([]string, 0, len(versionList)+4)
	result = append(result, fmt.Sprintf(
		"backend=%s target_metric=%s best_value=%.6f factors=%d epochs=%d batch_size=%d learning=%.6f reg=%.6f trial_count=%d",
		backend,
		targetMetric,
		tuneResult.Value,
		tuneResult.Config.Factors,
		tuneResult.Config.Epochs,
		tuneResult.Config.BatchSize,
		tuneResult.Config.Learning,
		tuneResult.Config.Reg,
		tuneResult.TrialCount,
	))
	result = append(result, fmt.Sprintf(
		"validation ndcg=%.6f precision=%.6f recall=%.6f",
		tuneResult.Score.NDCG,
		tuneResult.Score.Precision,
		tuneResult.Score.Recall,
	))
	result = append(result, fmt.Sprintf("artifact_dir=%s", artifactDir))
	if tuneConfigUpdateErr != nil {
		result = append(result, fmt.Sprintf("tune_config_update_failed=%v", tuneConfigUpdateErr))
	} else {
		result = append(result, fmt.Sprintf("tune_config_rows_updated=%d", tuneConfigUpdatedCount))
	}
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
		result = append(result, fmt.Sprintf("version=%s backend=%s total_users=%d cleared_subsets=%d", version, backend, len(collaborativeFilteringMap), clearedSubsetCount))
	}
	result = append(result, stats.BuildSummary())
	stats.LogSummary()
	return result, nil
}

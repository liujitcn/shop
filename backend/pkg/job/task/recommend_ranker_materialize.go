package task

import (
	"context"
	"fmt"
	"strings"
	"time"

	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	recommendCache "shop/pkg/recommend/cache"
	"shop/pkg/recommend/offline/materialize"
	recommendCtr "shop/pkg/recommend/offline/train/ctr"
	recommendTune "shop/pkg/recommend/offline/train/tune"

	"github.com/go-kratos/kratos/v2/log"
)

// RecommendRankerMaterialize 模型精排分数写缓存任务。
type RecommendRankerMaterialize struct {
	recommendModelVersionRepo *data.RecommendModelVersionRepo
	recommendRequestRepo      *data.RecommendRequestRepo
	recommendRequestItemRepo  *data.RecommendRequestItemRepo
	recommendGoodsActionRepo  *data.RecommendGoodsActionRepo
	goodsInfoRepo             *data.GoodsInfoRepo
	store                     recommendCache.Store
	materializer              *materialize.Materializer
	ctx                       context.Context
}

// NewRecommendRankerMaterialize 创建模型精排分数写缓存任务实例。
func NewRecommendRankerMaterialize(
	recommendModelVersionRepo *data.RecommendModelVersionRepo,
	recommendRequestRepo *data.RecommendRequestRepo,
	recommendRequestItemRepo *data.RecommendRequestItemRepo,
	recommendGoodsActionRepo *data.RecommendGoodsActionRepo,
	goodsInfoRepo *data.GoodsInfoRepo,
	store recommendCache.Store,
	materializer *materialize.Materializer,
) *RecommendRankerMaterialize {
	return &RecommendRankerMaterialize{
		recommendModelVersionRepo: recommendModelVersionRepo,
		recommendRequestRepo:      recommendRequestRepo,
		recommendRequestItemRepo:  recommendRequestItemRepo,
		recommendGoodsActionRepo:  recommendGoodsActionRepo,
		goodsInfoRepo:             goodsInfoRepo,
		store:                     store,
		materializer:              materializer,
		ctx:                       context.Background(),
	}
}

// Exec 执行模型精排分数写缓存任务。
func (t *RecommendRankerMaterialize) Exec(args map[string]string) ([]string, error) {
	log.Infof("Job RecommendRankerMaterialize Exec %+v", args)

	limit, err := parseRecommendMaterializeLimitArg(args["limit"])
	if err != nil {
		return []string{err.Error()}, err
	}
	version, err := parseRecommendMaterializeRequiredVersionArg(args["version"])
	if err != nil {
		return []string{err.Error()}, err
	}
	clearStale, err := parseRecommendMaterializeBoolArg(args["clearStale"], false)
	if err != nil {
		return []string{err.Error()}, err
	}
	stats := newRecommendMaterializeStats("RecommendRankerMaterialize", limit)
	stats.AddVersion(version)

	// 显式传入快照路径时，继续走已有的离线快照发布模式。
	if strings.TrimSpace(args["path"]) != "" {
		return t.execSnapshotPublish(args, version, clearStale, limit, stats)
	}
	return t.execTrainPublish(args, version, clearStale, limit, stats)
}

// execSnapshotPublish 执行基于 JSON 快照的 ranker 发布。
func (t *RecommendRankerMaterialize) execSnapshotPublish(args map[string]string, version string, clearStale bool, limit int64, stats *recommendMaterializeStats) ([]string, error) {
	stats.SetStage("load_ranker_snapshot")
	entryList, err := loadRecommendStageScoreEntryList(args["path"])
	if err != nil {
		return returnRecommendMaterializeFailure(stats, err)
	}
	stats.AddInputCount("entry_count", len(entryList))
	stats.AddInputCount("document_count", countRecommendStageScoreDocuments(entryList))

	publishedAt := time.Now()
	currentSubsetMap := make(map[string]struct{}, len(entryList))
	result := make([]string, 0, len(entryList)+2)
	for _, entry := range entryList {
		// 空快照条目不参与缓存发布。
		if entry == nil {
			continue
		}
		// 阶段分数快照必须带有效场景，避免写出无法读取的缓存键。
		if entry.Scene <= 0 {
			return returnRecommendMaterializeFailure(stats, errorsx.InvalidArgument("ranker 快照条目的 scene 必须大于 0"))
		}

		documentList := normalizeRecommendStageDocuments(entry.Documents, limit, publishedAt)
		subset := recommendCache.RankerSubset(entry.Scene, entry.ActorType, entry.ActorId, version)
		currentSubsetMap[subset] = struct{}{}
		stats.SetStage(fmt.Sprintf("publish_ranker_scene_%d_actor_type_%d_actor_%d", entry.Scene, entry.ActorType, entry.ActorId))
		err = t.materializer.MaterializeRanker(t.ctx, entry.Scene, entry.ActorType, entry.ActorId, version, documentList)
		if err != nil {
			return returnRecommendMaterializeFailure(stats, err)
		}
		stats.AddPublishedSubset(len(documentList))
		result = append(result, fmt.Sprintf(
			"scene=%d actor_type=%d actor_id=%d version=%s count=%d",
			entry.Scene,
			entry.ActorType,
			entry.ActorId,
			version,
			len(documentList),
		))
	}

	// 显式要求全量快照发布时，再清理当前版本下没有出现在快照里的旧子集合。
	if clearStale {
		stats.SetStage(fmt.Sprintf("clear_ranker_version_%s", version))
		clearedSubsetCount, clearErr := clearStaleVersionedSubsets(t.ctx, t.store, recommendCache.Ranker, version, currentSubsetMap)
		if clearErr != nil {
			return returnRecommendMaterializeFailure(stats, clearErr)
		}
		stats.AddClearedSubsets(clearedSubsetCount)
		result = append(result, fmt.Sprintf("version=%s cleared_subsets=%d", version, clearedSubsetCount))
	}

	result = append(result, stats.BuildSummary())
	stats.LogSummary()
	return result, nil
}

// execTrainPublish 执行真实样本训练并发布 ranker 分数。
func (t *RecommendRankerMaterialize) execTrainPublish(args map[string]string, version string, clearStale bool, limit int64, stats *recommendMaterializeStats) ([]string, error) {
	// 真实训练依赖 data 层样本装配能力，依赖缺失时不继续执行。
	if t.recommendRequestRepo == nil || t.recommendRequestItemRepo == nil || t.recommendGoodsActionRepo == nil || t.goodsInfoRepo == nil {
		return returnRecommendMaterializeFailure(stats, errorsx.InvalidArgument("ranker 训练依赖未注入"))
	}

	lookbackDays, err := parseRecommendTrainLookbackDaysArg(args["lookbackDays"], 30)
	if err != nil {
		return []string{err.Error()}, err
	}
	epochCount, err := parseRecommendTrainEpochArg(args["epochs"], 40)
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
	backend, err := parseRecommendTrainBackendArg(args["backend"], recommendCtr.BackendGoMLX)
	if err != nil {
		return []string{err.Error()}, err
	}
	stats.SetStage("load_ranker_training_data")
	startAt := time.Now().AddDate(0, 0, -lookbackDays)
	trainData, err := LoadRecommendRankerTrainData(
		t.ctx,
		t.recommendRequestRepo,
		t.recommendRequestItemRepo,
		t.recommendGoodsActionRepo,
		t.goodsInfoRepo,
		startAt,
	)
	if err != nil {
		return returnRecommendMaterializeFailure(stats, err)
	}
	stats.AddInputCount("request_count", len(trainData.RequestList))
	stats.AddInputCount("request_item_count", len(trainData.RequestItems))
	stats.AddInputCount("goods_action_count", len(trainData.ActionList))
	stats.AddInputCount("sample_count", len(trainData.SampleList))
	stats.AddInputCount("actor_scene_count", len(trainData.ActorStatsMap))
	stats.AddInputCount("put_on_goods_count", len(trainData.PutOnGoodsMap))
	positiveCount := CountRecommendRankerPositiveSamples(trainData.SampleList)
	stats.AddInputCount("positive_sample_count", positiveCount)

	// 训练样本为空、没有正样本或没有负样本时，不继续拟合 AFM 模型。
	if len(trainData.SampleList) == 0 || positiveCount == 0 || positiveCount >= len(trainData.SampleList) {
		stats.SetStage("skip_no_ranker_training_data")
		result := []string{"no ranker training data", stats.BuildSummary()}
		stats.LogSummary()
		return result, nil
	}

	stats.SetStage("build_ranker_dataset")
	dataset := recommendCtr.BuildDataset(trainData.SampleList)
	trainSet, testSet := dataset.Split(testRatio, int64(lookbackDays*1000+epochCount))
	stats.AddInputCount("dataset_user_count", dataset.CountUsers())
	stats.AddInputCount("dataset_item_count", dataset.CountItems())
	stats.AddInputCount("train_count", trainSet.Count())
	stats.AddInputCount("test_count", testSet.Count())

	baseConfig := recommendCtr.Config{
		BatchSize: 512,
		Backend:   backend,
		Factors:   16,
		Epochs:    epochCount,
		Jobs:      1,
		Verbose:   max(1, epochCount/4),
		Patience:  max(5, epochCount/3),
		Learning:  0.001,
		Reg:       0.0002,
		Optimizer: "adam",
		AutoScale: true,
		Seed:      int64(limit) + int64(epochCount) + int64(lookbackDays),
	}
	targetMetric := strings.TrimSpace(strings.ToLower(args["targetMetric"]))
	// 没有显式指定调参指标时，默认按 AUC 选最优参数。
	if targetMetric == "" {
		targetMetric = "auc"
	}
	stats.SetStage("tune_ranker_model")
	tuneResult, err := recommendTune.TuneAFM(t.ctx, trainSet, testSet, baseConfig, recommendTune.AFMOptions{
		TrialCount:   trialCount,
		TargetMetric: targetMetric,
	})
	if err != nil {
		return returnRecommendMaterializeFailure(stats, err)
	}

	stats.SetStage("fit_ranker_model")
	model := recommendCtr.NewAFM(tuneResult.Config)
	model.Fit(t.ctx, dataset, nil)
	documentMap := BuildRecommendRankerDocuments(model, trainData, limit)
	stats.AddInputCount("published_actor_scene_count", len(documentMap))
	modelSnapshot, err := model.ExportSnapshot()
	if err != nil {
		return returnRecommendMaterializeFailure(stats, err)
	}
	stats.SetStage("write_ranker_artifact")
	artifactCreatedAt := time.Now()
	artifactDir, err := writeRecommendTrainArtifacts(
		"ranker",
		version,
		artifactCreatedAt,
		&recommendTrainArtifactManifest{
			ModelType:    "afm",
			Backend:      backend,
			TargetMetric: targetMetric,
			BestValue:    tuneResult.Value,
			Score: map[string]float64{
				"precision": float64(tuneResult.Score.Precision),
				"recall":    float64(tuneResult.Score.Recall),
				"accuracy":  float64(tuneResult.Score.Accuracy),
				"auc":       float64(tuneResult.Score.AUC),
			},
			Counts: map[string]int{
				"requestCount":             len(trainData.RequestList),
				"requestItemCount":         len(trainData.RequestItems),
				"goodsActionCount":         len(trainData.ActionList),
				"sampleCount":              len(trainData.SampleList),
				"actorSceneCount":          len(trainData.ActorStatsMap),
				"putOnGoodsCount":          len(trainData.PutOnGoodsMap),
				"positiveSampleCount":      positiveCount,
				"datasetUserCount":         dataset.CountUsers(),
				"datasetItemCount":         dataset.CountItems(),
				"trainCount":               trainSet.Count(),
				"testCount":                testSet.Count(),
				"publishedActorSceneCount": len(documentMap),
			},
		},
		modelSnapshot,
		buildRecommendRankerArtifactSnapshot(documentMap),
	)
	if err != nil {
		return returnRecommendMaterializeFailure(stats, err)
	}
	stats.SetStage("write_ranker_tune_summary")
	tuneConfigUpdatedCount, tuneConfigUpdateErr := writeRecommendTuneLatestForVersion(
		t.ctx,
		t.recommendModelVersionRepo,
		version,
		buildRecommendTuneLatestSummary(
			"ranker",
			"afm",
			backend,
			artifactDir,
			artifactCreatedAt,
			version,
			nil,
			tuneResult.Value,
			map[string]float64{
				"precision": float64(tuneResult.Score.Precision),
				"recall":    float64(tuneResult.Score.Recall),
				"accuracy":  float64(tuneResult.Score.Accuracy),
				"auc":       float64(tuneResult.Score.AUC),
			},
		),
	)
	// 训练摘要回写失败不阻断缓存发布，只记录告警方便后续排查。
	if tuneConfigUpdateErr != nil {
		log.Errorf("writeRankerTuneLatest %v", tuneConfigUpdateErr)
	}

	publishedAt := time.Now()
	currentSubsetMap := make(map[string]struct{}, len(documentMap))
	result := make([]string, 0, len(documentMap)+4)
	result = append(result, fmt.Sprintf(
		"version=%s backend=%s target_metric=%s best_value=%.6f factors=%d epochs=%d batch_size=%d learning=%.6f reg=%.6f trial_count=%d",
		version,
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
		"validation precision=%.6f recall=%.6f accuracy=%.6f auc=%.6f",
		tuneResult.Score.Precision,
		tuneResult.Score.Recall,
		tuneResult.Score.Accuracy,
		tuneResult.Score.AUC,
	))
	result = append(result, fmt.Sprintf("artifact_dir=%s", artifactDir))
	if tuneConfigUpdateErr != nil {
		result = append(result, fmt.Sprintf("tune_config_update_failed=%v", tuneConfigUpdateErr))
	} else {
		result = append(result, fmt.Sprintf("tune_config_rows_updated=%d", tuneConfigUpdatedCount))
	}
	for actorKey, documentList := range documentMap {
		// 训练结果为空子集合时，不继续写缓存。
		if len(documentList) == 0 {
			continue
		}
		normalizedDocumentList := normalizeRecommendStageDocuments(documentList, limit, publishedAt)
		subset := recommendCache.RankerSubset(actorKey.Scene, actorKey.ActorType, actorKey.ActorId, version)
		currentSubsetMap[subset] = struct{}{}
		stats.SetStage(fmt.Sprintf("publish_ranker_scene_%d_actor_type_%d_actor_%d", actorKey.Scene, actorKey.ActorType, actorKey.ActorId))
		err = t.materializer.MaterializeRanker(t.ctx, actorKey.Scene, actorKey.ActorType, actorKey.ActorId, version, normalizedDocumentList)
		if err != nil {
			return returnRecommendMaterializeFailure(stats, err)
		}
		stats.AddPublishedSubset(len(normalizedDocumentList))
		result = append(result, fmt.Sprintf(
			"scene=%d actor_type=%d actor_id=%d version=%s count=%d",
			actorKey.Scene,
			actorKey.ActorType,
			actorKey.ActorId,
			version,
			len(normalizedDocumentList),
		))
	}

	// 显式要求全量发布时，再清理当前版本下不在新结果里的旧子集合。
	if clearStale {
		stats.SetStage(fmt.Sprintf("clear_ranker_version_%s", version))
		clearedSubsetCount, clearErr := clearStaleVersionedSubsets(t.ctx, t.store, recommendCache.Ranker, version, currentSubsetMap)
		if clearErr != nil {
			return returnRecommendMaterializeFailure(stats, clearErr)
		}
		stats.AddClearedSubsets(clearedSubsetCount)
		result = append(result, fmt.Sprintf("version=%s cleared_subsets=%d", version, clearedSubsetCount))
	}

	result = append(result, stats.BuildSummary())
	stats.LogSummary()
	return result, nil
}

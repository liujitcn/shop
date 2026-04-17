package tune

import (
	"context"
	"strings"

	"shop/pkg/recommend/offline/train/ctr"

	"github.com/c-bata/goptuna"
	"github.com/c-bata/goptuna/tpe"
)

const (
	// targetMetricAUC 表示按 AUC 选择最优 AFM 参数。
	targetMetricAUC = "auc"
	// targetMetricPrecision 表示按精确率选择最优 AFM 参数。
	targetMetricPrecision = "precision"
	// targetMetricRecall 表示按召回率选择最优 AFM 参数。
	targetMetricRecall = "recall"
	// targetMetricAccuracy 表示按准确率选择最优 AFM 参数。
	targetMetricAccuracy = "accuracy"
)

// AFMOptions 表示 AFM 自动调参配置。
type AFMOptions struct {
	TrialCount   int    // 调参试验次数。
	TargetMetric string // 调参目标指标。
}

// AFMResult 表示 AFM 调参输出结果。
type AFMResult struct {
	Config     ctr.Config // 最优训练参数。
	Score      ctr.Score  // 最优参数在验证集上的指标。
	Value      float64    // 最优目标值。
	TrialCount int        // 实际执行的试验次数。
}

// TuneAFM 对 AFM 训练参数做自动调参。
func TuneAFM(ctx context.Context, trainSet *ctr.Dataset, testSet *ctr.Dataset, baseConfig ctr.Config, options AFMOptions) (*AFMResult, error) {
	normalizedMetric := normalizeAFMTargetMetric(options.TargetMetric)
	// 没有训练集时，直接返回基础参数，避免调参流程空跑。
	if trainSet == nil || trainSet.Count() == 0 {
		return &AFMResult{
			Config:     baseConfig,
			Score:      ctr.Score{},
			Value:      0,
			TrialCount: 0,
		}, nil
	}

	// 没有验证集或只要求单次训练时，直接按基础参数跑一次。
	if testSet == nil || testSet.Count() == 0 || options.TrialCount <= 1 {
		score := fitAFM(ctx, trainSet, testSet, baseConfig)
		return &AFMResult{
			Config:     baseConfig,
			Score:      score,
			Value:      selectAFMMetricValue(score, normalizedMetric),
			TrialCount: 1,
		}, nil
	}

	study, err := goptuna.CreateStudy(
		"recommend_ranker_afm",
		goptuna.StudyOptionDirection(goptuna.StudyDirectionMaximize),
		goptuna.StudyOptionSampler(tpe.NewSampler()),
	)
	if err != nil {
		return nil, err
	}
	if ctx != nil {
		study.WithContext(ctx)
	}

	bestResult := &AFMResult{
		Config: baseConfig,
		Score:  ctr.Score{},
		Value:  -1,
	}
	err = study.Optimize(func(trial goptuna.Trial) (float64, error) {
		// 调参上下文已取消时，尽快终止当前试验。
		if ctx != nil && ctx.Err() != nil {
			return 0, ctx.Err()
		}

		config, configErr := suggestAFMConfig(trial, baseConfig)
		if configErr != nil {
			return 0, configErr
		}
		score := fitAFM(ctx, trainSet, testSet, config)
		value := selectAFMMetricValue(score, normalizedMetric)
		// 当前试验指标更优时，更新最优参数快照。
		if value >= bestResult.Value {
			bestResult = &AFMResult{
				Config: config,
				Score:  score,
				Value:  value,
			}
		}
		return value, nil
	}, options.TrialCount)
	if err != nil {
		return nil, err
	}
	bestResult.TrialCount = options.TrialCount
	return bestResult, nil
}

// normalizeAFMTargetMetric 归一化 AFM 调参目标指标。
func normalizeAFMTargetMetric(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case targetMetricPrecision:
		return targetMetricPrecision
	case targetMetricRecall:
		return targetMetricRecall
	case targetMetricAccuracy:
		return targetMetricAccuracy
	default:
		return targetMetricAUC
	}
}

// suggestAFMConfig 为当前试验生成 AFM 参数。
func suggestAFMConfig(trial goptuna.Trial, baseConfig ctr.Config) (ctr.Config, error) {
	config := baseConfig
	factors, err := trial.SuggestInt("factors", 8, 64)
	if err != nil {
		return ctr.Config{}, err
	}
	epochs, err := trial.SuggestInt("epochs", 20, 120)
	if err != nil {
		return ctr.Config{}, err
	}
	batchSize, err := trial.SuggestInt("batch_size", 128, 2048)
	if err != nil {
		return ctr.Config{}, err
	}
	learning, err := trial.SuggestLogFloat("learning", 0.0001, 0.05)
	if err != nil {
		return ctr.Config{}, err
	}
	reg, err := trial.SuggestLogFloat("reg", 0.00001, 0.05)
	if err != nil {
		return ctr.Config{}, err
	}

	config.Factors = factors
	config.Epochs = epochs
	config.BatchSize = roundBatchSize(batchSize)
	config.Learning = float32(learning)
	config.Reg = float32(reg)
	return config, nil
}

// roundBatchSize 将试验采样的 batchSize 规整为常见训练批大小。
func roundBatchSize(value int) int {
	switch {
	case value <= 128:
		return 128
	case value <= 256:
		return 256
	case value <= 512:
		return 512
	case value <= 1024:
		return 1024
	default:
		return 2048
	}
}

// fitAFM 使用给定参数训练并评估一个 AFM 模型。
func fitAFM(ctx context.Context, trainSet *ctr.Dataset, testSet *ctr.Dataset, config ctr.Config) ctr.Score {
	model := ctr.NewAFM(config)
	return model.Fit(ctx, trainSet, testSet)
}

// selectAFMMetricValue 选择当前调参目标对应的指标值。
func selectAFMMetricValue(score ctr.Score, metric string) float64 {
	switch metric {
	case targetMetricPrecision:
		return float64(score.Precision)
	case targetMetricRecall:
		return float64(score.Recall)
	case targetMetricAccuracy:
		return float64(score.Accuracy)
	default:
		return float64(score.AUC)
	}
}

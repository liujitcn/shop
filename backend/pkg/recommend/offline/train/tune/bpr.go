package tune

import (
	"context"
	"strings"

	recommendCf "shop/pkg/recommend/offline/train/cf"

	"github.com/c-bata/goptuna"
	"github.com/c-bata/goptuna/tpe"
)

const (
	// targetMetricNDCG 表示按 NDCG 选择最优 BPR 参数。
	targetMetricNDCG = "ndcg"
)

// BPROptions 表示 BPR 自动调参配置。
type BPROptions struct {
	TrialCount   int
	TargetMetric string
	TopK         int
	Candidates   int
}

// BPRResult 表示 BPR 调参输出结果。
type BPRResult struct {
	Config     recommendCf.Config
	Score      recommendCf.Score
	Value      float64
	TrialCount int
}

// TuneBPR 对 BPR 训练参数做自动调参。
func TuneBPR(ctx context.Context, trainSet *recommendCf.Dataset, testSet *recommendCf.Dataset, baseConfig recommendCf.Config, options BPROptions) (*BPRResult, error) {
	normalizedMetric := normalizeBPRTargetMetric(options.TargetMetric)
	// 没有训练集时，直接返回基础参数，避免调参流程空跑。
	if trainSet == nil || trainSet.Count() == 0 {
		return &BPRResult{
			Config:     baseConfig,
			Score:      recommendCf.Score{},
			Value:      0,
			TrialCount: 0,
		}, nil
	}

	// 没有验证集或只要求单次训练时，直接按基础参数跑一次。
	if testSet == nil || testSet.Count() == 0 || options.TrialCount <= 1 {
		score := fitBPR(ctx, trainSet, testSet, baseConfig, options)
		return &BPRResult{
			Config:     baseConfig,
			Score:      score,
			Value:      selectBPRMetricValue(score, normalizedMetric),
			TrialCount: 1,
		}, nil
	}

	study, err := goptuna.CreateStudy(
		"recommend_cf_bpr",
		goptuna.StudyOptionDirection(goptuna.StudyDirectionMaximize),
		goptuna.StudyOptionSampler(tpe.NewSampler()),
	)
	if err != nil {
		return nil, err
	}
	// 存在上层上下文时，把取消信号透传给调参器。
	if ctx != nil {
		study.WithContext(ctx)
	}

	bestResult := &BPRResult{
		Config: baseConfig,
		Score:  recommendCf.Score{},
		Value:  -1,
	}
	err = study.Optimize(func(trial goptuna.Trial) (float64, error) {
		// 调参上下文已取消时，尽快终止当前试验。
		if ctx != nil && ctx.Err() != nil {
			return 0, ctx.Err()
		}

		config, configErr := suggestBPRConfig(trial, baseConfig)
		if configErr != nil {
			return 0, configErr
		}
		score := fitBPR(ctx, trainSet, testSet, config, options)
		value := selectBPRMetricValue(score, normalizedMetric)
		// 当前试验指标更优时，更新最优参数快照。
		if value >= bestResult.Value {
			bestResult = &BPRResult{
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

// normalizeBPRTargetMetric 归一化 BPR 调参目标指标。
func normalizeBPRTargetMetric(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case targetMetricPrecision:
		return targetMetricPrecision
	case targetMetricRecall:
		return targetMetricRecall
	default:
		return targetMetricNDCG
	}
}

// suggestBPRConfig 为当前试验生成 BPR 参数。
func suggestBPRConfig(trial goptuna.Trial, baseConfig recommendCf.Config) (recommendCf.Config, error) {
	config := baseConfig
	factors, err := trial.SuggestInt("factors", 8, 64)
	if err != nil {
		return recommendCf.Config{}, err
	}
	epochs, err := trial.SuggestInt("epochs", 20, 120)
	if err != nil {
		return recommendCf.Config{}, err
	}
	batchSize, err := trial.SuggestInt("batch_size", 128, 2048)
	if err != nil {
		return recommendCf.Config{}, err
	}
	learning, err := trial.SuggestLogFloat("learning", 0.001, 0.1)
	if err != nil {
		return recommendCf.Config{}, err
	}
	reg, err := trial.SuggestLogFloat("reg", 0.0001, 0.1)
	if err != nil {
		return recommendCf.Config{}, err
	}
	initStdDev, err := trial.SuggestLogFloat("init_stddev", 0.0005, 0.05)
	if err != nil {
		return recommendCf.Config{}, err
	}

	config.Factors = factors
	config.Epochs = epochs
	config.BatchSize = roundBatchSize(batchSize)
	config.Learning = float32(learning)
	config.Reg = float32(reg)
	config.InitStdDev = float32(initStdDev)
	return config, nil
}

// fitBPR 使用给定参数训练并评估一个 BPR 模型。
func fitBPR(ctx context.Context, trainSet *recommendCf.Dataset, testSet *recommendCf.Dataset, config recommendCf.Config, options BPROptions) recommendCf.Score {
	model := recommendCf.Fit(ctx, trainSet.Interactions(), config)
	evaluator := testSet
	// 没有显式验证集时，回退到训练集做最小可用评估。
	if evaluator == nil || evaluator.Count() == 0 {
		evaluator = trainSet
	}
	return recommendCf.Evaluate(model, trainSet, evaluator, recommendCf.EvaluateConfig{
		TopK:       options.TopK,
		Candidates: options.Candidates,
		Seed:       config.Seed,
	})
}

// selectBPRMetricValue 选择当前调参目标对应的指标值。
func selectBPRMetricValue(score recommendCf.Score, metric string) float64 {
	switch metric {
	case targetMetricPrecision:
		return float64(score.Precision)
	case targetMetricRecall:
		return float64(score.Recall)
	default:
		return float64(score.NDCG)
	}
}

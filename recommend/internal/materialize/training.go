package materialize

import (
	"context"
	"errors"
	"math"
	recommendv1 "recommend/api/gen/go/recommend/v1"
	"recommend/contract"
	cachex "recommend/internal/cache"
	cacheleveldb "recommend/internal/cache/leveldb"
	"recommend/internal/core"
	"recommend/internal/rank"
	"time"

	goleveldb "github.com/syndtr/goleveldb/leveldb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type trainingExample struct {
	features map[string]float64
	label    float64
	weight   float64
}

// TrainRanking 训练学习排序模型并写入运行态缓存。
func TrainRanking(ctx context.Context, dependencies core.Dependencies, config core.ServiceConfig, request core.TrainRankingRequest) (*core.BuildResult, error) {
	err := validateTrainingDependencies(dependencies)
	if err != nil {
		return nil, err
	}

	manager, err := cacheleveldb.OpenManager(ctx, dependencies.Cache)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = manager.Close()
	}()

	traceStore := &cachex.TraceStore{Driver: manager}
	runtimeStore := &cachex.RuntimeStore{Driver: manager}
	scenes := normalizeScenes(request.Scenes, defaultScenes)
	startAt, endAt := resolveTrainingWindow(request.StatDate)
	updatedAt := time.Now()
	keyCount := int64(0)

	for _, scene := range scenes {
		examples, err := loadTrainingExamples(ctx, dependencies, traceStore, scene, startAt, endAt)
		if err != nil {
			return nil, err
		}
		// 单个场景样本不足时，不强行产出高噪声模型，继续使用规则排序。
		if len(examples) < normalizeTrainingMinSamples(config.Training.MinSampleCount) {
			continue
		}

		state := trainSceneRankingModel(scene, config, examples, updatedAt)
		err = runtimeStore.SaveRankingModelState(string(scene), rank.ResolveRankingModelName(config), state)
		if err != nil {
			return nil, err
		}
		keyCount++
	}

	return buildResult("training", keyCount, updatedAt), nil
}

// validateTrainingDependencies 校验学习排序训练依赖。
func validateTrainingDependencies(dependencies core.Dependencies) error {
	if dependencies.Recommend == nil {
		return errors.New("recommend: 推荐数据源未配置")
	}
	if dependencies.Cache == nil {
		return errors.New("recommend: 缓存数据源未配置")
	}
	return nil
}

// loadTrainingExamples 加载单个场景的训练样本。
func loadTrainingExamples(
	ctx context.Context,
	dependencies core.Dependencies,
	traceStore *cachex.TraceStore,
	scene core.Scene,
	startAt time.Time,
	endAt time.Time,
) ([]trainingExample, error) {
	requestFacts, err := dependencies.Recommend.ListRequestFacts(ctx, string(scene), startAt, endAt)
	if err != nil {
		return nil, err
	}
	actionFacts, err := dependencies.Recommend.ListActionFacts(ctx, string(scene), startAt, endAt)
	if err != nil {
		return nil, err
	}

	labelMap := buildActionLabelMap(actionFacts)
	examples := make([]trainingExample, 0, len(requestFacts)*5)
	for _, requestFact := range requestFacts {
		if requestFact == nil || requestFact.RequestId == "" {
			continue
		}

		detail, err := traceStore.GetTraceDetailByRequestId(requestFact.RequestId)
		if err != nil {
			if errors.Is(err, goleveldb.ErrNotFound) {
				continue
			}
			return nil, err
		}
		for _, scoreDetail := range detail.GetScoreDetails() {
			if scoreDetail == nil || scoreDetail.GetGoodsId() <= 0 {
				continue
			}
			labelWeight := labelMap[requestFact.RequestId][scoreDetail.GetGoodsId()]
			example := trainingExample{
				features: rank.BuildFeatureMapFromScoreDetail(scoreDetail),
				weight:   resolveTrainingSampleWeight(labelWeight),
			}
			if labelWeight > 0 {
				example.label = 1
			}
			examples = append(examples, example)
		}
	}
	return examples, nil
}

// buildActionLabelMap 构建请求商品维度的监督标签映射。
func buildActionLabelMap(actionFacts []*contract.ActionFact) map[string]map[int64]int32 {
	result := make(map[string]map[int64]int32, len(actionFacts))
	for _, item := range actionFacts {
		if item == nil || item.RequestId == "" || item.GoodsId <= 0 {
			continue
		}
		label := resolveActionLabel(item.EventType)
		if label == 0 {
			continue
		}
		goodsLabelMap, ok := result[item.RequestId]
		if !ok {
			goodsLabelMap = make(map[int64]int32)
			result[item.RequestId] = goodsLabelMap
		}
		if label > goodsLabelMap[item.GoodsId] {
			goodsLabelMap[item.GoodsId] = label
		}
	}
	return result
}

// resolveActionLabel 将行为类型映射为监督标签等级。
func resolveActionLabel(eventType string) int32 {
	switch eventType {
	case string(core.BehaviorClick):
		return 1
	case string(core.BehaviorOrderCreate):
		return 2
	case string(core.BehaviorOrderPay):
		return 3
	default:
		return 0
	}
}

// resolveTrainingSampleWeight 解析单个训练样本的梯度权重。
func resolveTrainingSampleWeight(label int32) float64 {
	switch label {
	case 1:
		return 1
	case 2:
		return 1.5
	case 3:
		return 2
	default:
		return 1
	}
}

// normalizeTrainingMinSamples 归一化训练最小样本数。
func normalizeTrainingMinSamples(minSamples int32) int {
	if minSamples <= 0 {
		return 1
	}
	return int(minSamples)
}

// normalizeTrainingEpochs 归一化训练轮数。
func normalizeTrainingEpochs(epochs int32) int {
	if epochs <= 0 {
		return 1
	}
	return int(epochs)
}

// normalizeTrainingRate 归一化训练学习率。
func normalizeTrainingRate(rate float64) float64 {
	if rate <= 0 {
		return 0.05
	}
	return rate
}

// resolveTrainingWindow 解析训练统计窗口。
func resolveTrainingWindow(statDate time.Time) (time.Time, time.Time) {
	if statDate.IsZero() {
		statDate = time.Now()
	}
	startAt := time.Date(statDate.Year(), statDate.Month(), statDate.Day(), 0, 0, 0, 0, statDate.Location())
	return startAt, startAt.AddDate(0, 0, 1)
}

// trainSceneRankingModel 训练单个场景的学习排序模型。
func trainSceneRankingModel(
	scene core.Scene,
	config core.ServiceConfig,
	examples []trainingExample,
	updatedAt time.Time,
) *recommendv1.RecommendRankingModelState {
	featureNames := rank.RankingFeatureNames()
	linearWeights := make(map[string]float64, len(featureNames))
	interactionWeights := make(map[string]float64)
	learningRate := normalizeTrainingRate(config.Training.LearningRate)
	epochs := normalizeTrainingEpochs(config.Training.Epochs)
	bias := 0.0

	for epochIndex := 0; epochIndex < epochs; epochIndex++ {
		for _, example := range examples {
			rawScore := bias
			for _, featureName := range featureNames {
				rawScore += linearWeights[featureName] * example.features[featureName]
			}
			for leftIndex := 0; leftIndex < len(featureNames); leftIndex++ {
				leftFeature := featureNames[leftIndex]
				leftValue := example.features[leftFeature]
				if leftValue == 0 {
					continue
				}
				for rightIndex := leftIndex + 1; rightIndex < len(featureNames); rightIndex++ {
					rightFeature := featureNames[rightIndex]
					rightValue := example.features[rightFeature]
					if rightValue == 0 {
						continue
					}
					rawScore += interactionWeights[rank.PairFeatureKey(leftFeature, rightFeature)] * leftValue * rightValue
				}
			}

			gradientBase := (sigmoid(rawScore) - example.label) * example.weight
			bias -= learningRate * gradientBase
			for _, featureName := range featureNames {
				value := example.features[featureName]
				if value == 0 {
					continue
				}
				linearWeights[featureName] -= learningRate * gradientBase * value
			}
			for leftIndex := 0; leftIndex < len(featureNames); leftIndex++ {
				leftFeature := featureNames[leftIndex]
				leftValue := example.features[leftFeature]
				if leftValue == 0 {
					continue
				}
				for rightIndex := leftIndex + 1; rightIndex < len(featureNames); rightIndex++ {
					rightFeature := featureNames[rightIndex]
					rightValue := example.features[rightFeature]
					if rightValue == 0 {
						continue
					}
					pairKey := rank.PairFeatureKey(leftFeature, rightFeature)
					interactionWeights[pairKey] -= learningRate * gradientBase * leftValue * rightValue
				}
			}
		}
	}

	return buildRankingModelState(scene, config, updatedAt, examples, bias, linearWeights, interactionWeights)
}

// buildRankingModelState 将训练结果转换为缓存模型状态。
func buildRankingModelState(
	scene core.Scene,
	config core.ServiceConfig,
	updatedAt time.Time,
	examples []trainingExample,
	bias float64,
	linearWeights map[string]float64,
	interactionWeights map[string]float64,
) *recommendv1.RecommendRankingModelState {
	positiveSampleCount := int64(0)
	for _, example := range examples {
		if example.label > 0 {
			positiveSampleCount++
		}
	}

	state := &recommendv1.RecommendRankingModelState{
		Meta:                buildPoolMeta(string(scene), 0, 0, updatedAt),
		ModelName:           rank.ResolveRankingModelName(config),
		RankingMode:         string(core.RankingModeFm),
		SampleCount:         int64(len(examples)),
		PositiveSampleCount: positiveSampleCount,
		Bias:                bias,
		LinearWeights:       make([]*recommendv1.RecommendFeatureWeight, 0, len(linearWeights)),
		InteractionWeights:  make([]*recommendv1.RecommendInteractionWeight, 0, len(interactionWeights)),
		TrainedAt:           timestamppb.New(updatedAt),
	}
	for _, featureName := range rank.RankingFeatureNames() {
		state.LinearWeights = append(state.LinearWeights, &recommendv1.RecommendFeatureWeight{
			Feature: featureName,
			Weight:  linearWeights[featureName],
		})
	}
	for _, pairKey := range rank.SortedInteractionKeys() {
		leftFeature, rightFeature := splitPairKey(pairKey)
		state.InteractionWeights = append(state.InteractionWeights, &recommendv1.RecommendInteractionWeight{
			LeftFeature:  leftFeature,
			RightFeature: rightFeature,
			Weight:       interactionWeights[pairKey],
		})
	}
	return state
}

// splitPairKey 解析交叉特征键。
func splitPairKey(pairKey string) (string, string) {
	for index := 0; index < len(pairKey); index++ {
		if pairKey[index] != '*' {
			continue
		}
		return pairKey[:index], pairKey[index+1:]
	}
	return pairKey, ""
}

// sigmoid 计算逻辑回归激活值。
func sigmoid(value float64) float64 {
	if value >= 0 {
		result := math.Exp(-value)
		return 1 / (1 + result)
	}
	result := math.Exp(value)
	return result / (1 + result)
}

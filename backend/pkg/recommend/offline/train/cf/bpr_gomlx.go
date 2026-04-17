package cf

import (
	"context"
	"fmt"
	"strings"

	gomlxBackends "github.com/gomlx/gomlx/backends"
	"github.com/gomlx/gomlx/backends/simplego"
	gomlxDtypes "github.com/gomlx/gomlx/pkg/core/dtypes"
	gomlxGraph "github.com/gomlx/gomlx/pkg/core/graph"
	gomlxContext "github.com/gomlx/gomlx/pkg/ml/context"
	gomlxDatasets "github.com/gomlx/gomlx/pkg/ml/datasets"
	gomlxTrain "github.com/gomlx/gomlx/pkg/ml/train"
	gomlxLosses "github.com/gomlx/gomlx/pkg/ml/train/losses"
	gomlxOptimizers "github.com/gomlx/gomlx/pkg/ml/train/optimizers"

	"shop/pkg/recommend/offline/train/util"
)

// bprTriple 表示一条 pairwise 排序训练样本。
type bprTriple struct {
	UserIndex     int32
	PositiveIndex int32
	NegativeIndex int32
}

// gomlxBPRRunner 保存 gomlx BPR 训练执行器。
type gomlxBPRRunner struct {
	backend            gomlxBackends.Backend
	ctx                *gomlxContext.Context
	trainer            *gomlxTrain.Trainer
	config             Config
	initialUserFactors [][]float32
	initialItemFactors [][]float32
}

// fitWithGoMLX 使用 gomlx/simplego 训练 BPR。
func fitWithGoMLX(ctx context.Context, model *Model) *Model {
	if model == nil {
		return nil
	}
	feedbackCount := model.feedbackCount()
	// 没有正反馈时，直接返回初始化模型，避免构造空训练集。
	if feedbackCount == 0 {
		return model
	}

	runner, err := newGoMLXBPRRunner(model.config, model.userFactors, model.itemFactors)
	if err != nil {
		panic(err)
	}
	defer runner.Close()

	loop := gomlxTrain.NewLoop(runner.trainer)
	rng := util.NewRandomGenerator(model.config.Seed)
	batchSize := min(model.config.BatchSize, feedbackCount)
	for epoch := 0; epoch < model.config.Epochs; epoch++ {
		// 上层任务取消时，提前结束训练，但仍保留当前已经学到的参数。
		if ctx != nil && ctx.Err() != nil {
			break
		}
		tripleList := sampleBPRTriples(model, feedbackCount, rng)
		// 当前轮次没有有效三元组时，直接结束训练，避免空数据集触发异常。
		if len(tripleList) == 0 {
			break
		}
		trainDataset, err := newGoMLXBPRDataset(runner.backend, tripleList)
		if err != nil {
			panic(err)
		}
		trainDataset.Shuffle().BatchSize(batchSize, false)
		if _, err = loop.RunEpochs(trainDataset, 1); err != nil {
			panic(fmt.Errorf("gomlx bpr train epoch %d failed: %w", epoch+1, err))
		}
	}

	userFactors, itemFactors, err := runner.MaterializeFactors()
	if err != nil {
		panic(err)
	}
	model.userFactors = userFactors
	model.itemFactors = itemFactors
	return model
}

// newGoMLXBPRRunner 创建 gomlx BPR 训练执行器。
func newGoMLXBPRRunner(config Config, userFactors [][]float32, itemFactors [][]float32) (*gomlxBPRRunner, error) {
	// 用户或商品因子为空时，无法构建矩阵分解训练图。
	if len(userFactors) == 0 || len(itemFactors) == 0 {
		return nil, fmt.Errorf("gomlx bpr requires non-empty user and item factors")
	}
	backend, err := simplego.New("ops_parallel")
	if err != nil {
		return nil, err
	}
	ctx := gomlxContext.New()
	// 显式提供种子时，固定随机流，保证训练过程可复现。
	if config.Seed != 0 {
		ctx.SetParam(gomlxContext.ParamInitialSeed, config.Seed)
		if err = ctx.SetRNGStateFromSeed(config.Seed); err != nil {
			return nil, err
		}
	}
	runner := &gomlxBPRRunner{
		backend:            backend,
		ctx:                ctx,
		config:             config,
		initialUserFactors: cloneMatrix32(userFactors),
		initialItemFactors: cloneMatrix32(itemFactors),
	}
	runner.trainer = gomlxTrain.NewTrainer(
		backend,
		ctx,
		runner.buildModel,
		gomlxLosses.BinaryCrossentropyLogits,
		runner.newOptimizer(),
		nil,
		nil,
	)
	return runner, nil
}

// buildModel 构建 gomlx BPR 排序训练图。
func (r *gomlxBPRRunner) buildModel(ctx *gomlxContext.Context, _ any, inputs []*gomlxGraph.Node) []*gomlxGraph.Node {
	// BPR 训练图固定接收用户、正样本、负样本三个索引输入。
	if len(inputs) != 3 {
		panic(fmt.Sprintf("gomlx bpr expects 3 inputs, got %d", len(inputs)))
	}

	userTable := ctx.In("user").VariableWithValue("embeddings", r.initialUserFactors).ValueGraph(inputs[0].Graph())
	itemTable := ctx.In("item").VariableWithValue("embeddings", r.initialItemFactors).ValueGraph(inputs[0].Graph())

	userEmbedding := gomlxGraph.Gather(userTable, gomlxGraph.InsertAxes(gomlxGraph.ConvertDType(inputs[0], gomlxDtypes.Int32), -1))
	positiveEmbedding := gomlxGraph.Gather(itemTable, gomlxGraph.InsertAxes(gomlxGraph.ConvertDType(inputs[1], gomlxDtypes.Int32), -1))
	negativeEmbedding := gomlxGraph.Gather(itemTable, gomlxGraph.InsertAxes(gomlxGraph.ConvertDType(inputs[2], gomlxDtypes.Int32), -1))

	diff := gomlxGraph.ReduceSum(
		gomlxGraph.Mul(userEmbedding, gomlxGraph.Sub(positiveEmbedding, negativeEmbedding)),
		1,
	)
	// 有正则项时，把当前三元组涉及的用户与商品向量 L2 损失并入总目标。
	if r.config.Reg > 0 {
		regLoss := gomlxGraph.ReduceAllMean(
			gomlxGraph.Add(
				gomlxGraph.Add(
					gomlxGraph.ReduceSum(gomlxGraph.Square(userEmbedding), 1),
					gomlxGraph.ReduceSum(gomlxGraph.Square(positiveEmbedding), 1),
				),
				gomlxGraph.ReduceSum(gomlxGraph.Square(negativeEmbedding), 1),
			),
		)
		gomlxTrain.AddLoss(ctx, gomlxGraph.MulScalar(regLoss, float64(r.config.Reg)))
	}
	return []*gomlxGraph.Node{diff}
}

// newOptimizer 根据配置创建 gomlx 优化器。
func (r *gomlxBPRRunner) newOptimizer() gomlxOptimizers.Interface {
	switch strings.TrimSpace(strings.ToLower(r.config.Optimizer)) {
	// SGD 分支关闭默认学习率衰减，尽量贴近当前 pairwise SGD 训练行为。
	case "sgd":
		return gomlxOptimizers.StochasticGradientDescent().
			WithDecay(false).
			WithLearningRate(float64(r.config.Learning)).
			Done()
	default:
		return gomlxOptimizers.Adam().
			LearningRate(float64(r.config.Learning)).
			Done()
	}
}

// MaterializeFactors 把 gomlx 训练后的因子矩阵转回普通二维切片。
func (r *gomlxBPRRunner) MaterializeFactors() ([][]float32, [][]float32, error) {
	if r == nil || r.ctx == nil {
		return nil, nil, fmt.Errorf("gomlx bpr runner is nil")
	}
	userVar := r.ctx.GetVariableByScopeAndName("/user", "embeddings")
	itemVar := r.ctx.GetVariableByScopeAndName("/item", "embeddings")
	// 训练尚未真正开始时，直接回退到初始化因子，避免取消任务后丢失已有状态。
	if userVar == nil || itemVar == nil {
		return cloneMatrix32(r.initialUserFactors), cloneMatrix32(r.initialItemFactors), nil
	}
	return copyMatrix32(userVar.MustValue().Value()), copyMatrix32(itemVar.MustValue().Value()), nil
}

// Close 释放 gomlx BPR 训练后端资源。
func (r *gomlxBPRRunner) Close() {
	if r == nil || r.backend == nil {
		return
	}
	r.backend.Finalize()
}

// sampleBPRTriples 依据当前反馈集合采样一轮 pairwise 训练三元组。
func sampleBPRTriples(model *Model, sampleCount int, rng util.RandomGenerator) []bprTriple {
	if model == nil || sampleCount <= 0 {
		return []bprTriple{}
	}
	result := make([]bprTriple, 0, sampleCount)
	for len(result) < sampleCount {
		userIndex := rng.Intn(len(model.userIds))
		// 没有正反馈的用户不参与当前轮次 pairwise 采样。
		if len(model.userFeedback[userIndex]) == 0 {
			continue
		}
		positiveIndex := model.userFeedback[userIndex][rng.Intn(len(model.userFeedback[userIndex]))]
		negativeIndex := sampleNegativeIndex(rng, len(model.itemIds), model.itemSetByUser[userIndex])
		result = append(result, bprTriple{
			UserIndex:     int32(userIndex),
			PositiveIndex: int32(positiveIndex),
			NegativeIndex: int32(negativeIndex),
		})
	}
	return result
}

// newGoMLXBPRDataset 把 pairwise 训练样本转换为 gomlx 内存数据集。
func newGoMLXBPRDataset(backend gomlxBackends.Backend, tripleList []bprTriple) (*gomlxDatasets.InMemoryDataset, error) {
	userIndexList := make([]int32, len(tripleList))
	positiveIndexList := make([]int32, len(tripleList))
	negativeIndexList := make([]int32, len(tripleList))
	labelList := make([]float32, len(tripleList))
	for index, item := range tripleList {
		userIndexList[index] = item.UserIndex
		positiveIndexList[index] = item.PositiveIndex
		negativeIndexList[index] = item.NegativeIndex
		// BPR 目标固定为“正样本分数高于负样本分数”。
		labelList[index] = 1
	}
	return gomlxDatasets.InMemoryFromData(
		backend,
		"bpr_train",
		[]any{userIndexList, positiveIndexList, negativeIndexList},
		[]any{labelList},
	)
}

// cloneMatrix32 深拷贝二维因子矩阵，避免训练期间回写原始初始化切片。
func cloneMatrix32(value [][]float32) [][]float32 {
	result := make([][]float32, len(value))
	for index := range value {
		result[index] = append([]float32{}, value[index]...)
	}
	return result
}

// copyMatrix32 把 gomlx 返回的矩阵值复制为独立的 float32 二维切片。
func copyMatrix32(value any) [][]float32 {
	switch typedValue := value.(type) {
	case [][]float32:
		return cloneMatrix32(typedValue)
	case [][]float64:
		result := make([][]float32, len(typedValue))
		for rowIndex := range typedValue {
			result[rowIndex] = make([]float32, len(typedValue[rowIndex]))
			for colIndex, item := range typedValue[rowIndex] {
				result[rowIndex][colIndex] = float32(item)
			}
		}
		return result
	default:
		// 未识别的矩阵类型按空矩阵处理，避免把未知值继续传给推荐链路。
		return [][]float32{}
	}
}

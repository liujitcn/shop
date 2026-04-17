package ctr

import (
	"context"
	"fmt"
	"strings"

	gomlxBackends "github.com/gomlx/gomlx/backends"
	"github.com/gomlx/gomlx/backends/simplego"
	gomlxDtypes "github.com/gomlx/gomlx/pkg/core/dtypes"
	gomlxGraph "github.com/gomlx/gomlx/pkg/core/graph"
	"github.com/gomlx/gomlx/pkg/core/shapes"
	gomlxContext "github.com/gomlx/gomlx/pkg/ml/context"
	gomlxInitializers "github.com/gomlx/gomlx/pkg/ml/context/initializers"
	gomlxDatasets "github.com/gomlx/gomlx/pkg/ml/datasets"
	gomlxTrain "github.com/gomlx/gomlx/pkg/ml/train"
	gomlxLosses "github.com/gomlx/gomlx/pkg/ml/train/losses"
	gomlxOptimizers "github.com/gomlx/gomlx/pkg/ml/train/optimizers"

	"shop/pkg/recommend/offline/train/nn"
)

// gomlxAFMRunner 保存 gomlx AFM 训练后可复用的推理执行器。
type gomlxAFMRunner struct {
	backend      gomlxBackends.Backend
	ctx          *gomlxContext.Context
	trainer      *gomlxTrain.Trainer
	predictExec  *gomlxContext.Exec
	config       Config
	numFeatures  int
	embeddingDim []int
}

// fitWithGoMLX 使用 gomlx/simplego 训练 AFM 模型。
func (m *AFM) fitWithGoMLX(ctx context.Context, trainSet *Dataset, testSet *Dataset) Score {
	// 训练集为空时，直接返回空评估，避免创建无意义的 gomlx 图。
	if trainSet == nil || trainSet.Count() == 0 {
		return Score{}
	}
	m.init(trainSet)
	// gomlx 后端只复用编码元信息与缩放器，不再保留 native 权重副本。
	m.B = nil
	m.W = nil
	m.V = nil
	m.A = nil
	m.E = nil

	runner, err := newGoMLXAFMRunner(m.config, m.numFeatures, m.embeddingDim)
	if err != nil {
		panic(err)
	}
	m.gomlxRunner = runner

	rowList := m.prepareEncodedRows(trainSet)
	actualBatchSize := min(m.config.BatchSize, len(rowList))
	// 没有可训练批次时，直接返回空结果，避免创建空数据集。
	if actualBatchSize <= 0 {
		return Score{}
	}
	trainDataset, err := newGoMLXTrainDataset(runner.backend, rowList, m.numDimension, m.embeddingDim)
	if err != nil {
		panic(err)
	}
	trainDataset.Shuffle().BatchSize(actualBatchSize, false)

	evaluator := testSet
	// 没有独立验证集时，退化为在训练集上做基础评估。
	if evaluator == nil || evaluator.Count() == 0 {
		evaluator = trainSet
	}
	bestScore := Score{}
	bestEpoch := 0
	loop := gomlxTrain.NewLoop(runner.trainer)

	for epoch := 1; epoch <= m.config.Epochs; epoch++ {
		// 上层任务取消时，尽快结束离线训练，避免后台任务继续占资源。
		if ctx != nil && ctx.Err() != nil {
			return bestScore
		}
		if _, err = loop.RunEpochs(trainDataset, 1); err != nil {
			panic(fmt.Errorf("gomlx afm train epoch %d failed: %w", epoch, err))
		}

		// 只在设定间隔或最后一轮评估，保持和 native 训练行为一致。
		if epoch%m.config.Verbose != 0 && epoch != m.config.Epochs {
			continue
		}
		score := evaluateClassification(m, evaluator)
		// 仅在 AUC 提升时刷新最优模型快照指标。
		if score.AUC >= bestScore.AUC {
			bestScore = score
			bestEpoch = epoch
		}
		// 开启早停时，连续若干轮没有提升就提前结束。
		if m.config.Patience > 0 && epoch-bestEpoch >= m.config.Patience {
			break
		}
	}
	biasTensor, linearTensor, factorTensor, attentionList, encoderList, err := runner.MaterializeNativeModel()
	if err != nil {
		panic(err)
	}
	// 训练完成后同步保留一份原生张量，便于序列化和无 gomlx 上下文回放。
	m.B = biasTensor
	m.W = &nn.EmbeddingLayer{W: linearTensor}
	m.V = &nn.EmbeddingLayer{W: factorTensor}
	m.A = attentionList
	m.E = encoderList
	return bestScore
}

// newGoMLXAFMRunner 创建 gomlx AFM 训练与推理执行器。
func newGoMLXAFMRunner(config Config, numFeatures int, embeddingDim []int) (*gomlxAFMRunner, error) {
	// 特征空间为空时，无法创建线性项和因子项嵌入表。
	if numFeatures <= 0 {
		return nil, fmt.Errorf("gomlx afm requires positive numFeatures")
	}
	backend, err := simplego.New("ops_parallel")
	if err != nil {
		return nil, err
	}

	ctx := gomlxContext.New()
	// 显式提供种子时，固定初始化与随机流，保证训练可复现。
	if config.Seed != 0 {
		ctx.SetParam(gomlxContext.ParamInitialSeed, config.Seed)
		if err = ctx.SetRNGStateFromSeed(config.Seed); err != nil {
			return nil, err
		}
	}

	runner := &gomlxAFMRunner{
		backend:      backend,
		ctx:          ctx,
		config:       config,
		numFeatures:  numFeatures,
		embeddingDim: append([]int{}, embeddingDim...),
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
	runner.predictExec = gomlxContext.MustNewExec(
		backend,
		ctx.Reuse(),
		func(execCtx *gomlxContext.Context, inputs []*gomlxGraph.Node) *gomlxGraph.Node {
			return buildGoMLXAFMLogits(execCtx, inputs, config, numFeatures, embeddingDim)
		},
	)
	return runner, nil
}

// buildModel 构建 gomlx AFM 训练图。
func (r *gomlxAFMRunner) buildModel(ctx *gomlxContext.Context, _ any, inputs []*gomlxGraph.Node) []*gomlxGraph.Node {
	expectedInputCount := 2 + activeEmbeddingCount(r.embeddingDim)
	// 训练图固定接收“稀疏特征 + 向量特征”输入，不满足时直接暴露实现错误。
	if len(inputs) != expectedInputCount {
		panic(fmt.Sprintf("gomlx afm expects %d inputs, got %d", expectedInputCount, len(inputs)))
	}
	return []*gomlxGraph.Node{
		buildGoMLXAFMLogits(ctx, inputs, r.config, r.numFeatures, r.embeddingDim),
	}
}

// newOptimizer 根据配置创建 gomlx 优化器。
func (r *gomlxAFMRunner) newOptimizer() gomlxOptimizers.Interface {
	switch strings.TrimSpace(strings.ToLower(r.config.Optimizer)) {
	// SGD 分支关闭学习率衰减，尽量贴近当前 native 训练核行为。
	case "sgd":
		return gomlxOptimizers.StochasticGradientDescent().
			WithDecay(false).
			WithLearningRate(float64(r.config.Learning)).
			Done()
	default:
		return gomlxOptimizers.Adam().
			LearningRate(float64(r.config.Learning)).
			WeightDecay(float64(r.config.Reg)).
			Done()
	}
}

// Predict 使用 gomlx 推理执行器输出一批原始分值。
func (r *gomlxAFMRunner) Predict(rowList []encodedRow, numDimension int, embeddingDim []int) []float32 {
	// 推理执行器或批量维度无效时，直接返回空结果，避免触发非法图执行。
	if r == nil || r.predictExec == nil || len(rowList) == 0 || numDimension <= 0 {
		return []float32{}
	}
	batchInputs, _ := buildGoMLXBatch(rowList, numDimension, embeddingDim)
	outputTensor := r.predictExec.MustExec1(batchInputs...)
	defer outputTensor.MustFinalizeAll()
	return tensorToFloat32Slice(outputTensor.Value())
}

// newGoMLXTrainDataset 把编码样本转换为 gomlx 内存数据集。
func newGoMLXTrainDataset(backend gomlxBackends.Backend, rowList []encodedRow, numDimension int, embeddingDim []int) (*gomlxDatasets.InMemoryDataset, error) {
	batchInputs, targetData := buildGoMLXBatch(rowList, numDimension, embeddingDim)
	return gomlxDatasets.InMemoryFromData(
		backend,
		"afm_train",
		batchInputs,
		[]any{targetData},
	)
}

// buildGoMLXBatch 把编码样本展开成 gomlx 可直接消费的二维批量输入。
func buildGoMLXBatch(rowList []encodedRow, numDimension int, embeddingDim []int) ([]any, []float32) {
	indicesData := make([][]int32, len(rowList))
	valuesData := make([][]float32, len(rowList))
	targetData := make([]float32, len(rowList))
	embeddingDataList := make([][][]float32, len(embeddingDim))
	for embeddingIndex, dim := range embeddingDim {
		if dim <= 0 {
			continue
		}
		embeddingDataList[embeddingIndex] = make([][]float32, len(rowList))
		for rowIndex := range rowList {
			embeddingDataList[embeddingIndex][rowIndex] = make([]float32, dim)
		}
	}
	for rowIndex, row := range rowList {
		indicesData[rowIndex] = make([]int32, numDimension)
		valuesData[rowIndex] = make([]float32, numDimension)
		copy(indicesData[rowIndex], row.indices)
		copy(valuesData[rowIndex], row.values)
		for embeddingIndex, value := range row.embeddings {
			if embeddingIndex >= len(embeddingDim) || embeddingDim[embeddingIndex] <= 0 || len(embeddingDataList[embeddingIndex]) == 0 {
				continue
			}
			copy(embeddingDataList[embeddingIndex][rowIndex], value)
		}
		targetData[rowIndex] = row.target
	}
	inputs := []any{indicesData, valuesData}
	for embeddingIndex, dim := range embeddingDim {
		if dim <= 0 {
			continue
		}
		inputs = append(inputs, embeddingDataList[embeddingIndex])
	}
	return inputs, targetData
}

// buildGoMLXAFMLogits 构建与当前 native 实现同口径的 AFM 打分图。
func buildGoMLXAFMLogits(ctx *gomlxContext.Context, inputs []*gomlxGraph.Node, config Config, numFeatures int, embeddingDim []int) *gomlxGraph.Node {
	indices := inputs[0]
	values := inputs[1]
	g := values.Graph()
	indexInput := gomlxGraph.ConvertDType(indices, gomlxDtypes.Int32)
	valueInput := gomlxGraph.ConvertDType(values, gomlxDtypes.Float32)
	indexInput = gomlxGraph.InsertAxes(indexInput, -1)
	valueInput = gomlxGraph.InsertAxes(valueInput, -1)

	linearCtx := ctx.In("linear")
	linearCtx = linearCtx.WithInitializer(gomlxInitializers.RandomNormalFn(linearCtx, float64(config.InitStdDev)))
	linearTable := linearCtx.
		VariableWithShape("embeddings", shapes.Make(gomlxDtypes.Float32, numFeatures, 1)).
		ValueGraph(g)

	factorCtx := ctx.In("factor")
	factorCtx = factorCtx.WithInitializer(gomlxInitializers.RandomNormalFn(factorCtx, float64(config.InitStdDev)))
	factorTable := factorCtx.
		VariableWithShape("embeddings", shapes.Make(gomlxDtypes.Float32, numFeatures, config.Factors)).
		ValueGraph(g)

	bias := ctx.In("bias").
		WithInitializer(gomlxInitializers.Zero).
		VariableWithShape("value", shapes.Make(gomlxDtypes.Float32)).
		ValueGraph(g)

	linearTerm := gomlxGraph.ReduceSum(
		gomlxGraph.Mul(gomlxGraph.Gather(linearTable, indexInput), valueInput),
		1,
		2,
	)
	weightedEmbedding := gomlxGraph.Mul(gomlxGraph.Gather(factorTable, indexInput), valueInput)
	sumEmbedding := gomlxGraph.ReduceSum(weightedEmbedding, 1)
	interactionTerm := gomlxGraph.ReduceSum(
		gomlxGraph.MulScalar(
			gomlxGraph.Sub(
				gomlxGraph.Square(sumEmbedding),
				gomlxGraph.ReduceSum(gomlxGraph.Square(weightedEmbedding), 1),
			),
			0.5,
		),
		1,
	)
	output := gomlxGraph.Add(gomlxGraph.Add(linearTerm, interactionTerm), bias)
	inputOffset := 2
	for embeddingIndex, dim := range embeddingDim {
		if dim <= 0 {
			continue
		}
		attentionOutput := buildGoMLXAttention(ctx, inputs[inputOffset], config, embeddingIndex, dim)
		encodedNorm := buildGoMLXLinear(
			ctx,
			fmt.Sprintf("embedding_%d_encoder", embeddingIndex),
			attentionOutput,
			dim,
			config.Factors,
			config.InitStdDev,
		)
		output = gomlxGraph.Add(output, gomlxGraph.ReduceSum(gomlxGraph.Mul(sumEmbedding, encodedNorm), 1))
		inputOffset++
	}
	return output
}

// tensorToFloat32Slice 把 gomlx 输出转换成 float32 切片。
func tensorToFloat32Slice(value any) []float32 {
	switch typedValue := value.(type) {
	case []float32:
		result := make([]float32, len(typedValue))
		copy(result, typedValue)
		return result
	case []float64:
		result := make([]float32, len(typedValue))
		for index, item := range typedValue {
			result[index] = float32(item)
		}
		return result
	default:
		// 未识别的返回类型按空结果处理，避免把不确定数据写入排序链路。
		return []float32{}
	}
}

// MaterializeNativeModel 把 gomlx 权重转回仓库内原生模型层。
func (r *gomlxAFMRunner) MaterializeNativeModel() (*nn.Tensor, *nn.Tensor, *nn.Tensor, []nn.Layer, []nn.Layer, error) {
	if r == nil || r.ctx == nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("gomlx afm runner is nil")
	}
	biasVar := r.ctx.GetVariableByScopeAndName("/bias", "value")
	linearVar := r.ctx.GetVariableByScopeAndName("/linear", "embeddings")
	factorVar := r.ctx.GetVariableByScopeAndName("/factor", "embeddings")
	// 训练尚未真正执行时，当前变量可能还没初始化。
	if biasVar == nil || linearVar == nil || factorVar == nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("gomlx afm variables are not initialized")
	}
	biasTensor, err := tensorFromGoMLXValue(biasVar.MustValue().Value())
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	linearTensor, err := tensorFromGoMLXValue(linearVar.MustValue().Value())
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	factorTensor, err := tensorFromGoMLXValue(factorVar.MustValue().Value())
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	attentionList := make([]nn.Layer, 0, len(r.embeddingDim))
	encoderList := make([]nn.Layer, 0, len(r.embeddingDim))
	for embeddingIndex, dim := range r.embeddingDim {
		if dim <= 0 {
			attentionList = append(attentionList, nil)
			encoderList = append(encoderList, nil)
			continue
		}
		attentionLayer, attentionErr := r.materializeAttentionLayer(embeddingIndex, dim)
		if attentionErr != nil {
			return nil, nil, nil, nil, nil, attentionErr
		}
		encoderLayer, encoderErr := r.materializeLinearLayer(fmt.Sprintf("embedding_%d_encoder", embeddingIndex))
		if encoderErr != nil {
			return nil, nil, nil, nil, nil, encoderErr
		}
		attentionList = append(attentionList, attentionLayer)
		encoderList = append(encoderList, encoderLayer)
	}
	return biasTensor, linearTensor, factorTensor, attentionList, encoderList, nil
}

// buildGoMLXLinear 构建一个带偏置的线性层图。
func buildGoMLXLinear(ctx *gomlxContext.Context, scope string, input *gomlxGraph.Node, in int, out int, stddev float32) *gomlxGraph.Node {
	g := input.Graph()
	weightScope := ctx.In(scope).In("kernel")
	weightScope = weightScope.WithInitializer(gomlxInitializers.RandomNormalFn(weightScope, float64(stddev)))
	weight := weightScope.
		VariableWithShape("value", shapes.Make(gomlxDtypes.Float32, in, out)).
		ValueGraph(g)
	bias := ctx.In(scope).In("bias").
		WithInitializer(gomlxInitializers.Zero).
		VariableWithShape("value", shapes.Make(gomlxDtypes.Float32, out)).
		ValueGraph(g)
	// gomlx 的 Add 对非标量要求 rank 对齐，这里显式补一个 batch 轴，保持和 native 线性层同口径广播。
	bias = gomlxGraph.InsertAxes(bias, 0)
	return gomlxGraph.Add(gomlxGraph.MatMul(input, weight), bias)
}

// buildGoMLXAttention 构建与原生实现一致的注意力投影分支。
func buildGoMLXAttention(ctx *gomlxContext.Context, input *gomlxGraph.Node, config Config, embeddingIndex int, dim int) *gomlxGraph.Node {
	scope := fmt.Sprintf("embedding_%d_attention", embeddingIndex)
	hidden := buildGoMLXLinear(ctx, scope+"_projection", input, dim, config.Factors, config.InitStdDev)
	hidden = gomlxGraph.Max(hidden, gomlxGraph.ScalarZero(input.Graph(), input.DType()))
	attentionScope := ctx.In(scope).In("h")
	attentionScope = attentionScope.WithInitializer(gomlxInitializers.RandomNormalFn(attentionScope, float64(config.InitStdDev)))
	h := attentionScope.
		VariableWithShape("value", shapes.Make(gomlxDtypes.Float32, config.Factors, dim)).
		ValueGraph(input.Graph())
	attentionWeight := gomlxGraph.Softmax(gomlxGraph.MatMul(hidden, h), 1)
	return gomlxGraph.Mul(attentionWeight, input)
}

// activeEmbeddingCount 返回有效命名向量数量。
func activeEmbeddingCount(embeddingDim []int) int {
	total := 0
	for _, dim := range embeddingDim {
		if dim > 0 {
			total++
		}
	}
	return total
}

// materializeLinearLayer 把 gomlx 线性层权重转回原生线性层。
func (r *gomlxAFMRunner) materializeLinearLayer(scope string) (nn.Layer, error) {
	weightVar := r.ctx.GetVariableByScopeAndName("/"+scope+"/kernel", "value")
	biasVar := r.ctx.GetVariableByScopeAndName("/"+scope+"/bias", "value")
	if weightVar == nil || biasVar == nil {
		return nil, fmt.Errorf("gomlx linear variables are not initialized: %s", scope)
	}
	weightTensor, err := tensorFromGoMLXValue(weightVar.MustValue().Value())
	if err != nil {
		return nil, err
	}
	biasTensor, err := tensorFromGoMLXValue(biasVar.MustValue().Value())
	if err != nil {
		return nil, err
	}
	return &nn.LinearLayer{
		W: weightTensor,
		B: biasTensor,
	}, nil
}

// materializeAttentionLayer 把 gomlx 注意力层权重转回原生注意力层。
func (r *gomlxAFMRunner) materializeAttentionLayer(embeddingIndex int, dim int) (nn.Layer, error) {
	scope := fmt.Sprintf("embedding_%d_attention", embeddingIndex)
	linearLayer, err := r.materializeLinearLayer(scope + "_projection")
	if err != nil {
		return nil, err
	}
	hVar := r.ctx.GetVariableByScopeAndName("/"+scope+"/h", "value")
	if hVar == nil {
		return nil, fmt.Errorf("gomlx attention variable is not initialized: %s", scope)
	}
	hTensor, err := tensorFromGoMLXValue(hVar.MustValue().Value())
	if err != nil {
		return nil, err
	}
	attention := nn.NewAttention(dim, r.config.Factors)
	attention.W = linearLayer
	attention.H = hTensor
	return attention, nil
}

// tensorFromGoMLXValue 把 gomlx 返回值转换成原生张量。
func tensorFromGoMLXValue(value any) (*nn.Tensor, error) {
	switch typedValue := value.(type) {
	case float32:
		return nn.NewScalar(typedValue), nil
	case float64:
		return nn.NewScalar(float32(typedValue)), nil
	case []float32:
		return nn.NewTensor(append([]float32{}, typedValue...), len(typedValue)), nil
	case []float64:
		data := make([]float32, len(typedValue))
		for index, item := range typedValue {
			data[index] = float32(item)
		}
		return nn.NewTensor(data, len(data)), nil
	case [][]float32:
		return buildMatrixTensor32(typedValue)
	case [][]float64:
		data := make([][]float32, len(typedValue))
		for rowIndex := range typedValue {
			data[rowIndex] = make([]float32, len(typedValue[rowIndex]))
			for colIndex, item := range typedValue[rowIndex] {
				data[rowIndex][colIndex] = float32(item)
			}
		}
		return buildMatrixTensor32(data)
	default:
		return nil, fmt.Errorf("unsupported gomlx tensor type %T", value)
	}
}

// buildMatrixTensor32 把二维矩阵转换成原生张量。
func buildMatrixTensor32(value [][]float32) (*nn.Tensor, error) {
	if len(value) == 0 {
		return nil, fmt.Errorf("gomlx matrix is empty")
	}
	colCount := len(value[0])
	if colCount == 0 {
		return nil, fmt.Errorf("gomlx matrix column is empty")
	}
	data := make([]float32, 0, len(value)*colCount)
	for rowIndex := range value {
		// 矩阵各行长度不一致时，无法还原合法张量。
		if len(value[rowIndex]) != colCount {
			return nil, fmt.Errorf("gomlx matrix row %d size is invalid", rowIndex)
		}
		data = append(data, value[rowIndex]...)
	}
	return nn.NewTensor(data, len(value), colCount), nil
}

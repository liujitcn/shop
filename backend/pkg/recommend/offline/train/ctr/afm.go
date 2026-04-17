package ctr

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"sync"
	"time"

	"shop/pkg/recommend/offline/train/nn"
)

// Score 表示二分类评估结果。
type Score struct {
	Precision float32
	Recall    float32
	Accuracy  float32
	AUC       float32
}

// Config 表示 AFM 训练参数。
type Config struct {
	BatchSize  int
	Backend    string
	Factors    int
	Epochs     int
	Jobs       int
	Verbose    int
	Patience   int
	Learning   float32
	Reg        float32
	InitMean   float32
	InitStdDev float32
	Optimizer  string
	AutoScale  bool
	Seed       int64
}

const (
	// BackendNative 表示使用当前仓库内置训练核。
	BackendNative = "native"
	// BackendGoMLX 表示使用 gomlx/simplego 训练后端。
	BackendGoMLX = "gomlx"
)

// fillDefault 补齐未显式配置的训练参数。
func (c Config) fillDefault() Config {
	if c.BatchSize <= 0 {
		c.BatchSize = 1024
	}
	if c.Factors <= 0 {
		c.Factors = 16
	}
	if c.Epochs <= 0 {
		c.Epochs = 50
	}
	if c.Jobs <= 0 {
		c.Jobs = 1
	}
	if c.Verbose <= 0 {
		c.Verbose = 10
	}
	if c.Learning <= 0 {
		c.Learning = 0.001
	}
	if c.Reg <= 0 {
		c.Reg = 0.0002
	}
	if c.InitStdDev <= 0 {
		c.InitStdDev = 0.01
	}
	if c.Optimizer == "" {
		c.Optimizer = "adam"
	}
	if c.NormalizeBackend() == "" {
		c.Backend = BackendNative
	}
	return c
}

// NormalizeBackend 返回归一化后的训练后端名称。
func (c Config) NormalizeBackend() string {
	return strings.TrimSpace(strings.ToLower(c.Backend))
}

// AFM 表示注意力因子分解机模型。
type AFM struct {
	config Config

	B *nn.Tensor
	W nn.Layer
	V nn.Layer
	A []nn.Layer
	E []nn.Layer

	index          *UnifiedMapIndex
	scalers        map[int32]*AutoScaler
	mu             sync.RWMutex
	numFeatures    int
	numDimension   int
	embeddingDim   []int
	embeddingCount int
	gomlxRunner    *gomlxAFMRunner
}

// NewAFM 创建 AFM 模型实例。
func NewAFM(config Config) *AFM {
	config = config.fillDefault()
	return &AFM{
		config:         config,
		scalers:        make(map[int32]*AutoScaler),
		embeddingDim:   []int{},
		embeddingCount: 0,
	}
}

// Fit 使用训练集和测试集拟合 AFM 模型。
func (m *AFM) Fit(ctx context.Context, trainSet *Dataset, testSet *Dataset) Score {
	// gomlx 后端启用时，优先走 gomlx/simplego 训练链路。
	if m.config.NormalizeBackend() == BackendGoMLX {
		return m.fitWithGoMLX(ctx, trainSet, testSet)
	}
	return m.fitWithNative(ctx, trainSet, testSet)
}

// fitWithNative 使用当前仓库内置训练核拟合 AFM 模型。
func (m *AFM) fitWithNative(ctx context.Context, trainSet *Dataset, testSet *Dataset) Score {
	if trainSet == nil || trainSet.Count() == 0 {
		return Score{}
	}
	m.gomlxRunner = nil
	m.init(trainSet)
	m.W.SetJobs(m.config.Jobs)
	m.V.SetJobs(m.config.Jobs)
	for _, layer := range m.A {
		layer.SetJobs(m.config.Jobs)
	}
	for _, layer := range m.E {
		layer.SetJobs(m.config.Jobs)
	}

	inputs := m.prepareRows(trainSet)
	evaluator := testSet
	if evaluator == nil {
		evaluator = trainSet
	}
	bestScore := evaluateClassification(m, evaluator)
	bestEpoch := 0

	var optimizer nn.Optimizer
	switch m.config.Optimizer {
	case "sgd":
		optimizer = nn.NewSGD(m.parameters(), m.config.Learning)
	default:
		optimizer = nn.NewAdam(m.parameters(), m.config.Learning)
	}
	optimizer.SetWeightDecay(m.config.Reg)
	optimizer.SetJobs(m.config.Jobs)

	for epoch := 1; epoch <= m.config.Epochs; epoch++ {
		startedAt := time.Now()
		loss := float32(0)
		for offset := 0; offset < trainSet.Count(); offset += m.config.BatchSize {
			if ctx != nil && ctx.Err() != nil {
				return bestScore
			}
			end := min(offset+m.config.BatchSize, trainSet.Count())
			output := m.Forward(
				inputs.indices.Slice(offset, end),
				inputs.values.Slice(offset, end),
				sliceTensorList(inputs.embeddings, offset, end),
				m.config.Jobs,
			)
			target := inputs.target.Slice(offset, end)
			batchLoss := nn.BCEWithLogits(target, output, nil)
			loss += batchLoss.Data()[0]
			optimizer.ZeroGrad()
			batchLoss.Backward()
			optimizer.Step()
		}

		// 只在指定间隔或最后一轮评估，避免训练时频繁阻塞。
		if epoch%m.config.Verbose != 0 && epoch != m.config.Epochs {
			continue
		}
		score := evaluateClassification(m, evaluator)
		if score.AUC >= bestScore.AUC {
			bestScore = score
			bestEpoch = epoch
		}
		// 开启早停时，连续若干轮未提升则结束训练。
		if m.config.Patience > 0 && epoch-bestEpoch >= m.config.Patience {
			break
		}
		_ = loss
		_ = startedAt
	}
	return bestScore
}

// PredictBatch 对样本批量输出打分。
func (m *AFM) PredictBatch(sampleList []Sample) []float32 {
	encodedList := m.encodeInferenceSamples(sampleList)
	if len(encodedList) == 0 {
		return []float32{}
	}
	return m.predictEncodedRows(encodedList)
}

// Forward 计算一批样本的 AFM 原始分值。
func (m *AFM) Forward(indices *nn.Tensor, values *nn.Tensor, embeddings []*nn.Tensor, jobs int) *nn.Tensor {
	batchSize := indices.Shape()[0]
	v := m.V.Forward(indices)
	x := nn.Reshape(values, batchSize, m.numDimension, 1)
	vx := nn.BMM(v, x, true, false, jobs)
	sumSquare := nn.Square(vx)
	e2 := nn.Square(v)
	x2 := nn.Square(x)
	squareSum := nn.BMM(e2, x2, true, false, jobs)
	sum := nn.Sub(sumSquare, squareSum)
	sum = nn.Sum(sum, 1)
	sum = nn.Mul(sum, nn.NewScalar(0.5))
	w := m.W.Forward(indices)
	linear := nn.BMM(w, x, true, false, jobs)
	output := nn.Add(nn.Reshape(linear, batchSize), nn.Reshape(sum, batchSize), m.B)
	for i, embedding := range embeddings {
		encodedNorm := m.E[i].Forward(m.A[i].Forward(embedding))
		encodedNorm = nn.Reshape(encodedNorm, batchSize, m.config.Factors, 1)
		output = nn.Add(output, nn.Reshape(nn.BMM(vx, encodedNorm, true, false, jobs), batchSize))
	}
	return nn.Flatten(output)
}

// prepareRows 将训练数据转成张量。
func (m *AFM) prepareRows(dataset *Dataset) tensorBatch {
	rowList := m.prepareEncodedRows(dataset)
	indicesTensor, valuesTensor, embeddingTensor, targetTensor := m.convertRowsToTensors(rowList)
	return tensorBatch{
		indices:    indicesTensor,
		values:     valuesTensor,
		embeddings: embeddingTensor,
		target:     targetTensor,
	}
}

// init 根据训练集初始化模型参数。
func (m *AFM) init(trainSet *Dataset) {
	rand.Seed(m.config.Seed)
	m.index = trainSet.Index
	m.numFeatures = int(trainSet.Index.Len())
	m.numDimension = 0
	for _, row := range trainSet.rows {
		m.numDimension = max(m.numDimension, len(row.indices))
	}
	m.B = nn.Zeros()
	m.W = nn.NewEmbedding(m.numFeatures, 1)
	m.V = nn.NewEmbedding(m.numFeatures, m.config.Factors)
	m.embeddingDim = append([]int{}, trainSet.GetEmbeddingDim()...)
	m.embeddingCount = len(m.embeddingDim)
	m.A = make([]nn.Layer, 0, m.embeddingCount)
	m.E = make([]nn.Layer, 0, m.embeddingCount)
	for _, dim := range m.embeddingDim {
		if dim <= 0 {
			m.A = append(m.A, nil)
			m.E = append(m.E, nil)
			continue
		}
		m.A = append(m.A, nn.NewAttention(dim, m.config.Factors))
		m.E = append(m.E, nn.NewLinear(dim, m.config.Factors))
	}
	if m.config.AutoScale {
		m.fitScalers(trainSet)
	}
}

// fitScalers 为数值特征拟合缩放器。
func (m *AFM) fitScalers(trainSet *Dataset) {
	m.scalers = make(map[int32]*AutoScaler)
	featureValues := make(map[int32][]float32)
	for _, row := range trainSet.rows {
		for i, index := range row.indices {
			featureValues[index] = append(featureValues[index], row.values[i])
		}
	}
	for index, values := range featureValues {
		isNumerical := false
		for _, value := range values {
			if value != 1 {
				isNumerical = true
				break
			}
		}
		if !isNumerical {
			continue
		}
		scaler := NewAutoScaler()
		scaler.Fit(values)
		m.scalers[index] = scaler
	}
}

// parameters 返回当前模型需要更新的参数列表。
func (m *AFM) parameters() []*nn.Tensor {
	result := []*nn.Tensor{m.B}
	result = append(result, m.W.Parameters()...)
	result = append(result, m.V.Parameters()...)
	for i := range m.A {
		result = append(result, m.A[i].Parameters()...)
		result = append(result, m.E[i].Parameters()...)
	}
	return result
}

// encodeInferenceSample 使用训练期索引编码推理样本。
func (m *AFM) encodeInferenceSample(sample Sample) encodedRow {
	if m.index == nil {
		return encodedRow{}
	}
	dataset := &Dataset{Index: m.index}
	row := dataset.encodeSample(sample)
	for index, featureIndex := range row.indices {
		if scaler, ok := m.scalers[featureIndex]; ok {
			row.values[index] = scaler.Transform(row.values[index])
		}
	}
	return row
}

// encodeInferenceSamples 使用训练期索引批量编码推理样本。
func (m *AFM) encodeInferenceSamples(sampleList []Sample) []encodedRow {
	result := make([]encodedRow, 0, len(sampleList))
	for _, sample := range sampleList {
		result = append(result, m.encodeInferenceSample(sample))
	}
	return result
}

// prepareEncodedRows 将数据集样本转成带缩放的编码结果。
func (m *AFM) prepareEncodedRows(dataset *Dataset) []encodedRow {
	rowList := make([]encodedRow, 0, dataset.Count())
	for _, row := range dataset.rows {
		clonedRow := encodedRow{
			indices:    append([]int32{}, row.indices...),
			values:     append([]float32{}, row.values...),
			embeddings: cloneEmbeddingRows(row.embeddings),
			target:     row.target,
		}
		for index, featureIndex := range clonedRow.indices {
			if scaler, ok := m.scalers[featureIndex]; ok {
				clonedRow.values[index] = scaler.Transform(clonedRow.values[index])
			}
		}
		rowList = append(rowList, clonedRow)
	}
	return rowList
}

// predictEncodedRows 对已编码样本批量输出原始分值。
func (m *AFM) predictEncodedRows(rowList []encodedRow) []float32 {
	// 当前模型已经切到 gomlx 后端时，复用 gomlx 推理执行器输出分值。
	if m.gomlxRunner != nil {
		return m.gomlxRunner.Predict(rowList, m.numDimension, m.embeddingDim)
	}
	indicesTensor, valuesTensor, embeddingTensor, _ := m.convertRowsToTensors(rowList)
	output := m.Forward(indicesTensor, valuesTensor, embeddingTensor, m.config.Jobs)
	scoreList := make([]float32, len(output.Data()))
	copy(scoreList, output.Data())
	return scoreList
}

// convertRowsToTensors 把编码样本列表转成训练张量。
func (m *AFM) convertRowsToTensors(rowList []encodedRow) (*nn.Tensor, *nn.Tensor, []*nn.Tensor, *nn.Tensor) {
	alignedIndices := make([]float32, len(rowList)*m.numDimension)
	alignedValues := make([]float32, len(rowList)*m.numDimension)
	alignedTarget := make([]float32, len(rowList))
	embeddingTensorList := make([]*nn.Tensor, 0, len(m.embeddingDim))
	embeddingDataList := make([][]float32, len(m.embeddingDim))
	for index, dim := range m.embeddingDim {
		if dim <= 0 {
			continue
		}
		embeddingDataList[index] = make([]float32, len(rowList)*dim)
	}
	for i, row := range rowList {
		for j := range row.indices {
			alignedIndices[i*m.numDimension+j] = float32(row.indices[j])
			alignedValues[i*m.numDimension+j] = row.values[j]
		}
		for embeddingIndex, embeddingValue := range row.embeddings {
			if embeddingIndex >= len(m.embeddingDim) || m.embeddingDim[embeddingIndex] <= 0 || len(embeddingDataList[embeddingIndex]) == 0 {
				continue
			}
			offset := i * m.embeddingDim[embeddingIndex]
			copy(embeddingDataList[embeddingIndex][offset:offset+m.embeddingDim[embeddingIndex]], embeddingValue)
		}
		alignedTarget[i] = row.target
	}
	for index, dim := range m.embeddingDim {
		if dim <= 0 {
			continue
		}
		embeddingTensorList = append(embeddingTensorList, nn.NewTensor(embeddingDataList[index], len(rowList), dim))
	}
	return nn.NewTensor(alignedIndices, len(rowList), m.numDimension), nn.NewTensor(alignedValues, len(rowList), m.numDimension), embeddingTensorList, nn.NewTensor(alignedTarget, len(rowList))
}

// tensorBatch 表示训练阶段预编码后的张量集合。
type tensorBatch struct {
	indices    *nn.Tensor
	values     *nn.Tensor
	embeddings []*nn.Tensor
	target     *nn.Tensor
}

// cloneEmbeddingRows 深拷贝样本中的命名向量列表。
func cloneEmbeddingRows(embeddings [][]float32) [][]float32 {
	result := make([][]float32, len(embeddings))
	for index := range embeddings {
		result[index] = append([]float32{}, embeddings[index]...)
	}
	return result
}

// sliceTensorList 对一组张量做统一切片。
func sliceTensorList(tensorList []*nn.Tensor, begin int, end int) []*nn.Tensor {
	result := make([]*nn.Tensor, 0, len(tensorList))
	for _, item := range tensorList {
		if item == nil {
			continue
		}
		result = append(result, item.Slice(begin, end))
	}
	return result
}

// evaluateClassification 评估当前模型在数据集上的分类表现。
func evaluateClassification(model *AFM, dataset *Dataset) Score {
	if model == nil || dataset == nil || dataset.Count() == 0 {
		return Score{}
	}
	prediction := model.predictEncodedRows(model.prepareEncodedRows(dataset))
	positiveList := make([]float32, 0, dataset.PositiveCount)
	negativeList := make([]float32, 0, dataset.NegativeCount)
	for index, row := range dataset.rows {
		if row.target > 0 {
			positiveList = append(positiveList, prediction[index])
			continue
		}
		negativeList = append(negativeList, prediction[index])
	}
	return Score{
		Precision: precision(positiveList, negativeList),
		Recall:    recall(positiveList),
		Accuracy:  accuracy(positiveList, negativeList),
		AUC:       auc(positiveList, negativeList),
	}
}

// precision 计算精确率。
func precision(positiveList []float32, negativeList []float32) float32 {
	var truePositive float32
	var falsePositive float32
	for _, value := range positiveList {
		if value > 0 {
			truePositive++
		}
	}
	for _, value := range negativeList {
		if value > 0 {
			falsePositive++
		}
	}
	if truePositive+falsePositive == 0 {
		return 0
	}
	return truePositive / (truePositive + falsePositive)
}

// recall 计算召回率。
func recall(positiveList []float32) float32 {
	var truePositive float32
	var falseNegative float32
	for _, value := range positiveList {
		if value > 0 {
			truePositive++
			continue
		}
		falseNegative++
	}
	if truePositive+falseNegative == 0 {
		return 0
	}
	return truePositive / (truePositive + falseNegative)
}

// accuracy 计算准确率。
func accuracy(positiveList []float32, negativeList []float32) float32 {
	total := len(positiveList) + len(negativeList)
	if total == 0 {
		return 0
	}
	correct := 0
	for _, value := range positiveList {
		if value > 0 {
			correct++
		}
	}
	for _, value := range negativeList {
		if value < 0 {
			correct++
		}
	}
	return float32(correct) / float32(total)
}

// auc 计算二分类 AUC。
func auc(positiveList []float32, negativeList []float32) float32 {
	if len(positiveList) == 0 || len(negativeList) == 0 {
		return 0
	}
	lessCount := float32(0)
	for _, positiveValue := range positiveList {
		for _, negativeValue := range negativeList {
			if positiveValue > negativeValue {
				lessCount++
			}
		}
	}
	return lessCount / float32(len(positiveList)*len(negativeList))
}

// String 返回简要评估结果。
func (s Score) String() string {
	return fmt.Sprintf("precision=%.4f recall=%.4f accuracy=%.4f auc=%.4f", s.Precision, s.Recall, s.Accuracy, s.AUC)
}

// Validate 检查模型当前是否已经完成训练初始化。
func (m *AFM) Validate() error {
	if m == nil || m.index == nil {
		return fmt.Errorf("afm model is not fitted")
	}
	if m.numFeatures <= 0 || m.numDimension <= 0 {
		return fmt.Errorf("afm model dimension is invalid")
	}
	return nil
}

// SigmoidScore 将原始分值映射到 0-1 区间。
func SigmoidScore(value float32) float32 {
	return 1 / (1 + float32(math.Exp(float64(-value))))
}

package ctr

import (
	"fmt"
	"sort"
	"strconv"

	"shop/pkg/recommend/offline/train/nn"
)

// Snapshot 表示 AFM 模型可序列化快照。
type Snapshot struct {
	Config       Config                    `json:"config"`       // 当前训练配置。
	NumFeatures  int                       `json:"numFeatures"`  // 特征空间大小。
	NumDimension int                       `json:"numDimension"` // 单样本最大特征维度。
	EmbeddingDim []int                     `json:"embeddingDim"` // 命名向量维度列表。
	Index        *unifiedMapIndexSnapshot  `json:"index"`        // 训练期统一索引快照。
	Scalers      []scalerEntrySnapshot     `json:"scalers"`      // 数值缩放器快照列表。
	Bias         *tensorSnapshot           `json:"bias"`         // 全局偏置张量。
	Linear       *tensorSnapshot           `json:"linear"`       // 线性项嵌入表。
	Factor       *tensorSnapshot           `json:"factor"`       // 因子项嵌入表。
	Attention    []*attentionLayerSnapshot `json:"attention"`    // 注意力层快照。
	Encoder      []*linearLayerSnapshot    `json:"encoder"`      // 向量编码层快照。
}

// tensorSnapshot 表示张量快照。
type tensorSnapshot struct {
	Shape []int     `json:"shape"` // 张量形状。
	Data  []float32 `json:"data"`  // 张量扁平数据。
}

// linearLayerSnapshot 表示线性层快照。
type linearLayerSnapshot struct {
	Weight *tensorSnapshot `json:"weight"` // 权重张量。
	Bias   *tensorSnapshot `json:"bias"`   // 偏置张量。
}

// attentionLayerSnapshot 表示注意力层快照。
type attentionLayerSnapshot struct {
	Projection *linearLayerSnapshot `json:"projection"` // 投影层快照。
	H          *tensorSnapshot      `json:"h"`          // 注意力张量。
}

// sparseIndexSnapshot 表示稀疏索引快照。
type sparseIndexSnapshot struct {
	Names []string `json:"names"` // 编码顺序对应的名称列表。
}

// unifiedMapIndexSnapshot 表示 AFM 统一索引快照。
type unifiedMapIndexSnapshot struct {
	UserIndex      *sparseIndexSnapshot `json:"userIndex"`      // 用户索引。
	ItemIndex      *sparseIndexSnapshot `json:"itemIndex"`      // 商品索引。
	UserLabelIndex *sparseIndexSnapshot `json:"userLabelIndex"` // 用户特征索引。
	ItemLabelIndex *sparseIndexSnapshot `json:"itemLabelIndex"` // 商品特征索引。
	ContextIndex   *sparseIndexSnapshot `json:"contextIndex"`   // 上下文特征索引。
	EmbeddingIndex *sparseIndexSnapshot `json:"embeddingIndex"` // 向量特征索引。
}

// scalerEntrySnapshot 表示一个缩放器快照条目。
type scalerEntrySnapshot struct {
	FeatureIndex int32               `json:"featureIndex"` // 特征编号。
	Scaler       *autoScalerSnapshot `json:"scaler"`       // 缩放器状态。
}

// autoScalerSnapshot 表示自动缩放器快照。
type autoScalerSnapshot struct {
	UseLog bool          `json:"useLog"` // 是否使用对数缩放。
	MinMax *MinMaxScaler `json:"minMax"` // MinMax 缩放器。
	Robust *RobustScaler `json:"robust"` // 鲁棒缩放器。
}

// ExportSnapshot 导出当前 AFM 模型快照。
func (m *AFM) ExportSnapshot() (*Snapshot, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	biasTensor, linearTensor, factorTensor, attentionList, encoderList, err := m.exportCoreTensors()
	if err != nil {
		return nil, err
	}
	attentionSnapshotList, err := exportAttentionSnapshotList(attentionList)
	if err != nil {
		return nil, err
	}
	encoderSnapshotList, err := exportLinearLayerSnapshotList(encoderList)
	if err != nil {
		return nil, err
	}
	return &Snapshot{
		Config:       m.config,
		NumFeatures:  m.numFeatures,
		NumDimension: m.numDimension,
		EmbeddingDim: append([]int{}, m.embeddingDim...),
		Index:        exportUnifiedMapIndexSnapshot(m.index),
		Scalers:      exportScalerSnapshotList(m.scalers),
		Bias:         exportTensorSnapshot(biasTensor),
		Linear:       exportTensorSnapshot(linearTensor),
		Factor:       exportTensorSnapshot(factorTensor),
		Attention:    attentionSnapshotList,
		Encoder:      encoderSnapshotList,
	}, nil
}

// BuildModel 根据快照恢复 AFM 模型。
func (s *Snapshot) BuildModel() (*AFM, error) {
	// 快照为空时，无法恢复模型。
	if s == nil {
		return nil, fmt.Errorf("afm snapshot is nil")
	}
	if s.Index == nil {
		return nil, fmt.Errorf("afm snapshot index is nil")
	}
	biasTensor, err := buildTensorFromSnapshot(s.Bias)
	if err != nil {
		return nil, err
	}
	linearTensor, err := buildTensorFromSnapshot(s.Linear)
	if err != nil {
		return nil, err
	}
	factorTensor, err := buildTensorFromSnapshot(s.Factor)
	if err != nil {
		return nil, err
	}

	model := NewAFM(s.Config)
	model.index = s.Index.buildUnifiedMapIndex()
	model.scalers = buildScalerMapFromSnapshot(s.Scalers)
	model.numFeatures = s.NumFeatures
	// 快照未显式写特征空间大小时，回退到线性表第一维。
	if model.numFeatures <= 0 {
		linearShape := linearTensor.Shape()
		if len(linearShape) == 0 {
			return nil, fmt.Errorf("afm snapshot linear shape is invalid")
		}
		model.numFeatures = linearShape[0]
	}
	model.numDimension = s.NumDimension
	model.B = biasTensor
	model.W = &nn.EmbeddingLayer{W: linearTensor}
	model.V = &nn.EmbeddingLayer{W: factorTensor}
	model.embeddingDim = append([]int{}, s.EmbeddingDim...)
	model.embeddingCount = len(model.embeddingDim)
	model.A, err = buildAttentionLayerListFromSnapshot(s.Attention, model.embeddingDim)
	if err != nil {
		return nil, err
	}
	model.E, err = buildLinearLayerListFromSnapshot(s.Encoder)
	if err != nil {
		return nil, err
	}
	model.gomlxRunner = nil
	if err = model.Validate(); err != nil {
		return nil, err
	}
	return model, nil
}

// exportCoreTensors 导出当前模型需要持久化的核心张量与注意力层。
func (m *AFM) exportCoreTensors() (*nn.Tensor, *nn.Tensor, *nn.Tensor, []nn.Layer, []nn.Layer, error) {
	if m == nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("afm model is nil")
	}
	// gomlx 训练后若尚未落回原生张量，先从 gomlx 上下文中抽取一份可持久化权重。
	if m.B == nil || m.W == nil || m.V == nil {
		if m.gomlxRunner == nil {
			return nil, nil, nil, nil, nil, fmt.Errorf("afm core tensors are empty")
		}
		biasTensor, linearTensor, factorTensor, attentionList, encoderList, err := m.gomlxRunner.MaterializeNativeModel()
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}
		m.B = biasTensor
		m.W = &nn.EmbeddingLayer{W: linearTensor}
		m.V = &nn.EmbeddingLayer{W: factorTensor}
		m.A = attentionList
		m.E = encoderList
	}

	linearTensor, err := exportEmbeddingTensor(m.W)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	factorTensor, err := exportEmbeddingTensor(m.V)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	return cloneTensor(m.B), linearTensor, factorTensor, cloneLayerList(m.A), cloneLayerList(m.E), nil
}

// exportEmbeddingTensor 导出嵌入层张量副本。
func exportEmbeddingTensor(layer nn.Layer) (*nn.Tensor, error) {
	embeddingLayer, ok := layer.(*nn.EmbeddingLayer)
	if !ok || embeddingLayer == nil || embeddingLayer.W == nil {
		return nil, fmt.Errorf("afm embedding layer is invalid")
	}
	return cloneTensor(embeddingLayer.W), nil
}

// cloneTensor 深拷贝一个训练张量。
func cloneTensor(tensor *nn.Tensor) *nn.Tensor {
	if tensor == nil {
		return nil
	}
	data := append([]float32{}, tensor.Data()...)
	return nn.NewTensor(data, tensor.Shape()...)
}

// exportTensorSnapshot 导出张量快照。
func exportTensorSnapshot(tensor *nn.Tensor) *tensorSnapshot {
	if tensor == nil {
		return nil
	}
	return &tensorSnapshot{
		Shape: append([]int{}, tensor.Shape()...),
		Data:  append([]float32{}, tensor.Data()...),
	}
}

// buildTensorFromSnapshot 根据张量快照恢复张量。
func buildTensorFromSnapshot(snapshot *tensorSnapshot) (*nn.Tensor, error) {
	if snapshot == nil {
		return nil, fmt.Errorf("afm tensor snapshot is nil")
	}
	if len(snapshot.Shape) == 0 {
		if len(snapshot.Data) != 1 {
			return nil, fmt.Errorf("afm scalar snapshot data size is invalid")
		}
		return nn.NewScalar(snapshot.Data[0]), nil
	}
	expectedSize := 1
	for _, item := range snapshot.Shape {
		expectedSize *= item
	}
	if expectedSize != len(snapshot.Data) {
		return nil, fmt.Errorf("afm tensor snapshot shape does not match data size")
	}
	return nn.NewTensor(append([]float32{}, snapshot.Data...), snapshot.Shape...), nil
}

// exportUnifiedMapIndexSnapshot 导出统一索引快照。
func exportUnifiedMapIndexSnapshot(index *UnifiedMapIndex) *unifiedMapIndexSnapshot {
	if index == nil {
		return nil
	}
	return &unifiedMapIndexSnapshot{
		UserIndex:      exportSparseIndexSnapshot(index.userIndex),
		ItemIndex:      exportSparseIndexSnapshot(index.itemIndex),
		UserLabelIndex: exportSparseIndexSnapshot(index.userLabelIndex),
		ItemLabelIndex: exportSparseIndexSnapshot(index.itemLabelIndex),
		ContextIndex:   exportSparseIndexSnapshot(index.ctxLabelIndex),
		EmbeddingIndex: exportSparseIndexSnapshot(index.embeddingIndex),
	}
}

// exportSparseIndexSnapshot 导出稀疏索引快照。
func exportSparseIndexSnapshot(index *sparseIndex) *sparseIndexSnapshot {
	if index == nil {
		return nil
	}
	return &sparseIndexSnapshot{
		Names: append([]string{}, index.names...),
	}
}

// buildUnifiedMapIndex 根据快照恢复统一索引。
func (s *unifiedMapIndexSnapshot) buildUnifiedMapIndex() *UnifiedMapIndex {
	if s == nil {
		return nil
	}
	return &UnifiedMapIndex{
		userIndex:      s.UserIndex.buildSparseIndex(),
		itemIndex:      s.ItemIndex.buildSparseIndex(),
		userLabelIndex: s.UserLabelIndex.buildSparseIndex(),
		itemLabelIndex: s.ItemLabelIndex.buildSparseIndex(),
		ctxLabelIndex:  s.ContextIndex.buildSparseIndex(),
		embeddingIndex: s.EmbeddingIndex.buildSparseIndex(),
	}
}

// buildSparseIndex 根据快照恢复稀疏索引。
func (s *sparseIndexSnapshot) buildSparseIndex() *sparseIndex {
	index := newSparseIndex()
	if s == nil {
		return index
	}
	index.names = append([]string{}, s.Names...)
	index.numbers = make(map[string]int32, len(index.names))
	for position, name := range index.names {
		index.numbers[name] = int32(position)
	}
	return index
}

// exportScalerSnapshotList 导出缩放器快照列表。
func exportScalerSnapshotList(scalerMap map[int32]*AutoScaler) []scalerEntrySnapshot {
	if len(scalerMap) == 0 {
		return []scalerEntrySnapshot{}
	}
	indexList := make([]int32, 0, len(scalerMap))
	for featureIndex := range scalerMap {
		indexList = append(indexList, featureIndex)
	}
	sort.Slice(indexList, func(i int, j int) bool {
		return indexList[i] < indexList[j]
	})
	result := make([]scalerEntrySnapshot, 0, len(indexList))
	for _, featureIndex := range indexList {
		scaler := scalerMap[featureIndex]
		if scaler == nil {
			continue
		}
		result = append(result, scalerEntrySnapshot{
			FeatureIndex: featureIndex,
			Scaler: &autoScalerSnapshot{
				UseLog: scaler.UseLog,
				MinMax: &MinMaxScaler{
					Min: scaler.MinMax.Min,
					Max: scaler.MinMax.Max,
				},
				Robust: &RobustScaler{
					Median: scaler.Robust.Median,
					Q1:     scaler.Robust.Q1,
					Q3:     scaler.Robust.Q3,
					IQR:    scaler.Robust.IQR,
				},
			},
		})
	}
	return result
}

// buildScalerMapFromSnapshot 根据快照恢复缩放器映射。
func buildScalerMapFromSnapshot(entryList []scalerEntrySnapshot) map[int32]*AutoScaler {
	result := make(map[int32]*AutoScaler, len(entryList))
	for _, item := range entryList {
		if item.Scaler == nil {
			continue
		}
		minMax := MinMaxScaler{}
		if item.Scaler.MinMax != nil {
			minMax = MinMaxScaler{
				Min: item.Scaler.MinMax.Min,
				Max: item.Scaler.MinMax.Max,
			}
		}
		robust := RobustScaler{}
		if item.Scaler.Robust != nil {
			robust = RobustScaler{
				Median: item.Scaler.Robust.Median,
				Q1:     item.Scaler.Robust.Q1,
				Q3:     item.Scaler.Robust.Q3,
				IQR:    item.Scaler.Robust.IQR,
			}
		}
		result[item.FeatureIndex] = &AutoScaler{
			UseLog: item.Scaler.UseLog,
			MinMax: minMax,
			Robust: robust,
		}
	}
	return result
}

// cloneLayerList 复制层切片，避免导出阶段直接复用原切片。
func cloneLayerList(layerList []nn.Layer) []nn.Layer {
	if len(layerList) == 0 {
		return []nn.Layer{}
	}
	result := make([]nn.Layer, len(layerList))
	copy(result, layerList)
	return result
}

// exportLinearLayerSnapshotList 导出线性层快照列表。
func exportLinearLayerSnapshotList(layerList []nn.Layer) ([]*linearLayerSnapshot, error) {
	result := make([]*linearLayerSnapshot, 0, len(layerList))
	for _, item := range layerList {
		snapshot, err := exportLinearLayerSnapshot(item)
		if err != nil {
			return nil, err
		}
		result = append(result, snapshot)
	}
	return result, nil
}

// exportAttentionSnapshotList 导出注意力层快照列表。
func exportAttentionSnapshotList(layerList []nn.Layer) ([]*attentionLayerSnapshot, error) {
	result := make([]*attentionLayerSnapshot, 0, len(layerList))
	for _, item := range layerList {
		snapshot, err := exportAttentionLayerSnapshot(item)
		if err != nil {
			return nil, err
		}
		result = append(result, snapshot)
	}
	return result, nil
}

// exportLinearLayerSnapshot 导出单个线性层快照。
func exportLinearLayerSnapshot(layer nn.Layer) (*linearLayerSnapshot, error) {
	if layer == nil {
		return nil, nil
	}
	linearLayer, ok := layer.(*nn.LinearLayer)
	if !ok || linearLayer == nil || linearLayer.W == nil || linearLayer.B == nil {
		return nil, fmt.Errorf("afm linear layer is invalid")
	}
	return &linearLayerSnapshot{
		Weight: exportTensorSnapshot(cloneTensor(linearLayer.W)),
		Bias:   exportTensorSnapshot(cloneTensor(linearLayer.B)),
	}, nil
}

// exportAttentionLayerSnapshot 导出单个注意力层快照。
func exportAttentionLayerSnapshot(layer nn.Layer) (*attentionLayerSnapshot, error) {
	if layer == nil {
		return nil, nil
	}
	attentionLayer, ok := layer.(*nn.Attention)
	if !ok || attentionLayer == nil || attentionLayer.H == nil {
		return nil, fmt.Errorf("afm attention layer is invalid")
	}
	projectionSnapshot, err := exportLinearLayerSnapshot(attentionLayer.W)
	if err != nil {
		return nil, err
	}
	return &attentionLayerSnapshot{
		Projection: projectionSnapshot,
		H:          exportTensorSnapshot(cloneTensor(attentionLayer.H)),
	}, nil
}

// buildLinearLayerListFromSnapshot 根据快照恢复线性层列表。
func buildLinearLayerListFromSnapshot(snapshotList []*linearLayerSnapshot) ([]nn.Layer, error) {
	result := make([]nn.Layer, 0, len(snapshotList))
	for _, item := range snapshotList {
		layer, err := buildLinearLayerFromSnapshot(item)
		if err != nil {
			return nil, err
		}
		result = append(result, layer)
	}
	return result, nil
}

// buildAttentionLayerListFromSnapshot 根据快照恢复注意力层列表。
func buildAttentionLayerListFromSnapshot(snapshotList []*attentionLayerSnapshot, embeddingDim []int) ([]nn.Layer, error) {
	if len(snapshotList) == 0 {
		return []nn.Layer{}, nil
	}
	result := make([]nn.Layer, 0, len(snapshotList))
	for index, item := range snapshotList {
		dim := 0
		if index < len(embeddingDim) {
			dim = embeddingDim[index]
		}
		layer, err := buildAttentionLayerFromSnapshot(item, dim)
		if err != nil {
			return nil, err
		}
		result = append(result, layer)
	}
	return result, nil
}

// buildLinearLayerFromSnapshot 根据快照恢复线性层。
func buildLinearLayerFromSnapshot(snapshot *linearLayerSnapshot) (nn.Layer, error) {
	if snapshot == nil {
		return nil, nil
	}
	weightTensor, err := buildTensorFromSnapshot(snapshot.Weight)
	if err != nil {
		return nil, err
	}
	biasTensor, err := buildTensorFromSnapshot(snapshot.Bias)
	if err != nil {
		return nil, err
	}
	return &nn.LinearLayer{
		W: weightTensor,
		B: biasTensor,
	}, nil
}

// buildAttentionLayerFromSnapshot 根据快照恢复注意力层。
func buildAttentionLayerFromSnapshot(snapshot *attentionLayerSnapshot, dim int) (nn.Layer, error) {
	if snapshot == nil {
		return nil, nil
	}
	if dim <= 0 {
		return nil, fmt.Errorf("afm attention dim is invalid")
	}
	projectionLayer, err := buildLinearLayerFromSnapshot(snapshot.Projection)
	if err != nil {
		return nil, err
	}
	hTensor, err := buildTensorFromSnapshot(snapshot.H)
	if err != nil {
		return nil, err
	}
	hShape := hTensor.Shape()
	if len(hShape) != 2 {
		return nil, fmt.Errorf("afm attention tensor shape is invalid")
	}
	attention := nn.NewAttention(dim, hShape[0])
	attention.W = projectionLayer
	attention.H = hTensor
	return attention, nil
}

// String 返回缩放器条目的简要文本。
func (s scalerEntrySnapshot) String() string {
	return strconv.FormatInt(int64(s.FeatureIndex), 10)
}

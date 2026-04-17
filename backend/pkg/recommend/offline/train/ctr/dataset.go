package ctr

import (
	"math/rand"
	"strings"
)

// Label 表示一个可编码的稀疏或数值特征。
type Label struct {
	Name  string
	Value float32
}

// Embedding 表示一组命名向量特征。
type Embedding struct {
	Name  string
	Value []float32
}

// Sample 表示一条 CTR 训练或推理样本。
type Sample struct {
	UserId        string
	ItemId        string
	UserLabels    []Label
	ItemLabels    []Label
	ContextLabels []Label
	Embeddings    []Embedding
	Target        float32
}

// encodedRow 表示已经编码完成的一条样本。
type encodedRow struct {
	indices    []int32
	values     []float32
	embeddings [][]float32
	target     float32
}

// Dataset 表示 AFM 训练所需的数据集。
type Dataset struct {
	Index         *UnifiedMapIndex
	rows          []encodedRow
	embeddingDim  []int
	PositiveCount int
	NegativeCount int
}

// BuildDataset 根据样本列表构建可训练数据集。
func BuildDataset(sampleList []Sample) *Dataset {
	index := newUnifiedMapIndex()
	for _, sample := range sampleList {
		if strings.TrimSpace(sample.UserId) != "" {
			index.AddUser(strings.TrimSpace(sample.UserId))
		}
		if strings.TrimSpace(sample.ItemId) != "" {
			index.AddItem(strings.TrimSpace(sample.ItemId))
		}
		for _, item := range sample.UserLabels {
			if strings.TrimSpace(item.Name) == "" {
				continue
			}
			index.AddUserLabel(strings.TrimSpace(item.Name))
		}
		for _, item := range sample.ItemLabels {
			if strings.TrimSpace(item.Name) == "" {
				continue
			}
			index.AddItemLabel(strings.TrimSpace(item.Name))
		}
		for _, item := range sample.ContextLabels {
			if strings.TrimSpace(item.Name) == "" {
				continue
			}
			index.AddContextLabel(strings.TrimSpace(item.Name))
		}
		for _, item := range sample.Embeddings {
			if strings.TrimSpace(item.Name) == "" || len(item.Value) == 0 {
				continue
			}
			index.AddEmbedding(strings.TrimSpace(item.Name))
		}
	}

	result := &Dataset{
		Index:        index,
		rows:         make([]encodedRow, 0, len(sampleList)),
		embeddingDim: make([]int, int(index.CountEmbeddings())),
	}
	for _, sample := range sampleList {
		row := result.encodeSample(sample)
		// 无法编码用户或商品时，不纳入训练数据集。
		if len(row.indices) == 0 {
			continue
		}
		result.rows = append(result.rows, row)
		updateEmbeddingDimensions(result.embeddingDim, row.embeddings)
		if row.target > 0 {
			result.PositiveCount++
		} else {
			result.NegativeCount++
		}
	}
	return result
}

// Split 按比例随机切分训练集与测试集。
func (d *Dataset) Split(ratio float32, seed int64) (*Dataset, *Dataset) {
	trainSet := &Dataset{
		Index:        d.Index,
		rows:         make([]encodedRow, 0, len(d.rows)),
		embeddingDim: append([]int{}, d.embeddingDim...),
	}
	testSet := &Dataset{
		Index:        d.Index,
		rows:         make([]encodedRow, 0, len(d.rows)),
		embeddingDim: append([]int{}, d.embeddingDim...),
	}
	if d == nil || len(d.rows) == 0 || ratio <= 0 {
		trainSet.rows = append(trainSet.rows, d.rows...)
		trainSet.PositiveCount = d.PositiveCount
		trainSet.NegativeCount = d.NegativeCount
		return trainSet, testSet
	}

	rng := rand.New(rand.NewSource(seed))
	testSize := int(float32(len(d.rows)) * ratio)
	selected := make(map[int]struct{}, testSize)
	for len(selected) < testSize {
		selected[rng.Intn(len(d.rows))] = struct{}{}
	}
	for index, row := range d.rows {
		_, inTest := selected[index]
		if inTest {
			testSet.rows = append(testSet.rows, row)
			if row.target > 0 {
				testSet.PositiveCount++
			} else {
				testSet.NegativeCount++
			}
			continue
		}
		trainSet.rows = append(trainSet.rows, row)
		if row.target > 0 {
			trainSet.PositiveCount++
		} else {
			trainSet.NegativeCount++
		}
	}
	return trainSet, testSet
}

// Count 返回样本数量。
func (d *Dataset) Count() int {
	if d == nil {
		return 0
	}
	return len(d.rows)
}

// CountUsers 返回用户数量。
func (d *Dataset) CountUsers() int {
	if d == nil || d.Index == nil {
		return 0
	}
	return int(d.Index.CountUsers())
}

// CountItems 返回商品数量。
func (d *Dataset) CountItems() int {
	if d == nil || d.Index == nil {
		return 0
	}
	return int(d.Index.CountItems())
}

// Get 返回指定位置的编码样本。
func (d *Dataset) Get(index int) ([]int32, []float32, float32) {
	row := d.rows[index]
	return row.indices, row.values, row.target
}

// GetEmbeddingDim 返回各命名向量的维度列表。
func (d *Dataset) GetEmbeddingDim() []int {
	if d == nil {
		return []int{}
	}
	return append([]int{}, d.embeddingDim...)
}

// encodeSample 将原始样本编码成统一特征空间。
func (d *Dataset) encodeSample(sample Sample) encodedRow {
	indices := make([]int32, 0, 2+len(sample.UserLabels)+len(sample.ItemLabels)+len(sample.ContextLabels))
	values := make([]float32, 0, cap(indices))
	embeddings := make([][]float32, int(d.Index.CountEmbeddings()))

	if userIndex := d.Index.EncodeUser(strings.TrimSpace(sample.UserId)); userIndex != sparseNotFound {
		indices = append(indices, userIndex)
		values = append(values, 1)
	}
	if itemIndex := d.Index.EncodeItem(strings.TrimSpace(sample.ItemId)); itemIndex != sparseNotFound {
		indices = append(indices, itemIndex)
		values = append(values, 1)
	}
	appendEncodedLabels(indices, values, sample.UserLabels, d.Index.EncodeUserLabel, &indices, &values)
	appendEncodedLabels(indices, values, sample.ItemLabels, d.Index.EncodeItemLabel, &indices, &values)
	appendEncodedLabels(indices, values, sample.ContextLabels, d.Index.EncodeContextLabel, &indices, &values)
	for _, item := range sample.Embeddings {
		name := strings.TrimSpace(item.Name)
		if name == "" || len(item.Value) == 0 {
			continue
		}
		embeddingIndex := d.Index.EncodeEmbedding(name)
		if embeddingIndex == sparseNotFound {
			continue
		}
		embeddings[embeddingIndex] = append([]float32{}, item.Value...)
	}
	return encodedRow{
		indices:    indices,
		values:     values,
		embeddings: embeddings,
		target:     sample.Target,
	}
}

// appendEncodedLabels 把特征列表编码并追加到样本中。
func appendEncodedLabels(_ []int32, _ []float32, labels []Label, encoder func(string) int32, indices *[]int32, values *[]float32) {
	for _, item := range labels {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		index := encoder(name)
		if index == sparseNotFound {
			continue
		}
		value := item.Value
		// 未显式指定数值时，统一回退为 one-hot。
		if value == 0 {
			value = 1
		}
		*indices = append(*indices, index)
		*values = append(*values, value)
	}
}

// updateEmbeddingDimensions 根据样本中的向量更新各命名向量的最大维度。
func updateEmbeddingDimensions(embeddingDim []int, embeddings [][]float32) {
	for index, value := range embeddings {
		if len(value) > embeddingDim[index] {
			embeddingDim[index] = len(value)
		}
	}
}

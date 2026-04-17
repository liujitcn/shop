package ctr

// sparseNotFound 表示稀疏特征尚未被索引。
const sparseNotFound = int32(-1)

// sparseIndex 负责维护字符串到稠密编号的映射。
type sparseIndex struct {
	numbers map[string]int32
	names   []string
}

// newSparseIndex 创建一个新的稀疏索引。
func newSparseIndex() *sparseIndex {
	return &sparseIndex{
		numbers: make(map[string]int32),
		names:   make([]string, 0),
	}
}

// add 为名称分配稠密编号。
func (i *sparseIndex) add(name string) {
	if name == "" {
		return
	}
	if _, ok := i.numbers[name]; ok {
		return
	}
	i.numbers[name] = int32(len(i.names))
	i.names = append(i.names, name)
}

// encode 返回名称对应的稠密编号。
func (i *sparseIndex) encode(name string) int32 {
	if name == "" {
		return sparseNotFound
	}
	value, ok := i.numbers[name]
	if !ok {
		return sparseNotFound
	}
	return value
}

// len 返回当前索引容量。
func (i *sparseIndex) len() int32 {
	if i == nil {
		return 0
	}
	return int32(len(i.names))
}

// UnifiedMapIndex 维护用户、商品和特征在同一编码空间的编号。
type UnifiedMapIndex struct {
	userIndex      *sparseIndex
	itemIndex      *sparseIndex
	userLabelIndex *sparseIndex
	itemLabelIndex *sparseIndex
	ctxLabelIndex  *sparseIndex
	embeddingIndex *sparseIndex
}

// newUnifiedMapIndex 创建统一索引。
func newUnifiedMapIndex() *UnifiedMapIndex {
	return &UnifiedMapIndex{
		userIndex:      newSparseIndex(),
		itemIndex:      newSparseIndex(),
		userLabelIndex: newSparseIndex(),
		itemLabelIndex: newSparseIndex(),
		ctxLabelIndex:  newSparseIndex(),
		embeddingIndex: newSparseIndex(),
	}
}

// AddUser 注册用户编号。
func (i *UnifiedMapIndex) AddUser(userId string) {
	i.userIndex.add(userId)
}

// AddItem 注册商品编号。
func (i *UnifiedMapIndex) AddItem(itemId string) {
	i.itemIndex.add(itemId)
}

// AddUserLabel 注册用户特征名称。
func (i *UnifiedMapIndex) AddUserLabel(name string) {
	i.userLabelIndex.add(name)
}

// AddItemLabel 注册商品特征名称。
func (i *UnifiedMapIndex) AddItemLabel(name string) {
	i.itemLabelIndex.add(name)
}

// AddContextLabel 注册上下文特征名称。
func (i *UnifiedMapIndex) AddContextLabel(name string) {
	i.ctxLabelIndex.add(name)
}

// AddEmbedding 注册向量特征名称。
func (i *UnifiedMapIndex) AddEmbedding(name string) {
	i.embeddingIndex.add(name)
}

// EncodeUser 返回用户编号。
func (i *UnifiedMapIndex) EncodeUser(userId string) int32 {
	return i.userIndex.encode(userId)
}

// EncodeItem 返回商品编号。
func (i *UnifiedMapIndex) EncodeItem(itemId string) int32 {
	value := i.itemIndex.encode(itemId)
	if value == sparseNotFound {
		return value
	}
	return i.userIndex.len() + value
}

// EncodeUserLabel 返回用户特征编号。
func (i *UnifiedMapIndex) EncodeUserLabel(name string) int32 {
	value := i.userLabelIndex.encode(name)
	if value == sparseNotFound {
		return value
	}
	return i.userIndex.len() + i.itemIndex.len() + value
}

// EncodeItemLabel 返回商品特征编号。
func (i *UnifiedMapIndex) EncodeItemLabel(name string) int32 {
	value := i.itemLabelIndex.encode(name)
	if value == sparseNotFound {
		return value
	}
	return i.userIndex.len() + i.itemIndex.len() + i.userLabelIndex.len() + value
}

// EncodeContextLabel 返回上下文特征编号。
func (i *UnifiedMapIndex) EncodeContextLabel(name string) int32 {
	value := i.ctxLabelIndex.encode(name)
	if value == sparseNotFound {
		return value
	}
	return i.userIndex.len() + i.itemIndex.len() + i.userLabelIndex.len() + i.itemLabelIndex.len() + value
}

// EncodeEmbedding 返回向量特征编号。
func (i *UnifiedMapIndex) EncodeEmbedding(name string) int32 {
	return i.embeddingIndex.encode(name)
}

// Len 返回统一编码空间大小。
func (i *UnifiedMapIndex) Len() int32 {
	return i.userIndex.len() + i.itemIndex.len() + i.userLabelIndex.len() + i.itemLabelIndex.len() + i.ctxLabelIndex.len()
}

// CountUsers 返回用户数量。
func (i *UnifiedMapIndex) CountUsers() int32 {
	return i.userIndex.len()
}

// CountItems 返回商品数量。
func (i *UnifiedMapIndex) CountItems() int32 {
	return i.itemIndex.len()
}

// CountUserLabels 返回用户特征数量。
func (i *UnifiedMapIndex) CountUserLabels() int32 {
	return i.userLabelIndex.len()
}

// CountItemLabels 返回商品特征数量。
func (i *UnifiedMapIndex) CountItemLabels() int32 {
	return i.itemLabelIndex.len()
}

// CountContextLabels 返回上下文特征数量。
func (i *UnifiedMapIndex) CountContextLabels() int32 {
	return i.ctxLabelIndex.len()
}

// CountEmbeddings 返回向量特征数量。
func (i *UnifiedMapIndex) CountEmbeddings() int32 {
	return i.embeddingIndex.len()
}

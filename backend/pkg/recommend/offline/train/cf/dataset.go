package cf

import (
	"math"
	"sort"
	"strings"

	"shop/pkg/recommend/offline/train/util"
)

// Dataset 表示 BPR 训练与评估使用的聚合正反馈数据集。
type Dataset struct {
	interactionList []Interaction
	userIds         []string
	itemIds         []string
	userItemWeight  map[string]map[string]int
}

// BuildDataset 把原始行为聚合成 user-item 级数据集。
func BuildDataset(interactionList []Interaction) *Dataset {
	userItemWeight := make(map[string]map[string]int)
	itemIdSet := make(map[string]struct{})
	for _, item := range interactionList {
		userId := strings.TrimSpace(item.UserId)
		itemId := strings.TrimSpace(item.ItemId)
		// 主体或商品为空时，当前行为不能进入训练快照。
		if userId == "" || itemId == "" {
			continue
		}
		weight := item.Weight
		// 权重非法时，统一按 1 条正反馈处理。
		if weight <= 0 {
			weight = 1
		}
		// 当前用户第一次出现时，先初始化其聚合桶。
		if _, ok := userItemWeight[userId]; !ok {
			userItemWeight[userId] = make(map[string]int)
		}
		userItemWeight[userId][itemId] += weight
		itemIdSet[itemId] = struct{}{}
	}
	userIds := sortedStringKeys(userItemWeight)
	itemIds := sortedStringKeys(itemIdSet)
	return &Dataset{
		interactionList: buildInteractionList(userItemWeight),
		userIds:         userIds,
		itemIds:         itemIds,
		userItemWeight:  userItemWeight,
	}
}

// Count 返回当前数据集的唯一正反馈条数。
func (d *Dataset) Count() int {
	if d == nil {
		return 0
	}
	return len(d.interactionList)
}

// CountUsers 返回当前数据集的用户数。
func (d *Dataset) CountUsers() int {
	if d == nil {
		return 0
	}
	return len(d.userIds)
}

// CountItems 返回当前数据集的商品数。
func (d *Dataset) CountItems() int {
	if d == nil {
		return 0
	}
	return len(d.itemIds)
}

// Interactions 返回当前数据集的聚合交互列表副本。
func (d *Dataset) Interactions() []Interaction {
	if d == nil {
		return []Interaction{}
	}
	result := make([]Interaction, len(d.interactionList))
	copy(result, d.interactionList)
	return result
}

// UserIds 返回当前数据集的用户列表副本。
func (d *Dataset) UserIds() []string {
	if d == nil {
		return []string{}
	}
	result := make([]string, len(d.userIds))
	copy(result, d.userIds)
	return result
}

// ItemIds 返回当前数据集的商品列表副本。
func (d *Dataset) ItemIds() []string {
	if d == nil {
		return []string{}
	}
	result := make([]string, len(d.itemIds))
	copy(result, d.itemIds)
	return result
}

// UserItemSet 返回指定用户的正反馈商品集合副本。
func (d *Dataset) UserItemSet(userId string) map[string]struct{} {
	result := make(map[string]struct{})
	// 数据集为空时，直接返回空集合。
	if d == nil {
		return result
	}
	itemWeightMap, ok := d.userItemWeight[strings.TrimSpace(userId)]
	// 当前用户没有正反馈时，返回空集合。
	if !ok {
		return result
	}
	for itemId := range itemWeightMap {
		result[itemId] = struct{}{}
	}
	return result
}

// UserItems 返回指定用户的正反馈商品列表。
func (d *Dataset) UserItems(userId string) []string {
	// 数据集为空时，直接返回空列表。
	if d == nil {
		return []string{}
	}
	itemWeightMap, ok := d.userItemWeight[strings.TrimSpace(userId)]
	// 当前用户没有正反馈时，返回空列表。
	if !ok {
		return []string{}
	}
	return sortedStringKeys(itemWeightMap)
}

// Split 按用户粒度把数据集拆分成训练集和验证集。
func (d *Dataset) Split(testRatio float32, seed int64) (*Dataset, *Dataset) {
	// 比例非法或数据为空时，直接回退为“全量训练、空验证”。
	if d == nil || d.Count() == 0 || testRatio <= 0 || testRatio >= 1 {
		return BuildDataset(d.Interactions()), BuildDataset(nil)
	}

	rng := util.NewRandomGenerator(seed)
	trainUserItemWeight := make(map[string]map[string]int)
	testUserItemWeight := make(map[string]map[string]int)
	for _, userId := range d.userIds {
		itemWeightMap := d.userItemWeight[userId]
		itemIds := sortedStringKeys(itemWeightMap)
		// 只有 1 个正反馈商品的用户无法安全切验证集，全部保留在训练集。
		if len(itemIds) < 2 {
			for _, itemId := range itemIds {
				appendInteractionWeight(trainUserItemWeight, userId, itemId, itemWeightMap[itemId])
			}
			continue
		}

		shuffledItemIds := append([]string{}, itemIds...)
		shuffleStrings(rng, shuffledItemIds)
		testCount := int(math.Round(float64(len(shuffledItemIds)) * float64(testRatio)))
		// 验证集至少保留 1 条，同时不能把该用户全部正反馈都切走。
		if testCount <= 0 {
			testCount = 1
		}
		if testCount >= len(shuffledItemIds) {
			testCount = len(shuffledItemIds) - 1
		}

		testItemIdSet := make(map[string]struct{}, testCount)
		for _, itemId := range shuffledItemIds[:testCount] {
			testItemIdSet[itemId] = struct{}{}
		}
		for _, itemId := range itemIds {
			// 命中验证集的商品写入 test，其余保留在 train。
			if _, ok := testItemIdSet[itemId]; ok {
				appendInteractionWeight(testUserItemWeight, userId, itemId, itemWeightMap[itemId])
				continue
			}
			appendInteractionWeight(trainUserItemWeight, userId, itemId, itemWeightMap[itemId])
		}
	}

	// 某些商品若只落到验证集，模型将无法学习其向量，需要回退到训练集。
	moveOrphanItemsToTrain(trainUserItemWeight, testUserItemWeight)
	return buildDatasetFromWeightMap(trainUserItemWeight), buildDatasetFromWeightMap(testUserItemWeight)
}

// buildDatasetFromWeightMap 根据聚合权重映射生成数据集。
func buildDatasetFromWeightMap(userItemWeight map[string]map[string]int) *Dataset {
	// 权重映射为空时，返回结构完整的空数据集。
	if len(userItemWeight) == 0 {
		return &Dataset{
			interactionList: []Interaction{},
			userIds:         []string{},
			itemIds:         []string{},
			userItemWeight:  map[string]map[string]int{},
		}
	}
	itemIdSet := make(map[string]struct{})
	for _, itemWeightMap := range userItemWeight {
		for itemId := range itemWeightMap {
			itemIdSet[itemId] = struct{}{}
		}
	}
	return &Dataset{
		interactionList: buildInteractionList(userItemWeight),
		userIds:         sortedStringKeys(userItemWeight),
		itemIds:         sortedStringKeys(itemIdSet),
		userItemWeight:  cloneUserItemWeightMap(userItemWeight),
	}
}

// buildInteractionList 把聚合权重映射展开成稳定顺序的交互列表。
func buildInteractionList(userItemWeight map[string]map[string]int) []Interaction {
	result := make([]Interaction, 0)
	for _, userId := range sortedStringKeys(userItemWeight) {
		itemWeightMap := userItemWeight[userId]
		for _, itemId := range sortedStringKeys(itemWeightMap) {
			result = append(result, Interaction{
				UserId: userId,
				ItemId: itemId,
				Weight: itemWeightMap[itemId],
			})
		}
	}
	return result
}

// appendInteractionWeight 向 user-item 聚合映射追加一条正反馈权重。
func appendInteractionWeight(target map[string]map[string]int, userId string, itemId string, weight int) {
	// 当前用户第一次写入时，先创建其权重映射。
	if _, ok := target[userId]; !ok {
		target[userId] = make(map[string]int)
	}
	target[userId][itemId] += weight
}

// moveOrphanItemsToTrain 把只存在于验证集的商品回退到训练集。
func moveOrphanItemsToTrain(trainUserItemWeight map[string]map[string]int, testUserItemWeight map[string]map[string]int) {
	trainItemCount := make(map[string]int)
	for _, itemWeightMap := range trainUserItemWeight {
		for itemId := range itemWeightMap {
			trainItemCount[itemId]++
		}
	}
	for userId, itemWeightMap := range testUserItemWeight {
		for itemId, weight := range itemWeightMap {
			// 训练集里已经存在该商品时，当前验证条目可以继续保留。
			if trainItemCount[itemId] > 0 {
				continue
			}
			appendInteractionWeight(trainUserItemWeight, userId, itemId, weight)
			delete(itemWeightMap, itemId)
			trainItemCount[itemId]++
		}
		// 当前用户验证集已被清空时，删除空映射避免留下无效主体。
		if len(itemWeightMap) == 0 {
			delete(testUserItemWeight, userId)
		}
	}
}

// cloneUserItemWeightMap 深拷贝 user-item 权重映射。
func cloneUserItemWeightMap(source map[string]map[string]int) map[string]map[string]int {
	result := make(map[string]map[string]int, len(source))
	for userId, itemWeightMap := range source {
		result[userId] = make(map[string]int, len(itemWeightMap))
		for itemId, weight := range itemWeightMap {
			result[userId][itemId] = weight
		}
	}
	return result
}

// sortedStringKeys 返回字符串映射的有序键列表。
func sortedStringKeys[T any](source map[string]T) []string {
	result := make([]string, 0, len(source))
	for key := range source {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}

// shuffleStrings 使用给定随机源原地打乱字符串切片。
func shuffleStrings(rng util.RandomGenerator, list []string) {
	for index := len(list) - 1; index > 0; index-- {
		swapIndex := rng.Intn(index + 1)
		list[index], list[swapIndex] = list[swapIndex], list[index]
	}
}

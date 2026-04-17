package cf

import (
	"context"
	"math"
	"sort"
	"strings"

	"shop/pkg/recommend/offline/train/floats"
	"shop/pkg/recommend/offline/train/util"
)

// Interaction 表示一条隐式正反馈。
type Interaction struct {
	UserId string
	ItemId string
	Weight int
}

// Config 表示 BPR 训练参数。
type Config struct {
	BatchSize  int
	Backend    string
	Factors    int
	Epochs     int
	Learning   float32
	Reg        float32
	InitMean   float32
	InitStdDev float32
	Optimizer  string
	Seed       int64
}

const (
	// BackendNative 表示使用当前仓库内置训练核。
	BackendNative = "native"
	// BackendGoMLX 表示使用 gomlx/simplego 训练后端。
	BackendGoMLX = "gomlx"
)

// fillDefault 补齐未配置的 BPR 参数。
func (c Config) fillDefault() Config {
	if c.BatchSize <= 0 {
		c.BatchSize = 1024
	}
	if c.Factors <= 0 {
		c.Factors = 16
	}
	if c.Epochs <= 0 {
		c.Epochs = 100
	}
	if c.Learning <= 0 {
		c.Learning = 0.05
	}
	if c.Reg <= 0 {
		c.Reg = 0.01
	}
	if c.InitStdDev <= 0 {
		c.InitStdDev = 0.001
	}
	if c.Optimizer == "" {
		c.Optimizer = "sgd"
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

// Model 表示训练完成后的 BPR 模型。
type Model struct {
	config        Config
	userIndex     map[string]int
	itemIndex     map[string]int
	userIds       []string
	itemIds       []string
	userFactors   [][]float32
	itemFactors   [][]float32
	userFeedback  [][]int
	itemSetByUser []map[int]struct{}
}

// Fit 根据隐式反馈训练 BPR 模型。
func Fit(ctx context.Context, interactionList []Interaction, config Config) *Model {
	config = config.fillDefault()
	model := buildModel(interactionList, config)
	if model == nil || len(model.userIds) == 0 || len(model.itemIds) == 0 {
		return model
	}
	// gomlx 后端启用时，优先走 gomlx/simplego 训练链路。
	if config.NormalizeBackend() == BackendGoMLX {
		return fitWithGoMLX(ctx, model)
	}
	return fitWithNative(ctx, model)
}

// fitWithNative 使用当前仓库内置训练核训练 BPR。
func fitWithNative(ctx context.Context, model *Model) *Model {
	if model == nil {
		return nil
	}
	config := model.config
	rng := util.NewRandomGenerator(config.Seed)
	temp := make([]float32, config.Factors)
	userBuffer := make([]float32, config.Factors)
	positiveBuffer := make([]float32, config.Factors)
	negativeBuffer := make([]float32, config.Factors)
	feedbackCount := model.feedbackCount()
	if feedbackCount == 0 {
		return model
	}

	for epoch := 0; epoch < config.Epochs; epoch++ {
		for step := 0; step < feedbackCount; step++ {
			if ctx != nil && ctx.Err() != nil {
				return model
			}
			userIndex := rng.Intn(len(model.userIds))
			if len(model.userFeedback[userIndex]) == 0 {
				continue
			}
			positiveIndex := model.userFeedback[userIndex][rng.Intn(len(model.userFeedback[userIndex]))]
			negativeIndex := sampleNegativeIndex(rng, len(model.itemIds), model.itemSetByUser[userIndex])
			diff := dot(model.userFactors[userIndex], model.itemFactors[positiveIndex]) - dot(model.userFactors[userIndex], model.itemFactors[negativeIndex])
			grad := float32(math.Exp(float64(-diff)) / (1 + math.Exp(float64(-diff))))

			copy(userBuffer, model.userFactors[userIndex])
			copy(positiveBuffer, model.itemFactors[positiveIndex])
			copy(negativeBuffer, model.itemFactors[negativeIndex])

			floats.MulConstAddTo(userBuffer, grad, positiveBuffer, temp)
			floats.MulConstAdd(positiveBuffer, -config.Reg, temp)
			floats.MulConstAdd(temp, config.Learning, model.itemFactors[positiveIndex])

			floats.MulConstAddTo(userBuffer, -grad, negativeBuffer, temp)
			floats.MulConstAdd(negativeBuffer, -config.Reg, temp)
			floats.MulConstAdd(temp, config.Learning, model.itemFactors[negativeIndex])

			floats.SubTo(positiveBuffer, negativeBuffer, temp)
			floats.MulConst(temp, grad)
			floats.MulConstAdd(userBuffer, -config.Reg, temp)
			floats.MulConstAdd(temp, config.Learning, model.userFactors[userIndex])
		}
	}
	return model
}

// feedbackCount 返回训练阶段总采样步数。
func (m *Model) feedbackCount() int {
	if m == nil {
		return 0
	}
	feedbackCount := 0
	for _, list := range m.userFeedback {
		feedbackCount += len(list)
	}
	return feedbackCount
}

// Recommend 为指定用户输出 topN 商品及分数。
func (m *Model) Recommend(userId string, limit int, excludeItemIdSet map[string]struct{}) []ScoredItem {
	if m == nil || limit <= 0 {
		return []ScoredItem{}
	}
	userIndex, ok := m.userIndex[strings.TrimSpace(userId)]
	if !ok {
		return []ScoredItem{}
	}
	result := make([]ScoredItem, 0, len(m.itemIds))
	for itemIndex, itemId := range m.itemIds {
		if _, excluded := excludeItemIdSet[itemId]; excluded {
			continue
		}
		if _, interacted := m.itemSetByUser[userIndex][itemIndex]; interacted {
			continue
		}
		result = append(result, ScoredItem{
			ItemId: itemId,
			Score:  dot(m.userFactors[userIndex], m.itemFactors[itemIndex]),
		})
	}
	sort.SliceStable(result, func(i int, j int) bool {
		if result[i].Score == result[j].Score {
			return result[i].ItemId < result[j].ItemId
		}
		return result[i].Score > result[j].Score
	})
	if len(result) > limit {
		return result[:limit]
	}
	return result
}

// UserIds 返回当前模型包含的用户列表。
func (m *Model) UserIds() []string {
	if m == nil {
		return []string{}
	}
	result := make([]string, 0, len(m.userIds))
	result = append(result, m.userIds...)
	return result
}

// ItemIds 返回当前模型包含的商品列表。
func (m *Model) ItemIds() []string {
	if m == nil {
		return []string{}
	}
	result := make([]string, 0, len(m.itemIds))
	result = append(result, m.itemIds...)
	return result
}

// ScoredItem 表示带训练分值的商品。
type ScoredItem struct {
	ItemId string
	Score  float32
}

// Rank 在给定候选商品集合上输出按分值排序的 topN 结果。
func (m *Model) Rank(userId string, candidateItemIds []string, limit int) []ScoredItem {
	// 模型为空、窗口非法或候选为空时，不执行候选重排。
	if m == nil || limit <= 0 || len(candidateItemIds) == 0 {
		return []ScoredItem{}
	}
	userIndex, ok := m.userIndex[strings.TrimSpace(userId)]
	// 用户不在模型索引内时，当前轮次无法计算打分。
	if !ok {
		return []ScoredItem{}
	}
	result := make([]ScoredItem, 0, len(candidateItemIds))
	seenItemIdSet := make(map[string]struct{}, len(candidateItemIds))
	for _, itemId := range candidateItemIds {
		normalizedItemId := strings.TrimSpace(itemId)
		// 候选商品为空或当前轮次已经加入过时，不重复参与排序。
		if normalizedItemId == "" {
			continue
		}
		// 同一商品只保留一次，避免重复候选影响排序稳定性。
		if _, exists := seenItemIdSet[normalizedItemId]; exists {
			continue
		}
		itemIndex, itemExists := m.itemIndex[normalizedItemId]
		// 模型里不存在的商品无法计算分值，直接跳过。
		if !itemExists {
			continue
		}
		seenItemIdSet[normalizedItemId] = struct{}{}
		result = append(result, ScoredItem{
			ItemId: normalizedItemId,
			Score:  dot(m.userFactors[userIndex], m.itemFactors[itemIndex]),
		})
	}
	sort.SliceStable(result, func(i int, j int) bool {
		if result[i].Score == result[j].Score {
			return result[i].ItemId < result[j].ItemId
		}
		return result[i].Score > result[j].Score
	})
	if len(result) > limit {
		return result[:limit]
	}
	return result
}

// buildModel 构建 BPR 训练所需的索引和矩阵。
func buildModel(interactionList []Interaction, config Config) *Model {
	userIndex := make(map[string]int)
	itemIndex := make(map[string]int)
	userIds := make([]string, 0)
	itemIds := make([]string, 0)
	userFeedback := make([][]int, 0)
	itemSetByUser := make([]map[int]struct{}, 0)

	appendUser := func(userId string) int {
		if index, ok := userIndex[userId]; ok {
			return index
		}
		index := len(userIds)
		userIndex[userId] = index
		userIds = append(userIds, userId)
		userFeedback = append(userFeedback, []int{})
		itemSetByUser = append(itemSetByUser, make(map[int]struct{}))
		return index
	}
	appendItem := func(itemId string) int {
		if index, ok := itemIndex[itemId]; ok {
			return index
		}
		index := len(itemIds)
		itemIndex[itemId] = index
		itemIds = append(itemIds, itemId)
		return index
	}

	for _, interaction := range interactionList {
		userId := strings.TrimSpace(interaction.UserId)
		itemId := strings.TrimSpace(interaction.ItemId)
		if userId == "" || itemId == "" {
			continue
		}
		userIdIndex := appendUser(userId)
		itemIdIndex := appendItem(itemId)
		weight := interaction.Weight
		if weight <= 0 {
			weight = 1
		}
		for i := 0; i < weight; i++ {
			userFeedback[userIdIndex] = append(userFeedback[userIdIndex], itemIdIndex)
		}
		itemSetByUser[userIdIndex][itemIdIndex] = struct{}{}
	}
	if len(userIds) == 0 || len(itemIds) == 0 {
		return &Model{}
	}

	rng := util.NewRandomGenerator(config.Seed)
	return &Model{
		config:        config,
		userIndex:     userIndex,
		itemIndex:     itemIndex,
		userIds:       userIds,
		itemIds:       itemIds,
		userFactors:   rng.NormalMatrix(len(userIds), config.Factors, config.InitMean, config.InitStdDev),
		itemFactors:   rng.NormalMatrix(len(itemIds), config.Factors, config.InitMean, config.InitStdDev),
		userFeedback:  userFeedback,
		itemSetByUser: itemSetByUser,
	}
}

// sampleNegativeIndex 从未交互商品里采样一个负样本。
func sampleNegativeIndex(rng util.RandomGenerator, itemCount int, excluded map[int]struct{}) int {
	if len(excluded) >= itemCount {
		return 0
	}
	for {
		index := rng.Intn(itemCount)
		if _, ok := excluded[index]; ok {
			continue
		}
		return index
	}
}

// dot 计算两个向量的点积。
func dot(left []float32, right []float32) float32 {
	return floats.Dot(left, right)
}

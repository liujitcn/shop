package cf

import (
	"fmt"
	"sort"
)

// Snapshot 表示 BPR 模型可序列化快照。
type Snapshot struct {
	Config            Config              `json:"config"`            // 当前训练配置。
	UserIds           []string            `json:"userIds"`           // 用户编号列表。
	ItemIds           []string            `json:"itemIds"`           // 商品编号列表。
	UserFactors       [][]float32         `json:"userFactors"`       // 用户隐向量矩阵。
	ItemFactors       [][]float32         `json:"itemFactors"`       // 商品隐向量矩阵。
	SeenItemIdsByUser map[string][]string `json:"seenItemIdsByUser"` // 用户历史正反馈商品集合。
}

// ExportSnapshot 导出当前 BPR 模型快照。
func (m *Model) ExportSnapshot() (*Snapshot, error) {
	// 模型为空时，无法导出训练快照。
	if m == nil {
		return nil, fmt.Errorf("bpr model is nil")
	}
	if len(m.userIds) != len(m.userFactors) {
		return nil, fmt.Errorf("bpr user factor size mismatch")
	}
	if len(m.itemIds) != len(m.itemFactors) {
		return nil, fmt.Errorf("bpr item factor size mismatch")
	}
	seenItemIdsByUser := make(map[string][]string, len(m.userIds))
	for userIndex, userId := range m.userIds {
		itemIds := make([]string, 0, len(m.itemSetByUser[userIndex]))
		for itemIndex := range m.itemSetByUser[userIndex] {
			if itemIndex < 0 || itemIndex >= len(m.itemIds) {
				continue
			}
			itemIds = append(itemIds, m.itemIds[itemIndex])
		}
		sort.Strings(itemIds)
		seenItemIdsByUser[userId] = itemIds
	}
	return &Snapshot{
		Config:            m.config,
		UserIds:           append([]string{}, m.userIds...),
		ItemIds:           append([]string{}, m.itemIds...),
		UserFactors:       cloneMatrix32(m.userFactors),
		ItemFactors:       cloneMatrix32(m.itemFactors),
		SeenItemIdsByUser: seenItemIdsByUser,
	}, nil
}

// BuildModel 根据快照恢复 BPR 模型。
func (s *Snapshot) BuildModel() (*Model, error) {
	// 快照为空时，无法恢复模型。
	if s == nil {
		return nil, fmt.Errorf("bpr snapshot is nil")
	}
	if len(s.UserIds) != len(s.UserFactors) {
		return nil, fmt.Errorf("bpr snapshot user factor size mismatch")
	}
	if len(s.ItemIds) != len(s.ItemFactors) {
		return nil, fmt.Errorf("bpr snapshot item factor size mismatch")
	}

	userIndex := make(map[string]int, len(s.UserIds))
	userIds := append([]string{}, s.UserIds...)
	for index, userId := range userIds {
		userIndex[userId] = index
	}
	itemIndex := make(map[string]int, len(s.ItemIds))
	itemIds := append([]string{}, s.ItemIds...)
	for index, itemId := range itemIds {
		itemIndex[itemId] = index
	}

	userFeedback := make([][]int, len(userIds))
	itemSetByUser := make([]map[int]struct{}, len(userIds))
	for userPosition, userId := range userIds {
		itemSetByUser[userPosition] = make(map[int]struct{})
		itemIdsByUser := append([]string{}, s.SeenItemIdsByUser[userId]...)
		sort.Strings(itemIdsByUser)
		for _, itemId := range itemIdsByUser {
			itemPosition, ok := itemIndex[itemId]
			// 快照里引用了不存在的商品时，当前条目直接跳过。
			if !ok {
				continue
			}
			itemSetByUser[userPosition][itemPosition] = struct{}{}
			// 推理恢复只需要知道“见过哪些商品”，这里按唯一商品回放一次即可。
			userFeedback[userPosition] = append(userFeedback[userPosition], itemPosition)
		}
	}
	return &Model{
		config:        s.Config.fillDefault(),
		userIndex:     userIndex,
		itemIndex:     itemIndex,
		userIds:       userIds,
		itemIds:       itemIds,
		userFactors:   cloneMatrix32(s.UserFactors),
		itemFactors:   cloneMatrix32(s.ItemFactors),
		userFeedback:  userFeedback,
		itemSetByUser: itemSetByUser,
	}, nil
}

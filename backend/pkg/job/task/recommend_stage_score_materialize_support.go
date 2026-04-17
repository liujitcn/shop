package task

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"time"

	"shop/pkg/errorsx"
	recommendCache "shop/pkg/recommend/cache"
)

// recommendStageScoreSnapshot 表示阶段分数快照文件。
type recommendStageScoreSnapshot struct {
	Entries []*recommendStageScoreEntry `json:"entries"` // 阶段分数条目列表。
}

// recommendStageScoreEntry 表示一个阶段分数子集合快照。
type recommendStageScoreEntry struct {
	Scene       int32                  `json:"scene"`        // 推荐场景。
	ActorType   int32                  `json:"actor_type"`   // 推荐主体类型。
	ActorId     int64                  `json:"actor_id"`     // 推荐主体编号。
	RequestHash string                 `json:"request_hash"` // 请求哈希，仅 LLM 二次重排使用。
	Documents   []recommendCache.Score `json:"documents"`    // 当前子集合的分数文档。
}

// loadRecommendStageScoreEntryList 读取阶段分数快照条目列表。
func loadRecommendStageScoreEntryList(path string) ([]*recommendStageScoreEntry, error) {
	trimmedPath := strings.TrimSpace(path)
	// 快照路径为空时，无法定位离线产物文件。
	if trimmedPath == "" {
		return nil, errorsx.InvalidArgument("path 不能为空")
	}

	fileByte, err := os.ReadFile(trimmedPath)
	if err != nil {
		return nil, err
	}

	snapshot := &recommendStageScoreSnapshot{}
	err = json.Unmarshal(fileByte, snapshot)
	// 外层对象解析失败时，再兼容直接传条目数组的快照格式。
	if err == nil && len(snapshot.Entries) > 0 {
		return snapshot.Entries, nil
	}

	entryList := make([]*recommendStageScoreEntry, 0)
	err = json.Unmarshal(fileByte, &entryList)
	if err != nil {
		return nil, err
	}
	return entryList, nil
}

// parseRecommendMaterializeRequiredVersionArg 解析必填的缓存版本参数。
func parseRecommendMaterializeRequiredVersionArg(value string) (string, error) {
	version := strings.TrimSpace(value)
	// 阶段分数快照发布必须显式指定目标缓存版本，避免误写到默认空间。
	if version == "" {
		return "", errorsx.InvalidArgument("version 不能为空")
	}
	return recommendCache.NormalizeVersion(version), nil
}

// parseRecommendMaterializeBoolArg 解析布尔类型任务参数。
func parseRecommendMaterializeBoolArg(value string, defaultValue bool) (bool, error) {
	trimmedValue := strings.TrimSpace(value)
	// 未传参数时，回退到调用方给出的默认值。
	if trimmedValue == "" {
		return defaultValue, nil
	}

	parsed, err := strconv.ParseBool(trimmedValue)
	if err != nil {
		return false, errorsx.InvalidArgument("布尔参数格式错误")
	}
	return parsed, nil
}

// countRecommendStageScoreDocuments 统计阶段分数快照中的文档数量。
func countRecommendStageScoreDocuments(entryList []*recommendStageScoreEntry) int {
	total := 0
	for _, entry := range entryList {
		// 空条目不参与文档数量统计。
		if entry == nil {
			continue
		}
		total += len(entry.Documents)
	}
	return total
}

// normalizeRecommendStageDocuments 规整阶段分数文档并按限制截断。
func normalizeRecommendStageDocuments(documents []recommendCache.Score, limit int64, fallbackTime time.Time) []recommendCache.Score {
	result := make([]recommendCache.Score, 0, len(documents))
	documentMap := make(map[string]recommendCache.Score, len(documents))
	for _, item := range documents {
		documentId := strings.TrimSpace(item.Id)
		// 文档主键为空时，不继续写入阶段分数缓存。
		if documentId == "" {
			continue
		}
		item.Id = documentId
		// 输入文档未提供时间戳时，统一回退到当前发布时间。
		if item.Timestamp.IsZero() {
			item.Timestamp = fallbackTime
		}
		existing, exists := documentMap[item.Id]
		// 同一商品在快照里重复出现时，保留分数更高的一条。
		if exists && existing.Score >= item.Score {
			continue
		}
		documentMap[item.Id] = item
	}

	for _, item := range documentMap {
		result = append(result, item)
	}
	recommendCache.SortDocuments(result)
	// 发布上限生效时，只保留当前子集合的 TopN 文档。
	if limit > 0 && int64(len(result)) > limit {
		return result[:limit]
	}
	return result
}

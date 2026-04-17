package task

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	recommendCache "shop/pkg/recommend/cache"
)

var recommendTrainArtifactRootDirResolver = resolveRecommendTrainArtifactRootDir

// recommendTrainArtifactManifest 表示一次训练产物的说明文件。
type recommendTrainArtifactManifest struct {
	Task                string             `json:"task"`                          // 任务名。
	ModelType           string             `json:"modelType"`                     // 模型类型。
	CreatedAt           string             `json:"createdAt"`                     // 产物生成时间。
	Version             string             `json:"version,omitempty"`             // 单版本任务版本号。
	Versions            []string           `json:"versions,omitempty"`            // 多版本任务版本列表。
	Backend             string             `json:"backend"`                       // 训练后端。
	TargetMetric        string             `json:"targetMetric,omitempty"`        // 调参目标指标。
	BestValue           float64            `json:"bestValue,omitempty"`           // 最优目标值。
	Score               map[string]float64 `json:"score,omitempty"`               // 验证集指标。
	Counts              map[string]int     `json:"counts,omitempty"`              // 输入与产出规模。
	ModelFile           string             `json:"modelFile,omitempty"`           // 模型快照文件名。
	PublishSnapshotFile string             `json:"publishSnapshotFile,omitempty"` // 发布快照文件名。
}

// recommendCollaborativeFilteringSnapshot 表示协同过滤发布快照文件。
type recommendCollaborativeFilteringSnapshot struct {
	Entries []*recommendCollaborativeFilteringEntry `json:"entries"` // 用户维度快照条目。
}

// recommendCollaborativeFilteringEntry 表示协同过滤单个用户的发布快照。
type recommendCollaborativeFilteringEntry struct {
	UserId    int64                  `json:"userId"`    // 用户编号。
	Documents []recommendCache.Score `json:"documents"` // 当前用户推荐文档。
}

// resolveRecommendTrainArtifactRootDir 解析训练产物根目录。
func resolveRecommendTrainArtifactRootDir() string {
	_, currentFilePath, _, ok := runtime.Caller(0)
	if !ok {
		return filepath.Clean("data/recommend/train")
	}
	currentDir := filepath.Dir(currentFilePath)
	return filepath.Clean(filepath.Join(currentDir, "..", "..", "..", "data", "recommend", "train"))
}

// newRecommendTrainArtifactRunDir 创建一次训练运行的产物目录。
func newRecommendTrainArtifactRunDir(taskName string, version string, createdAt time.Time) (string, error) {
	rootDir := recommendTrainArtifactRootDirResolver()
	if rootDir == "" {
		return "", fmt.Errorf("recommend artifact root dir is empty")
	}
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	pathParts := []string{rootDir, sanitizeRecommendArtifactPathPart(taskName)}
	// 当前任务提供了版本号时，再继续按版本拆目录，便于历史回溯。
	if strings.TrimSpace(version) != "" {
		pathParts = append(pathParts, sanitizeRecommendArtifactPathPart(version))
	}
	pathParts = append(pathParts, createdAt.Format("20060102_150405_000000000"))
	runDir := filepath.Join(pathParts...)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return "", err
	}
	return runDir, nil
}

// sanitizeRecommendArtifactPathPart 规整产物路径片段。
func sanitizeRecommendArtifactPathPart(value string) string {
	normalizedValue := strings.TrimSpace(strings.ToLower(value))
	if normalizedValue == "" {
		return "default"
	}
	return strings.Map(func(item rune) rune {
		switch {
		case item >= 'a' && item <= 'z':
			return item
		case item >= '0' && item <= '9':
			return item
		case item == '-' || item == '_':
			return item
		default:
			return '_'
		}
	}, normalizedValue)
}

// writeRecommendTrainArtifactJSON 将任意训练产物写成 JSON 文件。
func writeRecommendTrainArtifactJSON(dir string, filename string, payload any) (string, error) {
	fileByte, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	filePath := filepath.Join(dir, filename)
	if err = os.WriteFile(filePath, append(fileByte, '\n'), 0o600); err != nil {
		return "", err
	}
	return filePath, nil
}

// writeRecommendTrainArtifacts 写出一次训练运行的模型、发布快照和说明文件。
func writeRecommendTrainArtifacts(
	taskName string,
	version string,
	createdAt time.Time,
	manifest *recommendTrainArtifactManifest,
	modelPayload any,
	publishSnapshotPayload any,
) (string, error) {
	runDir, err := newRecommendTrainArtifactRunDir(taskName, version, createdAt)
	if err != nil {
		return "", err
	}
	if manifest == nil {
		manifest = &recommendTrainArtifactManifest{}
	}
	manifest.Task = taskName
	manifest.CreatedAt = createdAt.Format(time.RFC3339Nano)
	if strings.TrimSpace(version) != "" {
		manifest.Version = strings.TrimSpace(version)
	}
	if modelPayload != nil {
		if _, err = writeRecommendTrainArtifactJSON(runDir, "model.json", modelPayload); err != nil {
			return "", err
		}
		manifest.ModelFile = "model.json"
	}
	if publishSnapshotPayload != nil {
		if _, err = writeRecommendTrainArtifactJSON(runDir, "publish_snapshot.json", publishSnapshotPayload); err != nil {
			return "", err
		}
		manifest.PublishSnapshotFile = "publish_snapshot.json"
	}
	if _, err = writeRecommendTrainArtifactJSON(runDir, "manifest.json", manifest); err != nil {
		return "", err
	}
	return runDir, nil
}

// buildRecommendCollaborativeFilteringArtifactSnapshot 构建协同过滤发布快照。
func buildRecommendCollaborativeFilteringArtifactSnapshot(documentMap map[int64][]recommendCache.Score) *recommendCollaborativeFilteringSnapshot {
	userIdList := make([]int64, 0, len(documentMap))
	for userId := range documentMap {
		userIdList = append(userIdList, userId)
	}
	sort.Slice(userIdList, func(i int, j int) bool {
		return userIdList[i] < userIdList[j]
	})
	snapshot := &recommendCollaborativeFilteringSnapshot{
		Entries: make([]*recommendCollaborativeFilteringEntry, 0, len(userIdList)),
	}
	for _, userId := range userIdList {
		documentList := append([]recommendCache.Score{}, documentMap[userId]...)
		snapshot.Entries = append(snapshot.Entries, &recommendCollaborativeFilteringEntry{
			UserId:    userId,
			Documents: documentList,
		})
	}
	return snapshot
}

// buildRecommendRankerArtifactSnapshot 构建 ranker 发布快照。
func buildRecommendRankerArtifactSnapshot(documentMap map[RecommendRankerActorKey][]recommendCache.Score) *recommendStageScoreSnapshot {
	actorKeyList := make([]RecommendRankerActorKey, 0, len(documentMap))
	for actorKey := range documentMap {
		actorKeyList = append(actorKeyList, actorKey)
	}
	sort.Slice(actorKeyList, func(i int, j int) bool {
		if actorKeyList[i].Scene == actorKeyList[j].Scene {
			if actorKeyList[i].ActorType == actorKeyList[j].ActorType {
				return actorKeyList[i].ActorId < actorKeyList[j].ActorId
			}
			return actorKeyList[i].ActorType < actorKeyList[j].ActorType
		}
		return actorKeyList[i].Scene < actorKeyList[j].Scene
	})
	snapshot := &recommendStageScoreSnapshot{
		Entries: make([]*recommendStageScoreEntry, 0, len(actorKeyList)),
	}
	for _, actorKey := range actorKeyList {
		documentList := append([]recommendCache.Score{}, documentMap[actorKey]...)
		snapshot.Entries = append(snapshot.Entries, &recommendStageScoreEntry{
			Scene:     actorKey.Scene,
			ActorType: actorKey.ActorType,
			ActorId:   actorKey.ActorId,
			Documents: documentList,
		})
	}
	return snapshot
}

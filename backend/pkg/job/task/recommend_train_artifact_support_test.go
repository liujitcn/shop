package task

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestWriteRecommendTrainArtifacts 验证训练产物会按约定目录写出说明文件和快照。
func TestWriteRecommendTrainArtifacts(t *testing.T) {
	previousResolver := recommendTrainArtifactRootDirResolver
	recommendTrainArtifactRootDirResolver = func() string {
		return t.TempDir()
	}
	defer func() {
		recommendTrainArtifactRootDirResolver = previousResolver
	}()

	createdAt := time.Date(2026, 4, 17, 12, 30, 45, 123000000, time.Local)
	runDir, err := writeRecommendTrainArtifacts(
		"ranker",
		"v1",
		createdAt,
		&recommendTrainArtifactManifest{
			ModelType: "afm",
			Backend:   "gomlx",
		},
		map[string]any{"factors": 16},
		map[string]any{"entries": []any{}},
	)
	if err != nil {
		t.Fatalf("write train artifacts: %v", err)
	}

	for _, filename := range []string{"model.json", "publish_snapshot.json", "manifest.json"} {
		filePath := filepath.Join(runDir, filename)
		if _, statErr := os.Stat(filePath); statErr != nil {
			t.Fatalf("artifact file missing %s: %v", filename, statErr)
		}
	}

	manifestFilePath := filepath.Join(runDir, "manifest.json")
	manifestFileByte, err := os.ReadFile(manifestFilePath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	manifest := &recommendTrainArtifactManifest{}
	if err = json.Unmarshal(manifestFileByte, manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	if manifest.Task != "ranker" || manifest.Version != "v1" {
		t.Fatalf("unexpected manifest: %+v", manifest)
	}
}

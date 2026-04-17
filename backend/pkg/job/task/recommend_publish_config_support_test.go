package task

import (
	"encoding/json"
	"testing"
	"time"

	recommendDomain "shop/pkg/recommend/domain"
)

// TestMergeRecommendPublishConfigJSON 验证发布补丁会保留调参与其他发布字段。
func TestMergeRecommendPublishConfigJSON(t *testing.T) {
	configJSON, err := mergeRecommendPublishConfigJSON(`{
  "tune": {
    "enabled": true,
    "target_metric": "auc",
    "trial_count": 9,
    "latest": {
      "task": "ranker"
    }
  },
  "publish": {
    "rollback_version": "rollback-v1",
    "gray_ratio": 0.2
  }
}`, recommendPublishConfigPatch{
		CacheVersion:       "gray-v2",
		HasCacheVersion:    true,
		ClearRollback:      true,
		GrayRatio:          1,
		HasGrayRatio:       true,
		PublishedBy:        "tester",
		HasPublishedBy:     true,
		PublishedReason:    "正式发布",
		HasPublishedReason: true,
		PublishedAt:        time.Date(2026, 4, 17, 12, 30, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("merge publish config json: %v", err)
	}

	config := &recommendDomain.StrategyVersionConfig{}
	if err = json.Unmarshal([]byte(configJSON), config); err != nil {
		t.Fatalf("unmarshal merged publish config: %v", err)
	}
	if config.Tune == nil || config.Tune.Latest == nil || config.Tune.Latest.Task != "ranker" {
		t.Fatalf("tune config lost after publish merge: %+v", config.Tune)
	}
	if config.Publish == nil {
		t.Fatalf("publish config missing")
	}
	if config.Publish.CacheVersion != "gray-v2" || config.Publish.RollbackVersion != "" {
		t.Fatalf("unexpected publish version config: %+v", config.Publish)
	}
	if config.Publish.GrayRatio != 1 || config.Publish.PublishedBy != "tester" || config.Publish.PublishedReason != "正式发布" {
		t.Fatalf("unexpected publish meta: %+v", config.Publish)
	}
	if config.Publish.PublishedAt != "2026-04-17T12:30:00Z" {
		t.Fatalf("unexpected publishedAt: %+v", config.Publish)
	}
}

// TestParseRecommendSceneListArg 验证场景参数支持数字与枚举名混传。
func TestParseRecommendSceneListArg(t *testing.T) {
	sceneList, err := parseRecommendSceneListArg("HOME, 3, HOME")
	if err != nil {
		t.Fatalf("parse scene list: %v", err)
	}
	if len(sceneList) != 2 || sceneList[0] != 1 || sceneList[1] != 3 {
		t.Fatalf("unexpected scene list: %+v", sceneList)
	}
}

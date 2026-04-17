package task

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"shop/api/gen/go/common"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	recommendDomain "shop/pkg/recommend/domain"

	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// recommendVersionPublishTestEnv 表示发布任务测试所需的最小运行环境。
type recommendVersionPublishTestEnv struct {
	ctx  context.Context
	repo *data.RecommendModelVersionRepo
	task *RecommendVersionPublish
}

// newRecommendVersionPublishTestEnv 创建发布任务测试环境。
func newRecommendVersionPublishTestEnv(t *testing.T) *recommendVersionPublishTestEnv {
	t.Helper()

	dsn := filepath.Join(t.TempDir(), "recommend-version-publish.db")
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sqlite db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})
	if err = db.AutoMigrate(&models.RecommendModelVersion{}); err != nil {
		t.Fatalf("auto migrate recommend model version: %v", err)
	}

	dataLayer := data.NewData(&databaseGorm.Client{DB: db})
	repo := data.NewRecommendModelVersionRepo(dataLayer)
	return &recommendVersionPublishTestEnv{
		ctx:  context.Background(),
		repo: repo,
		task: NewRecommendVersionPublish(data.NewTransaction(dataLayer), repo),
	}
}

// mustCreateVersion 写入测试用版本记录。
func (e *recommendVersionPublishTestEnv) mustCreateVersion(t *testing.T, entity *models.RecommendModelVersion) *models.RecommendModelVersion {
	t.Helper()
	if err := e.repo.Create(e.ctx, entity); err != nil {
		t.Fatalf("create recommend model version: %v", err)
	}
	return entity
}

// mustLoadVersion 读取最新版本记录。
func (e *recommendVersionPublishTestEnv) mustLoadVersion(t *testing.T, id int64) *models.RecommendModelVersion {
	t.Helper()
	entity, err := e.repo.FindById(e.ctx, id)
	if err != nil {
		t.Fatalf("find recommend model version: %v", err)
	}
	return entity
}

// mustParseRecommendVersionConfig 解析版本配置 JSON。
func mustParseRecommendVersionConfig(t *testing.T, configJSON string) *recommendDomain.StrategyVersionConfig {
	t.Helper()
	config := &recommendDomain.StrategyVersionConfig{}
	if err := json.Unmarshal([]byte(configJSON), config); err != nil {
		t.Fatalf("unmarshal recommend version config: %v", err)
	}
	return config
}

// TestRecommendVersionPublishExecSwitchVersion 验证正式切换会激活目标版本并禁用旧版本。
func TestRecommendVersionPublishExecSwitchVersion(t *testing.T) {
	env := newRecommendVersionPublishTestEnv(t)
	target := env.mustCreateVersion(t, &models.RecommendModelVersion{
		ID:        101,
		ModelName: "home_rank",
		ModelType: "afm",
		Version:   "gray-v2",
		Scene:     int32(common.RecommendScene_HOME),
		ConfigJSON: `{
  "tune": {
    "latest": {
      "task": "ranker"
    }
  },
  "publish": {
    "rollback_version": "rollback-v1"
  }
}`,
		Status:    int32(common.Status_DISABLE),
		CreatedAt: time.Date(2026, 4, 17, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 4, 17, 10, 0, 0, 0, time.UTC),
	})
	previous := env.mustCreateVersion(t, &models.RecommendModelVersion{
		ID:        102,
		ModelName: "home_rank",
		ModelType: "afm",
		Version:   "gray-v1",
		Scene:     int32(common.RecommendScene_HOME),
		ConfigJSON: `{
  "publish": {
    "cache_version": "gray-v1",
    "rollback_version": "rollback-v0"
  }
}`,
		Status:    int32(common.Status_ENABLE),
		CreatedAt: time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC),
	})

	result, err := env.task.Exec(map[string]string{
		"scene":           "HOME",
		"version":         "gray-v2",
		"publishedBy":     "ops",
		"publishedReason": "正式发布",
	})
	if err != nil {
		t.Fatalf("exec publish task: %v", err)
	}

	expected := strings.Join([]string{
		"scenes=1 updated_rows=1 activated_rows=1 disabled_rows=1",
		"version=gray-v2",
		"cache_version=gray-v2",
	}, "|")
	// 结果摘要需要体现正式切换、自动激活与清理回滚版本的动作。
	if strings.Join(result, "|") != expected {
		t.Fatalf("unexpected publish result: %+v", result)
	}

	target = env.mustLoadVersion(t, target.ID)
	targetConfig := mustParseRecommendVersionConfig(t, target.ConfigJSON)
	// 目标版本发布后必须进入启用态。
	if target.Status != int32(common.Status_ENABLE) {
		t.Fatalf("unexpected target status: %d", target.Status)
	}
	// 发布配置必须切到目标缓存版本并清空旧回滚版本。
	if targetConfig.Publish == nil {
		t.Fatalf("target publish config missing")
	}
	// 发布时间、发布人与发布原因都应该回写到当前目标版本。
	if targetConfig.Publish.CacheVersion != "gray-v2" || targetConfig.Publish.RollbackVersion != "" || targetConfig.Publish.PublishedBy != "ops" || targetConfig.Publish.PublishedReason != "正式发布" || targetConfig.Publish.PublishedAt == "" {
		t.Fatalf("unexpected target publish config: %+v", targetConfig.Publish)
	}
	// 发布动作不能覆盖已有训练摘要。
	if targetConfig.Tune == nil || targetConfig.Tune.Latest == nil || targetConfig.Tune.Latest.Task != "ranker" {
		t.Fatalf("unexpected target tune config: %+v", targetConfig.Tune)
	}

	previous = env.mustLoadVersion(t, previous.ID)
	// 同场景旧启用版本应该被自动禁用。
	if previous.Status != int32(common.Status_DISABLE) {
		t.Fatalf("unexpected previous status: %d", previous.Status)
	}
}

// TestRecommendVersionPublishExecRejectsAmbiguousVersion 验证同场景多条同版本记录会拒绝发布。
func TestRecommendVersionPublishExecRejectsAmbiguousVersion(t *testing.T) {
	env := newRecommendVersionPublishTestEnv(t)
	env.mustCreateVersion(t, &models.RecommendModelVersion{
		ID:         201,
		ModelName:  "home_rank_main",
		ModelType:  "afm",
		Version:    "gray-v2",
		Scene:      int32(common.RecommendScene_HOME),
		Status:     int32(common.Status_DISABLE),
		CreatedAt:  time.Date(2026, 4, 17, 9, 0, 0, 0, time.UTC),
		UpdatedAt:  time.Date(2026, 4, 17, 9, 0, 0, 0, time.UTC),
		ConfigJSON: `{}`,
	})
	env.mustCreateVersion(t, &models.RecommendModelVersion{
		ID:         202,
		ModelName:  "home_rank_backup",
		ModelType:  "fm",
		Version:    "gray-v2",
		Scene:      int32(common.RecommendScene_HOME),
		Status:     int32(common.Status_DISABLE),
		CreatedAt:  time.Date(2026, 4, 17, 8, 0, 0, 0, time.UTC),
		UpdatedAt:  time.Date(2026, 4, 17, 8, 0, 0, 0, time.UTC),
		ConfigJSON: `{}`,
	})

	_, err := env.task.Exec(map[string]string{
		"scene":           "HOME",
		"version":         "gray-v2",
		"publishedReason": "正式发布",
	})
	// 目标版本不唯一时，必须中断发布，避免误切到错误模型。
	if err == nil {
		t.Fatalf("expected ambiguous version error")
	}
	// 错误信息需要明确提示补充过滤条件。
	if !strings.Contains(err.Error(), "请补充 modelName 或 modelType") {
		t.Fatalf("unexpected ambiguous version error: %v", err)
	}
}

// TestRecommendVersionPublishExecPatchCurrentEnabledVersions 验证未显式传 version 时只补丁当前启用版本。
func TestRecommendVersionPublishExecPatchCurrentEnabledVersions(t *testing.T) {
	env := newRecommendVersionPublishTestEnv(t)
	home := env.mustCreateVersion(t, &models.RecommendModelVersion{
		ID:        301,
		ModelName: "home_rank",
		ModelType: "afm",
		Version:   "gray-v1",
		Scene:     int32(common.RecommendScene_HOME),
		ConfigJSON: `{
  "publish": {
    "cache_version": "gray-v1"
  }
}`,
		Status:    int32(common.Status_ENABLE),
		CreatedAt: time.Date(2026, 4, 17, 7, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 4, 17, 7, 0, 0, 0, time.UTC),
	})
	cart := env.mustCreateVersion(t, &models.RecommendModelVersion{
		ID:        302,
		ModelName: "cart_rank",
		ModelType: "bpr",
		Version:   "gray-cart-v1",
		Scene:     int32(common.RecommendScene_CART),
		ConfigJSON: `{
  "publish": {
    "cache_version": "gray-cart-v1"
  },
  "tune": {
    "latest": {
      "task": "collaborative_filtering"
    }
  }
}`,
		Status:    int32(common.Status_ENABLE),
		CreatedAt: time.Date(2026, 4, 17, 7, 30, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 4, 17, 7, 30, 0, 0, time.UTC),
	})
	disabled := env.mustCreateVersion(t, &models.RecommendModelVersion{
		ID:        303,
		ModelName: "cart_rank",
		ModelType: "bpr",
		Version:   "gray-cart-v0",
		Scene:     int32(common.RecommendScene_CART),
		ConfigJSON: `{
  "publish": {
    "cache_version": "gray-cart-v0"
  }
}`,
		Status:    int32(common.Status_DISABLE),
		CreatedAt: time.Date(2026, 4, 17, 6, 30, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 4, 17, 6, 30, 0, 0, time.UTC),
	})

	result, err := env.task.Exec(map[string]string{
		"scenes":          "HOME, 3",
		"rollbackVersion": "rollback-v2",
		"grayRatio":       "0.25",
		"publishedReason": "灰度回滚",
	})
	if err != nil {
		t.Fatalf("exec patch publish task: %v", err)
	}

	expected := strings.Join([]string{
		"scenes=1,3 updated_rows=2 activated_rows=0 disabled_rows=0",
		"rollback_version=rollback-v2",
	}, "|")
	// 当前启用版本补丁只应该更新命中场景的启用记录。
	if strings.Join(result, "|") != expected {
		t.Fatalf("unexpected patch publish result: %+v", result)
	}

	home = env.mustLoadVersion(t, home.ID)
	cart = env.mustLoadVersion(t, cart.ID)
	disabled = env.mustLoadVersion(t, disabled.ID)
	homeConfig := mustParseRecommendVersionConfig(t, home.ConfigJSON)
	cartConfig := mustParseRecommendVersionConfig(t, cart.ConfigJSON)
	disabledConfig := mustParseRecommendVersionConfig(t, disabled.ConfigJSON)
	// 当前启用版本补丁不能改变两个目标版本的启用状态。
	if home.Status != int32(common.Status_ENABLE) || cart.Status != int32(common.Status_ENABLE) {
		t.Fatalf("unexpected enabled status after patch: home=%d cart=%d", home.Status, cart.Status)
	}
	// 命中场景的启用版本都应该收到统一的回滚与灰度配置。
	if homeConfig.Publish == nil || cartConfig.Publish == nil {
		t.Fatalf("publish config missing after patch: home=%+v cart=%+v", homeConfig.Publish, cartConfig.Publish)
	}
	// 非正式切换不能覆盖既有缓存版本。
	if homeConfig.Publish.CacheVersion != "gray-v1" || cartConfig.Publish.CacheVersion != "gray-cart-v1" {
		t.Fatalf("unexpected cache version after patch: home=%+v cart=%+v", homeConfig.Publish, cartConfig.Publish)
	}
	// 回滚版本、灰度比例和发布说明需要同步写入两个目标场景。
	if homeConfig.Publish.RollbackVersion != "rollback-v2" || cartConfig.Publish.RollbackVersion != "rollback-v2" || homeConfig.Publish.GrayRatio != 0.25 || cartConfig.Publish.GrayRatio != 0.25 || homeConfig.Publish.PublishedReason != "灰度回滚" || cartConfig.Publish.PublishedReason != "灰度回滚" {
		t.Fatalf("unexpected publish patch config: home=%+v cart=%+v", homeConfig.Publish, cartConfig.Publish)
	}
	// 当前启用版本补丁不能覆盖购物车场景已有训练摘要。
	if cartConfig.Tune == nil || cartConfig.Tune.Latest == nil || cartConfig.Tune.Latest.Task != "collaborative_filtering" {
		t.Fatalf("unexpected cart tune config after patch: %+v", cartConfig.Tune)
	}
	// 非当前启用版本不应该被误改。
	if disabled.Status != int32(common.Status_DISABLE) || disabledConfig.Publish == nil || disabledConfig.Publish.CacheVersion != "gray-cart-v0" || disabledConfig.Publish.RollbackVersion != "" {
		t.Fatalf("unexpected disabled version state: status=%d config=%+v", disabled.Status, disabledConfig.Publish)
	}
}

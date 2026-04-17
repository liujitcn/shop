package task

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"shop/api/gen/go/common"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/go-kratos/kratos/v2/log"
)

// RecommendVersionPublish 推荐版本发布与回滚任务。
type RecommendVersionPublish struct {
	tx                        data.Transaction
	recommendModelVersionRepo *data.RecommendModelVersionRepo
	ctx                       context.Context
}

// NewRecommendVersionPublish 创建推荐版本发布与回滚任务实例。
func NewRecommendVersionPublish(
	tx data.Transaction,
	recommendModelVersionRepo *data.RecommendModelVersionRepo,
) *RecommendVersionPublish {
	return &RecommendVersionPublish{
		tx:                        tx,
		recommendModelVersionRepo: recommendModelVersionRepo,
		ctx:                       context.Background(),
	}
}

// Exec 执行推荐版本发布与回滚任务。
func (t *RecommendVersionPublish) Exec(args map[string]string) ([]string, error) {
	log.Infof("Job RecommendVersionPublish Exec %+v", args)

	sceneArg := strings.TrimSpace(args["scene"])
	// 兼容场景列表参数使用 scenes 复数写法。
	if sceneArg == "" {
		sceneArg = strings.TrimSpace(args["scenes"])
	}
	sceneList, err := parseRecommendSceneListArg(sceneArg)
	if err != nil {
		return []string{err.Error()}, err
	}
	version := normalizeRecommendPublishVersionValue(args["version"])
	cacheVersion := strings.TrimSpace(args["cacheVersion"])
	rollbackVersion := strings.TrimSpace(args["rollbackVersion"])
	clearRollback, err := parseRecommendMaterializeBoolArg(args["clearRollback"], false)
	if err != nil {
		return []string{err.Error()}, err
	}
	grayRatio, hasGrayRatio, err := parseRecommendGrayRatioArg(args["grayRatio"])
	if err != nil {
		return []string{err.Error()}, err
	}
	publishedBy := strings.TrimSpace(args["publishedBy"])
	publishedReason := strings.TrimSpace(args["publishedReason"])
	modelName := strings.TrimSpace(args["modelName"])
	modelType := strings.TrimSpace(args["modelType"])
	hasCacheVersion := cacheVersion != ""
	hasRollbackVersion := rollbackVersion != ""
	hasPublishedBy := publishedBy != ""
	hasPublishedReason := publishedReason != ""

	// 至少需要一个发布相关动作，避免把空任务误打到线上。
	if version == "" && !hasCacheVersion && !hasRollbackVersion && !clearRollback && !hasGrayRatio && !hasPublishedBy && !hasPublishedReason {
		err = errorsx.InvalidArgument("发布任务至少需要 version、cacheVersion、rollbackVersion、clearRollback、grayRatio、publishedBy、publishedReason 中的一个参数")
		return []string{err.Error()}, err
	}

	updatedRows := 0
	activatedRows := 0
	disabledRows := 0
	targetSceneList := make([]int32, 0)
	publishedAt := time.Now()
	err = t.tx.Transaction(t.ctx, func(ctx context.Context) error {
		var targetList []*models.RecommendModelVersion
		var loadErr error
		// 显式传目标版本时，先切换到该版本记录再更新发布配置。
		if version != "" {
			targetList, loadErr = loadRecommendPublishTargetVersions(ctx, t.recommendModelVersionRepo, version, sceneList, modelName, modelType)
		} else {
			// 未显式传版本时，只更新当前已启用版本的发布配置。
			targetList, loadErr = loadRecommendCurrentEnabledVersions(ctx, t.recommendModelVersionRepo, sceneList)
		}
		if loadErr != nil {
			return loadErr
		}
		if len(targetList) == 0 {
			return errorsx.ResourceNotFound("未找到可发布的推荐版本")
		}

		targetSceneList = targetSceneList[:0]
		for _, target := range targetList {
			targetSceneList = append(targetSceneList, target.Scene)
			patch := recommendPublishConfigPatch{
				CacheVersion:       cacheVersion,
				HasCacheVersion:    hasCacheVersion,
				RollbackVersion:    rollbackVersion,
				HasRollbackVersion: hasRollbackVersion,
				ClearRollback:      clearRollback,
				GrayRatio:          grayRatio,
				HasGrayRatio:       hasGrayRatio,
				PublishedBy:        publishedBy,
				HasPublishedBy:     hasPublishedBy,
				PublishedReason:    publishedReason,
				HasPublishedReason: hasPublishedReason,
				PublishedAt:        publishedAt,
			}
			// 当前执行正式版本切换但未显式指定 cacheVersion 时，默认把缓存版本切到目标版本。
			if version != "" && !patch.HasCacheVersion {
				patch.CacheVersion = target.Version
				patch.HasCacheVersion = true
			}
			// 当前执行正式版本切换且未显式指定回滚版本时，自动清理旧回滚配置。
			if version != "" && !patch.HasRollbackVersion && !patch.ClearRollback {
				patch.ClearRollback = true
			}

			// 当前执行正式版本切换时，先禁用同场景下其余启用版本，确保线上只命中目标版本。
			if version != "" {
				sceneDisabledCount, disableErr := disableRecommendEnabledVersionsForScene(ctx, t.recommendModelVersionRepo, target.Scene, target.ID)
				if disableErr != nil {
					return disableErr
				}
				disabledRows += sceneDisabledCount
			}

			mergedConfigJSON, mergeErr := mergeRecommendPublishConfigJSON(target.ConfigJSON, patch)
			if mergeErr != nil {
				return mergeErr
			}
			changed := false
			activated := false
			if target.ConfigJSON != mergedConfigJSON {
				target.ConfigJSON = mergedConfigJSON
				changed = true
			}
			// 当前执行正式版本切换时，目标版本必须处于启用态。
			if version != "" && target.Status != int32(common.Status_ENABLE) {
				target.Status = int32(common.Status_ENABLE)
				changed = true
				activated = true
			}
			if !changed {
				continue
			}
			if updateErr := t.recommendModelVersionRepo.UpdateById(ctx, target); updateErr != nil {
				return updateErr
			}
			updatedRows++
			if activated {
				activatedRows++
			}
		}
		return nil
	})
	if err != nil {
		return []string{err.Error()}, err
	}

	result := make([]string, 0, 4)
	result = append(result, fmt.Sprintf(
		"scenes=%s updated_rows=%d activated_rows=%d disabled_rows=%d",
		formatRecommendSceneList(targetSceneList),
		updatedRows,
		activatedRows,
		disabledRows,
	))
	if version != "" {
		result = append(result, fmt.Sprintf("version=%s", version))
	}
	if version != "" || hasCacheVersion {
		effectiveCacheVersion := cacheVersion
		if effectiveCacheVersion == "" {
			effectiveCacheVersion = version
		}
		result = append(result, fmt.Sprintf("cache_version=%s", normalizeRecommendPublishVersionValue(effectiveCacheVersion)))
	}
	if hasRollbackVersion {
		result = append(result, fmt.Sprintf("rollback_version=%s", normalizeRecommendPublishVersionValue(rollbackVersion)))
	}
	if clearRollback {
		result = append(result, "rollback_version_cleared=true")
	}
	return result, nil
}

// formatRecommendSceneList 格式化场景列表，便于任务摘要输出。
func formatRecommendSceneList(sceneList []int32) string {
	if len(sceneList) == 0 {
		return ""
	}
	partList := make([]string, 0, len(sceneList))
	for _, scene := range sceneList {
		partList = append(partList, strconv.Itoa(int(scene)))
	}
	return strings.Join(partList, ",")
}

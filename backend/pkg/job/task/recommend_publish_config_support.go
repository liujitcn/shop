package task

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"shop/api/gen/go/common"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	recommendCache "shop/pkg/recommend/cache"
	recommendDomain "shop/pkg/recommend/domain"

	"github.com/liujitcn/gorm-kit/repo"
)

// recommendPublishConfigPatch 表示一次发布配置写回补丁。
type recommendPublishConfigPatch struct {
	CacheVersion       string    // 当前希望生效的缓存版本。
	HasCacheVersion    bool      // 当前是否显式更新缓存版本。
	RollbackVersion    string    // 当前希望写入的回滚版本。
	HasRollbackVersion bool      // 当前是否显式更新回滚版本。
	ClearRollback      bool      // 当前是否清空回滚版本。
	GrayRatio          float64   // 当前灰度比例。
	HasGrayRatio       bool      // 当前是否显式更新灰度比例。
	PublishedBy        string    // 当前发布人。
	HasPublishedBy     bool      // 当前是否显式更新发布人。
	PublishedReason    string    // 当前发布说明。
	HasPublishedReason bool      // 当前是否显式更新发布说明。
	PublishedAt        time.Time // 当前发布时间。
}

// parseRecommendSceneListArg 解析推荐场景列表参数。
func parseRecommendSceneListArg(sceneValue string) ([]int32, error) {
	trimmedSceneValue := strings.TrimSpace(sceneValue)
	// 未显式传场景时，交给调用方决定作用范围。
	if trimmedSceneValue == "" {
		return nil, nil
	}
	partList := strings.Split(trimmedSceneValue, ",")
	sceneList := make([]int32, 0, len(partList))
	sceneSet := make(map[int32]struct{}, len(partList))
	for _, part := range partList {
		trimmedPart := strings.TrimSpace(part)
		// 空片段不参与场景解析。
		if trimmedPart == "" {
			continue
		}
		scene, err := parseRecommendSceneArg(trimmedPart)
		if err != nil {
			return nil, err
		}
		_, exists := sceneSet[scene]
		// 重复场景只保留一份，避免重复更新同一条版本记录。
		if exists {
			continue
		}
		sceneSet[scene] = struct{}{}
		sceneList = append(sceneList, scene)
	}
	if len(sceneList) == 0 {
		return nil, errorsx.InvalidArgument("scene 不能为空")
	}
	slices.Sort(sceneList)
	return sceneList, nil
}

// parseRecommendSceneArg 解析单个推荐场景参数。
func parseRecommendSceneArg(value string) (int32, error) {
	trimmedValue := strings.TrimSpace(value)
	// 先按数字场景编号解析，兼容任务参数直接传枚举值。
	sceneValue, err := strconv.ParseInt(trimmedValue, 10, 32)
	if err == nil {
		scene := int32(sceneValue)
		_, exists := common.RecommendScene_name[scene]
		if scene > 0 && exists {
			return scene, nil
		}
		return 0, errorsx.InvalidArgument(fmt.Sprintf("scene 不支持 %s", trimmedValue))
	}
	enumKey := strings.ToUpper(trimmedValue)
	enumSceneValue, exists := common.RecommendScene_value[enumKey]
	if !exists || enumSceneValue <= 0 {
		return 0, errorsx.InvalidArgument(fmt.Sprintf("scene 不支持 %s", trimmedValue))
	}
	return int32(enumSceneValue), nil
}

// parseRecommendGrayRatioArg 解析灰度比例参数。
func parseRecommendGrayRatioArg(value string) (float64, bool, error) {
	trimmedValue := strings.TrimSpace(value)
	if trimmedValue == "" {
		return 0, false, nil
	}
	grayRatio, err := strconv.ParseFloat(trimmedValue, 64)
	if err != nil {
		return 0, false, errorsx.InvalidArgument("grayRatio 格式错误")
	}
	// 当前灰度比例必须落在 0 到 1 之间，避免写出无效比例。
	if grayRatio < 0 || grayRatio > 1 {
		return 0, false, errorsx.InvalidArgument("grayRatio 必须在 0 到 1 之间")
	}
	return grayRatio, true, nil
}

// mergeRecommendPublishConfigJSON 将发布补丁合并到版本配置 JSON。
func mergeRecommendPublishConfigJSON(configJSON string, patch recommendPublishConfigPatch) (string, error) {
	config := &recommendDomain.StrategyVersionConfig{}
	trimmedConfigJSON := strings.TrimSpace(configJSON)
	// 原配置存在时，先按既有结构解析，避免覆盖其他策略字段。
	if trimmedConfigJSON != "" {
		if err := json.Unmarshal([]byte(trimmedConfigJSON), config); err != nil {
			return "", fmt.Errorf("unmarshal recommend version config: %w", err)
		}
	}
	// 当前还没有发布配置时，先初始化容器再写发布补丁。
	if config.Publish == nil {
		config.Publish = &recommendDomain.PublishStrategy{}
	}
	if patch.HasCacheVersion {
		config.Publish.CacheVersion = normalizeRecommendPublishVersionValue(patch.CacheVersion)
	}
	// 当前显式要求清空回滚版本时，优先清空，避免保留旧回滚配置。
	if patch.ClearRollback {
		config.Publish.RollbackVersion = ""
	} else if patch.HasRollbackVersion {
		config.Publish.RollbackVersion = normalizeRecommendPublishVersionValue(patch.RollbackVersion)
	}
	if patch.HasGrayRatio {
		config.Publish.GrayRatio = patch.GrayRatio
	}
	if patch.HasPublishedBy {
		config.Publish.PublishedBy = strings.TrimSpace(patch.PublishedBy)
	}
	if patch.HasPublishedReason {
		config.Publish.PublishedReason = strings.TrimSpace(patch.PublishedReason)
	}
	// 当前发布时间存在时，再更新最近一次发布时间。
	if !patch.PublishedAt.IsZero() {
		config.Publish.PublishedAt = patch.PublishedAt.Format(time.RFC3339Nano)
	}
	configByte, err := json.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("marshal recommend version config: %w", err)
	}
	return string(configByte), nil
}

// normalizeRecommendPublishVersionValue 规整发布配置里的版本值。
func normalizeRecommendPublishVersionValue(version string) string {
	trimmedVersion := strings.TrimSpace(version)
	// 空版本用于清空配置，不做 default 归一化。
	if trimmedVersion == "" {
		return ""
	}
	return recommendCache.NormalizeVersion(trimmedVersion)
}

// loadRecommendPublishTargetVersions 加载目标发布版本记录。
func loadRecommendPublishTargetVersions(
	ctx context.Context,
	recommendModelVersionRepo *data.RecommendModelVersionRepo,
	version string,
	sceneList []int32,
	modelName string,
	modelType string,
) ([]*models.RecommendModelVersion, error) {
	normalizedVersion := normalizeRecommendPublishVersionValue(version)
	if recommendModelVersionRepo == nil || normalizedVersion == "" {
		return nil, nil
	}
	query := recommendModelVersionRepo.Query(ctx).RecommendModelVersion
	opts := make([]repo.QueryOption, 0, 5)
	opts = append(opts, repo.Where(query.Version.Eq(normalizedVersion)))
	if len(sceneList) > 0 {
		opts = append(opts, repo.Where(query.Scene.In(sceneList...)))
	}
	if strings.TrimSpace(modelName) != "" {
		opts = append(opts, repo.Where(query.ModelName.Eq(strings.TrimSpace(modelName))))
	}
	if strings.TrimSpace(modelType) != "" {
		opts = append(opts, repo.Where(query.ModelType.Eq(strings.TrimSpace(modelType))))
	}
	opts = append(opts, repo.Order(query.Scene.Asc()))
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))
	list, err := recommendModelVersionRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return selectRecommendPublishUniqueTargets(list, sceneList, normalizedVersion)
}

// loadRecommendCurrentEnabledVersions 加载当前已启用的版本记录。
func loadRecommendCurrentEnabledVersions(
	ctx context.Context,
	recommendModelVersionRepo *data.RecommendModelVersionRepo,
	sceneList []int32,
) ([]*models.RecommendModelVersion, error) {
	if recommendModelVersionRepo == nil {
		return nil, nil
	}
	query := recommendModelVersionRepo.Query(ctx).RecommendModelVersion
	opts := make([]repo.QueryOption, 0, 4)
	opts = append(opts, repo.Where(query.Status.Eq(int32(common.Status_ENABLE))))
	if len(sceneList) > 0 {
		opts = append(opts, repo.Where(query.Scene.In(sceneList...)))
	}
	opts = append(opts, repo.Order(query.Scene.Asc()))
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))
	list, err := recommendModelVersionRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	if len(sceneList) > 0 {
		return selectRecommendPublishUniqueTargets(list, sceneList, "")
	}
	return pickLatestRecommendModelVersionByScene(list), nil
}

// selectRecommendPublishUniqueTargets 为每个场景挑出唯一可发布的目标版本。
func selectRecommendPublishUniqueTargets(
	list []*models.RecommendModelVersion,
	requiredSceneList []int32,
	version string,
) ([]*models.RecommendModelVersion, error) {
	groupedMap := make(map[int32][]*models.RecommendModelVersion, len(list))
	for _, item := range list {
		// 非法版本记录不参与正式发布选择。
		if item == nil || item.ID <= 0 || item.Scene <= 0 {
			continue
		}
		groupedMap[item.Scene] = append(groupedMap[item.Scene], item)
	}
	if len(requiredSceneList) > 0 {
		for _, scene := range requiredSceneList {
			if len(groupedMap[scene]) == 0 {
				return nil, errorsx.ResourceNotFound(fmt.Sprintf("scene=%d 未找到可发布版本", scene))
			}
		}
	}
	result := make([]*models.RecommendModelVersion, 0, len(groupedMap))
	for scene, currentList := range groupedMap {
		// 同一场景命中多条同版本记录时，要求调用方先缩小过滤条件，避免错误发布。
		if len(currentList) > 1 {
			return nil, errorsx.Conflict(fmt.Sprintf("scene=%d version=%s 存在多条版本记录，请补充 modelName 或 modelType", scene, version))
		}
		result = append(result, currentList[0])
	}
	slices.SortFunc(result, func(left *models.RecommendModelVersion, right *models.RecommendModelVersion) int {
		return int(left.Scene - right.Scene)
	})
	return result, nil
}

// disableRecommendEnabledVersionsForScene 禁用同场景下除保留记录外的其余启用版本。
func disableRecommendEnabledVersionsForScene(
	ctx context.Context,
	recommendModelVersionRepo *data.RecommendModelVersionRepo,
	scene int32,
	keepId int64,
) (int, error) {
	if recommendModelVersionRepo == nil || scene <= 0 {
		return 0, nil
	}
	query := recommendModelVersionRepo.Query(ctx).RecommendModelVersion
	opts := make([]repo.QueryOption, 0, 3)
	opts = append(opts, repo.Where(query.Scene.Eq(scene)))
	opts = append(opts, repo.Where(query.Status.Eq(int32(common.Status_ENABLE))))
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))
	list, err := recommendModelVersionRepo.List(ctx, opts...)
	if err != nil {
		return 0, err
	}
	disabledCount := 0
	for _, item := range list {
		// 保留记录或非法记录不参与禁用。
		if item == nil || item.ID <= 0 || item.ID == keepId {
			continue
		}
		item.Status = int32(common.Status_DISABLE)
		if err := recommendModelVersionRepo.UpdateById(ctx, item); err != nil {
			return disabledCount, err
		}
		disabledCount++
	}
	return disabledCount, nil
}

package task

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"shop/api/gen/go/common"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	recommendCache "shop/pkg/recommend/cache"
	recommendDomain "shop/pkg/recommend/domain"

	"github.com/liujitcn/gorm-kit/repo"
)

// buildRecommendTuneLatestSummary 构建最近一次真实训练的版本摘要。
func buildRecommendTuneLatestSummary(
	taskName string,
	modelType string,
	backend string,
	artifactDir string,
	trainedAt time.Time,
	version string,
	versionList []string,
	bestValue float64,
	score map[string]float64,
) *recommendDomain.TuneLatestSummary {
	summary := &recommendDomain.TuneLatestSummary{
		Task:        strings.TrimSpace(taskName),
		ModelType:   strings.TrimSpace(strings.ToLower(modelType)),
		Backend:     strings.TrimSpace(strings.ToLower(backend)),
		ArtifactDir: strings.TrimSpace(artifactDir),
		BestValue:   bestValue,
		Score:       cloneRecommendTuneScoreMap(score),
	}
	// 单版本任务显式带了版本号时，才写入版本字段，避免空值被归一成 default。
	if strings.TrimSpace(version) != "" {
		summary.Version = recommendCache.NormalizeVersion(version)
	}
	// 训练时间非零时，再补齐标准时间戳，便于线上调试上下文直接透出。
	if !trainedAt.IsZero() {
		summary.TrainedAt = trainedAt.Format(time.RFC3339Nano)
	}
	normalizedVersionList := normalizeRecommendTuneVersionList(versionList)
	if len(normalizedVersionList) > 0 {
		summary.Versions = normalizedVersionList
	}
	return summary
}

// cloneRecommendTuneScoreMap 复制训练指标映射，避免调用方后续修改原始结果。
func cloneRecommendTuneScoreMap(score map[string]float64) map[string]float64 {
	if len(score) == 0 {
		return nil
	}
	clonedMap := make(map[string]float64, len(score))
	for key, value := range score {
		normalizedKey := strings.TrimSpace(strings.ToLower(key))
		// 指标名为空时，不继续写入训练摘要。
		if normalizedKey == "" {
			continue
		}
		clonedMap[normalizedKey] = value
	}
	if len(clonedMap) == 0 {
		return nil
	}
	return clonedMap
}

// normalizeRecommendTuneVersionList 规整多版本训练任务的版本列表。
func normalizeRecommendTuneVersionList(versionList []string) []string {
	if len(versionList) == 0 {
		return nil
	}
	versionSet := make(map[string]struct{}, len(versionList))
	result := make([]string, 0, len(versionList))
	for _, version := range versionList {
		// 空版本不参与版本列表回写，避免被缓存层归一成 default。
		if strings.TrimSpace(version) == "" {
			continue
		}
		normalizedVersion := recommendCache.NormalizeVersion(version)
		_, exists := versionSet[normalizedVersion]
		// 重复版本只保留一份，避免配置 JSON 冗余膨胀。
		if exists {
			continue
		}
		versionSet[normalizedVersion] = struct{}{}
		result = append(result, normalizedVersion)
	}
	sort.Strings(result)
	return result
}

// mergeRecommendTuneLatestConfigJSON 将最近训练摘要合并到版本配置 JSON。
func mergeRecommendTuneLatestConfigJSON(configJSON string, latest *recommendDomain.TuneLatestSummary) (string, error) {
	config := &recommendDomain.StrategyVersionConfig{}
	trimmedConfigJSON := strings.TrimSpace(configJSON)
	// 原配置存在时，先按既有结构解析，避免覆盖其他策略字段。
	if trimmedConfigJSON != "" {
		if err := json.Unmarshal([]byte(trimmedConfigJSON), config); err != nil {
			return "", fmt.Errorf("unmarshal recommend version config: %w", err)
		}
	}
	// 当前还没有调参配置时，先初始化容器再写最近训练摘要。
	if config.Tune == nil {
		config.Tune = &recommendDomain.TuneStrategy{}
	}
	config.Tune.Latest = latest
	configByte, err := json.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("marshal recommend version config: %w", err)
	}
	return string(configByte), nil
}

// writeRecommendTuneLatestForEnabledVersions 将最近训练摘要回写到当前启用版本配置。
func writeRecommendTuneLatestForEnabledVersions(
	ctx context.Context,
	recommendModelVersionRepo *data.RecommendModelVersionRepo,
	versionList []string,
	latest *recommendDomain.TuneLatestSummary,
) (int, error) {
	normalizedVersionList := normalizeRecommendTuneVersionList(versionList)
	if recommendModelVersionRepo == nil || latest == nil || len(normalizedVersionList) == 0 {
		return 0, nil
	}
	query := recommendModelVersionRepo.Query(ctx).RecommendModelVersion
	opts := make([]repo.QueryOption, 0, 4)
	opts = append(opts, repo.Where(query.Version.In(normalizedVersionList...)))
	opts = append(opts, repo.Where(query.Status.Eq(int32(common.Status_ENABLE))))
	opts = append(opts, repo.Order(query.Scene.Asc()))
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))
	list, err := recommendModelVersionRepo.List(ctx, opts...)
	if err != nil {
		return 0, err
	}
	return updateRecommendTuneLatestForModelVersionList(ctx, recommendModelVersionRepo, pickLatestRecommendModelVersionByScene(list), latest)
}

// writeRecommendTuneLatestForVersion 将最近训练摘要回写到指定版本配置。
func writeRecommendTuneLatestForVersion(
	ctx context.Context,
	recommendModelVersionRepo *data.RecommendModelVersionRepo,
	version string,
	latest *recommendDomain.TuneLatestSummary,
) (int, error) {
	if recommendModelVersionRepo == nil || latest == nil || strings.TrimSpace(version) == "" {
		return 0, nil
	}
	normalizedVersion := recommendCache.NormalizeVersion(version)
	query := recommendModelVersionRepo.Query(ctx).RecommendModelVersion
	opts := make([]repo.QueryOption, 0, 3)
	opts = append(opts, repo.Where(query.Version.Eq(normalizedVersion)))
	opts = append(opts, repo.Order(query.Scene.Asc()))
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))
	list, err := recommendModelVersionRepo.List(ctx, opts...)
	if err != nil {
		return 0, err
	}
	return updateRecommendTuneLatestForModelVersionList(ctx, recommendModelVersionRepo, pickLatestRecommendModelVersionByScene(list), latest)
}

// pickLatestRecommendModelVersionByScene 按场景挑出最新一条版本记录。
func pickLatestRecommendModelVersionByScene(list []*models.RecommendModelVersion) []*models.RecommendModelVersion {
	result := make([]*models.RecommendModelVersion, 0, len(list))
	sceneSet := make(map[int32]struct{}, len(list))
	for _, item := range list {
		// 非法记录无法安全回写配置，直接跳过。
		if item == nil || item.ID <= 0 {
			continue
		}
		_, exists := sceneSet[item.Scene]
		// 同场景只保留当前排序下的首条记录，也就是最新版本记录。
		if exists {
			continue
		}
		sceneSet[item.Scene] = struct{}{}
		result = append(result, item)
	}
	return result
}

// updateRecommendTuneLatestForModelVersionList 批量回写最近训练摘要。
func updateRecommendTuneLatestForModelVersionList(
	ctx context.Context,
	recommendModelVersionRepo *data.RecommendModelVersionRepo,
	list []*models.RecommendModelVersion,
	latest *recommendDomain.TuneLatestSummary,
) (int, error) {
	updatedCount := 0
	for _, item := range list {
		mergedConfigJSON, err := mergeRecommendTuneLatestConfigJSON(item.ConfigJSON, latest)
		if err != nil {
			return updatedCount, fmt.Errorf("merge version config scene=%d version=%s: %w", item.Scene, item.Version, err)
		}
		// 合并结果没有变化时，不重复更新数据库。
		if item.ConfigJSON == mergedConfigJSON {
			continue
		}
		item.ConfigJSON = mergedConfigJSON
		if err := recommendModelVersionRepo.UpdateById(ctx, item); err != nil {
			return updatedCount, fmt.Errorf("update version config scene=%d version=%s: %w", item.Scene, item.Version, err)
		}
		updatedCount++
	}
	return updatedCount, nil
}

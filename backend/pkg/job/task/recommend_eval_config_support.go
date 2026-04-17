package task

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	recommendDomain "shop/pkg/recommend/domain"
)

// buildRecommendTuneLatestEvalSummary 构建最近一次评估日报摘要。
func buildRecommendTuneLatestEvalSummary(report *models.RecommendEvalReport) *recommendDomain.TuneLatestEvalSummary {
	// 评估报告为空时，不生成评估摘要。
	if report == nil {
		return nil
	}
	summary := &recommendDomain.TuneLatestEvalSummary{
		SampleSize:    report.SampleSize,
		RequestCount:  report.RequestCount,
		ExposureCount: report.ExposureCount,
		ClickCount:    report.ClickCount,
		OrderCount:    report.OrderCount,
		PayCount:      report.PayCount,
		Ctr:           report.Ctr,
		Cvr:           report.Cvr,
		Ndcg:          report.Ndcg,
		Precision:     report.PrecisionScore,
		Recall:        report.RecallScore,
	}
	// 报告日期存在时，再补齐日期字符串，便于直接写入版本配置。
	if !report.ReportDate.IsZero() {
		summary.ReportDate = report.ReportDate.Format(time.DateOnly)
	}
	// 策略名称存在时，再补齐评估对应的策略标识。
	if strings.TrimSpace(report.StrategyName) != "" {
		summary.StrategyName = strings.TrimSpace(report.StrategyName)
	}
	return summary
}

// mergeRecommendTuneLatestEvalConfigJSON 将最近评估摘要合并到版本配置 JSON。
func mergeRecommendTuneLatestEvalConfigJSON(configJSON string, latestEval *recommendDomain.TuneLatestEvalSummary) (string, error) {
	config := &recommendDomain.StrategyVersionConfig{}
	trimmedConfigJSON := strings.TrimSpace(configJSON)
	// 原配置存在时，先按既有结构解析，避免覆盖其他策略字段。
	if trimmedConfigJSON != "" {
		if err := json.Unmarshal([]byte(trimmedConfigJSON), config); err != nil {
			return "", fmt.Errorf("unmarshal recommend version config: %w", err)
		}
	}
	// 当前还没有调参配置时，先初始化容器再写最近评估摘要。
	if config.Tune == nil {
		config.Tune = &recommendDomain.TuneStrategy{}
	}
	config.Tune.LatestEval = latestEval
	configByte, err := json.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("marshal recommend version config: %w", err)
	}
	return string(configByte), nil
}

// writeRecommendTuneLatestEvalForSceneVersions 将最近评估摘要回写到各场景版本配置。
func writeRecommendTuneLatestEvalForSceneVersions(
	ctx context.Context,
	recommendModelVersionRepo *data.RecommendModelVersionRepo,
	strategyByScene map[int32]*models.RecommendModelVersion,
	reportList []*models.RecommendEvalReport,
) (int, error) {
	// 缺少仓储、版本映射或报告列表时，不需要执行回写。
	if recommendModelVersionRepo == nil || len(strategyByScene) == 0 || len(reportList) == 0 {
		return 0, nil
	}
	updatedCount := 0
	for _, report := range reportList {
		// 当前评估报告为空时，无法回写摘要。
		if report == nil {
			continue
		}
		modelVersion := strategyByScene[report.Scene]
		// 当前场景没有启用版本时，只保留评估报表，不回写版本配置。
		if modelVersion == nil || modelVersion.ID <= 0 {
			continue
		}
		mergedConfigJSON, err := mergeRecommendTuneLatestEvalConfigJSON(modelVersion.ConfigJSON, buildRecommendTuneLatestEvalSummary(report))
		if err != nil {
			return updatedCount, fmt.Errorf("merge eval config scene=%d version=%s: %w", modelVersion.Scene, modelVersion.Version, err)
		}
		// 合并结果没有变化时，不重复更新数据库。
		if modelVersion.ConfigJSON == mergedConfigJSON {
			continue
		}
		modelVersion.ConfigJSON = mergedConfigJSON
		if err := recommendModelVersionRepo.UpdateById(ctx, modelVersion); err != nil {
			return updatedCount, fmt.Errorf("update eval config scene=%d version=%s: %w", modelVersion.Scene, modelVersion.Version, err)
		}
		updatedCount++
	}
	return updatedCount, nil
}

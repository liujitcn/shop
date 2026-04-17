package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	adminApi "shop/api/gen/go/admin"
	commonApi "shop/api/gen/go/common"
	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"shop/pkg/job/task"
	recommendCache "shop/pkg/recommend/cache"
	recommendDomain "shop/pkg/recommend/domain"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/liujitcn/gorm-kit/repo"
	"gorm.io/gorm"
)

// recommendModelVersionPublishExecutor 定义推荐版本发布任务执行能力。
type recommendModelVersionPublishExecutor interface {
	Exec(args map[string]string) ([]string, error)
}

// RecommendModelVersionCase 推荐模型版本业务实例。
type RecommendModelVersionCase struct {
	*biz.BaseCase
	*data.RecommendModelVersionRepo
	recommendModelVersionPublishTask recommendModelVersionPublishExecutor
}

// NewRecommendModelVersionCase 创建推荐模型版本业务实例。
func NewRecommendModelVersionCase(
	baseCase *biz.BaseCase,
	recommendModelVersionRepo *data.RecommendModelVersionRepo,
	recommendModelVersionPublishTask *task.RecommendVersionPublish,
) *RecommendModelVersionCase {
	return &RecommendModelVersionCase{
		BaseCase:                         baseCase,
		RecommendModelVersionRepo:        recommendModelVersionRepo,
		recommendModelVersionPublishTask: recommendModelVersionPublishTask,
	}
}

// PageRecommendModelVersion 查询推荐版本分页列表。
func (c *RecommendModelVersionCase) PageRecommendModelVersion(ctx context.Context, req *adminApi.PageRecommendModelVersionRequest) (*adminApi.PageRecommendModelVersionResponse, error) {
	query := c.Query(ctx).RecommendModelVersion
	opts := make([]repo.QueryOption, 0, 7)
	opts = append(opts, repo.Order(query.Scene.Asc()))
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))

	modelName := strings.TrimSpace(req.GetModelName())
	// 传入模型名称时，按模型名称模糊匹配推荐版本。
	if modelName != "" {
		opts = append(opts, repo.Where(query.ModelName.Like("%"+modelName+"%")))
	}

	modelType := strings.TrimSpace(req.GetModelType())
	// 传入模型类型时，按模型类型精确过滤推荐版本。
	if modelType != "" {
		opts = append(opts, repo.Where(query.ModelType.Eq(modelType)))
	}

	version := strings.TrimSpace(req.GetVersion())
	// 传入版本号时，按版本号模糊匹配推荐版本。
	if version != "" {
		opts = append(opts, repo.Where(query.Version.Like("%"+version+"%")))
	}

	// 显式传入场景时，只返回目标场景的版本记录。
	if req.Scene != nil {
		opts = append(opts, repo.Where(query.Scene.Eq(int32(req.GetScene()))))
	}

	// 显式传入状态时，只返回目标状态的版本记录。
	if req.Status != nil {
		opts = append(opts, repo.Where(query.Status.Eq(int32(req.GetStatus()))))
	}

	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	result := make([]*adminApi.RecommendModelVersion, 0, len(list))
	for _, item := range list {
		// 空记录不继续参与响应组装。
		if item == nil {
			continue
		}
		result = append(result, toAdminRecommendModelVersion(item))
	}
	return &adminApi.PageRecommendModelVersionResponse{
		List:  result,
		Total: int32(total),
	}, nil
}

// PublishRecommendModelVersion 发布推荐版本。
func (c *RecommendModelVersionCase) PublishRecommendModelVersion(ctx context.Context, req *adminApi.UpdateRecommendModelVersionPublishRequest) (*adminApi.UpdateRecommendModelVersionPublishResponse, error) {
	if req.GetId() <= 0 {
		return nil, errorsx.InvalidArgument("推荐版本ID不能为空")
	}

	entity, err := c.FindById(ctx, req.GetId())
	if err != nil {
		// 目标版本不存在时，直接返回资源不存在错误。
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.ResourceNotFound("推荐版本不存在").WithCause(err)
		}
		return nil, err
	}

	args, err := buildRecommendModelVersionPublishArgs(entity, req)
	if err != nil {
		return nil, err
	}
	summary, err := c.recommendModelVersionPublishTask.Exec(args)
	if err != nil {
		return nil, err
	}
	return &adminApi.UpdateRecommendModelVersionPublishResponse{
		Summary: summary,
	}, nil
}

// buildRecommendModelVersionPublishArgs 组装发布任务参数。
func buildRecommendModelVersionPublishArgs(entity *models.RecommendModelVersion, req *adminApi.UpdateRecommendModelVersionPublishRequest) (map[string]string, error) {
	if entity == nil || entity.ID <= 0 {
		return nil, errorsx.InvalidArgument("推荐版本不存在")
	}

	args := map[string]string{
		"scene": strconv.Itoa(int(entity.Scene)),
	}
	action := req.GetAction()
	// 未显式传发布动作时，默认按正式发布所选版本处理。
	if action == adminApi.RecommendModelVersionPublishAction_RECOMMEND_MODEL_VERSION_PUBLISH_ACTION_UNKNOWN {
		action = adminApi.RecommendModelVersionPublishAction_RECOMMEND_MODEL_VERSION_PUBLISH_ACTION_PUBLISH
	}

	switch action {
	// 正式发布时，目标版本、模型名和模型类型都按当前选中记录回填，避免同版本歧义。
	case adminApi.RecommendModelVersionPublishAction_RECOMMEND_MODEL_VERSION_PUBLISH_ACTION_PUBLISH:
		args["version"] = strings.TrimSpace(entity.Version)
		args["modelName"] = strings.TrimSpace(entity.ModelName)
		args["modelType"] = strings.TrimSpace(entity.ModelType)
	// 设置回滚版本时，未显式传回滚版本则默认把当前选中版本设为回滚目标。
	case adminApi.RecommendModelVersionPublishAction_RECOMMEND_MODEL_VERSION_PUBLISH_ACTION_SET_ROLLBACK:
		rollbackVersion := strings.TrimSpace(req.GetRollbackVersion())
		if rollbackVersion == "" {
			rollbackVersion = strings.TrimSpace(entity.Version)
		}
		args["rollbackVersion"] = rollbackVersion
	// 清空回滚版本时，只透传 clearRollback 标记即可。
	case adminApi.RecommendModelVersionPublishAction_RECOMMEND_MODEL_VERSION_PUBLISH_ACTION_CLEAR_ROLLBACK:
		args["clearRollback"] = "true"
	default:
		return nil, errorsx.InvalidArgument(fmt.Sprintf("不支持的发布动作：%s", action.String()))
	}

	cacheVersion := strings.TrimSpace(req.GetCacheVersion())
	// 显式传入缓存版本时，按请求覆盖任务默认行为。
	if cacheVersion != "" {
		args["cacheVersion"] = cacheVersion
	}

	// 显式传入灰度比例时，按请求更新发布配置。
	if req.GrayRatio != nil {
		args["grayRatio"] = strconv.FormatFloat(req.GetGrayRatio(), 'f', -1, 64)
	}

	publishedBy := strings.TrimSpace(req.GetPublishedBy())
	// 传入发布人时，回写最近一次发布人的审计信息。
	if publishedBy != "" {
		args["publishedBy"] = publishedBy
	}

	publishedReason := strings.TrimSpace(req.GetPublishedReason())
	// 传入发布说明时，回写最近一次发布说明。
	if publishedReason != "" {
		args["publishedReason"] = publishedReason
	}
	return args, nil
}

// parseRecommendModelVersionConfigJSON 解析推荐版本配置 JSON。
func parseRecommendModelVersionConfigJSON(configJSON string) *recommendDomain.StrategyVersionConfig {
	config := &recommendDomain.StrategyVersionConfig{}
	trimmedConfigJSON := strings.TrimSpace(configJSON)
	// 原始配置为空时，直接返回空配置对象。
	if trimmedConfigJSON == "" {
		return config
	}
	err := json.Unmarshal([]byte(trimmedConfigJSON), config)
	if err != nil {
		log.Errorf("parseRecommendModelVersionConfigJSON %v", err)
		return &recommendDomain.StrategyVersionConfig{}
	}
	return config
}

// toAdminRecommendModelVersion 将版本记录转换为后台分页项。
func toAdminRecommendModelVersion(entity *models.RecommendModelVersion) *adminApi.RecommendModelVersion {
	if entity == nil {
		return nil
	}
	config := parseRecommendModelVersionConfigJSON(entity.ConfigJSON)
	effectiveVersion := recommendCache.NormalizeVersion(entity.Version)
	// 存在发布配置时，优先按发布配置推导当前生效版本。
	if config.Publish != nil {
		effectiveVersion = recommendCache.NormalizeVersion(config.Publish.ResolveEffectiveVersion(entity.Version))
	}
	return &adminApi.RecommendModelVersion{
		Id:               entity.ID,
		ModelName:        entity.ModelName,
		ModelType:        entity.ModelType,
		Version:          entity.Version,
		Scene:            commonApi.RecommendScene(entity.Scene),
		Status:           commonApi.Status(entity.Status),
		EffectiveVersion: effectiveVersion,
		ConfigJson:       entity.ConfigJSON,
		Publish:          toAdminRecommendModelVersionPublishConfig(config.Publish),
		Tune:             toAdminRecommendModelVersionTuneConfig(config.Tune),
		CreatedAt:        formatAdminTime(entity.CreatedAt),
		UpdatedAt:        formatAdminTime(entity.UpdatedAt),
	}
}

// toAdminRecommendModelVersionPublishConfig 转换发布配置。
func toAdminRecommendModelVersionPublishConfig(config *recommendDomain.PublishStrategy) *adminApi.RecommendModelVersionPublishConfig {
	if config == nil {
		return nil
	}
	return &adminApi.RecommendModelVersionPublishConfig{
		CacheVersion:    strings.TrimSpace(config.CacheVersion),
		RollbackVersion: strings.TrimSpace(config.RollbackVersion),
		GrayRatio:       config.GrayRatio,
		PublishedBy:     strings.TrimSpace(config.PublishedBy),
		PublishedReason: strings.TrimSpace(config.PublishedReason),
		PublishedAt:     strings.TrimSpace(config.PublishedAt),
	}
}

// toAdminRecommendModelVersionTuneConfig 转换调参配置。
func toAdminRecommendModelVersionTuneConfig(config *recommendDomain.TuneStrategy) *adminApi.RecommendModelVersionTuneConfig {
	if config == nil {
		return nil
	}
	return &adminApi.RecommendModelVersionTuneConfig{
		Enabled:      config.Enabled,
		TargetMetric: strings.TrimSpace(config.TargetMetric),
		TrialCount:   config.TrialCount,
		Latest:       toAdminRecommendModelVersionTuneLatestSummary(config.Latest),
		LatestEval:   toAdminRecommendModelVersionTuneLatestEvalSummary(config.LatestEval),
	}
}

// toAdminRecommendModelVersionTuneLatestSummary 转换最近一次训练摘要。
func toAdminRecommendModelVersionTuneLatestSummary(summary *recommendDomain.TuneLatestSummary) *adminApi.RecommendModelVersionTuneLatestSummary {
	if summary == nil {
		return nil
	}
	score := make(map[string]float64, len(summary.Score))
	for key, value := range summary.Score {
		normalizedKey := strings.TrimSpace(strings.ToLower(key))
		// 指标名为空时，不继续写入后台展示结果。
		if normalizedKey == "" {
			continue
		}
		score[normalizedKey] = value
	}
	return &adminApi.RecommendModelVersionTuneLatestSummary{
		Task:        strings.TrimSpace(summary.Task),
		ModelType:   strings.TrimSpace(summary.ModelType),
		Backend:     strings.TrimSpace(summary.Backend),
		ArtifactDir: strings.TrimSpace(summary.ArtifactDir),
		TrainedAt:   strings.TrimSpace(summary.TrainedAt),
		Version:     strings.TrimSpace(summary.Version),
		Versions:    append([]string{}, summary.Versions...),
		BestValue:   summary.BestValue,
		Score:       score,
	}
}

// toAdminRecommendModelVersionTuneLatestEvalSummary 转换最近一次评估摘要。
func toAdminRecommendModelVersionTuneLatestEvalSummary(summary *recommendDomain.TuneLatestEvalSummary) *adminApi.RecommendModelVersionTuneLatestEvalSummary {
	if summary == nil {
		return nil
	}
	return &adminApi.RecommendModelVersionTuneLatestEvalSummary{
		ReportDate:    strings.TrimSpace(summary.ReportDate),
		StrategyName:  strings.TrimSpace(summary.StrategyName),
		SampleSize:    summary.SampleSize,
		RequestCount:  summary.RequestCount,
		ExposureCount: summary.ExposureCount,
		ClickCount:    summary.ClickCount,
		OrderCount:    summary.OrderCount,
		PayCount:      summary.PayCount,
		Ctr:           summary.Ctr,
		Cvr:           summary.Cvr,
		Ndcg:          summary.Ndcg,
		Precision:     summary.Precision,
		Recall:        summary.Recall,
	}
}

// formatAdminTime 格式化后台时间字符串。
func formatAdminTime(value time.Time) string {
	// 零时间不输出字符串，避免前端误判为合法时间。
	if value.IsZero() {
		return ""
	}
	return value.Format(time.DateTime)
}

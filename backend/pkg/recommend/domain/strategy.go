package domain

import (
	"strings"
	"time"
)

const (
	// RankerTypeNone 表示不启用模型精排，命名与 参考实现 的 ranker.type 保持一致。
	RankerTypeNone = "none"
	// RankerTypeFM 表示使用 FM 作为模型精排类型，命名与 参考实现 的 ranker.type 保持一致。
	RankerTypeFM = "fm"
)

// RankerStrategy 表示阶段 7 的模型精排配置。
type RankerStrategy struct {
	Enabled bool    `json:"enabled"` // 是否启用当前模型精排阶段。
	Type    string  `json:"type"`    // 当前模型精排类型，当前与 参考实现 对齐为 none / fm。
	TopN    int64   `json:"top_n"`   // 当前阶段只处理粗排 TopN。
	Weight  float64 `json:"weight"`  // 当前阶段写回最终分时的权重。
}

// NormalizeType 返回归一化后的模型精排类型。
func (c *RankerStrategy) NormalizeType() string {
	normalizedType := strings.TrimSpace(strings.ToLower(c.Type))
	// 当前类型为空时，统一回退为不启用精排。
	if normalizedType == "" {
		return RankerTypeNone
	}
	return normalizedType
}

// IsEnabled 判断当前模型精排阶段是否有效启用。
func (c *RankerStrategy) IsEnabled() bool {
	// 显式关闭或类型为 none 时，都不进入模型精排阶段。
	return c != nil && c.Enabled && c.NormalizeType() != RankerTypeNone
}

// ResolveTopN 返回当前模型精排阶段实际使用的 TopN。
func (c *RankerStrategy) ResolveTopN(defaultTopN int64) int64 {
	// 未配置或配置非法时，回退到调用方给定的默认窗口。
	if c == nil || c.TopN <= 0 {
		return defaultTopN
	}
	return c.TopN
}

// ResolveWeight 返回当前模型精排阶段实际使用的权重。
func (c *RankerStrategy) ResolveWeight(defaultWeight float64) float64 {
	// 权重非法时，回退到调用方给定的默认值。
	if c == nil || c.Weight <= 0 {
		return defaultWeight
	}
	return c.Weight
}

// LlmRerankStrategy 表示阶段 7 的 LLM TopN 二次重排配置。
type LlmRerankStrategy struct {
	Enabled             bool     `json:"enabled"`               // 是否启用当前 LLM 重排阶段。
	Model               string   `json:"model"`                 // 当前 LLM 重排所用模型名。
	TopN                int64    `json:"top_n"`                 // 当前阶段只处理前 TopN 候选。
	Weight              float64  `json:"weight"`                // 当前阶段写回最终分时的权重。
	CacheTTLSeconds     int64    `json:"cache_ttl_seconds"`     // 当前阶段结果缓存秒数。
	SystemPrompt        string   `json:"system_prompt"`         // 当前阶段使用的系统提示词。
	PromptTemplate      string   `json:"prompt_template"`       // 当前阶段使用的用户提示词模板。
	CandidateFilterExpr string   `json:"candidate_filter_expr"` // 当前阶段候选过滤表达式。
	ScoreExpr           string   `json:"score_expr"`            // 当前阶段分数表达式。
	ScoreScript         string   `json:"score_script"`          // 当前阶段分数脚本。
	TimeoutSeconds      int64    `json:"timeout_seconds"`       // 当前阶段在线调用超时秒数。
	MaxCompletionTokens int64    `json:"max_completion_tokens"` // 当前阶段最大输出 token 数。
	Temperature         *float64 `json:"temperature,omitempty"` // 当前阶段温度参数，显式传 0 表示尽量稳定输出。
}

// IsEnabled 判断当前 LLM 重排阶段是否有效启用。
func (c *LlmRerankStrategy) IsEnabled() bool {
	return c != nil && c.Enabled
}

// ResolveTopN 返回当前 LLM 重排阶段实际使用的 TopN。
func (c *LlmRerankStrategy) ResolveTopN(defaultTopN int64) int64 {
	// 未配置或配置非法时，回退到调用方给定的默认窗口。
	if c == nil || c.TopN <= 0 {
		return defaultTopN
	}
	return c.TopN
}

// ResolveWeight 返回当前 LLM 重排阶段实际使用的权重。
func (c *LlmRerankStrategy) ResolveWeight(defaultWeight float64) float64 {
	// 权重非法时，回退到调用方给定的默认值。
	if c == nil || c.Weight <= 0 {
		return defaultWeight
	}
	return c.Weight
}

// ResolveModel 返回当前 LLM 重排阶段实际使用的模型名。
func (c *LlmRerankStrategy) ResolveModel(defaultModel string) string {
	// 当前未配置模型时，回退到调用方给定的默认模型。
	if c == nil || strings.TrimSpace(c.Model) == "" {
		return strings.TrimSpace(defaultModel)
	}
	return strings.TrimSpace(c.Model)
}

// ResolveTimeout 返回当前 LLM 重排阶段实际使用的超时时间。
func (c *LlmRerankStrategy) ResolveTimeout(defaultTimeout time.Duration) time.Duration {
	// 未配置或配置非法时，回退到调用方给定的默认超时。
	if c == nil || c.TimeoutSeconds <= 0 {
		return defaultTimeout
	}
	return time.Duration(c.TimeoutSeconds) * time.Second
}

// ResolveMaxCompletionTokens 返回当前 LLM 重排阶段实际使用的最大输出 token 数。
func (c *LlmRerankStrategy) ResolveMaxCompletionTokens(defaultTokens int64) int64 {
	// 未配置或配置非法时，回退到调用方给定的默认值。
	if c == nil || c.MaxCompletionTokens <= 0 {
		return defaultTokens
	}
	return c.MaxCompletionTokens
}

// ResolveTemperature 返回当前 LLM 重排阶段实际使用的温度参数。
func (c *LlmRerankStrategy) ResolveTemperature(defaultTemperature float64) float64 {
	// 当前未显式配置温度时，回退到调用方给定的默认值。
	if c == nil || c.Temperature == nil {
		return defaultTemperature
	}
	return *c.Temperature
}

// TuneLatestSummary 表示最近一次真实训练的摘要快照。
type TuneLatestSummary struct {
	Task        string             `json:"task,omitempty"`         // 最近一次训练对应的任务名。
	ModelType   string             `json:"model_type,omitempty"`   // 最近一次训练输出的模型类型。
	Backend     string             `json:"backend,omitempty"`      // 最近一次训练使用的后端。
	ArtifactDir string             `json:"artifact_dir,omitempty"` // 最近一次训练产物目录。
	TrainedAt   string             `json:"trained_at,omitempty"`   // 最近一次训练完成时间。
	Version     string             `json:"version,omitempty"`      // 单版本训练任务的版本号。
	Versions    []string           `json:"versions,omitempty"`     // 多版本训练任务涉及的版本列表。
	BestValue   float64            `json:"best_value"`             // 最近一次训练最优目标值。
	Score       map[string]float64 `json:"score,omitempty"`        // 最近一次训练验证指标。
}

// BuildContext 构建最近一次真实训练的调试上下文。
func (c *TuneLatestSummary) BuildContext() map[string]any {
	// 最近一次训练摘要为空时，不生成调试上下文。
	if c == nil {
		return map[string]any{}
	}
	contextMap := map[string]any{
		"bestValue": c.BestValue,
	}
	if strings.TrimSpace(c.Task) != "" {
		contextMap["task"] = strings.TrimSpace(c.Task)
	}
	if strings.TrimSpace(c.ModelType) != "" {
		contextMap["modelType"] = strings.TrimSpace(strings.ToLower(c.ModelType))
	}
	if strings.TrimSpace(c.Backend) != "" {
		contextMap["backend"] = strings.TrimSpace(strings.ToLower(c.Backend))
	}
	if strings.TrimSpace(c.ArtifactDir) != "" {
		contextMap["artifactDir"] = strings.TrimSpace(c.ArtifactDir)
	}
	if strings.TrimSpace(c.TrainedAt) != "" {
		contextMap["trainedAt"] = strings.TrimSpace(c.TrainedAt)
	}
	if strings.TrimSpace(c.Version) != "" {
		contextMap["version"] = strings.TrimSpace(c.Version)
	}
	// 多版本训练任务才补版本列表，避免单版本上下文重复。
	if len(c.Versions) > 0 {
		contextMap["versions"] = append([]string{}, c.Versions...)
	}
	// 最近一次训练有验证指标时，再补充分数字段。
	if len(c.Score) > 0 {
		scoreMap := make(map[string]float64, len(c.Score))
		for key, value := range c.Score {
			normalizedKey := strings.TrimSpace(strings.ToLower(key))
			// 指标名为空时，不继续写入调试上下文。
			if normalizedKey == "" {
				continue
			}
			scoreMap[normalizedKey] = value
		}
		if len(scoreMap) > 0 {
			contextMap["score"] = scoreMap
		}
	}
	return contextMap
}

// TuneLatestEvalSummary 表示最近一次评估日报的摘要快照。
type TuneLatestEvalSummary struct {
	ReportDate    string  `json:"report_date,omitempty"`   // 最近一次评估报告日期。
	StrategyName  string  `json:"strategy_name,omitempty"` // 最近一次评估使用的策略名称。
	SampleSize    int64   `json:"sample_size"`             // 最近一次评估样本量。
	RequestCount  int64   `json:"request_count"`           // 最近一次评估推荐请求数。
	ExposureCount int64   `json:"exposure_count"`          // 最近一次评估曝光数。
	ClickCount    int64   `json:"click_count"`             // 最近一次评估点击数。
	OrderCount    int64   `json:"order_count"`             // 最近一次评估下单数。
	PayCount      int64   `json:"pay_count"`               // 最近一次评估支付数。
	Ctr           float64 `json:"ctr"`                     // 最近一次评估 CTR。
	Cvr           float64 `json:"cvr"`                     // 最近一次评估 CVR。
	Ndcg          float64 `json:"ndcg"`                    // 最近一次评估 NDCG。
	Precision     float64 `json:"precision"`               // 最近一次评估 Precision。
	Recall        float64 `json:"recall"`                  // 最近一次评估 Recall。
}

// BuildContext 构建最近一次评估日报的调试上下文。
func (c *TuneLatestEvalSummary) BuildContext() map[string]any {
	// 最近一次评估摘要为空时，不生成调试上下文。
	if c == nil {
		return map[string]any{}
	}
	contextMap := map[string]any{
		"sampleSize":    c.SampleSize,
		"requestCount":  c.RequestCount,
		"exposureCount": c.ExposureCount,
		"clickCount":    c.ClickCount,
		"orderCount":    c.OrderCount,
		"payCount":      c.PayCount,
		"ctr":           c.Ctr,
		"cvr":           c.Cvr,
		"ndcg":          c.Ndcg,
		"precision":     c.Precision,
		"recall":        c.Recall,
	}
	// 评估日期存在时，再补充报告日期，便于在线排障查看口径时间。
	if strings.TrimSpace(c.ReportDate) != "" {
		contextMap["reportDate"] = strings.TrimSpace(c.ReportDate)
	}
	// 策略名称存在时，再补充评估对应的策略标识。
	if strings.TrimSpace(c.StrategyName) != "" {
		contextMap["strategyName"] = strings.TrimSpace(c.StrategyName)
	}
	return contextMap
}

// TuneStrategy 表示阶段 8 的自动调参配置。
type TuneStrategy struct {
	Enabled      bool                   `json:"enabled"`               // 是否启用自动调参。
	TargetMetric string                 `json:"target_metric"`         // 当前自动调参优化目标，例如 ndcg / ctr / cvr。
	TrialCount   int32                  `json:"trial_count"`           // 当前自动调参尝试次数。
	Latest       *TuneLatestSummary     `json:"latest,omitempty"`      // 最近一次真实训练摘要。
	LatestEval   *TuneLatestEvalSummary `json:"latest_eval,omitempty"` // 最近一次评估日报摘要。
}

// BuildContext 构建自动调参调试上下文。
func (c *TuneStrategy) BuildContext() map[string]any {
	// 调参配置为空时，不生成调试上下文。
	if c == nil {
		return map[string]any{}
	}
	contextMap := map[string]any{
		"enabled":      c.Enabled,
		"targetMetric": strings.TrimSpace(strings.ToLower(c.TargetMetric)),
		"trialCount":   c.TrialCount,
	}
	// 最近一次训练摘要存在时，再把真实训练结果挂到调试上下文。
	if latestContext := c.Latest.BuildContext(); len(latestContext) > 0 {
		contextMap["latest"] = latestContext
	}
	// 最近一次评估摘要存在时，再把离线评估结果挂到调试上下文。
	if latestEvalContext := c.LatestEval.BuildContext(); len(latestEvalContext) > 0 {
		contextMap["latestEval"] = latestEvalContext
	}
	return contextMap
}

// PublishStrategy 表示阶段 8 的版本发布与回滚配置。
type PublishStrategy struct {
	CacheVersion    string  `json:"cache_version"`    // 当前版本希望读取的缓存版本。
	RollbackVersion string  `json:"rollback_version"` // 当前版本需要快速回滚时读取的目标版本。
	GrayRatio       float64 `json:"gray_ratio"`       // 当前版本灰度比例。
	PublishedBy     string  `json:"published_by"`     // 当前版本发布人。
	PublishedReason string  `json:"published_reason"` // 当前版本发布说明。
	PublishedAt     string  `json:"published_at"`     // 当前版本最近一次发布时间。
}

// ResolveEffectiveVersion 返回当前发布配置驱动下的实际读取版本。
func (c *PublishStrategy) ResolveEffectiveVersion(defaultVersion string) string {
	// 当前显式配置了回滚版本时，优先走回滚版本。
	if c != nil && strings.TrimSpace(c.RollbackVersion) != "" {
		return strings.TrimSpace(c.RollbackVersion)
	}
	// 当前显式配置了缓存版本时，继续读取配置指定版本。
	if c != nil && strings.TrimSpace(c.CacheVersion) != "" {
		return strings.TrimSpace(c.CacheVersion)
	}
	return strings.TrimSpace(defaultVersion)
}

// StrategyVersionConfig 表示版本配置中的扩展策略字段。
type StrategyVersionConfig struct {
	RecallProbe *RecallProbeStrategy `json:"recall_probe"` // 召回探针配置。
	Ranker      *RankerStrategy      `json:"ranker"`       // 模型精排配置。
	LlmRerank   *LlmRerankStrategy   `json:"llm_rerank"`   // LLM 二次重排配置。
	Tune        *TuneStrategy        `json:"tune"`         // 自动调参配置。
	Publish     *PublishStrategy     `json:"publish"`      // 发布与回滚配置。
}

// SceneStrategyContext 表示某个场景当前生效的在线策略上下文。
type SceneStrategyContext struct {
	Scene              int32                     // 当前推荐场景。
	Version            string                    // 当前场景启用版本。
	EffectiveVersion   string                    // 当前场景实际读取的缓存版本。
	VersionPublishedAt time.Time                 // 当前版本发布时间。
	Config             *StrategyVersionConfig    // 当前版本扩展策略配置。
	PublishResolution  *PublishVersionResolution // 当前请求命中的发布版本决策结果。
}

// BuildPublishContext 构建当前场景版本的发布调试上下文。
func (c *SceneStrategyContext) BuildPublishContext() map[string]any {
	// 策略上下文为空时，不生成发布调试上下文。
	if c == nil {
		return map[string]any{}
	}
	publishContext := map[string]any{
		"scene":            c.Scene,
		"sceneVersion":     c.Version,
		"effectiveVersion": c.EffectiveVersion,
	}
	// 当前版本发布时间存在时，再补充发布时间。
	if !c.VersionPublishedAt.IsZero() {
		publishContext["versionPublishedAt"] = c.VersionPublishedAt.Format(time.RFC3339Nano)
	}
	// 当前没有发布配置时，直接返回基础版本上下文。
	if c.Config == nil || c.Config.Publish == nil {
		return publishContext
	}
	if strings.TrimSpace(c.Config.Publish.CacheVersion) != "" {
		publishContext["cacheVersion"] = strings.TrimSpace(c.Config.Publish.CacheVersion)
	}
	if strings.TrimSpace(c.Config.Publish.RollbackVersion) != "" {
		publishContext["rollbackVersion"] = strings.TrimSpace(c.Config.Publish.RollbackVersion)
	}
	if c.PublishResolution != nil {
		if c.PublishResolution.BaselineVersion != "" {
			publishContext["baselineVersion"] = c.PublishResolution.BaselineVersion
		}
		if c.PublishResolution.GrayVersion != "" {
			publishContext["grayVersion"] = c.PublishResolution.GrayVersion
		}
		publishContext["grayRatio"] = c.PublishResolution.GrayRatio
		if c.PublishResolution.GrayEnabled {
			publishContext["grayEnabled"] = true
			publishContext["grayHit"] = c.PublishResolution.GrayHit
			if c.PublishResolution.GrayBucketResolved {
				publishContext["grayBucket"] = c.PublishResolution.GrayBucket
			}
		}
	} else {
		publishContext["grayRatio"] = c.Config.Publish.GrayRatio
	}
	if strings.TrimSpace(c.Config.Publish.PublishedBy) != "" {
		publishContext["publishedBy"] = strings.TrimSpace(c.Config.Publish.PublishedBy)
	}
	if strings.TrimSpace(c.Config.Publish.PublishedReason) != "" {
		publishContext["publishedReason"] = strings.TrimSpace(c.Config.Publish.PublishedReason)
	}
	if strings.TrimSpace(c.Config.Publish.PublishedAt) != "" {
		publishContext["publishedAt"] = strings.TrimSpace(c.Config.Publish.PublishedAt)
	}
	return publishContext
}

// BuildTuneContext 构建当前场景版本的调参调试上下文。
func (c *SceneStrategyContext) BuildTuneContext() map[string]any {
	// 策略上下文为空或没有调参配置时，不生成调参调试上下文。
	if c == nil || c.Config == nil || c.Config.Tune == nil {
		return map[string]any{}
	}
	return c.Config.Tune.BuildContext()
}

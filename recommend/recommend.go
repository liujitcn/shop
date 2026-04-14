package recommend

import (
	"fmt"
	"recommend/contract"
	"recommend/internal/core"
)

// Option 表示推荐实例初始化选项。
type Option func(*Recommend) error

// Recommend 表示商城推荐工具实例。
type Recommend struct {
	dependencies Dependencies
	config       Config
}

// New 创建推荐工具实例。
func New(options ...Option) (*Recommend, error) {
	instance := &Recommend{
		config: DefaultConfig(),
	}
	for _, option := range options {
		// 空 option 不参与实例初始化，避免调用方在条件拼装时额外判空。
		if option == nil {
			continue
		}
		err := option(instance)
		if err != nil {
			return nil, err
		}
	}
	err := validateConfig(instance.config)
	if err != nil {
		return nil, err
	}
	return instance, nil
}

// DefaultConfig 返回推荐工具的默认配置。
func DefaultConfig() Config {
	return cloneConfig(core.DefaultServiceConfig())
}

// WithDependencies 指定推荐实例使用的全部依赖。
func WithDependencies(dependencies Dependencies) Option {
	return func(instance *Recommend) error {
		instance.dependencies = dependencies
		return nil
	}
}

// WithConfig 指定推荐实例使用的完整配置。
func WithConfig(config Config) Option {
	return func(instance *Recommend) error {
		instance.config = cloneConfig(config)
		return nil
	}
}

// WithQueryConfig 指定在线请求默认值配置。
func WithQueryConfig(config QueryConfig) Option {
	return func(instance *Recommend) error {
		instance.config.Query = config
		return nil
	}
}

// WithRankingConfig 指定在线排序配置。
func WithRankingConfig(config RankingConfig) Option {
	return func(instance *Recommend) error {
		config.SceneWeights = cloneSceneWeights(config.SceneWeights)
		instance.config.Ranking = config
		return nil
	}
}

// WithMaterializeConfig 指定离线构建配置。
func WithMaterializeConfig(config MaterializeConfig) Option {
	return func(instance *Recommend) error {
		config.DefaultScenes = cloneScenes(config.DefaultScenes)
		instance.config.Materialize = config
		return nil
	}
}

// WithEvaluateConfig 指定离线评估配置。
func WithEvaluateConfig(config EvaluateConfig) Option {
	return func(instance *Recommend) error {
		instance.config.Evaluate = config
		return nil
	}
}

// WithExplainConfig 指定 explain 配置。
func WithExplainConfig(config ExplainConfig) Option {
	return func(instance *Recommend) error {
		instance.config.Explain = config
		return nil
	}
}

// WithSyncConfig 指定运行态同步配置。
func WithSyncConfig(config SyncConfig) Option {
	return func(instance *Recommend) error {
		instance.config.Sync = config
		return nil
	}
}

// WithStrategyConfig 指定策略层扩展配置。
func WithStrategyConfig(config StrategyConfig) Option {
	return func(instance *Recommend) error {
		config.NonPersonalizedSources = append([]string(nil), config.NonPersonalizedSources...)
		instance.config.Strategy = config
		return nil
	}
}

// WithTrainingConfig 指定训练链路扩展配置。
func WithTrainingConfig(config TrainingConfig) Option {
	return func(instance *Recommend) error {
		instance.config.Training = config
		return nil
	}
}

// WithVectorConfig 指定向量召回扩展配置。
func WithVectorConfig(config VectorConfig) Option {
	return func(instance *Recommend) error {
		instance.config.Vector = config
		return nil
	}
}

// WithVectorSource 指定向量召回数据源。
func WithVectorSource(source contract.VectorSource) Option {
	return func(instance *Recommend) error {
		instance.dependencies.Vector = source
		return nil
	}
}

// WithLlmReranker 指定 LLM 重排器依赖。
func WithLlmReranker(reranker contract.LlmReranker) Option {
	return func(instance *Recommend) error {
		instance.dependencies.Reranker = reranker
		return nil
	}
}

// WithGoodsSource 指定商品数据源。
func WithGoodsSource(source contract.GoodsSource) Option {
	return func(instance *Recommend) error {
		instance.dependencies.Goods = source
		return nil
	}
}

// WithUserSource 指定用户数据源。
func WithUserSource(source contract.UserSource) Option {
	return func(instance *Recommend) error {
		instance.dependencies.User = source
		return nil
	}
}

// WithOrderSource 指定订单数据源。
func WithOrderSource(source contract.OrderSource) Option {
	return func(instance *Recommend) error {
		instance.dependencies.Order = source
		return nil
	}
}

// WithBehaviorSource 指定行为数据源。
func WithBehaviorSource(source contract.BehaviorSource) Option {
	return func(instance *Recommend) error {
		instance.dependencies.Behavior = source
		return nil
	}
}

// WithRecommendSource 指定推荐事实数据源。
func WithRecommendSource(source contract.RecommendSource) Option {
	return func(instance *Recommend) error {
		instance.dependencies.Recommend = source
		return nil
	}
}

// WithCacheSource 指定缓存布局数据源。
func WithCacheSource(source contract.CacheSource) Option {
	return func(instance *Recommend) error {
		instance.dependencies.Cache = source
		return nil
	}
}

// validateConfig 校验推荐实例配置。
func validateConfig(config Config) error {
	switch config.Ranking.Mode {
	case "", RankingModeRule, RankingModeFm, RankingModeLlm, RankingModeCustom:
	default:
		return fmt.Errorf("recommend: 不支持的排序模式 %q", config.Ranking.Mode)
	}
	if config.Ranking.LlmBlendWeight < 0 || config.Ranking.LlmBlendWeight > 1 {
		return fmt.Errorf("recommend: llm 融合权重必须位于 0 到 1 之间，当前值=%v", config.Ranking.LlmBlendWeight)
	}
	if config.Training.MinSampleCount < 0 {
		return fmt.Errorf("recommend: 最小训练样本数不能为负数，当前值=%d", config.Training.MinSampleCount)
	}
	if config.Training.Epochs < 0 {
		return fmt.Errorf("recommend: 训练轮数不能为负数，当前值=%d", config.Training.Epochs)
	}
	if config.Training.LearningRate < 0 {
		return fmt.Errorf("recommend: 训练学习率不能为负数，当前值=%v", config.Training.LearningRate)
	}
	return nil
}

// cloneConfig 复制推荐配置，避免不同实例共享 map / slice。
func cloneConfig(config Config) Config {
	result := config
	result.Ranking.SceneWeights = cloneSceneWeights(config.Ranking.SceneWeights)
	result.Materialize.DefaultScenes = cloneScenes(config.Materialize.DefaultScenes)
	result.Strategy.NonPersonalizedSources = append([]string(nil), config.Strategy.NonPersonalizedSources...)
	return result
}

// cloneSceneWeights 复制场景权重配置。
func cloneSceneWeights(sceneWeights map[Scene]ScoreWeights) map[Scene]ScoreWeights {
	if len(sceneWeights) == 0 {
		return nil
	}
	result := make(map[Scene]ScoreWeights, len(sceneWeights))
	for scene, weights := range sceneWeights {
		result[scene] = weights
	}
	return result
}

// cloneScenes 复制场景列表。
func cloneScenes(scenes []Scene) []Scene {
	if len(scenes) == 0 {
		return nil
	}
	return append([]Scene(nil), scenes...)
}

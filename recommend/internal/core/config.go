package core

const (
	// defaultPageNumValue 表示默认页码。
	defaultPageNumValue = 1
	// defaultPageSizeValue 表示默认分页大小。
	defaultPageSizeValue = 10
	// defaultBuildLimitValue 表示离线构建默认候选上限。
	defaultBuildLimitValue = 50
	// defaultNeighborLimitValue 表示相似用户构建默认邻居上限。
	defaultNeighborLimitValue = 20
	// defaultEvaluateTopKValue 表示离线评估默认 topK。
	defaultEvaluateTopKValue = 10
	// defaultMaxPerCategoryValue 表示默认类目打散上限。
	defaultMaxPerCategoryValue = 2
	// defaultLlmCandidateLimitValue 表示 LLM 重排默认候选上限。
	defaultLlmCandidateLimitValue = 20
	// defaultRecentGoodsCountValue 表示运行态保留的默认最近商品数量。
	defaultRecentGoodsCountValue = 20
	// defaultExposurePenaltyValue 表示单次曝光默认惩罚值。
	defaultExposurePenaltyValue = 0.2
	// defaultOrderCreatePenaltyValue 表示下单默认惩罚值。
	defaultOrderCreatePenaltyValue = 0.3
	// defaultOrderPayPenaltyValue 表示支付默认惩罚值。
	defaultOrderPayPenaltyValue = 0.6
	// defaultLlmBlendWeightValue 表示规则分与 LLM 分的默认融合权重。
	defaultLlmBlendWeightValue = 0.7
	// defaultTrainingMinSamplesValue 表示学习排序默认最小样本量。
	defaultTrainingMinSamplesValue = 10
	// defaultTrainingEpochsValue 表示学习排序默认训练轮数。
	defaultTrainingEpochsValue = 30
	// defaultTrainingRateValue 表示学习排序默认学习率。
	defaultTrainingRateValue = 0.08
	// defaultVectorRecallLimitValue 表示向量召回默认候选上限。
	defaultVectorRecallLimitValue = 20
)

// RankingMode 表示在线排序模式。
type RankingMode string

const (
	// RankingModeRule 表示使用规则权重排序。
	RankingModeRule RankingMode = "rule"
	// RankingModeFm 表示使用轻量 FM 二阶特征交叉模型排序。
	RankingModeFm RankingMode = "fm"
	// RankingModeLlm 表示使用 LLM 重排模式排序。
	RankingModeLlm RankingMode = "llm"
	// RankingModeCustom 表示预留给外部自定义排序器的模式标识。
	RankingModeCustom RankingMode = "custom"
)

// ScoreWeights 表示排序阶段使用的各路信号权重。
type ScoreWeights struct {
	// RelationWeight 表示商品关联召回信号的权重。
	RelationWeight float64
	// UserGoodsWeight 表示用户商品偏好信号的权重。
	UserGoodsWeight float64
	// CategoryWeight 表示用户类目偏好信号的权重。
	CategoryWeight float64
	// SceneHotWeight 表示场景热度信号的权重。
	SceneHotWeight float64
	// GlobalHotWeight 表示全站热度信号的权重。
	GlobalHotWeight float64
	// FreshnessWeight 表示商品新鲜度信号的权重。
	FreshnessWeight float64
	// SessionWeight 表示会话上下文信号的权重。
	SessionWeight float64
	// ExternalWeight 表示外部推荐池信号的权重。
	ExternalWeight float64
	// CollaborativeWeight 表示协同过滤信号的权重。
	CollaborativeWeight float64
	// UserNeighborWeight 表示相似用户信号的权重。
	UserNeighborWeight float64
	// VectorWeight 表示向量召回信号的权重。
	VectorWeight float64
	// ExposurePenalty 表示曝光惩罚的扣分权重。
	ExposurePenalty float64
	// RepeatPenalty 表示重复购买惩罚的扣分权重。
	RepeatPenalty float64
}

// QueryConfig 表示在线请求默认值配置。
type QueryConfig struct {
	// DefaultPageNum 表示未显式传入页码时使用的默认页码。
	DefaultPageNum int32
	// DefaultPageSize 表示未显式传入分页大小时使用的默认分页大小。
	DefaultPageSize int32
	// DefaultExplain 表示未显式指定时是否默认持久化 explain。
	DefaultExplain bool
}

// RankingConfig 表示在线排序配置。
type RankingConfig struct {
	// Mode 表示当前启用的排序模式。
	Mode RankingMode
	// MaxPerCategory 表示单个类目在主结果区允许出现的最大数量。
	MaxPerCategory int
	// LlmCandidateLimit 表示 LLM 重排阶段允许接收的候选数量上限。
	LlmCandidateLimit int32
	// LlmBlendWeight 表示规则分与 LLM 分的融合权重，取值范围为 0 到 1。
	LlmBlendWeight float64
	// SceneWeights 表示按场景覆写的排序权重。
	SceneWeights map[Scene]ScoreWeights
}

// MaterializeConfig 表示离线构建与物化默认配置。
type MaterializeConfig struct {
	// DefaultLimit 表示离线构建候选池时使用的默认商品数量。
	DefaultLimit int32
	// DefaultNeighborLimit 表示相似用户构建时使用的默认邻居数量。
	DefaultNeighborLimit int32
	// DefaultScenes 表示未显式指定时参与离线构建的默认场景集合。
	DefaultScenes []Scene
	// EnableEvaluateAfterRebuild 表示调用重建入口后是否自动执行离线评估。
	EnableEvaluateAfterRebuild bool
}

// EvaluateConfig 表示离线评估默认配置。
type EvaluateConfig struct {
	// DefaultTopK 表示未显式指定时离线评估使用的默认 topK。
	DefaultTopK int32
}

// ExplainConfig 表示 explain 链路配置。
type ExplainConfig struct {
	// StrictTracePersistence 表示显式请求 explain 时，trace 持久化失败是否直接返回错误。
	StrictTracePersistence bool
}

// SyncConfig 表示运行态同步配置。
type SyncConfig struct {
	// MaxRecentGoodsCount 表示最近行为序列保留的最大商品数量。
	MaxRecentGoodsCount int
	// ExposurePenalty 表示单次曝光事件追加的默认惩罚值。
	ExposurePenalty float64
	// OrderCreatePenalty 表示创建订单行为追加的默认复购惩罚值。
	OrderCreatePenalty float64
	// OrderPayPenalty 表示支付完成行为追加的默认复购惩罚值。
	OrderPayPenalty float64
}

// StrategyConfig 表示策略层扩展配置。
type StrategyConfig struct {
	// NonPersonalizedSources 表示非个性化构建默认合并的来源顺序。
	NonPersonalizedSources []string
	// EnableSourceScores 表示离线候选池是否保留各来源原始分值。
	EnableSourceScores bool
	// EnableExternalFallback 表示外部池缺失时是否回退事实源。
	EnableExternalFallback bool
}

// TrainingConfig 表示学习排序训练配置。
type TrainingConfig struct {
	// CollaborativeMode 表示协同过滤训练模式标识。
	CollaborativeMode string
	// CtrMode 表示 CTR / FM 训练模式标识。
	CtrMode string
	// EnableOptimization 表示是否启用训练优化流程。
	EnableOptimization bool
	// MinSampleCount 表示单个场景开始训练前要求的最小样本量。
	MinSampleCount int32
	// Epochs 表示轻量模型训练的迭代轮数。
	Epochs int32
	// LearningRate 表示轻量模型训练使用的学习率。
	LearningRate float64
}

// VectorConfig 表示向量召回配置。
type VectorConfig struct {
	// Enabled 表示是否启用向量召回链路。
	Enabled bool
	// RecallLimit 表示向量召回单次拉取的候选数量上限。
	RecallLimit int32
	// ProviderName 表示当前使用的向量提供方标识。
	ProviderName string
}

// ServiceConfig 表示推荐实例的总配置。
type ServiceConfig struct {
	// Query 表示在线请求默认值配置。
	Query QueryConfig
	// Ranking 表示在线排序配置。
	Ranking RankingConfig
	// Materialize 表示离线构建配置。
	Materialize MaterializeConfig
	// Evaluate 表示离线评估配置。
	Evaluate EvaluateConfig
	// Explain 表示 explain 相关配置。
	Explain ExplainConfig
	// Sync 表示运行态同步配置。
	Sync SyncConfig
	// Strategy 表示多路召回与离线池策略配置。
	Strategy StrategyConfig
	// Training 表示训练链路扩展配置。
	Training TrainingConfig
	// Vector 表示向量召回扩展配置。
	Vector VectorConfig
}

// DefaultSceneWeights 返回默认场景权重表。
func DefaultSceneWeights() map[Scene]ScoreWeights {
	return map[Scene]ScoreWeights{
		SceneHome: {
			UserGoodsWeight:     0.24,
			CategoryWeight:      0.16,
			SceneHotWeight:      0.14,
			GlobalHotWeight:     0.08,
			FreshnessWeight:     0.08,
			SessionWeight:       0.04,
			ExternalWeight:      0.10,
			CollaborativeWeight: 0.08,
			UserNeighborWeight:  0.08,
			VectorWeight:        0.10,
			ExposurePenalty:     1.0,
			RepeatPenalty:       1.0,
		},
		SceneGoodsDetail: {
			RelationWeight:      0.42,
			SceneHotWeight:      0.10,
			FreshnessWeight:     0.06,
			SessionWeight:       0.16,
			ExternalWeight:      0.12,
			CollaborativeWeight: 0.08,
			UserNeighborWeight:  0.04,
			VectorWeight:        0.12,
			ExposurePenalty:     1.0,
			RepeatPenalty:       1.0,
		},
		SceneCart: {
			RelationWeight:      0.34,
			SceneHotWeight:      0.12,
			FreshnessWeight:     0.06,
			SessionWeight:       0.18,
			ExternalWeight:      0.12,
			CollaborativeWeight: 0.08,
			UserNeighborWeight:  0.04,
			VectorWeight:        0.12,
			ExposurePenalty:     1.0,
			RepeatPenalty:       1.0,
		},
		SceneProfile: {
			UserGoodsWeight:     0.22,
			CategoryWeight:      0.18,
			GlobalHotWeight:     0.08,
			FreshnessWeight:     0.08,
			ExternalWeight:      0.10,
			CollaborativeWeight: 0.12,
			UserNeighborWeight:  0.12,
			VectorWeight:        0.10,
			ExposurePenalty:     1.0,
			RepeatPenalty:       1.0,
		},
		SceneOrderDetail: {
			RelationWeight:      0.38,
			SceneHotWeight:      0.14,
			FreshnessWeight:     0.08,
			ExternalWeight:      0.14,
			CollaborativeWeight: 0.10,
			UserNeighborWeight:  0.06,
			VectorWeight:        0.10,
			ExposurePenalty:     1.0,
			RepeatPenalty:       1.0,
		},
		SceneOrderPaid: {
			RelationWeight:      0.24,
			SceneHotWeight:      0.12,
			FreshnessWeight:     0.08,
			ExternalWeight:      0.16,
			CollaborativeWeight: 0.14,
			UserNeighborWeight:  0.14,
			VectorWeight:        0.10,
			ExposurePenalty:     1.0,
			RepeatPenalty:       1.2,
		},
	}
}

// DefaultServiceConfig 返回推荐实例的默认配置。
func DefaultServiceConfig() ServiceConfig {
	return ServiceConfig{
		Query: QueryConfig{
			DefaultPageNum:  defaultPageNumValue,
			DefaultPageSize: defaultPageSizeValue,
		},
		Ranking: RankingConfig{
			Mode:              RankingModeRule,
			MaxPerCategory:    defaultMaxPerCategoryValue,
			LlmCandidateLimit: defaultLlmCandidateLimitValue,
			LlmBlendWeight:    defaultLlmBlendWeightValue,
			SceneWeights:      DefaultSceneWeights(),
		},
		Materialize: MaterializeConfig{
			DefaultLimit:         defaultBuildLimitValue,
			DefaultNeighborLimit: defaultNeighborLimitValue,
			DefaultScenes: []Scene{
				SceneHome,
				SceneGoodsDetail,
				SceneCart,
				SceneProfile,
				SceneOrderDetail,
				SceneOrderPaid,
			},
		},
		Evaluate: EvaluateConfig{
			DefaultTopK: defaultEvaluateTopKValue,
		},
		Explain: ExplainConfig{
			StrictTracePersistence: true,
		},
		Sync: SyncConfig{
			MaxRecentGoodsCount: defaultRecentGoodsCountValue,
			ExposurePenalty:     defaultExposurePenaltyValue,
			OrderCreatePenalty:  defaultOrderCreatePenaltyValue,
			OrderPayPenalty:     defaultOrderPayPenaltyValue,
		},
		Strategy: StrategyConfig{
			NonPersonalizedSources: []string{"latest", "scene_hot", "global_hot"},
			EnableSourceScores:     true,
			EnableExternalFallback: true,
		},
		Training: TrainingConfig{
			CollaborativeMode:  "fact_pool",
			CtrMode:            "light_fm",
			EnableOptimization: true,
			MinSampleCount:     defaultTrainingMinSamplesValue,
			Epochs:             defaultTrainingEpochsValue,
			LearningRate:       defaultTrainingRateValue,
		},
		Vector: VectorConfig{
			Enabled:     true,
			RecallLimit: defaultVectorRecallLimitValue,
		},
	}
}

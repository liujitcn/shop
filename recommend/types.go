package recommend

import "recommend/internal/core"

// Dependencies 定义推荐工具运行所需的数据契约集合。
type Dependencies = core.Dependencies

// Config 定义推荐实例的总配置。
type Config = core.ServiceConfig

// QueryConfig 定义在线请求默认值配置。
type QueryConfig = core.QueryConfig

// RankingMode 表示在线排序模式。
type RankingMode = core.RankingMode

const (
	RankingModeRule   = core.RankingModeRule
	RankingModeFm     = core.RankingModeFm
	RankingModeLlm    = core.RankingModeLlm
	RankingModeCustom = core.RankingModeCustom
)

// ScoreWeights 定义排序阶段的各路信号权重。
type ScoreWeights = core.ScoreWeights

// RankingConfig 定义在线排序配置。
type RankingConfig = core.RankingConfig

// MaterializeConfig 定义离线构建配置。
type MaterializeConfig = core.MaterializeConfig

// EvaluateConfig 定义离线评估配置。
type EvaluateConfig = core.EvaluateConfig

// ExplainConfig 定义 explain 链路配置。
type ExplainConfig = core.ExplainConfig

// SyncConfig 定义运行态同步配置。
type SyncConfig = core.SyncConfig

// StrategyConfig 定义策略层扩展配置。
type StrategyConfig = core.StrategyConfig

// TrainingConfig 定义学习排序训练配置。
type TrainingConfig = core.TrainingConfig

// VectorConfig 定义向量召回配置。
type VectorConfig = core.VectorConfig

// Scene 表示商城推荐场景。
type Scene = core.Scene

const (
	SceneHome        = core.SceneHome
	SceneGoodsDetail = core.SceneGoodsDetail
	SceneCart        = core.SceneCart
	SceneProfile     = core.SceneProfile
	SceneOrderDetail = core.SceneOrderDetail
	SceneOrderPaid   = core.SceneOrderPaid
)

// ActorType 表示推荐主体类型。
type ActorType = core.ActorType

const (
	ActorTypeAnonymous = core.ActorTypeAnonymous
	ActorTypeUser      = core.ActorTypeUser
)

// BehaviorType 表示回传到推荐工具的行为类型。
type BehaviorType = core.BehaviorType

const (
	BehaviorView        = core.BehaviorView
	BehaviorClick       = core.BehaviorClick
	BehaviorCollect     = core.BehaviorCollect
	BehaviorAddCart     = core.BehaviorAddCart
	BehaviorOrderCreate = core.BehaviorOrderCreate
	BehaviorOrderPay    = core.BehaviorOrderPay
)

// Actor 表示由业务层解析出的推荐主体。
type Actor = core.Actor

// Pager 表示分页请求参数。
type Pager = core.Pager

// RecommendContext 表示场景相关的业务上下文。
type RecommendContext = core.RecommendContext

// RecommendRequest 表示推荐查询的公开入参。
type RecommendRequest = core.RecommendRequest

// RecommendItem 表示推荐结果中的单个排序商品。
type RecommendItem = core.RecommendItem

// RecommendResult 表示推荐查询的公开返回结果。
type RecommendResult = core.RecommendResult

// TraceStep 表示追踪结果中的单个步骤。
type TraceStep = core.TraceStep

// ScoreDetail 表示单个商品的最终评分明细。
type ScoreDetail = core.ScoreDetail

// ExplainRequest 表示追踪结果查询请求。
type ExplainRequest = core.ExplainRequest

// ExplainResult 表示追踪结果返回值。
type ExplainResult = core.ExplainResult

// BuildNonPersonalizedRequest 表示非个性化池构建请求。
type BuildNonPersonalizedRequest = core.BuildNonPersonalizedRequest

// BuildUserCandidateRequest 表示用户候选池构建请求。
type BuildUserCandidateRequest = core.BuildUserCandidateRequest

// BuildGoodsRelationRequest 表示商品关联池构建请求。
type BuildGoodsRelationRequest = core.BuildGoodsRelationRequest

// BuildUserToUserRequest 表示相似用户池和 user-to-user 候选构建请求。
type BuildUserToUserRequest = core.BuildUserToUserRequest

// BuildCollaborativeRequest 表示协同过滤池构建请求。
type BuildCollaborativeRequest = core.BuildCollaborativeRequest

// BuildExternalRequest 表示外部推荐池构建请求。
type BuildExternalRequest = core.BuildExternalRequest

// BuildVectorRequest 表示向量召回池构建请求。
type BuildVectorRequest = core.BuildVectorRequest

// BuildResult 表示一次构建动作的汇总结果。
type BuildResult = core.BuildResult

// TrainRankingRequest 表示学习排序模型训练请求。
type TrainRankingRequest = core.TrainRankingRequest

// RebuildRequest 表示一键重建离线池的请求。
type RebuildRequest = core.RebuildRequest

// RebuildResult 表示一键重建结果。
type RebuildResult = core.RebuildResult

// EvaluateRequest 表示离线评估请求。
type EvaluateRequest = core.EvaluateRequest

// SceneMetric 表示单个场景的离线评估指标。
type SceneMetric = core.SceneMetric

// EvaluateResult 表示离线评估结果。
type EvaluateResult = core.EvaluateResult

// ExposureSyncRequest 表示曝光回传后的运行态同步请求。
type ExposureSyncRequest = core.ExposureSyncRequest

// BehaviorSyncItem 表示行为回传中的单个商品。
type BehaviorSyncItem = core.BehaviorSyncItem

// BehaviorSyncRequest 表示行为回传后的运行态同步请求。
type BehaviorSyncRequest = core.BehaviorSyncRequest

// ActorBindRequest 表示匿名主体绑定后的运行态同步请求。
type ActorBindRequest = core.ActorBindRequest

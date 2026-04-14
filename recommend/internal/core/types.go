package core

import (
	"time"

	"recommend/contract"
)

// Dependencies 定义推荐工具运行所需的数据契约集合。
type Dependencies struct {
	// Goods 表示商品属性与商品列表数据源。
	Goods contract.GoodsSource
	// User 表示用户画像数据源。
	User contract.UserSource
	// Order 表示订单上下文与最近支付商品数据源。
	Order contract.OrderSource
	// Behavior 表示行为事件与会话事件数据源。
	Behavior contract.BehaviorSource
	// Recommend 表示推荐事实、聚合结果与离线评估事实数据源。
	Recommend contract.RecommendSource
	// Vector 表示向量召回与 embedding 检索数据源。
	Vector contract.VectorSource
	// Reranker 表示 LLM 重排器依赖。
	Reranker contract.LlmReranker
	// Cache 表示推荐缓存布局与 LevelDB 落盘数据源。
	Cache contract.CacheSource
}

// Scene 表示商城推荐场景。
type Scene string

const (
	// SceneHome 表示首页推荐场景。
	SceneHome Scene = "home"
	// SceneGoodsDetail 表示商品详情推荐场景。
	SceneGoodsDetail Scene = "goods_detail"
	// SceneCart 表示购物车推荐场景。
	SceneCart Scene = "cart"
	// SceneProfile 表示个人中心推荐场景。
	SceneProfile Scene = "profile"
	// SceneOrderDetail 表示订单详情推荐场景。
	SceneOrderDetail Scene = "order_detail"
	// SceneOrderPaid 表示支付完成推荐场景。
	SceneOrderPaid Scene = "order_paid"
)

// ActorType 表示推荐主体类型。
type ActorType int32

const (
	// ActorTypeAnonymous 表示匿名主体。
	ActorTypeAnonymous ActorType = 0
	// ActorTypeUser 表示登录用户主体。
	ActorTypeUser ActorType = 1
)

// BehaviorType 表示回传到推荐工具的行为类型。
type BehaviorType string

const (
	// BehaviorView 表示浏览行为。
	BehaviorView BehaviorType = "view"
	// BehaviorClick 表示点击行为。
	BehaviorClick BehaviorType = "click"
	// BehaviorCollect 表示收藏行为。
	BehaviorCollect BehaviorType = "collect"
	// BehaviorAddCart 表示加购行为。
	BehaviorAddCart BehaviorType = "add_cart"
	// BehaviorOrderCreate 表示下单行为。
	BehaviorOrderCreate BehaviorType = "order_create"
	// BehaviorOrderPay 表示支付行为。
	BehaviorOrderPay BehaviorType = "order_pay"
)

// Actor 表示由业务层解析出的推荐主体。
type Actor struct {
	// Type 表示主体类型，例如匿名主体或登录用户主体。
	Type ActorType
	// Id 表示主体主键编号。
	Id int64
	// SessionId 表示同一主体下的具体会话编号。
	SessionId string
}

// Pager 表示分页请求参数。
type Pager struct {
	// PageNum 表示分页页码，从 1 开始计数。
	PageNum int32
	// PageSize 表示单页返回的商品数量。
	PageSize int32
}

// RecommendContext 表示场景相关的业务上下文。
type RecommendContext struct {
	// RequestId 表示由业务层生成的推荐请求编号。
	RequestId string
	// GoodsId 表示商品详情、订单详情等场景中的锚点商品编号。
	GoodsId int64
	// OrderId 表示订单相关场景中的订单编号。
	OrderId int64
	// CartGoodsIds 表示购物车场景中的上下文商品编号集合。
	CartGoodsIds []int64
	// ExternalStrategy 表示外部推荐池使用的策略标识。
	ExternalStrategy string
	// Attributes 表示供扩展策略消费的业务透传属性。
	Attributes map[string]string
}

// RecommendRequest 表示推荐查询的公开入参。
type RecommendRequest struct {
	// Scene 表示当前请求命中的推荐场景。
	Scene Scene
	// Actor 表示当前请求对应的推荐主体。
	Actor Actor
	// Pager 表示分页信息。
	Pager Pager
	// Context 表示场景上下文数据。
	Context RecommendContext
	// Explain 表示是否要求同时持久化 explain 明细。
	Explain bool
}

// RecommendItem 表示推荐结果中的单个排序商品。
type RecommendItem struct {
	// GoodsId 表示推荐商品编号。
	GoodsId int64
	// Score 表示当前商品最终排序分值。
	Score float64
	// RecallSources 表示命中当前商品的召回来源列表。
	RecallSources []string
}

// RecommendResult 表示推荐查询的公开返回结果。
type RecommendResult struct {
	// TraceId 表示本次推荐结果关联的 explain 追踪编号。
	TraceId string
	// Total 表示当前候选总量。
	Total int64
	// Items 表示当前分页返回的推荐商品列表。
	Items []RecommendItem
	// GoodsIds 表示当前分页返回的商品编号列表。
	GoodsIds []int64
	// RecallSources 表示当前分页命中的召回来源并集。
	RecallSources []string
}

// TraceStep 表示追踪结果中的单个步骤。
type TraceStep struct {
	// Stage 表示追踪步骤名称。
	Stage string
	// Reason 表示当前步骤的业务说明。
	Reason string
	// GoodsIds 表示当前步骤涉及的商品编号列表。
	GoodsIds []int64
}

// ScoreDetail 表示单个商品的最终评分明细。
type ScoreDetail struct {
	// GoodsId 表示评分明细对应的商品编号。
	GoodsId int64
	// FinalScore 表示当前商品最终排序分值。
	FinalScore float64
	// RelationScore 表示商品关联信号得分。
	RelationScore float64
	// UserGoodsScore 表示用户商品偏好信号得分。
	UserGoodsScore float64
	// CategoryScore 表示用户类目偏好信号得分。
	CategoryScore float64
	// SceneHotScore 表示场景热度信号得分。
	SceneHotScore float64
	// GlobalHotScore 表示全站热度信号得分。
	GlobalHotScore float64
	// FreshnessScore 表示商品新鲜度信号得分。
	FreshnessScore float64
	// SessionScore 表示会话上下文信号得分。
	SessionScore float64
	// ExternalScore 表示外部召回信号得分。
	ExternalScore float64
	// CollaborativeScore 表示协同过滤信号得分。
	CollaborativeScore float64
	// UserNeighborScore 表示相似用户信号得分。
	UserNeighborScore float64
	// VectorScore 表示向量召回信号得分。
	VectorScore float64
	// ExposurePenalty 表示曝光惩罚扣分值。
	ExposurePenalty float64
	// RepeatPenalty 表示重复购买惩罚扣分值。
	RepeatPenalty float64
	// RuleScore 表示规则排序阶段产出的基础分。
	RuleScore float64
	// FmScore 表示学习排序模型产出的预测分。
	FmScore float64
	// LlmScore 表示 LLM 重排阶段产出的相关性分。
	LlmScore float64
	// RecallSources 表示当前商品命中的召回来源列表。
	RecallSources []string
}

// ExplainRequest 表示追踪结果查询请求。
type ExplainRequest struct {
	// TraceId 表示优先用于精准回查的追踪编号。
	TraceId string
	// RequestId 表示用于回退查询的请求编号。
	RequestId string
	// Scene 表示调用方补充的场景信息。
	Scene Scene
	// Actor 表示调用方补充的主体信息。
	Actor Actor
}

// ExplainResult 表示追踪结果返回值。
type ExplainResult struct {
	// TraceId 表示最终命中的追踪编号。
	TraceId string
	// Scene 表示生成当前 explain 的推荐场景。
	Scene Scene
	// Steps 表示推荐链路的关键步骤列表。
	Steps []TraceStep
	// ScoreDetails 表示候选商品评分明细列表。
	ScoreDetails []ScoreDetail
	// ResultGoodsIds 表示最终返回结果中的商品编号列表。
	ResultGoodsIds []int64
}

// BuildNonPersonalizedRequest 表示非个性化池构建请求。
type BuildNonPersonalizedRequest struct {
	// Scenes 表示需要构建匿名通用候选池的场景集合。
	Scenes []Scene
	// StatDate 表示统计口径日期。
	StatDate time.Time
	// Limit 表示单个场景构建的候选上限。
	Limit int32
}

// BuildUserCandidateRequest 表示用户候选池构建请求。
type BuildUserCandidateRequest struct {
	// UserIds 表示需要重建候选池的用户编号集合。
	UserIds []int64
	// WindowDays 表示偏好聚合回看的时间窗口天数。
	WindowDays int32
	// Limit 表示单个用户构建的候选上限。
	Limit int32
}

// BuildGoodsRelationRequest 表示商品关联池构建请求。
type BuildGoodsRelationRequest struct {
	// GoodsIds 表示需要重建关联池的商品编号集合。
	GoodsIds []int64
	// WindowDays 表示关联聚合回看的时间窗口天数。
	WindowDays int32
	// Limit 表示单个商品构建的关联候选上限。
	Limit int32
}

// BuildUserToUserRequest 表示相似用户池构建请求。
type BuildUserToUserRequest struct {
	// UserIds 表示需要重建相似用户池的用户编号集合。
	UserIds []int64
	// WindowDays 表示相似度计算回看的时间窗口天数。
	WindowDays int32
	// NeighborLimit 表示单个用户保留的相似用户数量上限。
	NeighborLimit int32
	// Limit 表示单个用户保留的 user-to-user 商品数量上限。
	Limit int32
}

// BuildCollaborativeRequest 表示协同过滤池构建请求。
type BuildCollaborativeRequest struct {
	// UserIds 表示需要重建协同过滤池的用户编号集合。
	UserIds []int64
	// WindowDays 表示协同过滤训练或聚合回看的时间窗口天数。
	WindowDays int32
	// Limit 表示单个用户保留的协同过滤商品数量上限。
	Limit int32
}

// BuildExternalRequest 表示外部推荐池构建请求。
type BuildExternalRequest struct {
	// Scenes 表示需要重建外部池的场景集合。
	Scenes []Scene
	// Strategies 表示需要重建的外部策略标识集合。
	Strategies []string
	// ActorType 表示外部池主体类型。
	ActorType ActorType
	// ActorIds 表示需要重建的主体编号集合。
	ActorIds []int64
	// Limit 表示单个主体保留的外部商品数量上限。
	Limit int32
}

// BuildVectorRequest 表示向量召回池构建请求。
type BuildVectorRequest struct {
	// Scenes 表示需要构建向量召回池的场景集合。
	Scenes []Scene
	// UserIds 表示需要构建用户向量池的用户编号集合。
	UserIds []int64
	// GoodsIds 表示需要构建商品向量池的商品编号集合。
	GoodsIds []int64
	// Limit 表示单个向量池保留的候选上限。
	Limit int32
}

// BuildResult 表示一次构建动作的汇总结果。
type BuildResult struct {
	// Scope 表示本次构建动作的范围标识。
	Scope string
	// KeyCount 表示本次构建落库的缓存 key 数量。
	KeyCount int64
	// UpdatedAt 表示本次构建完成时间。
	UpdatedAt time.Time
}

// RebuildRequest 表示一键重建离线池的请求。
type RebuildRequest struct {
	// Scenes 表示需要参与重建的场景集合。
	Scenes []Scene
	// UserIds 表示需要重建用户类离线池的用户编号集合。
	UserIds []int64
	// GoodsIds 表示需要重建商品关联池的商品编号集合。
	GoodsIds []int64
	// Strategies 表示需要重建的外部策略集合。
	Strategies []string
	// ActorType 表示外部池使用的主体类型。
	ActorType ActorType
	// ActorIds 表示外部池使用的主体编号集合。
	ActorIds []int64
	// IncludeNonPersonalized 表示是否执行匿名通用池构建。
	IncludeNonPersonalized bool
	// IncludeUserCandidate 表示是否执行用户候选池构建。
	IncludeUserCandidate bool
	// IncludeGoodsRelation 表示是否执行商品关联池构建。
	IncludeGoodsRelation bool
	// IncludeUserToUser 表示是否执行相似用户池构建。
	IncludeUserToUser bool
	// IncludeCollaborative 表示是否执行协同过滤池构建。
	IncludeCollaborative bool
	// IncludeExternal 表示是否执行外部推荐池构建。
	IncludeExternal bool
	// IncludeVector 表示是否执行向量召回池构建。
	IncludeVector bool
	// IncludeTraining 表示是否执行学习排序模型训练。
	IncludeTraining bool
	// EvaluateAfterBuild 表示构建完成后是否立即执行离线评估。
	EvaluateAfterBuild bool
	// StatDate 表示构建和评估使用的统计日期。
	StatDate time.Time
	// Limit 表示构建时使用的统一候选上限。
	Limit int32
	// NeighborLimit 表示相似用户池构建时使用的统一邻居上限。
	NeighborLimit int32
	// TopK 表示离线评估时使用的 topK。
	TopK int32
}

// TrainRankingRequest 表示学习排序模型训练请求。
type TrainRankingRequest struct {
	// Scenes 表示需要训练排序模型的场景集合。
	Scenes []Scene
	// StatDate 表示训练样本使用的统计日期。
	StatDate time.Time
}

// RebuildResult 表示一键重建结果。
type RebuildResult struct {
	// Builds 表示本次执行的各个构建动作结果。
	Builds []BuildResult
	// Evaluation 表示可选的离线评估结果。
	Evaluation *EvaluateResult
}

// EvaluateRequest 表示离线评估请求。
type EvaluateRequest struct {
	// Scenes 表示参与评估的场景集合。
	Scenes []Scene
	// StatDate 表示评估统计日期。
	StatDate time.Time
	// TopK 表示排序指标口径使用的 topK。
	TopK int32
}

// SceneMetric 表示单个场景的离线评估指标。
type SceneMetric struct {
	// Scene 表示指标所属场景。
	Scene Scene
	// RequestCount 表示评估窗口内的有效请求数量。
	RequestCount int64
	// ExposureCount 表示评估窗口内的有效曝光商品数量。
	ExposureCount int64
	// ClickCount 表示评估窗口内的点击商品数量。
	ClickCount int64
	// OrderCount 表示评估窗口内的下单商品数量。
	OrderCount int64
	// PayCount 表示评估窗口内的支付商品数量。
	PayCount int64
	// Precision 表示排序结果的准确率均值。
	Precision float64
	// Recall 表示排序结果的召回率均值。
	Recall float64
	// Ndcg 表示排序结果的 NDCG 指标。
	Ndcg float64
	// Ctr 表示曝光到点击的转化率。
	Ctr float64
	// OrderRate 表示点击到下单的转化率。
	OrderRate float64
	// PayRate 表示点击到支付的转化率。
	PayRate float64
}

// EvaluateResult 表示离线评估结果。
type EvaluateResult struct {
	// GeneratedAt 表示本次评估结果生成时间。
	GeneratedAt time.Time
	// Scenes 表示按场景拆分的评估指标列表。
	Scenes []SceneMetric
}

// ExposureSyncRequest 表示曝光回传后的运行态同步请求。
type ExposureSyncRequest struct {
	// Actor 表示触发曝光事件的主体。
	Actor Actor
	// Scene 表示曝光事件所属场景。
	Scene Scene
	// RequestId 表示原始推荐请求编号。
	RequestId string
	// GoodsIds 表示本次曝光的商品编号集合。
	GoodsIds []int64
	// ReportedAt 表示曝光事件上报时间。
	ReportedAt time.Time
}

// BehaviorSyncItem 表示行为回传中的单个商品。
type BehaviorSyncItem struct {
	// GoodsId 表示行为涉及的商品编号。
	GoodsId int64
	// GoodsNum 表示行为涉及的商品数量。
	GoodsNum int64
}

// BehaviorSyncRequest 表示行为回传后的运行态同步请求。
type BehaviorSyncRequest struct {
	// Actor 表示触发行为事件的主体。
	Actor Actor
	// Scene 表示行为事件所属场景。
	Scene Scene
	// RequestId 表示原始推荐请求编号。
	RequestId string
	// EventType 表示当前行为类型。
	EventType BehaviorType
	// Items 表示当前行为涉及的商品集合。
	Items []BehaviorSyncItem
	// ReportedAt 表示行为事件上报时间。
	ReportedAt time.Time
}

// ActorBindRequest 表示匿名主体绑定后的运行态同步请求。
type ActorBindRequest struct {
	// AnonymousId 表示待归并的匿名主体编号。
	AnonymousId int64
	// UserId 表示归并目标的登录用户编号。
	UserId int64
	// BoundAt 表示主体绑定完成时间。
	BoundAt time.Time
}

package recommend

import (
	"time"

	"recommend/contract"
)

// Dependencies 定义推荐工具运行所需的数据契约集合。
type Dependencies struct {
	Goods     contract.GoodsSource
	User      contract.UserSource
	Order     contract.OrderSource
	Behavior  contract.BehaviorSource
	Recommend contract.RecommendSource
	Cache     contract.CacheSource
}

// Scene 表示商城推荐场景。
type Scene string

const (
	SceneHome        Scene = "home"
	SceneGoodsDetail Scene = "goods_detail"
	SceneCart        Scene = "cart"
	SceneProfile     Scene = "profile"
	SceneOrderDetail Scene = "order_detail"
	SceneOrderPaid   Scene = "order_paid"
)

// ActorType 表示推荐主体类型。
type ActorType int32

const (
	ActorTypeAnonymous ActorType = 0
	ActorTypeUser      ActorType = 1
)

// BehaviorType 表示回传到推荐工具的行为类型。
type BehaviorType string

const (
	BehaviorView        BehaviorType = "view"
	BehaviorClick       BehaviorType = "click"
	BehaviorCollect     BehaviorType = "collect"
	BehaviorAddCart     BehaviorType = "add_cart"
	BehaviorOrderCreate BehaviorType = "order_create"
	BehaviorOrderPay    BehaviorType = "order_pay"
)

// Actor 表示由业务层解析出的推荐主体。
type Actor struct {
	Type      ActorType
	Id        int64
	SessionId string
}

// Pager 表示分页请求参数。
type Pager struct {
	PageNum  int32
	PageSize int32
}

// RecommendContext 表示场景相关的业务上下文。
type RecommendContext struct {
	RequestId        string
	GoodsId          int64
	OrderId          int64
	CartGoodsIds     []int64
	ExternalStrategy string
	Attributes       map[string]string
}

// RecommendRequest 表示推荐查询的公开入参。
type RecommendRequest struct {
	Scene   Scene
	Actor   Actor
	Pager   Pager
	Context RecommendContext
	Explain bool
}

// RecommendItem 表示推荐结果中的单个排序商品。
type RecommendItem struct {
	GoodsId       int64
	Score         float64
	RecallSources []string
}

// RecommendResult 表示推荐查询的公开返回结果。
type RecommendResult struct {
	TraceId       string
	Total         int64
	Items         []RecommendItem
	GoodsIds      []int64
	RecallSources []string
}

// TraceStep 表示追踪结果中的单个步骤。
type TraceStep struct {
	Stage    string
	Reason   string
	GoodsIds []int64
}

// ScoreDetail 表示单个商品的最终评分明细。
type ScoreDetail struct {
	GoodsId         int64
	FinalScore      float64
	RelationScore   float64
	UserGoodsScore  float64
	CategoryScore   float64
	SceneHotScore   float64
	GlobalHotScore  float64
	FreshnessScore  float64
	ExposurePenalty float64
	RepeatPenalty   float64
	RecallSources   []string
}

// ExplainRequest 表示追踪结果查询请求。
type ExplainRequest struct {
	TraceId   string
	RequestId string
	Scene     Scene
	Actor     Actor
}

// ExplainResult 表示追踪结果返回值。
type ExplainResult struct {
	TraceId        string
	Scene          Scene
	Steps          []TraceStep
	ScoreDetails   []ScoreDetail
	ResultGoodsIds []int64
}

// BuildNonPersonalizedRequest 表示非个性化池构建请求。
type BuildNonPersonalizedRequest struct {
	Scenes   []Scene
	StatDate time.Time
	Limit    int32
}

// BuildUserCandidateRequest 表示用户候选池构建请求。
type BuildUserCandidateRequest struct {
	UserIds    []int64
	WindowDays int32
	Limit      int32
}

// BuildGoodsRelationRequest 表示商品关联池构建请求。
type BuildGoodsRelationRequest struct {
	GoodsIds   []int64
	WindowDays int32
	Limit      int32
}

// BuildUserToUserRequest 表示相似用户池构建请求。
type BuildUserToUserRequest struct {
	UserIds       []int64
	WindowDays    int32
	NeighborLimit int32
}

// BuildCollaborativeRequest 表示协同过滤池构建请求。
type BuildCollaborativeRequest struct {
	UserIds    []int64
	WindowDays int32
	Limit      int32
}

// BuildExternalRequest 表示外部推荐池构建请求。
type BuildExternalRequest struct {
	Scenes     []Scene
	Strategies []string
	ActorType  ActorType
	ActorIds   []int64
	Limit      int32
}

// BuildResult 表示一次构建动作的汇总结果。
type BuildResult struct {
	Scope     string
	KeyCount  int64
	UpdatedAt time.Time
}

// EvaluateRequest 表示离线评估请求。
type EvaluateRequest struct {
	Scenes   []Scene
	StatDate time.Time
	TopK     int32
}

// SceneMetric 表示单个场景的离线评估指标。
type SceneMetric struct {
	Scene         Scene
	RequestCount  int64
	ExposureCount int64
	ClickCount    int64
	OrderCount    int64
	PayCount      int64
	Precision     float64
	Recall        float64
	Ndcg          float64
	Ctr           float64
	OrderRate     float64
	PayRate       float64
}

// EvaluateResult 表示离线评估结果。
type EvaluateResult struct {
	GeneratedAt time.Time
	Scenes      []SceneMetric
}

// ExposureSyncRequest 表示曝光回传后的运行态同步请求。
type ExposureSyncRequest struct {
	Actor      Actor
	Scene      Scene
	RequestId  string
	GoodsIds   []int64
	ReportedAt time.Time
}

// BehaviorSyncItem 表示行为回传中的单个商品。
type BehaviorSyncItem struct {
	GoodsId  int64
	GoodsNum int64
}

// BehaviorSyncRequest 表示行为回传后的运行态同步请求。
type BehaviorSyncRequest struct {
	Actor      Actor
	Scene      Scene
	RequestId  string
	EventType  BehaviorType
	Items      []BehaviorSyncItem
	ReportedAt time.Time
}

// ActorBindRequest 表示匿名主体绑定后的运行态同步请求。
type ActorBindRequest struct {
	AnonymousId int64
	UserId      int64
	BoundAt     time.Time
}

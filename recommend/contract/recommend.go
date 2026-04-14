package contract

import (
	"context"
	"time"
)

// WeightedGoods 表示带分值的商品事实。
type WeightedGoods struct {
	// GoodsId 表示商品编号。
	GoodsId int64
	// Score 表示当前商品对应的聚合分值。
	Score float64
}

// WeightedCategory 表示带分值的类目偏好事实。
type WeightedCategory struct {
	// CategoryId 表示类目编号。
	CategoryId int64
	// Score 表示当前类目的偏好分值。
	Score float64
}

// WeightedUser 表示带分值的相似用户事实。
type WeightedUser struct {
	// UserId 表示相似用户编号。
	UserId int64
	// Score 表示当前相似用户的相似度分值。
	Score float64
}

// RequestFact 表示离线评估使用的推荐请求事实。
type RequestFact struct {
	// RequestId 表示推荐请求编号。
	RequestId string
	// Scene 表示请求所属推荐场景。
	Scene string
	// ActorType 表示请求主体类型。
	ActorType int32
	// ActorId 表示请求主体编号。
	ActorId int64
	// CreatedAt 表示请求创建时间。
	CreatedAt time.Time
	// GoodsIds 表示请求返回的商品编号列表。
	GoodsIds []int64
}

// ExposureFact 表示离线评估使用的推荐曝光事实。
type ExposureFact struct {
	// RequestId 表示曝光关联的推荐请求编号。
	RequestId string
	// Scene 表示曝光所属推荐场景。
	Scene string
	// ActorType 表示曝光主体类型。
	ActorType int32
	// ActorId 表示曝光主体编号。
	ActorId int64
	// CreatedAt 表示曝光创建时间。
	CreatedAt time.Time
	// GoodsIds 表示曝光的商品编号列表。
	GoodsIds []int64
}

// ActionFact 表示离线评估使用的推荐行为事实。
type ActionFact struct {
	// RequestId 表示行为关联的推荐请求编号。
	RequestId string
	// Scene 表示行为所属推荐场景。
	Scene string
	// ActorType 表示行为主体类型。
	ActorType int32
	// ActorId 表示行为主体编号。
	ActorId int64
	// GoodsId 表示行为关联的商品编号。
	GoodsId int64
	// EventType 表示行为类型。
	EventType string
	// GoodsNum 表示行为关联的商品数量。
	GoodsNum int64
	// CreatedAt 表示行为创建时间。
	CreatedAt time.Time
}

// VectorRecallRequest 表示向量召回使用的查询条件。
type VectorRecallRequest struct {
	// Scene 表示当前向量召回所属场景。
	Scene string
	// ActorType 表示发起召回的主体类型。
	ActorType int32
	// ActorId 表示发起召回的主体编号。
	ActorId int64
	// SessionId 表示发起召回的会话编号。
	SessionId string
	// SourceGoodsIds 表示向量查询使用的锚点商品编号集合。
	SourceGoodsIds []int64
	// Limit 表示单次向量召回返回的商品上限。
	Limit int32
	// Attributes 表示提供给向量服务的扩展属性。
	Attributes map[string]string
}

// VectorSource 定义向量召回或 embedding 检索数据源。
type VectorSource interface {
	ListVectorGoods(context.Context, VectorRecallRequest) ([]*WeightedGoods, error)
}

// LlmRerankCandidate 表示传给 LLM 重排器的候选商品。
type LlmRerankCandidate struct {
	// GoodsId 表示候选商品编号。
	GoodsId int64
	// CategoryId 表示候选商品所属类目编号。
	CategoryId int64
	// BaseScore 表示规则排序阶段产出的基础分。
	BaseScore float64
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
	// FreshnessScore 表示新鲜度信号得分。
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
	// RecallSources 表示当前候选命中的召回来源集合。
	RecallSources []string
}

// LlmRerankRequest 表示 LLM 重排请求。
type LlmRerankRequest struct {
	// Scene 表示当前重排所属场景。
	Scene string
	// ActorType 表示当前请求主体类型。
	ActorType int32
	// ActorId 表示当前请求主体编号。
	ActorId int64
	// SessionId 表示当前请求会话编号。
	SessionId string
	// GoodsId 表示详情类场景的锚点商品编号。
	GoodsId int64
	// OrderId 表示订单类场景的订单编号。
	OrderId int64
	// ExternalStrategy 表示外部推荐池策略标识。
	ExternalStrategy string
	// Attributes 表示调用方透传给重排器的上下文属性。
	Attributes map[string]string
	// Candidates 表示参与重排的候选商品集合。
	Candidates []*LlmRerankCandidate
}

// LlmRerankResult 表示 LLM 重排返回结果。
type LlmRerankResult struct {
	// GoodsId 表示被重排的商品编号。
	GoodsId int64
	// Score 表示 LLM 给出的相关性分值。
	Score float64
	// Reason 表示可选的重排解释文本。
	Reason string
}

// LlmReranker 定义 LLM 重排器契约。
type LlmReranker interface {
	Rerank(context.Context, LlmRerankRequest) ([]*LlmRerankResult, error)
}

// RecommendSource 定义推荐事实表与聚合数据来源。
type RecommendSource interface {
	ListSceneHotGoods(context.Context, string, time.Time, int32) ([]*WeightedGoods, error)
	ListGlobalHotGoods(context.Context, time.Time, int32) ([]*WeightedGoods, error)
	ListRelatedGoods(context.Context, int64, int32) ([]*WeightedGoods, error)
	ListUserGoodsPreference(context.Context, int64, int32) ([]*WeightedGoods, error)
	ListUserCategoryPreference(context.Context, int64, int32) ([]*WeightedCategory, error)
	ListNeighborUsers(context.Context, int64, int32) ([]*WeightedUser, error)
	ListUserToUserGoods(context.Context, int64, int32) ([]*WeightedGoods, error)
	ListCollaborativeGoods(context.Context, int64, int32) ([]*WeightedGoods, error)
	ListExternalGoods(context.Context, string, string, int32, int64, int32) ([]*WeightedGoods, error)
	ListRequestFacts(context.Context, string, time.Time, time.Time) ([]*RequestFact, error)
	ListExposureFacts(context.Context, string, time.Time, time.Time) ([]*ExposureFact, error)
	ListActionFacts(context.Context, string, time.Time, time.Time) ([]*ActionFact, error)
}

package contract

import (
	"context"
	"time"
)

// WeightedGoods 表示带分值的商品事实。
type WeightedGoods struct {
	GoodsId int64
	Score   float64
}

// WeightedCategory 表示带分值的类目偏好事实。
type WeightedCategory struct {
	CategoryId int64
	Score      float64
}

// WeightedUser 表示带分值的相似用户事实。
type WeightedUser struct {
	UserId int64
	Score  float64
}

// RequestFact 表示离线评估使用的推荐请求事实。
type RequestFact struct {
	RequestId string
	Scene     string
	ActorType int32
	ActorId   int64
	CreatedAt time.Time
	GoodsIds  []int64
}

// ExposureFact 表示离线评估使用的推荐曝光事实。
type ExposureFact struct {
	RequestId string
	Scene     string
	ActorType int32
	ActorId   int64
	CreatedAt time.Time
	GoodsIds  []int64
}

// ActionFact 表示离线评估使用的推荐行为事实。
type ActionFact struct {
	RequestId string
	Scene     string
	ActorType int32
	ActorId   int64
	GoodsId   int64
	EventType string
	GoodsNum  int64
	CreatedAt time.Time
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

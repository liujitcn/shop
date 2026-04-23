package dto

// RecommendActorType 表示推荐链路内部使用的主体类型。
type RecommendActorType int32

const (
	// AnonymousActorType 表示匿名推荐主体。
	AnonymousActorType RecommendActorType = 1
	// UserActorType 表示登录用户推荐主体。
	UserActorType RecommendActorType = 2
)

// RecommendStrategy 表示推荐链路内部使用的推荐策略标识。
type RecommendStrategy string

const (
	// RemoteStrategy 表示远端推荐策略。
	RemoteStrategy RecommendStrategy = "remote"
	// LocalStrategy 表示本地同类目推荐策略。
	LocalStrategy RecommendStrategy = "local"
)

// RecommendActor 表示推荐链路内部使用的推荐主体。
type RecommendActor struct {
	ActorType RecommendActorType `json:"actor_type"` // 推荐主体类型
	ActorId   int64              `json:"actor_id"`   // 推荐主体编号
}

// IsValid 判断当前推荐主体是否有效。
func (r *RecommendActor) IsValid() bool {
	return r != nil && r.ActorId > 0
}

// IsUser 判断当前推荐主体是否为登录用户。
func (r *RecommendActor) IsUser() bool {
	return r != nil && r.ActorType == UserActorType && r.ActorId > 0
}

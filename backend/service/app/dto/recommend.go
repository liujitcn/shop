package dto

import recommendEvent "shop/pkg/recommend/event"

// RecommendActor 表示推荐链路中的主体信息
type RecommendActor struct {
	ActorType int32
	ActorId   int64
}

// UserId 获取登录态推荐主体的用户标识。
func (a *RecommendActor) UserId() int64 {
	// 主体为空或不是登录用户时，不返回用户标识。
	if a == nil || a.ActorType != recommendEvent.ActorTypeUser {
		return 0
	}
	return a.ActorId
}

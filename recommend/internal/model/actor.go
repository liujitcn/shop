package model

import "recommend"

// Actor 表示推荐链路内部使用的主体信息。
type Actor struct {
	Type      recommend.ActorType
	Id        int64
	SessionId string
}

// ResolveActor 将公开主体结构转换为内部主体结构。
func ResolveActor(actor recommend.Actor) Actor {
	return Actor{
		Type:      actor.Type,
		Id:        actor.Id,
		SessionId: actor.SessionId,
	}
}

// IsAnonymous 判断当前主体是否为匿名主体。
func (a Actor) IsAnonymous() bool {
	return a.Type == recommend.ActorTypeAnonymous
}

// IsUser 判断当前主体是否为登录用户主体。
func (a Actor) IsUser() bool {
	return a.Type == recommend.ActorTypeUser && a.Id > 0
}

package model

import "recommend/internal/core"

// Actor 表示推荐链路内部使用的主体信息。
type Actor struct {
	// Type 表示当前主体类型，例如匿名主体或登录用户主体。
	Type core.ActorType
	// Id 表示当前主体编号。
	Id int64
	// SessionId 表示当前主体对应的会话编号。
	SessionId string
}

// ResolveActor 将公开主体结构转换为内部主体结构。
func ResolveActor(actor core.Actor) Actor {
	return Actor{
		Type:      actor.Type,
		Id:        actor.Id,
		SessionId: actor.SessionId,
	}
}

// IsAnonymous 判断当前主体是否为匿名主体。
func (a Actor) IsAnonymous() bool {
	return a.Type == core.ActorTypeAnonymous
}

// IsUser 判断当前主体是否为登录用户主体。
func (a Actor) IsUser() bool {
	return a.Type == core.ActorTypeUser && a.Id > 0
}

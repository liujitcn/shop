package domain

import recommendEvent "shop/pkg/recommend/event"

// Actor 表示推荐链路中的主体信息。
type Actor struct {
	ActorType int32 // 主体类型：匿名主体或登录主体
	ActorId   int64 // 主体编号：匿名主体编号或用户编号
}

// UserId 返回登录态主体对应的用户编号。
func (a *Actor) UserId() int64 {
	// 主体为空或不是登录主体时，不返回用户编号。
	if a == nil || a.ActorType != recommendEvent.ActorTypeUser {
		return 0
	}
	return a.ActorId
}

// IsAnonymous 判断当前主体是否为匿名主体。
func (a *Actor) IsAnonymous() bool {
	// 主体为空时，默认按匿名主体处理，避免调用方继续判空。
	if a == nil {
		return true
	}
	return a.ActorType == recommendEvent.ActorTypeAnonymous
}

// ResolveCacheActorType 返回缓存键构造需要的主体类型。
func (a *Actor) ResolveCacheActorType() int32 {
	// 主体为空时，统一按匿名主体处理。
	if a == nil {
		return recommendEvent.ActorTypeAnonymous
	}
	return a.ActorType
}

// ResolveCacheActorId 返回缓存键构造需要的主体编号。
func (a *Actor) ResolveCacheActorId() int64 {
	// 主体为空时，统一回退到 0，避免缓存键缺失。
	if a == nil {
		return 0
	}
	return a.ActorId
}

// RequestContext 表示一次推荐请求在领域层的上下文。
type RequestContext struct {
	RequestId     string         // 推荐请求编号
	Scene         int32          // 推荐场景
	Actor         *Actor         // 推荐主体
	OrderId       int64          // 场景相关订单编号
	GoodsId       int64          // 场景相关商品编号
	PageNum       int64          // 当前页码
	PageSize      int64          // 每页数量
	SourceContext map[string]any // 来源上下文扩展信息
}

// CloneSourceContext 返回请求上下文中的浅拷贝来源上下文。
func (c *RequestContext) CloneSourceContext() map[string]any {
	// 来源上下文为空时，返回空映射，避免调用方继续判空。
	if c == nil || len(c.SourceContext) == 0 {
		return map[string]any{}
	}
	result := make(map[string]any, len(c.SourceContext))
	for key, value := range c.SourceContext {
		result[key] = value
	}
	return result
}

// GoodsActionContext 表示商品行为链路中透传的推荐上下文。
type GoodsActionContext struct {
	Scene     int32  // 行为来源推荐场景
	RequestId string // 关联的推荐请求编号
	Position  int32  // 商品在推荐结果中的位置
}

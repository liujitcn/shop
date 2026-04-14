package model

import "recommend/internal/core"

const (
	// defaultPageNum 表示内部分页归一化时使用的默认页码。
	defaultPageNum = 1
	// defaultPageSize 表示内部分页归一化时使用的默认分页大小。
	defaultPageSize = 10
)

// Pager 表示内部归一化后的分页信息。
type Pager struct {
	// PageNum 表示分页页码。
	PageNum int32
	// PageSize 表示单页商品数量。
	PageSize int32
}

// RequestContext 表示内部归一化后的推荐上下文。
type RequestContext struct {
	// RequestId 表示推荐请求编号。
	RequestId string
	// GoodsId 表示详情类场景的锚点商品编号。
	GoodsId int64
	// OrderId 表示订单类场景的订单编号。
	OrderId int64
	// CartGoodsIds 表示购物车场景的上下文商品编号集合。
	CartGoodsIds []int64
	// ExternalStrategy 表示外部推荐池策略标识。
	ExternalStrategy string
	// Attributes 表示透传给内部召回和排序链路的扩展属性。
	Attributes map[string]string
}

// Request 表示进入推荐内核后的内部请求结构。
// 这里保留与 core 接近的字段形态，但职责是承接归一化后的运行态数据和内部方法，不直接对外暴露。
type Request struct {
	// Scene 表示当前请求场景。
	Scene Scene
	// Actor 表示当前请求主体。
	Actor Actor
	// Pager 表示当前请求分页参数。
	Pager Pager
	// Context 表示当前请求业务上下文。
	Context RequestContext
	// NeedExplain 表示当前请求是否要求持久化 explain 明细。
	NeedExplain bool
}

// ResolveRequest 将公开请求转换为内部请求。
func ResolveRequest(request core.RecommendRequest) Request {
	return Request{
		Scene:       ResolveScene(request.Scene),
		Actor:       ResolveActor(request.Actor),
		Pager:       resolvePager(request.Pager),
		Context:     resolveRequestContext(request.Context),
		NeedExplain: request.Explain,
	}
}

// Offset 返回当前分页对应的偏移量。
func (r Request) Offset() int {
	pageNum := r.Pager.PageNum
	pageSize := r.Pager.PageSize
	return int((pageNum - 1) * pageSize)
}

// Limit 返回当前分页对应的数量限制。
func (r Request) Limit() int {
	return int(r.Pager.PageSize)
}

// resolvePager 归一化分页参数。
func resolvePager(pager core.Pager) Pager {
	result := Pager{
		PageNum:  pager.PageNum,
		PageSize: pager.PageSize,
	}
	// 页码非法时，统一回退到第一页，避免在线推荐出现负偏移。
	if result.PageNum <= 0 {
		result.PageNum = defaultPageNum
	}
	// 每页数量非法时，统一回退到默认值，避免查询范围异常。
	if result.PageSize <= 0 {
		result.PageSize = defaultPageSize
	}
	return result
}

// resolveRequestContext 复制并归一化推荐上下文。
func resolveRequestContext(context core.RecommendContext) RequestContext {
	result := RequestContext{
		RequestId:        context.RequestId,
		GoodsId:          context.GoodsId,
		OrderId:          context.OrderId,
		ExternalStrategy: context.ExternalStrategy,
	}
	if len(context.CartGoodsIds) > 0 {
		result.CartGoodsIds = append(result.CartGoodsIds, context.CartGoodsIds...)
	}
	if len(context.Attributes) > 0 {
		result.Attributes = make(map[string]string, len(context.Attributes))
		for key, value := range context.Attributes {
			result.Attributes[key] = value
		}
	}
	return result
}

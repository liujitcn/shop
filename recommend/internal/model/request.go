package model

import "recommend"

const defaultPageNum = 1
const defaultPageSize = 10

// Pager 表示内部使用的分页信息。
type Pager struct {
	PageNum  int32
	PageSize int32
}

// RequestContext 表示内部使用的推荐上下文。
type RequestContext struct {
	RequestId        string
	GoodsId          int64
	OrderId          int64
	CartGoodsIds     []int64
	ExternalStrategy string
	Attributes       map[string]string
}

// Request 表示内部使用的推荐请求。
type Request struct {
	Scene       Scene
	Actor       Actor
	Pager       Pager
	Context     RequestContext
	NeedExplain bool
}

// ResolveRequest 将公开请求转换为内部请求。
func ResolveRequest(request recommend.RecommendRequest) Request {
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
func resolvePager(pager recommend.Pager) Pager {
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
func resolveRequestContext(context recommend.RecommendContext) RequestContext {
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

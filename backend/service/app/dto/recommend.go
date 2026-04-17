package dto

import (
	app "shop/api/gen/go/app"
	recommendDomain "shop/pkg/recommend/domain"
	recommendEvent "shop/pkg/recommend/event"
)

// RecommendActor 表示推荐链路中的主体信息。
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

// ToDomainActor 将应用层推荐主体桥接为领域主体对象。
func (a *RecommendActor) ToDomainActor() *recommendDomain.Actor {
	// 主体为空时，不再继续构建领域主体对象。
	if a == nil {
		return nil
	}
	return &recommendDomain.Actor{
		ActorType: a.ActorType,
		ActorId:   a.ActorId,
	}
}

// BuildRecommendGoodsRequest 将接口请求桥接为推荐领域请求对象。
func BuildRecommendGoodsRequest(req *app.RecommendGoodsRequest) *recommendDomain.GoodsRequest {
	if req == nil {
		return &recommendDomain.GoodsRequest{}
	}
	return &recommendDomain.GoodsRequest{
		Scene:    req.GetScene(),
		OrderId:  req.GetOrderId(),
		GoodsId:  req.GetGoodsId(),
		PageNum:  req.GetPageNum(),
		PageSize: req.GetPageSize(),
	}
}

// RecommendActorBindLogUserRow 表示推荐主体绑定日志里的用户查询结果。
type RecommendActorBindLogUserRow struct {
	UserId int64 `gorm:"column:user_id"`
}

// RecommendGoodsRelationKey 表示商品关联聚合的唯一键。
type RecommendGoodsRelationKey struct {
	GoodsId        int64
	RelatedGoodsId int64
	RelationType   string
}

// RecommendOrderRelationGroupKey 表示订单级商品关联的分组键。
type RecommendOrderRelationGroupKey struct {
	RequestId string
	EventType int32
}

// RecommendUserGoodsPreferenceKey 表示用户商品偏好的聚合键。
type RecommendUserGoodsPreferenceKey struct {
	UserId  int64
	GoodsId int64
}

// RecommendUserPreferenceKey 表示用户类目偏好的聚合键。
type RecommendUserPreferenceKey struct {
	UserId     int64
	CategoryId int64
}

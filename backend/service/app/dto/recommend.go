package dto

import (
	"shop/api/gen/go/common"
	"shop/pkg/gen/models"
	"time"
)

// RecommendActor 表示推荐链路中的主体信息
type RecommendActor struct {
	ActorType int32
	ActorId   int64
}

// RecommendGoodsActionEvent 表示推荐商品行为事件
type RecommendGoodsActionEvent struct {
	RecommendActor *RecommendActor                 // 推荐主体信息
	EventType      common.RecommendGoodsActionType // 商品行为事件类型
	EventTime      time.Time                       // 事件发生时间
	GoodsItems     []*models.RecommendGoodsAction  // 商品行为列表
}

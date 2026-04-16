package domain

import (
	"time"

	"shop/api/gen/go/common"
	"shop/pkg/gen/models"
)

// GoodsActionProjectionEvent 表示用于投影聚合的商品行为事件。
type GoodsActionProjectionEvent struct {
	ActorType  int32                           // 主体类型，当前用于区分匿名主体和登录主体。
	ActorId    int64                           // 主体编号，登录态时对应用户编号。
	EventType  common.RecommendGoodsActionType // 行为类型，用于决定偏好和关系权重。
	EventTime  time.Time                       // 事件发生时间，用于刷新投影记录时间边界。
	GoodsItems []*models.RecommendGoodsAction  // 行为明细列表，每条记录对应一个商品事实。
}

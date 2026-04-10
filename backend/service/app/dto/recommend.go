package dto

// RecommendActor 表示推荐链路中的主体信息。
type RecommendActor struct {
	ActorType int32
	ActorId   int64
	UserId    int64
}

// RecommendEvent 表示推荐行为事件。
type RecommendEvent struct {
	EventType  string                     `json:"eventType"`
	UserID     int64                      `json:"userId"`
	ActorType  int32                      `json:"actorType"`
	ActorID    int64                      `json:"actorId"`
	RequestID  string                     `json:"requestId,omitempty"`
	Scene      int32                      `json:"scene,omitempty"`
	GoodsID    int64                      `json:"goodsId,omitempty"`
	GoodsIDs   []int64                    `json:"goodsIds,omitempty"`
	GoodsNum   int64                      `json:"goodsNum,omitempty"`
	GoodsItems []*RecommendEventGoodsItem `json:"goodsItems,omitempty"`
	Position   int32                      `json:"position,omitempty"`
	OccurredAt int64                      `json:"occurredAt,omitempty"`
}

// RecommendEventGoodsItem 表示推荐事件中的商品项。
type RecommendEventGoodsItem struct {
	GoodsID   int64  `json:"goodsId,omitempty"`
	GoodsNum  int64  `json:"goodsNum,omitempty"`
	Scene     int32  `json:"scene,omitempty"`
	RequestID string `json:"requestId,omitempty"`
	Position  int32  `json:"position,omitempty"`
}

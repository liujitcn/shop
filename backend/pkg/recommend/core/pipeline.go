package core

// RecallRequest 表示通用召回阶段的输入参数。
type RecallRequest struct {
	ActorType int32
	ActorId   int64
	Scene     int32
	Limit     int
}

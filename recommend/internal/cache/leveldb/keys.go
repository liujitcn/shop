package leveldb

import "fmt"

const (
	keyPrefixCandidatePool  = "pool:candidate"
	keyPrefixRelatedGoods   = "pool:related_goods"
	keyPrefixUserCandidate  = "pool:user_candidate"
	keyPrefixUserNeighbor   = "pool:user_neighbor"
	keyPrefixCollaborative  = "pool:collaborative"
	keyPrefixExternal       = "pool:external"
	keyPrefixSessionState   = "runtime:session"
	keyPrefixPenaltyState   = "runtime:penalty"
	keyPrefixTraceDetail    = "trace:detail"
	keyPrefixTraceByRequest = "trace:request"
)

// CandidatePoolKey 返回通用候选池的缓存键。
func CandidatePoolKey(scene string, actorType int32, actorId int64) []byte {
	return []byte(fmt.Sprintf("%s:%s:%d:%d", keyPrefixCandidatePool, scene, actorType, actorId))
}

// RelatedGoodsPoolKey 返回商品关联池的缓存键。
func RelatedGoodsPoolKey(scene string, sourceGoodsId int64) []byte {
	return []byte(fmt.Sprintf("%s:%s:%d", keyPrefixRelatedGoods, scene, sourceGoodsId))
}

// UserCandidatePoolKey 返回用户候选池的缓存键。
func UserCandidatePoolKey(scene string, userId int64) []byte {
	return []byte(fmt.Sprintf("%s:%s:%d", keyPrefixUserCandidate, scene, userId))
}

// UserNeighborPoolKey 返回相似用户池的缓存键。
func UserNeighborPoolKey(userId int64) []byte {
	return []byte(fmt.Sprintf("%s:%d", keyPrefixUserNeighbor, userId))
}

// CollaborativePoolKey 返回协同过滤池的缓存键。
func CollaborativePoolKey(scene string, userId int64) []byte {
	return []byte(fmt.Sprintf("%s:%s:%d", keyPrefixCollaborative, scene, userId))
}

// ExternalPoolKey 返回外部推荐池的缓存键。
func ExternalPoolKey(scene, strategy string, actorType int32, actorId int64) []byte {
	return []byte(fmt.Sprintf("%s:%s:%s:%d:%d", keyPrefixExternal, scene, strategy, actorType, actorId))
}

// SessionStateKey 返回会话态的缓存键。
func SessionStateKey(actorType int32, actorId int64, sessionId string) []byte {
	return []byte(fmt.Sprintf("%s:%d:%d:%s", keyPrefixSessionState, actorType, actorId, sessionId))
}

// PenaltyStateKey 返回惩罚态的缓存键。
func PenaltyStateKey(scene string, actorType int32, actorId int64) []byte {
	return []byte(fmt.Sprintf("%s:%s:%d:%d", keyPrefixPenaltyState, scene, actorType, actorId))
}

// TraceDetailKey 返回追踪详情的缓存键。
func TraceDetailKey(traceId string) []byte {
	return []byte(fmt.Sprintf("%s:%s", keyPrefixTraceDetail, traceId))
}

// TraceByRequestKey 返回请求编号到追踪详情的索引键。
func TraceByRequestKey(requestId string) []byte {
	return []byte(fmt.Sprintf("%s:%s", keyPrefixTraceByRequest, requestId))
}

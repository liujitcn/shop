package engine

import (
	"context"
	"errors"
	recommendv1 "recommend/api/gen/go/recommend/v1"
	cachex "recommend/internal/cache"
	"recommend/internal/core"

	goleveldb "github.com/syndtr/goleveldb/leveldb"
)

const (
	defaultRecentGoodsCountValue = 20
)

// SyncExposure 在曝光事实落库后更新运行态惩罚与追踪数据。
func SyncExposure(ctx context.Context, dependencies core.Dependencies, config core.ServiceConfig, request core.ExposureSyncRequest) error {
	manager, err := openCacheManager(ctx, dependencies)
	if err != nil {
		return err
	}
	defer func() {
		_ = manager.Close()
	}()

	runtimeStore := &cachex.RuntimeStore{Driver: manager}
	traceStore := &cachex.TraceStore{Driver: manager}
	state, err := loadPenaltyState(runtimeStore, string(request.Scene), int32(request.Actor.Type), request.Actor.Id)
	if err != nil {
		return err
	}

	applyExposurePenalty(state, request.GoodsIds, resolveExposurePenalty(config))
	state.Meta = buildCacheMeta(string(request.Scene), int32(request.Actor.Type), request.Actor.Id, request.ReportedAt)
	err = runtimeStore.SavePenaltyState(string(request.Scene), int32(request.Actor.Type), request.Actor.Id, state)
	if err != nil {
		return err
	}

	return appendTraceStepByStore(traceStore, request.RequestId, "exposure_reported", "推荐曝光事件已同步到运行态惩罚", request.GoodsIds)
}

// SyncBehavior 在行为事实落库后更新会话态与惩罚态。
func SyncBehavior(ctx context.Context, dependencies core.Dependencies, config core.ServiceConfig, request core.BehaviorSyncRequest) error {
	manager, err := openCacheManager(ctx, dependencies)
	if err != nil {
		return err
	}
	defer func() {
		_ = manager.Close()
	}()

	runtimeStore := &cachex.RuntimeStore{Driver: manager}
	traceStore := &cachex.TraceStore{Driver: manager}
	sessionState, err := loadSessionState(runtimeStore, request.Actor)
	if err != nil {
		return err
	}
	sharedSessionState, err := loadSharedSessionState(runtimeStore, request.Actor)
	if err != nil {
		return err
	}
	penaltyState, err := loadPenaltyState(runtimeStore, string(request.Scene), int32(request.Actor.Type), request.Actor.Id)
	if err != nil {
		return err
	}

	goodsIds := collectBehaviorGoodsIds(request.Items)

	switch request.EventType {
	// 浏览行为只更新最近浏览序列，供后续会话召回和 explain 使用。
	case core.BehaviorView:
		sessionState.RecentViewGoodsIds = appendRecentGoodsIds(sessionState.GetRecentViewGoodsIds(), goodsIds, resolveRecentGoodsCount(config))
		sharedSessionState.RecentViewGoodsIds = appendRecentGoodsIds(sharedSessionState.GetRecentViewGoodsIds(), goodsIds, resolveRecentGoodsCount(config))
	// 点击行为说明用户对商品产生更强兴趣，进入点击序列。
	case core.BehaviorClick:
		sessionState.RecentClickGoodsIds = appendRecentGoodsIds(sessionState.GetRecentClickGoodsIds(), goodsIds, resolveRecentGoodsCount(config))
		sharedSessionState.RecentClickGoodsIds = appendRecentGoodsIds(sharedSessionState.GetRecentClickGoodsIds(), goodsIds, resolveRecentGoodsCount(config))
	// 加购行为既保留在加购序列，也能给后续推荐提供更强上下文。
	case core.BehaviorAddCart:
		sessionState.RecentCartGoodsIds = appendRecentGoodsIds(sessionState.GetRecentCartGoodsIds(), goodsIds, resolveRecentGoodsCount(config))
		sharedSessionState.RecentCartGoodsIds = appendRecentGoodsIds(sharedSessionState.GetRecentCartGoodsIds(), goodsIds, resolveRecentGoodsCount(config))
	// 创建订单后先写入较轻的复购惩罚，避免同一请求后再次推荐刚下单商品。
	case core.BehaviorOrderCreate:
		applyRepeatPenalty(penaltyState, request.Items, resolveOrderCreatePenalty(config))
	// 已支付行为采用更强复购惩罚，优先抑制短期重复曝光。
	case core.BehaviorOrderPay:
		applyRepeatPenalty(penaltyState, request.Items, resolveOrderPayPenalty(config))
	// 收藏行为当前没有独立运行态结构，先只补充 trace，避免无意义地挤占会话序列。
	case core.BehaviorCollect:
	default:
	}

	sessionState.Meta = buildCacheMeta(string(request.Scene), int32(request.Actor.Type), request.Actor.Id, request.ReportedAt)
	sharedSessionState.Meta = buildCacheMeta(string(request.Scene), int32(request.Actor.Type), request.Actor.Id, request.ReportedAt)
	penaltyState.Meta = buildCacheMeta(string(request.Scene), int32(request.Actor.Type), request.Actor.Id, request.ReportedAt)

	err = saveSessionStates(runtimeStore, request.Actor, sessionState, sharedSessionState)
	if err != nil {
		return err
	}
	err = runtimeStore.SavePenaltyState(string(request.Scene), int32(request.Actor.Type), request.Actor.Id, penaltyState)
	if err != nil {
		return err
	}

	return appendTraceStepByStore(traceStore, request.RequestId, "behavior_reported", "推荐行为事件已同步到运行态", goodsIds)
}

// SyncActorBind 在匿名主体绑定成功后归并运行态数据。
func SyncActorBind(ctx context.Context, dependencies core.Dependencies, config core.ServiceConfig, request core.ActorBindRequest) error {
	manager, err := openCacheManager(ctx, dependencies)
	if err != nil {
		return err
	}
	defer func() {
		_ = manager.Close()
	}()

	runtimeStore := &cachex.RuntimeStore{Driver: manager}
	err = mergeSharedSessionState(runtimeStore, config, request)
	if err != nil {
		return err
	}
	return mergePenaltyStates(runtimeStore, request)
}

// loadPenaltyState 读取惩罚态；缓存不存在时返回空状态。
func loadPenaltyState(
	store *cachex.RuntimeStore,
	scene string,
	actorType int32,
	actorId int64,
) (*recommendv1.RecommendPenaltyState, error) {
	state, err := store.GetPenaltyState(scene, actorType, actorId)
	if err == nil {
		ensurePenaltyMaps(state)
		return state, nil
	}
	// 运行态尚未初始化时，返回空状态即可，不需要把首次写入当作异常。
	if errors.Is(err, goleveldb.ErrNotFound) {
		return &recommendv1.RecommendPenaltyState{
			ExposurePenalty: make(map[int64]float64),
			RepeatPenalty:   make(map[int64]float64),
		}, nil
	}
	return nil, err
}

// loadSessionState 读取具体会话槽位的会话态。
func loadSessionState(store *cachex.RuntimeStore, actor core.Actor) (*recommendv1.RecommendSessionState, error) {
	return loadSessionStateBySessionId(store, actor, actor.SessionId)
}

// loadSharedSessionState 读取主体级共享会话态。
func loadSharedSessionState(store *cachex.RuntimeStore, actor core.Actor) (*recommendv1.RecommendSessionState, error) {
	return loadSessionStateBySessionId(store, actor, "")
}

// loadSessionStateBySessionId 按指定会话槽位读取会话态。
func loadSessionStateBySessionId(
	store *cachex.RuntimeStore,
	actor core.Actor,
	sessionId string,
) (*recommendv1.RecommendSessionState, error) {
	state, err := store.GetSessionState(int32(actor.Type), actor.Id, sessionId)
	if err == nil {
		return state, nil
	}
	// 首次回传行为前缓存不存在时，返回空会话态供后续继续累积。
	if errors.Is(err, goleveldb.ErrNotFound) {
		return &recommendv1.RecommendSessionState{}, nil
	}
	return nil, err
}

// ensurePenaltyMaps 确保惩罚态中的 map 已初始化。
func ensurePenaltyMaps(state *recommendv1.RecommendPenaltyState) {
	if state == nil {
		return
	}
	if state.ExposurePenalty == nil {
		state.ExposurePenalty = make(map[int64]float64)
	}
	if state.RepeatPenalty == nil {
		state.RepeatPenalty = make(map[int64]float64)
	}
}

// applyExposurePenalty 将曝光惩罚写入惩罚态。
func applyExposurePenalty(state *recommendv1.RecommendPenaltyState, goodsIds []int64, delta float64) {
	ensurePenaltyMaps(state)
	for _, goodsId := range goodsIds {
		// 非法商品编号或非法惩罚增量都不应进入运行态。
		if goodsId <= 0 || delta <= 0 {
			continue
		}
		state.ExposurePenalty[goodsId] += delta
	}
}

// applyRepeatPenalty 将复购惩罚写入惩罚态。
func applyRepeatPenalty(state *recommendv1.RecommendPenaltyState, items []core.BehaviorSyncItem, basePenalty float64) {
	ensurePenaltyMaps(state)
	for _, item := range items {
		// 非法商品编号或非法惩罚基数不参与重复购买惩罚累计。
		if item.GoodsId <= 0 || basePenalty <= 0 {
			continue
		}

		goodsNum := item.GoodsNum
		// 未携带数量时，按 1 件处理，避免把合法支付行为当成 0 惩罚。
		if goodsNum <= 0 {
			goodsNum = 1
		}
		state.RepeatPenalty[item.GoodsId] += float64(goodsNum) * basePenalty
	}
}

// collectBehaviorGoodsIds 提取行为回传中的商品编号。
func collectBehaviorGoodsIds(items []core.BehaviorSyncItem) []int64 {
	goodsIds := make([]int64, 0, len(items))
	for _, item := range items {
		// 非法商品编号不进入运行态更新集合。
		if item.GoodsId <= 0 {
			continue
		}
		goodsIds = append(goodsIds, item.GoodsId)
	}
	return goodsIds
}

// appendRecentGoodsIds 以“最新在前”的方式追加最近行为商品。
func appendRecentGoodsIds(current []int64, goodsIds []int64, limit int) []int64 {
	if limit <= 0 {
		return nil
	}

	result := append([]int64(nil), current...)
	for _, goodsId := range goodsIds {
		// 非法商品编号不进入最近行为序列。
		if goodsId <= 0 {
			continue
		}
		result = prependUniqueGoodsId(result, goodsId)
	}
	if len(result) > limit {
		result = result[:limit]
	}
	return result
}

// resolveRecentGoodsCount 返回运行态最近行为序列长度。
func resolveRecentGoodsCount(config core.ServiceConfig) int {
	if config.Sync.MaxRecentGoodsCount > 0 {
		return config.Sync.MaxRecentGoodsCount
	}
	return defaultRecentGoodsCountValue
}

// resolveExposurePenalty 返回默认曝光惩罚。
func resolveExposurePenalty(config core.ServiceConfig) float64 {
	if config.Sync.ExposurePenalty > 0 {
		return config.Sync.ExposurePenalty
	}
	return core.DefaultServiceConfig().Sync.ExposurePenalty
}

// resolveOrderCreatePenalty 返回下单行为惩罚。
func resolveOrderCreatePenalty(config core.ServiceConfig) float64 {
	if config.Sync.OrderCreatePenalty > 0 {
		return config.Sync.OrderCreatePenalty
	}
	return core.DefaultServiceConfig().Sync.OrderCreatePenalty
}

// resolveOrderPayPenalty 返回支付行为惩罚。
func resolveOrderPayPenalty(config core.ServiceConfig) float64 {
	if config.Sync.OrderPayPenalty > 0 {
		return config.Sync.OrderPayPenalty
	}
	return core.DefaultServiceConfig().Sync.OrderPayPenalty
}

// prependUniqueGoodsId 将商品编号移动到序列头部。
func prependUniqueGoodsId(current []int64, goodsId int64) []int64 {
	result := make([]int64, 0, len(current)+1)
	result = append(result, goodsId)
	for _, item := range current {
		// 已经提升到头部的商品不再重复写入。
		if item == goodsId {
			continue
		}
		result = append(result, item)
	}
	return result
}

// saveSessionStates 保存会话态和主体共享会话态。
func saveSessionStates(
	store *cachex.RuntimeStore,
	actor core.Actor,
	sessionState *recommendv1.RecommendSessionState,
	sharedSessionState *recommendv1.RecommendSessionState,
) error {
	// 具体会话槽位存在时，优先单独保存，便于后续按真实会话精细回放。
	if actor.SessionId != "" {
		err := store.SaveSessionState(int32(actor.Type), actor.Id, actor.SessionId, sessionState)
		if err != nil {
			return err
		}
	}
	return store.SaveSessionState(int32(actor.Type), actor.Id, "", sharedSessionState)
}

// mergeSharedSessionState 归并匿名主体与登录主体的共享会话态。
func mergeSharedSessionState(store *cachex.RuntimeStore, config core.ServiceConfig, request core.ActorBindRequest) error {
	anonymousActor := core.Actor{Type: core.ActorTypeAnonymous, Id: request.AnonymousId}
	userActor := core.Actor{Type: core.ActorTypeUser, Id: request.UserId}

	anonymousState, err := loadSharedSessionState(store, anonymousActor)
	if err != nil {
		return err
	}
	userState, err := loadSharedSessionState(store, userActor)
	if err != nil {
		return err
	}

	userState.RecentViewGoodsIds = appendRecentGoodsIds(userState.GetRecentViewGoodsIds(), anonymousState.GetRecentViewGoodsIds(), resolveRecentGoodsCount(config))
	userState.RecentClickGoodsIds = appendRecentGoodsIds(userState.GetRecentClickGoodsIds(), anonymousState.GetRecentClickGoodsIds(), resolveRecentGoodsCount(config))
	userState.RecentCartGoodsIds = appendRecentGoodsIds(userState.GetRecentCartGoodsIds(), anonymousState.GetRecentCartGoodsIds(), resolveRecentGoodsCount(config))
	userState.Meta = buildCacheMeta("", int32(core.ActorTypeUser), request.UserId, request.BoundAt)

	// 匿名共享会话态为空时，不需要额外覆盖用户态。
	if len(anonymousState.GetRecentViewGoodsIds()) == 0 &&
		len(anonymousState.GetRecentClickGoodsIds()) == 0 &&
		len(anonymousState.GetRecentCartGoodsIds()) == 0 {
		return nil
	}

	err = store.SaveSessionState(int32(core.ActorTypeUser), request.UserId, "", userState)
	if err != nil {
		return err
	}
	return store.DeleteSessionState(int32(core.ActorTypeAnonymous), request.AnonymousId, "")
}

// mergePenaltyStates 归并匿名主体与登录主体的场景惩罚态。
func mergePenaltyStates(store *cachex.RuntimeStore, request core.ActorBindRequest) error {
	for _, scene := range []core.Scene{
		core.SceneHome,
		core.SceneGoodsDetail,
		core.SceneCart,
		core.SceneProfile,
		core.SceneOrderDetail,
		core.SceneOrderPaid,
	} {
		anonymousState, err := loadPenaltyState(store, string(scene), int32(core.ActorTypeAnonymous), request.AnonymousId)
		if err != nil {
			return err
		}
		userState, err := loadPenaltyState(store, string(scene), int32(core.ActorTypeUser), request.UserId)
		if err != nil {
			return err
		}

		mergePenaltyMap(userState.ExposurePenalty, anonymousState.GetExposurePenalty())
		mergePenaltyMap(userState.RepeatPenalty, anonymousState.GetRepeatPenalty())
		userState.Meta = buildCacheMeta(string(scene), int32(core.ActorTypeUser), request.UserId, request.BoundAt)

		// 匿名场景惩罚为空时，不需要创建新的用户场景缓存。
		if len(anonymousState.GetExposurePenalty()) == 0 && len(anonymousState.GetRepeatPenalty()) == 0 {
			continue
		}

		err = store.SavePenaltyState(string(scene), int32(core.ActorTypeUser), request.UserId, userState)
		if err != nil {
			return err
		}
		err = store.DeletePenaltyState(string(scene), int32(core.ActorTypeAnonymous), request.AnonymousId)
		if err != nil && !errors.Is(err, goleveldb.ErrNotFound) {
			return err
		}
	}
	return nil
}

// mergePenaltyMap 将来源惩罚值累加到目标惩罚 map。
func mergePenaltyMap(target map[int64]float64, source map[int64]float64) {
	for goodsId, penalty := range source {
		// 非法商品编号或非法惩罚值不参与主体归并。
		if goodsId <= 0 || penalty <= 0 {
			continue
		}
		target[goodsId] += penalty
	}
}

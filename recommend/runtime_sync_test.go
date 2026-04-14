package recommend

import (
	"context"
	"math"
	"path/filepath"
	"recommend/contract"
	cachex "recommend/internal/cache"
	cacheleveldb "recommend/internal/cache/leveldb"
	"testing"
	"time"

	goleveldb "github.com/syndtr/goleveldb/leveldb"
)

func TestRecommendExplainAndExposurePenalty(t *testing.T) {
	ctx := context.Background()
	cacheSource := newRuntimeTestCacheSource(t)
	dependencies := Dependencies{
		Goods: &runtimeTestGoodsSource{
			goodsById: map[int64]*contract.Goods{
				1: {Id: 1, CategoryId: 11, OnSale: true, InStock: true},
				2: {Id: 2, CategoryId: 12, OnSale: true, InStock: true},
				3: {Id: 3, CategoryId: 13, OnSale: true, InStock: true},
			},
			latestGoods: []*contract.Goods{
				{Id: 3, CategoryId: 13, OnSale: true, InStock: true},
			},
		},
		Recommend: &runtimeTestRecommendSource{},
		Cache:     cacheSource,
	}
	recommender := newTestRecommend(t, WithDependencies(dependencies))

	firstResult, err := recommender.Recommend(ctx, RecommendRequest{
		Scene: SceneHome,
		Actor: Actor{
			Type: ActorTypeUser,
			Id:   101,
		},
		Pager: Pager{
			PageNum:  1,
			PageSize: 2,
		},
		Context: RecommendContext{
			RequestId: "request-1",
		},
		Explain: true,
	})
	if err != nil {
		t.Fatalf("首次推荐失败: %v", err)
	}
	if firstResult.TraceId != "request-1" {
		t.Fatalf("首次推荐 traceId 不符合预期: %+v", firstResult)
	}
	if len(firstResult.Items) != 2 || firstResult.Items[0].GoodsId != 1 {
		t.Fatalf("首次推荐结果不符合预期: %+v", firstResult.Items)
	}

	explainResult, err := recommender.Explain(ctx, ExplainRequest{
		RequestId: "request-1",
	})
	if err != nil {
		t.Fatalf("Explain 查询失败: %v", err)
	}
	if len(explainResult.Steps) < 2 {
		t.Fatalf("Explain 步骤数量不符合预期: %+v", explainResult)
	}
	if len(explainResult.ScoreDetails) == 0 {
		t.Fatalf("Explain 评分明细不应为空: %+v", explainResult)
	}
	if len(explainResult.ResultGoodsIds) != 2 || explainResult.ResultGoodsIds[0] != 1 {
		t.Fatalf("Explain 结果商品列表不符合预期: %+v", explainResult.ResultGoodsIds)
	}

	err = recommender.SyncExposure(ctx, ExposureSyncRequest{
		Actor: Actor{
			Type: ActorTypeUser,
			Id:   101,
		},
		Scene:      SceneHome,
		RequestId:  "request-1",
		GoodsIds:   []int64{1},
		ReportedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("同步曝光失败: %v", err)
	}

	secondResult, err := recommender.Recommend(ctx, RecommendRequest{
		Scene: SceneHome,
		Actor: Actor{
			Type: ActorTypeUser,
			Id:   101,
		},
		Pager: Pager{
			PageNum:  1,
			PageSize: 2,
		},
		Context: RecommendContext{
			RequestId: "request-2",
		},
	})
	if err != nil {
		t.Fatalf("二次推荐失败: %v", err)
	}
	if len(secondResult.Items) != 2 || secondResult.Items[0].GoodsId != 2 {
		t.Fatalf("曝光惩罚未生效，二次推荐结果不符合预期: %+v", secondResult.Items)
	}

	explainResult, err = recommender.Explain(ctx, ExplainRequest{
		RequestId: "request-1",
	})
	if err != nil {
		t.Fatalf("曝光后再次查询 Explain 失败: %v", err)
	}
	lastStep := explainResult.Steps[len(explainResult.Steps)-1]
	if lastStep.Stage != "exposure_reported" {
		t.Fatalf("曝光同步后 trace 步骤未补充: %+v", explainResult.Steps)
	}
}

func TestSyncBehaviorAndActorBind(t *testing.T) {
	ctx := context.Background()
	cacheSource := newRuntimeTestCacheSource(t)
	dependencies := Dependencies{
		Cache: cacheSource,
	}
	recommender := newTestRecommend(t, WithDependencies(dependencies))

	anonymousActor := Actor{
		Type:      ActorTypeAnonymous,
		Id:        900,
		SessionId: "session-a",
	}

	err := recommender.SyncBehavior(ctx, BehaviorSyncRequest{
		Actor:      anonymousActor,
		Scene:      SceneHome,
		RequestId:  "request-3",
		EventType:  BehaviorAddCart,
		Items:      []BehaviorSyncItem{{GoodsId: 21}},
		ReportedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("同步加购行为失败: %v", err)
	}

	err = recommender.SyncBehavior(ctx, BehaviorSyncRequest{
		Actor:      anonymousActor,
		Scene:      SceneHome,
		RequestId:  "request-4",
		EventType:  BehaviorOrderPay,
		Items:      []BehaviorSyncItem{{GoodsId: 31, GoodsNum: 2}},
		ReportedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("同步支付行为失败: %v", err)
	}

	manager := openRuntimeTestManager(t, cacheSource)
	store := &cachex.RuntimeStore{Driver: manager}

	sessionState, err := store.GetSessionState(int32(ActorTypeAnonymous), 900, "session-a")
	if err != nil {
		t.Fatalf("读取匿名主体具体会话态失败: %v", err)
	}
	if len(sessionState.GetRecentCartGoodsIds()) != 1 || sessionState.GetRecentCartGoodsIds()[0] != 21 {
		t.Fatalf("匿名主体具体会话态不符合预期: %+v", sessionState)
	}

	sharedState, err := store.GetSessionState(int32(ActorTypeAnonymous), 900, "")
	if err != nil {
		t.Fatalf("读取匿名主体共享会话态失败: %v", err)
	}
	if len(sharedState.GetRecentCartGoodsIds()) != 1 || sharedState.GetRecentCartGoodsIds()[0] != 21 {
		t.Fatalf("匿名主体共享会话态不符合预期: %+v", sharedState)
	}

	penaltyState, err := store.GetPenaltyState("home", int32(ActorTypeAnonymous), 900)
	if err != nil {
		t.Fatalf("读取匿名主体惩罚态失败: %v", err)
	}
	if math.Abs(penaltyState.GetRepeatPenalty()[31]-1.2) > 0.0001 {
		t.Fatalf("匿名主体重复购买惩罚不符合预期: %+v", penaltyState)
	}

	err = manager.Close()
	if err != nil {
		t.Fatalf("关闭匿名主体检查用 LevelDB 失败: %v", err)
	}

	err = recommender.SyncActorBind(ctx, ActorBindRequest{
		AnonymousId: 900,
		UserId:      1001,
		BoundAt:     time.Now(),
	})
	if err != nil {
		t.Fatalf("同步主体绑定失败: %v", err)
	}

	manager = openRuntimeTestManager(t, cacheSource)
	defer func() {
		closeErr := manager.Close()
		if closeErr != nil {
			t.Fatalf("关闭用户主体检查用 LevelDB 失败: %v", closeErr)
		}
	}()

	store = &cachex.RuntimeStore{Driver: manager}
	userState, err := store.GetSessionState(int32(ActorTypeUser), 1001, "")
	if err != nil {
		t.Fatalf("读取登录主体共享会话态失败: %v", err)
	}
	if len(userState.GetRecentCartGoodsIds()) != 1 || userState.GetRecentCartGoodsIds()[0] != 21 {
		t.Fatalf("登录主体共享会话态不符合预期: %+v", userState)
	}

	userPenaltyState, err := store.GetPenaltyState("home", int32(ActorTypeUser), 1001)
	if err != nil {
		t.Fatalf("读取登录主体惩罚态失败: %v", err)
	}
	if math.Abs(userPenaltyState.GetRepeatPenalty()[31]-1.2) > 0.0001 {
		t.Fatalf("登录主体重复购买惩罚不符合预期: %+v", userPenaltyState)
	}

	_, err = store.GetSessionState(int32(ActorTypeAnonymous), 900, "")
	if err != goleveldb.ErrNotFound {
		t.Fatalf("匿名主体共享会话态应已删除，实际错误: %v", err)
	}

	_, err = store.GetPenaltyState("home", int32(ActorTypeAnonymous), 900)
	if err != goleveldb.ErrNotFound {
		t.Fatalf("匿名主体惩罚态应已删除，实际错误: %v", err)
	}
}

func TestRecommendUsesRuntimeSessionContext(t *testing.T) {
	ctx := context.Background()
	cacheSource := newRuntimeTestCacheSource(t)
	dependencies := Dependencies{
		Goods: &runtimeTestGoodsSource{
			goodsById: map[int64]*contract.Goods{
				1: {Id: 1, CategoryId: 11, OnSale: true, InStock: true},
				2: {Id: 2, CategoryId: 12, OnSale: true, InStock: true},
			},
			latestGoods: []*contract.Goods{
				{Id: 1, CategoryId: 11, OnSale: true, InStock: true},
			},
		},
		Recommend: &runtimeTestRecommendSource{
			sceneHotGoods: []*contract.WeightedGoods{},
			relatedGoods: map[int64][]*contract.WeightedGoods{
				1: {
					{GoodsId: 2, Score: 5},
				},
			},
		},
		Cache: cacheSource,
	}
	recommender := newTestRecommend(t, WithDependencies(dependencies))

	err := recommender.SyncBehavior(ctx, BehaviorSyncRequest{
		Actor: Actor{
			Type:      ActorTypeUser,
			Id:        101,
			SessionId: "session-a",
		},
		Scene:      SceneGoodsDetail,
		RequestId:  "request-session",
		EventType:  BehaviorClick,
		Items:      []BehaviorSyncItem{{GoodsId: 1}},
		ReportedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("同步会话点击行为失败: %v", err)
	}

	var result *RecommendResult
	result, err = recommender.Recommend(ctx, RecommendRequest{
		Scene: SceneGoodsDetail,
		Actor: Actor{
			Type:      ActorTypeUser,
			Id:        101,
			SessionId: "session-a",
		},
		Pager: Pager{
			PageNum:  1,
			PageSize: 1,
		},
		Context: RecommendContext{
			RequestId: "request-session-recommend",
		},
	})
	if err != nil {
		t.Fatalf("运行态会话推荐失败: %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].GoodsId != 2 {
		t.Fatalf("运行态会话推荐结果不符合预期: %+v", result.Items)
	}
}

type runtimeTestGoodsSource struct {
	goodsById   map[int64]*contract.Goods
	latestGoods []*contract.Goods
}

func (s *runtimeTestGoodsSource) GetGoods(_ context.Context, goodsId int64) (*contract.Goods, error) {
	return s.goodsById[goodsId], nil
}

func (s *runtimeTestGoodsSource) ListGoods(_ context.Context, goodsIds []int64) ([]*contract.Goods, error) {
	list := make([]*contract.Goods, 0, len(goodsIds))
	for _, goodsId := range goodsIds {
		item, ok := s.goodsById[goodsId]
		if ok {
			list = append(list, item)
		}
	}
	return list, nil
}

func (s *runtimeTestGoodsSource) ListGoodsByCategoryIds(_ context.Context, _ []int64, _ int32) ([]*contract.Goods, error) {
	return nil, nil
}

func (s *runtimeTestGoodsSource) ListLatestGoods(_ context.Context, limit int32) ([]*contract.Goods, error) {
	list := append([]*contract.Goods(nil), s.latestGoods...)
	if len(list) > int(limit) {
		list = list[:limit]
	}
	return list, nil
}

type runtimeTestRecommendSource struct {
	sceneHotGoods  []*contract.WeightedGoods
	globalHotGoods []*contract.WeightedGoods
	relatedGoods   map[int64][]*contract.WeightedGoods
}

func (s *runtimeTestRecommendSource) ListSceneHotGoods(_ context.Context, _ string, _ time.Time, _ int32) ([]*contract.WeightedGoods, error) {
	if s != nil && s.sceneHotGoods != nil {
		return cloneWeightedGoods(s.sceneHotGoods), nil
	}
	return []*contract.WeightedGoods{
		{GoodsId: 1, Score: 6},
		{GoodsId: 2, Score: 5},
	}, nil
}

func (s *runtimeTestRecommendSource) ListGlobalHotGoods(_ context.Context, _ time.Time, _ int32) ([]*contract.WeightedGoods, error) {
	if s != nil && s.globalHotGoods != nil {
		return cloneWeightedGoods(s.globalHotGoods), nil
	}
	return nil, nil
}

func (s *runtimeTestRecommendSource) ListRelatedGoods(_ context.Context, goodsId int64, _ int32) ([]*contract.WeightedGoods, error) {
	if s != nil && s.relatedGoods != nil {
		return cloneWeightedGoods(s.relatedGoods[goodsId]), nil
	}
	return nil, nil
}

func (s *runtimeTestRecommendSource) ListUserGoodsPreference(_ context.Context, _ int64, _ int32) ([]*contract.WeightedGoods, error) {
	return nil, nil
}

func (s *runtimeTestRecommendSource) ListUserCategoryPreference(_ context.Context, _ int64, _ int32) ([]*contract.WeightedCategory, error) {
	return nil, nil
}

func (s *runtimeTestRecommendSource) ListNeighborUsers(_ context.Context, _ int64, _ int32) ([]*contract.WeightedUser, error) {
	return nil, nil
}

func (s *runtimeTestRecommendSource) ListUserToUserGoods(_ context.Context, _ int64, _ int32) ([]*contract.WeightedGoods, error) {
	return nil, nil
}

func (s *runtimeTestRecommendSource) ListCollaborativeGoods(_ context.Context, _ int64, _ int32) ([]*contract.WeightedGoods, error) {
	return nil, nil
}

func (s *runtimeTestRecommendSource) ListExternalGoods(_ context.Context, _, _ string, _ int32, _ int64, _ int32) ([]*contract.WeightedGoods, error) {
	return nil, nil
}

func (s *runtimeTestRecommendSource) ListRequestFacts(_ context.Context, _ string, _, _ time.Time) ([]*contract.RequestFact, error) {
	return nil, nil
}

func (s *runtimeTestRecommendSource) ListExposureFacts(_ context.Context, _ string, _, _ time.Time) ([]*contract.ExposureFact, error) {
	return nil, nil
}

func (s *runtimeTestRecommendSource) ListActionFacts(_ context.Context, _ string, _, _ time.Time) ([]*contract.ActionFact, error) {
	return nil, nil
}

type runtimeTestCacheSource struct {
	layout contract.LevelDbLayout
}

func (s *runtimeTestCacheSource) RecommendLevelDb(_ context.Context) (contract.LevelDbLayout, error) {
	return s.layout, nil
}

func newRuntimeTestCacheSource(t *testing.T) *runtimeTestCacheSource {
	t.Helper()

	rootPath := t.TempDir()
	return &runtimeTestCacheSource{
		layout: contract.LevelDbLayout{
			PoolPath:    filepath.Join(rootPath, "pool.db"),
			RuntimePath: filepath.Join(rootPath, "runtime.db"),
			TracePath:   filepath.Join(rootPath, "trace.db"),
		},
	}
}

func openRuntimeTestManager(t *testing.T, cacheSource *runtimeTestCacheSource) *cacheleveldb.Manager {
	t.Helper()

	manager, err := cacheleveldb.OpenManagerByLayout(cacheSource.layout)
	if err != nil {
		t.Fatalf("打开运行态测试 LevelDB 失败: %v", err)
	}
	return manager
}

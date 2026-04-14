package recall

import (
	"context"
	"path/filepath"
	"recommend/contract"
	cachex "recommend/internal/cache"
	cacheleveldb "recommend/internal/cache/leveldb"
	"recommend/internal/core"
	"recommend/internal/model"
	"testing"
	"time"

	recommendv1 "recommend/api/gen/go/recommend/v1"
)

func TestRecallLatest(t *testing.T) {
	dependencies := core.Dependencies{
		Goods: &fakeGoodsSource{
			latestGoods: []*contract.Goods{
				{Id: 1, CategoryId: 11},
				{Id: 2, CategoryId: 12},
			},
		},
	}

	list, err := RecallLatest(context.Background(), Request{
		Dependencies: dependencies,
		Limit:        2,
	})
	if err != nil {
		t.Fatalf("最新商品召回失败: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("最新商品召回数量不符合预期: %d", len(list))
	}
	if list[0].GoodsId() != 1 || list[0].RecallSourceList()[0] != RecallSourceLatest {
		t.Fatalf("最新商品召回结果不符合预期: %+v", list[0])
	}
}

// TestRecallLatestFromPool 校验最新商品召回优先读取离线池。
func TestRecallLatestFromPool(t *testing.T) {
	manager := openRecallTestManager(t)
	defer func() {
		err := manager.Close()
		if err != nil {
			t.Fatalf("关闭 recall 测试 LevelDB 失败: %v", err)
		}
	}()

	poolStore := &cachex.PoolStore{Driver: manager}
	var err error
	err = poolStore.SaveCandidatePool("home", int32(core.ActorTypeAnonymous), 0, &recommendv1.RecommendCandidatePool{
		Items: []*recommendv1.RecommendCandidateItem{
			{
				GoodsId:       3,
				Score:         1,
				RecallSources: []string{RecallSourceLatest},
				SourceScores: map[string]float64{
					RecallSourceLatest: 1,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("保存非个性化池失败: %v", err)
	}

	dependencies := buildRecallDependencies()
	var list []*model.Candidate
	list, err = RecallLatest(context.Background(), Request{
		Scene:        model.SceneHome,
		Dependencies: dependencies,
		PoolStore:    poolStore,
		Limit:        2,
	})
	if err != nil {
		t.Fatalf("从非个性化池召回最新商品失败: %v", err)
	}
	if len(list) != 1 || list[0].GoodsId() != 3 {
		t.Fatalf("最新商品池召回结果不符合预期: %+v", list)
	}
}

func TestRecallSceneHot(t *testing.T) {
	dependencies := buildRecallDependencies()
	list, err := RecallSceneHot(context.Background(), Request{
		Scene:         model.SceneHome,
		Dependencies:  dependencies,
		ReferenceTime: time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC),
		Limit:         3,
	})
	if err != nil {
		t.Fatalf("场景热销召回失败: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("场景热销召回数量不符合预期: %d", len(list))
	}
	if list[0].Score.SceneHotScore <= 0 {
		t.Fatalf("场景热销得分未正确回填: %+v", list[0])
	}
}

// TestRecallSceneHotFromPool 校验场景热销召回优先读取离线池并使用来源分值。
func TestRecallSceneHotFromPool(t *testing.T) {
	manager := openRecallTestManager(t)
	defer func() {
		err := manager.Close()
		if err != nil {
			t.Fatalf("关闭 recall 测试 LevelDB 失败: %v", err)
		}
	}()

	poolStore := &cachex.PoolStore{Driver: manager}
	var err error
	err = poolStore.SaveCandidatePool("home", int32(core.ActorTypeAnonymous), 0, &recommendv1.RecommendCandidatePool{
		Items: []*recommendv1.RecommendCandidateItem{
			{
				GoodsId:       1,
				Score:         6,
				RecallSources: []string{RecallSourceLatest, RecallSourceSceneHot},
				SourceScores: map[string]float64{
					RecallSourceLatest:   1,
					RecallSourceSceneHot: 5,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("保存非个性化池失败: %v", err)
	}

	dependencies := buildRecallDependencies()
	var list []*model.Candidate
	list, err = RecallSceneHot(context.Background(), Request{
		Scene:         model.SceneHome,
		Dependencies:  dependencies,
		ReferenceTime: time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC),
		PoolStore:     poolStore,
		Limit:         3,
	})
	if err != nil {
		t.Fatalf("从非个性化池召回场景热销失败: %v", err)
	}
	if len(list) != 1 || list[0].GoodsId() != 1 || list[0].Score.SceneHotScore != 5 {
		t.Fatalf("场景热销池召回结果不符合预期: %+v", list)
	}
}

// TestRecallSceneHotFallbackWhenPoolMissingSourceScores 校验旧格式合并池会自动回退事实源。
func TestRecallSceneHotFallbackWhenPoolMissingSourceScores(t *testing.T) {
	manager := openRecallTestManager(t)
	defer func() {
		err := manager.Close()
		if err != nil {
			t.Fatalf("关闭 recall 测试 LevelDB 失败: %v", err)
		}
	}()

	poolStore := &cachex.PoolStore{Driver: manager}
	var err error
	err = poolStore.SaveCandidatePool("home", int32(core.ActorTypeAnonymous), 0, &recommendv1.RecommendCandidatePool{
		Items: []*recommendv1.RecommendCandidateItem{
			{
				GoodsId:       1,
				Score:         6,
				RecallSources: []string{RecallSourceLatest, RecallSourceSceneHot},
			},
		},
	})
	if err != nil {
		t.Fatalf("保存旧格式非个性化池失败: %v", err)
	}

	dependencies := buildRecallDependencies()
	var list []*model.Candidate
	list, err = RecallSceneHot(context.Background(), Request{
		Scene:         model.SceneHome,
		Dependencies:  dependencies,
		ReferenceTime: time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC),
		PoolStore:     poolStore,
		Limit:         3,
	})
	if err != nil {
		t.Fatalf("旧格式非个性化池回退事实源失败: %v", err)
	}
	if len(list) != 2 || list[0].GoodsId() != 1 || list[0].Score.SceneHotScore != 5 {
		t.Fatalf("旧格式非个性化池回退结果不符合预期: %+v", list)
	}
}

// TestRecallGlobalHotFromPool 校验全站热销召回优先读取离线池并使用来源分值。
func TestRecallGlobalHotFromPool(t *testing.T) {
	manager := openRecallTestManager(t)
	defer func() {
		err := manager.Close()
		if err != nil {
			t.Fatalf("关闭 recall 测试 LevelDB 失败: %v", err)
		}
	}()

	poolStore := &cachex.PoolStore{Driver: manager}
	var err error
	err = poolStore.SaveCandidatePool("home", int32(core.ActorTypeAnonymous), 0, &recommendv1.RecommendCandidatePool{
		Items: []*recommendv1.RecommendCandidateItem{
			{
				GoodsId:       2,
				Score:         7,
				RecallSources: []string{RecallSourceSceneHot, RecallSourceGlobalHot},
				SourceScores: map[string]float64{
					RecallSourceSceneHot:  3,
					RecallSourceGlobalHot: 4,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("保存非个性化池失败: %v", err)
	}

	dependencies := buildRecallDependencies()
	var list []*model.Candidate
	list, err = RecallGlobalHot(context.Background(), Request{
		Scene:         model.SceneHome,
		Dependencies:  dependencies,
		ReferenceTime: time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC),
		PoolStore:     poolStore,
		Limit:         3,
	})
	if err != nil {
		t.Fatalf("从非个性化池召回全站热销失败: %v", err)
	}
	if len(list) != 1 || list[0].GoodsId() != 2 || list[0].Score.GlobalHotScore != 4 {
		t.Fatalf("全站热销池召回结果不符合预期: %+v", list)
	}
}

// TestRecallUserGoodsPreferenceFromPool 校验用户商品偏好召回优先读取用户候选池。
func TestRecallUserGoodsPreferenceFromPool(t *testing.T) {
	manager := openRecallTestManager(t)
	defer func() {
		err := manager.Close()
		if err != nil {
			t.Fatalf("关闭 recall 测试 LevelDB 失败: %v", err)
		}
	}()

	poolStore := &cachex.PoolStore{Driver: manager}
	var err error
	err = poolStore.SaveUserCandidatePool("home", 101, &recommendv1.RecommendUserCandidatePool{
		Items: []*recommendv1.RecommendCandidateItem{
			{
				GoodsId:       1,
				Score:         8,
				RecallSources: []string{RecallSourceUserGoods, RecallSourceUserCategory},
				SourceScores: map[string]float64{
					RecallSourceUserGoods:    6,
					RecallSourceUserCategory: 2,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("保存用户候选池失败: %v", err)
	}

	dependencies := buildRecallDependencies()
	var list []*model.Candidate
	list, err = RecallUserGoodsPreference(context.Background(), Request{
		Scene: model.SceneHome,
		Actor: model.Actor{
			Type: core.ActorTypeUser,
			Id:   101,
		},
		Dependencies: dependencies,
		PoolStore:    poolStore,
		Limit:        3,
	})
	if err != nil {
		t.Fatalf("从用户候选池召回商品偏好失败: %v", err)
	}
	if len(list) != 1 || list[0].GoodsId() != 1 || list[0].Score.UserGoodsScore != 6 {
		t.Fatalf("用户商品偏好池召回结果不符合预期: %+v", list)
	}
}

// TestRecallUserCategoryPreferenceFromPool 校验用户类目偏好召回优先读取用户候选池。
func TestRecallUserCategoryPreferenceFromPool(t *testing.T) {
	manager := openRecallTestManager(t)
	defer func() {
		err := manager.Close()
		if err != nil {
			t.Fatalf("关闭 recall 测试 LevelDB 失败: %v", err)
		}
	}()

	poolStore := &cachex.PoolStore{Driver: manager}
	var err error
	err = poolStore.SaveUserCandidatePool("home", 101, &recommendv1.RecommendUserCandidatePool{
		Items: []*recommendv1.RecommendCandidateItem{
			{
				GoodsId:       3,
				Score:         5,
				RecallSources: []string{RecallSourceUserCategory},
				SourceScores: map[string]float64{
					RecallSourceUserCategory: 5,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("保存用户候选池失败: %v", err)
	}

	dependencies := buildRecallDependencies()
	var list []*model.Candidate
	list, err = RecallUserCategoryPreference(context.Background(), Request{
		Scene: model.SceneHome,
		Actor: model.Actor{
			Type: core.ActorTypeUser,
			Id:   101,
		},
		Dependencies: dependencies,
		PoolStore:    poolStore,
		Limit:        3,
	})
	if err != nil {
		t.Fatalf("从用户候选池召回类目偏好失败: %v", err)
	}
	if len(list) != 1 || list[0].GoodsId() != 3 || list[0].Score.CategoryScore != 5 {
		t.Fatalf("用户类目偏好池召回结果不符合预期: %+v", list)
	}
}

// TestRecallUserToUserFromPool 校验 user-to-user 召回优先读取相似用户池中的商品项。
func TestRecallUserToUserFromPool(t *testing.T) {
	manager := openRecallTestManager(t)
	defer func() {
		err := manager.Close()
		if err != nil {
			t.Fatalf("关闭 recall 测试 LevelDB 失败: %v", err)
		}
	}()

	poolStore := &cachex.PoolStore{Driver: manager}
	var err error
	err = poolStore.SaveUserNeighborPool(101, &recommendv1.RecommendUserNeighborPool{
		NeighborUserIds: []int64{201, 202},
		Items: []*recommendv1.RecommendCandidateItem{
			{
				GoodsId:       3,
				Score:         4,
				RecallSources: []string{RecallSourceUserToUser},
				SourceScores: map[string]float64{
					RecallSourceUserToUser: 4,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("保存相似用户池失败: %v", err)
	}

	dependencies := buildRecallDependencies()
	var list []*model.Candidate
	list, err = RecallUserToUser(context.Background(), Request{
		Scene: model.SceneHome,
		Actor: model.Actor{
			Type: core.ActorTypeUser,
			Id:   101,
		},
		Dependencies: dependencies,
		PoolStore:    poolStore,
		Limit:        3,
	})
	if err != nil {
		t.Fatalf("从相似用户池召回 user-to-user 候选失败: %v", err)
	}
	if len(list) != 1 || list[0].GoodsId() != 3 || list[0].Score.UserNeighborScore != 4 {
		t.Fatalf("user-to-user 池召回结果不符合预期: %+v", list)
	}
}

// TestRecallUserToUserFallbackWhenPoolHasNoItems 校验旧格式相似用户池会自动回退事实源。
func TestRecallUserToUserFallbackWhenPoolHasNoItems(t *testing.T) {
	manager := openRecallTestManager(t)
	defer func() {
		err := manager.Close()
		if err != nil {
			t.Fatalf("关闭 recall 测试 LevelDB 失败: %v", err)
		}
	}()

	poolStore := &cachex.PoolStore{Driver: manager}
	var err error
	err = poolStore.SaveUserNeighborPool(101, &recommendv1.RecommendUserNeighborPool{
		NeighborUserIds: []int64{201, 202},
	})
	if err != nil {
		t.Fatalf("保存旧格式相似用户池失败: %v", err)
	}

	dependencies := buildRecallDependencies()
	var list []*model.Candidate
	list, err = RecallUserToUser(context.Background(), Request{
		Scene: model.SceneHome,
		Actor: model.Actor{
			Type: core.ActorTypeUser,
			Id:   101,
		},
		Dependencies: dependencies,
		PoolStore:    poolStore,
		Limit:        3,
	})
	if err != nil {
		t.Fatalf("旧格式相似用户池回退事实源失败: %v", err)
	}
	if len(list) != 1 || list[0].GoodsId() != 3 || list[0].Score.UserNeighborScore != 4 {
		t.Fatalf("旧格式相似用户池回退结果不符合预期: %+v", list)
	}
}

func TestRecallSessionContext(t *testing.T) {
	dependencies := buildRecallDependencies()
	list, err := RecallSessionContext(context.Background(), Request{
		Actor: model.Actor{
			Type: core.ActorTypeUser,
			Id:   101,
		},
		Dependencies: dependencies,
		Limit:        3,
	})
	if err != nil {
		t.Fatalf("会话召回失败: %v", err)
	}
	if len(list) == 0 {
		t.Fatal("会话召回应返回至少一个候选商品")
	}
	if list[0].Score.SessionScore <= 0 {
		t.Fatalf("会话召回得分未正确回填: %+v", list[0])
	}
}

// TestRecallSessionContextFromRuntimeState 校验会话召回优先读取运行态会话缓存。
func TestRecallSessionContextFromRuntimeState(t *testing.T) {
	manager := openRecallTestManager(t)
	defer func() {
		err := manager.Close()
		if err != nil {
			t.Fatalf("关闭 recall 测试 LevelDB 失败: %v", err)
		}
	}()

	runtimeStore := &cachex.RuntimeStore{Driver: manager}
	var err error
	err = runtimeStore.SaveSessionState(int32(core.ActorTypeUser), 101, "session-a", &recommendv1.RecommendSessionState{
		RecentCartGoodsIds: []int64{2},
	})
	if err != nil {
		t.Fatalf("保存运行态会话缓存失败: %v", err)
	}

	dependencies := buildRecallDependencies()
	var list []*model.Candidate
	list, err = RecallSessionContext(context.Background(), Request{
		Actor: model.Actor{
			Type:      core.ActorTypeUser,
			Id:        101,
			SessionId: "session-a",
		},
		Dependencies: dependencies,
		RuntimeStore: runtimeStore,
		Limit:        3,
	})
	if err != nil {
		t.Fatalf("从运行态会话缓存召回失败: %v", err)
	}
	if len(list) != 1 || list[0].GoodsId() != 1 || list[0].Score.SessionScore <= 0 {
		t.Fatalf("运行态会话缓存召回结果不符合预期: %+v", list)
	}
}

// TestRecallGoodsRelationFromPool 校验商品关联召回优先读取离线池。
func TestRecallGoodsRelationFromPool(t *testing.T) {
	manager := openRecallTestManager(t)
	defer func() {
		err := manager.Close()
		if err != nil {
			t.Fatalf("关闭 recall 测试 LevelDB 失败: %v", err)
		}
	}()

	poolStore := &cachex.PoolStore{Driver: manager}
	err := poolStore.SaveRelatedGoodsPool("goods_detail", 1, &recommendv1.RecommendRelatedGoodsPool{
		Items: []*recommendv1.RecommendCandidateItem{
			{GoodsId: 3, Score: 9, RecallSources: []string{RecallSourceGoodsRelation}},
		},
	})
	if err != nil {
		t.Fatalf("保存商品关联池失败: %v", err)
	}

	dependencies := buildRecallDependencies()
	list, err := RecallGoodsRelation(context.Background(), Request{
		Scene: model.SceneGoodsDetail,
		Context: model.RequestContext{
			GoodsId: 1,
		},
		Dependencies: dependencies,
		PoolStore:    poolStore,
	})
	if err != nil {
		t.Fatalf("从商品关联池召回失败: %v", err)
	}
	if len(list) != 1 || list[0].GoodsId() != 3 || list[0].Score.RelationScore != 9 {
		t.Fatalf("商品关联池召回结果不符合预期: %+v", list)
	}
}

// TestRecallCollaborativeFromPool 校验协同过滤召回优先读取离线池。
func TestRecallCollaborativeFromPool(t *testing.T) {
	manager := openRecallTestManager(t)
	defer func() {
		err := manager.Close()
		if err != nil {
			t.Fatalf("关闭 recall 测试 LevelDB 失败: %v", err)
		}
	}()

	poolStore := &cachex.PoolStore{Driver: manager}
	err := poolStore.SaveCollaborativePool("home", 101, &recommendv1.RecommendCollaborativePool{
		Items: []*recommendv1.RecommendCandidateItem{
			{GoodsId: 3, Score: 7, RecallSources: []string{RecallSourceCollaborative}},
		},
	})
	if err != nil {
		t.Fatalf("保存协同过滤池失败: %v", err)
	}

	dependencies := buildRecallDependencies()
	list, err := RecallCollaborative(context.Background(), Request{
		Scene: model.SceneHome,
		Actor: model.Actor{
			Type: core.ActorTypeUser,
			Id:   101,
		},
		Dependencies: dependencies,
		PoolStore:    poolStore,
	})
	if err != nil {
		t.Fatalf("从协同过滤池召回失败: %v", err)
	}
	if len(list) != 1 || list[0].GoodsId() != 3 || list[0].Score.CollaborativeScore != 7 {
		t.Fatalf("协同过滤池召回结果不符合预期: %+v", list)
	}
}

// TestRecallExternalFromPool 校验外部召回优先读取离线池。
func TestRecallExternalFromPool(t *testing.T) {
	manager := openRecallTestManager(t)
	defer func() {
		err := manager.Close()
		if err != nil {
			t.Fatalf("关闭 recall 测试 LevelDB 失败: %v", err)
		}
	}()

	poolStore := &cachex.PoolStore{Driver: manager}
	err := poolStore.SaveExternalPool("home", "campaign", 0, 0, &recommendv1.RecommendExternalPool{
		Items: []*recommendv1.RecommendCandidateItem{
			{GoodsId: 1, Score: 8, RecallSources: []string{RecallSourceExternal}},
		},
	})
	if err != nil {
		t.Fatalf("保存外部池失败: %v", err)
	}

	dependencies := buildRecallDependencies()
	list, err := RecallExternal(context.Background(), Request{
		Scene: model.SceneHome,
		Context: model.RequestContext{
			ExternalStrategy: "campaign",
		},
		Actor: model.Actor{
			Type: core.ActorTypeAnonymous,
			Id:   0,
		},
		Dependencies: dependencies,
		PoolStore:    poolStore,
	})
	if err != nil {
		t.Fatalf("从外部池召回失败: %v", err)
	}
	if len(list) != 1 || list[0].GoodsId() != 1 || list[0].Score.ExternalScore != 8 {
		t.Fatalf("外部池召回结果不符合预期: %+v", list)
	}
}

type fakeGoodsSource struct {
	goodsById       map[int64]*contract.Goods
	goodsByCategory map[int64][]*contract.Goods
	latestGoods     []*contract.Goods
}

func (s *fakeGoodsSource) GetGoods(_ context.Context, goodsId int64) (*contract.Goods, error) {
	return s.goodsById[goodsId], nil
}

func (s *fakeGoodsSource) ListGoods(_ context.Context, goodsIds []int64) ([]*contract.Goods, error) {
	list := make([]*contract.Goods, 0, len(goodsIds))
	for _, goodsId := range goodsIds {
		item, ok := s.goodsById[goodsId]
		if ok {
			list = append(list, item)
		}
	}
	return list, nil
}

func (s *fakeGoodsSource) ListGoodsByCategoryIds(_ context.Context, categoryIds []int64, limit int32) ([]*contract.Goods, error) {
	list := make([]*contract.Goods, 0)
	for _, categoryId := range categoryIds {
		list = append(list, s.goodsByCategory[categoryId]...)
	}
	if len(list) > int(limit) {
		list = list[:limit]
	}
	return list, nil
}

func (s *fakeGoodsSource) ListLatestGoods(_ context.Context, limit int32) ([]*contract.Goods, error) {
	list := append([]*contract.Goods(nil), s.latestGoods...)
	if len(list) > int(limit) {
		list = list[:limit]
	}
	return list, nil
}

type fakeRecommendSource struct{}

func (s *fakeRecommendSource) ListSceneHotGoods(_ context.Context, _ string, _ time.Time, _ int32) ([]*contract.WeightedGoods, error) {
	return []*contract.WeightedGoods{
		{GoodsId: 1, Score: 5},
		{GoodsId: 2, Score: 3},
	}, nil
}

func (s *fakeRecommendSource) ListGlobalHotGoods(_ context.Context, _ time.Time, _ int32) ([]*contract.WeightedGoods, error) {
	return []*contract.WeightedGoods{
		{GoodsId: 2, Score: 4},
	}, nil
}

func (s *fakeRecommendSource) ListRelatedGoods(_ context.Context, goodsId int64, _ int32) ([]*contract.WeightedGoods, error) {
	if goodsId == 1 {
		return []*contract.WeightedGoods{
			{GoodsId: 2, Score: 3},
			{GoodsId: 3, Score: 2},
		}, nil
	}
	return []*contract.WeightedGoods{
		{GoodsId: 1, Score: 1},
	}, nil
}

func (s *fakeRecommendSource) ListUserGoodsPreference(_ context.Context, _ int64, _ int32) ([]*contract.WeightedGoods, error) {
	return []*contract.WeightedGoods{
		{GoodsId: 1, Score: 6},
	}, nil
}

func (s *fakeRecommendSource) ListUserCategoryPreference(_ context.Context, _ int64, _ int32) ([]*contract.WeightedCategory, error) {
	return []*contract.WeightedCategory{
		{CategoryId: 11, Score: 5},
	}, nil
}

func (s *fakeRecommendSource) ListNeighborUsers(_ context.Context, _ int64, _ int32) ([]*contract.WeightedUser, error) {
	return []*contract.WeightedUser{
		{UserId: 201, Score: 3},
	}, nil
}

func (s *fakeRecommendSource) ListUserToUserGoods(_ context.Context, _ int64, _ int32) ([]*contract.WeightedGoods, error) {
	return []*contract.WeightedGoods{
		{GoodsId: 3, Score: 4},
	}, nil
}

func (s *fakeRecommendSource) ListCollaborativeGoods(_ context.Context, _ int64, _ int32) ([]*contract.WeightedGoods, error) {
	return []*contract.WeightedGoods{
		{GoodsId: 2, Score: 2},
	}, nil
}

func (s *fakeRecommendSource) ListExternalGoods(_ context.Context, _, _ string, _ int32, _ int64, _ int32) ([]*contract.WeightedGoods, error) {
	return []*contract.WeightedGoods{
		{GoodsId: 1, Score: 3},
	}, nil
}

func (s *fakeRecommendSource) ListRequestFacts(_ context.Context, _ string, _, _ time.Time) ([]*contract.RequestFact, error) {
	return nil, nil
}

func (s *fakeRecommendSource) ListExposureFacts(_ context.Context, _ string, _, _ time.Time) ([]*contract.ExposureFact, error) {
	return nil, nil
}

func (s *fakeRecommendSource) ListActionFacts(_ context.Context, _ string, _, _ time.Time) ([]*contract.ActionFact, error) {
	return nil, nil
}

type fakeBehaviorSource struct{}

func (s *fakeBehaviorSource) ListSessionEvents(_ context.Context, _ int32, _ int64, _ int32) ([]*contract.SessionEvent, error) {
	return []*contract.SessionEvent{
		{GoodsId: 1, EventType: "click"},
		{GoodsId: 2, EventType: "view"},
	}, nil
}

func (s *fakeBehaviorSource) ListBehaviorEvents(_ context.Context, _ int32, _ int64, _ time.Time, _ int32) ([]*contract.BehaviorEvent, error) {
	return nil, nil
}

func buildRecallDependencies() core.Dependencies {
	return core.Dependencies{
		Goods: &fakeGoodsSource{
			goodsById: map[int64]*contract.Goods{
				1: {Id: 1, CategoryId: 11},
				2: {Id: 2, CategoryId: 12},
				3: {Id: 3, CategoryId: 11},
			},
			goodsByCategory: map[int64][]*contract.Goods{
				11: {
					{Id: 1, CategoryId: 11},
					{Id: 3, CategoryId: 11},
				},
			},
		},
		Behavior:  &fakeBehaviorSource{},
		Recommend: &fakeRecommendSource{},
	}
}

// openRecallTestManager 打开 recall 包测试使用的 LevelDB 管理器。
func openRecallTestManager(t *testing.T) *cacheleveldb.Manager {
	t.Helper()

	rootPath := t.TempDir()
	layout := contract.LevelDbLayout{
		PoolPath:    filepath.Join(rootPath, "pool.db"),
		RuntimePath: filepath.Join(rootPath, "runtime.db"),
		TracePath:   filepath.Join(rootPath, "trace.db"),
	}

	manager, err := cacheleveldb.OpenManagerByLayout(layout)
	if err != nil {
		t.Fatalf("打开 recall 测试 LevelDB 管理器失败: %v", err)
	}
	return manager
}

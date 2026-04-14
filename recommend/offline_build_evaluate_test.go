package recommend

import (
	"context"
	"fmt"
	"math"
	"recommend/contract"
	cachex "recommend/internal/cache"
	"sort"
	"testing"
	"time"
)

// TestBuildOfflinePools 校验离线构建会把候选池正确写入缓存。
func TestBuildOfflinePools(t *testing.T) {
	ctx := context.Background()
	cacheSource := newRuntimeTestCacheSource(t)
	dependencies := Dependencies{
		Goods: &offlineTestGoodsSource{
			goodsById: map[int64]*contract.Goods{
				1: {Id: 1, CategoryId: 11},
				2: {Id: 2, CategoryId: 12},
				3: {Id: 3, CategoryId: 11},
				4: {Id: 4, CategoryId: 13},
			},
			goodsByCategory: map[int64][]*contract.Goods{
				11: {
					{Id: 1, CategoryId: 11},
					{Id: 3, CategoryId: 11},
				},
			},
			latestGoods: []*contract.Goods{
				{Id: 4, CategoryId: 13},
				{Id: 3, CategoryId: 11},
			},
		},
		Recommend: &offlineTestRecommendSource{
			sceneHotGoods: map[string][]*contract.WeightedGoods{
				"home": {
					{GoodsId: 1, Score: 5},
					{GoodsId: 2, Score: 4},
				},
			},
			globalHotGoods: []*contract.WeightedGoods{
				{GoodsId: 2, Score: 3},
			},
			relatedGoods: map[int64][]*contract.WeightedGoods{
				1: {
					{GoodsId: 2, Score: 6},
				},
			},
			userGoodsPref: map[int64][]*contract.WeightedGoods{
				101: {
					{GoodsId: 1, Score: 7},
				},
			},
			userCategoryPref: map[int64][]*contract.WeightedCategory{
				101: {
					{CategoryId: 11, Score: 2},
				},
			},
			neighborUsers: map[int64][]*contract.WeightedUser{
				101: {
					{UserId: 201, Score: 0.9},
					{UserId: 202, Score: 0.8},
				},
			},
			userToUserGoods: map[int64][]*contract.WeightedGoods{
				101: {
					{GoodsId: 3, Score: 4},
				},
			},
			collaborativeGoods: map[int64][]*contract.WeightedGoods{
				101: {
					{GoodsId: 3, Score: 8},
				},
			},
			externalGoods: map[string][]*contract.WeightedGoods{
				"home:campaign:0:0": {
					{GoodsId: 4, Score: 9},
				},
			},
		},
		Cache: cacheSource,
	}
	recommender := newTestRecommend(t, WithDependencies(dependencies))

	buildResult, err := recommender.BuildNonPersonalized(ctx, BuildNonPersonalizedRequest{
		Scenes: []Scene{SceneHome},
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("构建非个性化池失败: %v", err)
	}
	if buildResult.KeyCount != 1 {
		t.Fatalf("非个性化池构建写入数量不符合预期: %+v", buildResult)
	}

	_, err = recommender.BuildUserCandidate(ctx, BuildUserCandidateRequest{
		UserIds: []int64{101},
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("构建用户候选池失败: %v", err)
	}

	_, err = recommender.BuildGoodsRelation(ctx, BuildGoodsRelationRequest{
		GoodsIds: []int64{1},
		Limit:    10,
	})
	if err != nil {
		t.Fatalf("构建商品关联池失败: %v", err)
	}

	_, err = recommender.BuildUserToUser(ctx, BuildUserToUserRequest{
		UserIds:       []int64{101},
		NeighborLimit: 10,
	})
	if err != nil {
		t.Fatalf("构建相似用户池失败: %v", err)
	}

	_, err = recommender.BuildCollaborative(ctx, BuildCollaborativeRequest{
		UserIds: []int64{101},
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("构建协同过滤池失败: %v", err)
	}

	_, err = recommender.BuildExternal(ctx, BuildExternalRequest{
		Scenes:     []Scene{SceneHome},
		Strategies: []string{"campaign"},
		ActorType:  ActorTypeAnonymous,
		ActorIds:   []int64{0},
		Limit:      10,
	})
	if err != nil {
		t.Fatalf("构建外部池失败: %v", err)
	}

	manager := openRuntimeTestManager(t, cacheSource)
	defer func() {
		closeErr := manager.Close()
		if closeErr != nil {
			t.Fatalf("关闭构建测试 LevelDB 失败: %v", closeErr)
		}
	}()

	store := &cachex.PoolStore{Driver: manager}

	candidatePool, err := store.GetCandidatePool("home", int32(ActorTypeAnonymous), 0)
	if err != nil {
		t.Fatalf("读取非个性化池失败: %v", err)
	}
	if len(candidatePool.GetItems()) < 3 {
		t.Fatalf("非个性化池商品数量不符合预期: %+v", candidatePool)
	}
	if candidatePool.GetItems()[0].GetGoodsId() != 2 {
		t.Fatalf("非个性化池排序不符合预期: %+v", candidatePool.GetItems())
	}
	if candidatePool.GetItems()[0].GetSourceScores()["scene_hot"] != 4 || candidatePool.GetItems()[0].GetSourceScores()["global_hot"] != 3 {
		t.Fatalf("非个性化池来源分值不符合预期: %+v", candidatePool.GetItems()[0].GetSourceScores())
	}

	userCandidatePool, err := store.GetUserCandidatePool("home", 101)
	if err != nil {
		t.Fatalf("读取用户候选池失败: %v", err)
	}
	if len(userCandidatePool.GetItems()) == 0 || userCandidatePool.GetItems()[0].GetGoodsId() != 1 {
		t.Fatalf("用户候选池结果不符合预期: %+v", userCandidatePool.GetItems())
	}
	if userCandidatePool.GetItems()[0].GetSourceScores()["user_goods_pref"] != 7 || userCandidatePool.GetItems()[0].GetSourceScores()["user_category_pref"] != 2 {
		t.Fatalf("用户候选池来源分值不符合预期: %+v", userCandidatePool.GetItems()[0].GetSourceScores())
	}

	relatedPool, err := store.GetRelatedGoodsPool("goods_detail", 1)
	if err != nil {
		t.Fatalf("读取商品关联池失败: %v", err)
	}
	if len(relatedPool.GetItems()) != 1 || relatedPool.GetItems()[0].GetGoodsId() != 2 {
		t.Fatalf("商品关联池结果不符合预期: %+v", relatedPool.GetItems())
	}

	userNeighborPool, err := store.GetUserNeighborPool(101)
	if err != nil {
		t.Fatalf("读取相似用户池失败: %v", err)
	}
	if len(userNeighborPool.GetNeighborUserIds()) != 2 || userNeighborPool.GetNeighborUserIds()[0] != 201 {
		t.Fatalf("相似用户池结果不符合预期: %+v", userNeighborPool.GetNeighborUserIds())
	}
	if len(userNeighborPool.GetItems()) != 1 || userNeighborPool.GetItems()[0].GetGoodsId() != 3 {
		t.Fatalf("相似用户池商品结果不符合预期: %+v", userNeighborPool.GetItems())
	}
	if userNeighborPool.GetItems()[0].GetSourceScores()["user_to_user"] != 4 {
		t.Fatalf("相似用户池来源分值不符合预期: %+v", userNeighborPool.GetItems()[0].GetSourceScores())
	}

	collaborativePool, err := store.GetCollaborativePool("home", 101)
	if err != nil {
		t.Fatalf("读取协同过滤池失败: %v", err)
	}
	if len(collaborativePool.GetItems()) != 1 || collaborativePool.GetItems()[0].GetGoodsId() != 3 {
		t.Fatalf("协同过滤池结果不符合预期: %+v", collaborativePool.GetItems())
	}

	externalPool, err := store.GetExternalPool("home", "campaign", int32(ActorTypeAnonymous), 0)
	if err != nil {
		t.Fatalf("读取外部池失败: %v", err)
	}
	if len(externalPool.GetItems()) != 1 || externalPool.GetItems()[0].GetGoodsId() != 4 {
		t.Fatalf("外部池结果不符合预期: %+v", externalPool.GetItems())
	}
}

// TestEvaluateOffline 校验离线评估指标口径。
func TestEvaluateOffline(t *testing.T) {
	ctx := context.Background()
	dependencies := Dependencies{
		Recommend: &offlineTestRecommendSource{
			requestFacts: map[string][]*contract.RequestFact{
				"home": {
					{RequestId: "request-1", GoodsIds: []int64{1, 2, 3}},
					{RequestId: "request-2", GoodsIds: []int64{3, 4}},
				},
			},
			exposureFacts: map[string][]*contract.ExposureFact{
				"home": {
					{RequestId: "request-1", GoodsIds: []int64{1, 2}},
					{RequestId: "request-2", GoodsIds: []int64{3, 4}},
				},
			},
			actionFacts: map[string][]*contract.ActionFact{
				"home": {
					{RequestId: "request-1", GoodsId: 2, EventType: "click"},
					{RequestId: "request-1", GoodsId: 2, EventType: "order_create"},
					{RequestId: "request-2", GoodsId: 3, EventType: "order_pay"},
				},
			},
		},
	}
	recommender := newTestRecommend(t, WithDependencies(dependencies))

	result, err := recommender.EvaluateOffline(ctx, EvaluateRequest{
		Scenes: []Scene{SceneHome},
		TopK:   2,
	})
	if err != nil {
		t.Fatalf("离线评估失败: %v", err)
	}
	if len(result.Scenes) != 1 {
		t.Fatalf("离线评估场景数量不符合预期: %+v", result)
	}

	metric := result.Scenes[0]
	if metric.RequestCount != 2 || metric.ExposureCount != 4 {
		t.Fatalf("离线评估样本统计不符合预期: %+v", metric)
	}
	if metric.ClickCount != 1 || metric.OrderCount != 1 || metric.PayCount != 1 {
		t.Fatalf("离线评估行为统计不符合预期: %+v", metric)
	}
	if math.Abs(metric.Ctr-0.25) > 0.0001 {
		t.Fatalf("CTR 不符合预期: %+v", metric)
	}
	if math.Abs(metric.OrderRate-1.0) > 0.0001 || math.Abs(metric.PayRate-1.0) > 0.0001 {
		t.Fatalf("转化率指标不符合预期: %+v", metric)
	}
	if math.Abs(metric.Precision-0.5) > 0.0001 || math.Abs(metric.Recall-1.0) > 0.0001 {
		t.Fatalf("排序指标不符合预期: %+v", metric)
	}
	if metric.Ndcg <= 0 {
		t.Fatalf("NDCG 应大于 0: %+v", metric)
	}
}

// TestRebuild 校验一键重建会统一编排离线构建和离线评估。
func TestRebuild(t *testing.T) {
	ctx := context.Background()
	cacheSource := newRuntimeTestCacheSource(t)
	dependencies := Dependencies{
		Goods: &offlineTestGoodsSource{
			goodsById: map[int64]*contract.Goods{
				1: {Id: 1, CategoryId: 11},
				2: {Id: 2, CategoryId: 12},
				3: {Id: 3, CategoryId: 11},
				4: {Id: 4, CategoryId: 13},
			},
			goodsByCategory: map[int64][]*contract.Goods{
				11: {
					{Id: 1, CategoryId: 11},
					{Id: 3, CategoryId: 11},
				},
			},
			latestGoods: []*contract.Goods{
				{Id: 4, CategoryId: 13},
				{Id: 3, CategoryId: 11},
			},
		},
		Recommend: &offlineTestRecommendSource{
			sceneHotGoods: map[string][]*contract.WeightedGoods{
				"home": {
					{GoodsId: 1, Score: 5},
					{GoodsId: 2, Score: 4},
				},
			},
			globalHotGoods: []*contract.WeightedGoods{
				{GoodsId: 2, Score: 3},
			},
			relatedGoods: map[int64][]*contract.WeightedGoods{
				1: {
					{GoodsId: 2, Score: 6},
				},
			},
			userGoodsPref: map[int64][]*contract.WeightedGoods{
				101: {
					{GoodsId: 1, Score: 7},
				},
			},
			userCategoryPref: map[int64][]*contract.WeightedCategory{
				101: {
					{CategoryId: 11, Score: 2},
				},
			},
			neighborUsers: map[int64][]*contract.WeightedUser{
				101: {
					{UserId: 201, Score: 0.9},
				},
			},
			userToUserGoods: map[int64][]*contract.WeightedGoods{
				101: {
					{GoodsId: 3, Score: 4},
				},
			},
			collaborativeGoods: map[int64][]*contract.WeightedGoods{
				101: {
					{GoodsId: 3, Score: 8},
				},
			},
			externalGoods: map[string][]*contract.WeightedGoods{
				"home:campaign:0:0": {
					{GoodsId: 4, Score: 9},
				},
			},
			requestFacts: map[string][]*contract.RequestFact{
				"home": {
					{RequestId: "request-1", GoodsIds: []int64{1, 2}},
				},
			},
			exposureFacts: map[string][]*contract.ExposureFact{
				"home": {
					{RequestId: "request-1", GoodsIds: []int64{1, 2}},
				},
			},
			actionFacts: map[string][]*contract.ActionFact{
				"home": {
					{RequestId: "request-1", GoodsId: 1, EventType: "click"},
				},
			},
		},
		Cache: cacheSource,
	}
	recommender := newTestRecommend(
		t,
		WithDependencies(dependencies),
		WithMaterializeConfig(MaterializeConfig{
			DefaultLimit:               10,
			DefaultNeighborLimit:       10,
			DefaultScenes:              []Scene{SceneHome},
			EnableEvaluateAfterRebuild: true,
		}),
		WithEvaluateConfig(EvaluateConfig{DefaultTopK: 2}),
	)

	result, err := recommender.Rebuild(ctx, RebuildRequest{
		UserIds:    []int64{101},
		GoodsIds:   []int64{1},
		Strategies: []string{"campaign"},
		ActorType:  ActorTypeAnonymous,
		ActorIds:   []int64{0},
	})
	if err != nil {
		t.Fatalf("一键重建失败: %v", err)
	}
	if len(result.Builds) != 7 {
		t.Fatalf("一键重建未覆盖全部离线池: %+v", result.Builds)
	}
	if result.Builds[len(result.Builds)-1].Scope != "training" {
		t.Fatalf("一键重建未串联训练步骤: %+v", result.Builds)
	}
	if result.Evaluation == nil || len(result.Evaluation.Scenes) != 1 {
		t.Fatalf("一键重建未串联离线评估: %+v", result.Evaluation)
	}
}

type offlineTestGoodsSource struct {
	goodsById       map[int64]*contract.Goods
	goodsByCategory map[int64][]*contract.Goods
	latestGoods     []*contract.Goods
}

// GetGoods 按商品编号返回测试商品。
func (s *offlineTestGoodsSource) GetGoods(_ context.Context, goodsId int64) (*contract.Goods, error) {
	return s.goodsById[goodsId], nil
}

// ListGoods 按商品编号列表返回测试商品集合。
func (s *offlineTestGoodsSource) ListGoods(_ context.Context, goodsIds []int64) ([]*contract.Goods, error) {
	list := make([]*contract.Goods, 0, len(goodsIds))
	for _, goodsId := range goodsIds {
		if item, ok := s.goodsById[goodsId]; ok {
			list = append(list, item)
		}
	}
	sort.SliceStable(list, func(i int, j int) bool {
		return list[i].Id < list[j].Id
	})
	return list, nil
}

// ListGoodsByCategoryIds 按类目返回测试商品集合。
func (s *offlineTestGoodsSource) ListGoodsByCategoryIds(_ context.Context, categoryIds []int64, limit int32) ([]*contract.Goods, error) {
	list := make([]*contract.Goods, 0)
	for _, categoryId := range categoryIds {
		list = append(list, s.goodsByCategory[categoryId]...)
	}
	if len(list) > int(limit) {
		list = list[:limit]
	}
	return list, nil
}

// ListLatestGoods 返回最新商品列表。
func (s *offlineTestGoodsSource) ListLatestGoods(_ context.Context, limit int32) ([]*contract.Goods, error) {
	list := append([]*contract.Goods(nil), s.latestGoods...)
	if len(list) > int(limit) {
		list = list[:limit]
	}
	return list, nil
}

type offlineTestRecommendSource struct {
	sceneHotGoods      map[string][]*contract.WeightedGoods
	globalHotGoods     []*contract.WeightedGoods
	relatedGoods       map[int64][]*contract.WeightedGoods
	userGoodsPref      map[int64][]*contract.WeightedGoods
	userCategoryPref   map[int64][]*contract.WeightedCategory
	neighborUsers      map[int64][]*contract.WeightedUser
	userToUserGoods    map[int64][]*contract.WeightedGoods
	collaborativeGoods map[int64][]*contract.WeightedGoods
	externalGoods      map[string][]*contract.WeightedGoods
	requestFacts       map[string][]*contract.RequestFact
	exposureFacts      map[string][]*contract.ExposureFact
	actionFacts        map[string][]*contract.ActionFact
}

// ListSceneHotGoods 返回测试场景热销商品。
func (s *offlineTestRecommendSource) ListSceneHotGoods(_ context.Context, scene string, _ time.Time, _ int32) ([]*contract.WeightedGoods, error) {
	return cloneWeightedGoods(s.sceneHotGoods[scene]), nil
}

// ListGlobalHotGoods 返回测试全站热销商品。
func (s *offlineTestRecommendSource) ListGlobalHotGoods(_ context.Context, _ time.Time, _ int32) ([]*contract.WeightedGoods, error) {
	return cloneWeightedGoods(s.globalHotGoods), nil
}

// ListRelatedGoods 返回测试商品关联结果。
func (s *offlineTestRecommendSource) ListRelatedGoods(_ context.Context, goodsId int64, _ int32) ([]*contract.WeightedGoods, error) {
	return cloneWeightedGoods(s.relatedGoods[goodsId]), nil
}

// ListUserGoodsPreference 返回测试用户商品偏好。
func (s *offlineTestRecommendSource) ListUserGoodsPreference(_ context.Context, userId int64, _ int32) ([]*contract.WeightedGoods, error) {
	return cloneWeightedGoods(s.userGoodsPref[userId]), nil
}

// ListUserCategoryPreference 返回测试用户类目偏好。
func (s *offlineTestRecommendSource) ListUserCategoryPreference(_ context.Context, userId int64, _ int32) ([]*contract.WeightedCategory, error) {
	return cloneWeightedCategories(s.userCategoryPref[userId]), nil
}

// ListNeighborUsers 返回测试相似用户列表。
func (s *offlineTestRecommendSource) ListNeighborUsers(_ context.Context, userId int64, _ int32) ([]*contract.WeightedUser, error) {
	return cloneWeightedUsers(s.neighborUsers[userId]), nil
}

// ListUserToUserGoods 返回测试 user-to-user 商品结果。
func (s *offlineTestRecommendSource) ListUserToUserGoods(_ context.Context, userId int64, _ int32) ([]*contract.WeightedGoods, error) {
	return cloneWeightedGoods(s.userToUserGoods[userId]), nil
}

// ListCollaborativeGoods 返回测试协同过滤商品结果。
func (s *offlineTestRecommendSource) ListCollaborativeGoods(_ context.Context, userId int64, _ int32) ([]*contract.WeightedGoods, error) {
	return cloneWeightedGoods(s.collaborativeGoods[userId]), nil
}

// ListExternalGoods 返回测试外部推荐商品结果。
func (s *offlineTestRecommendSource) ListExternalGoods(_ context.Context, scene string, strategy string, actorType int32, actorId int64, _ int32) ([]*contract.WeightedGoods, error) {
	key := buildExternalKey(scene, strategy, actorType, actorId)
	return cloneWeightedGoods(s.externalGoods[key]), nil
}

// ListRequestFacts 返回测试推荐请求事实。
func (s *offlineTestRecommendSource) ListRequestFacts(_ context.Context, scene string, _, _ time.Time) ([]*contract.RequestFact, error) {
	return cloneRequestFacts(s.requestFacts[scene]), nil
}

// ListExposureFacts 返回测试推荐曝光事实。
func (s *offlineTestRecommendSource) ListExposureFacts(_ context.Context, scene string, _, _ time.Time) ([]*contract.ExposureFact, error) {
	return cloneExposureFacts(s.exposureFacts[scene]), nil
}

// ListActionFacts 返回测试推荐行为事实。
func (s *offlineTestRecommendSource) ListActionFacts(_ context.Context, scene string, _, _ time.Time) ([]*contract.ActionFact, error) {
	return cloneActionFacts(s.actionFacts[scene]), nil
}

// cloneWeightedGoods 复制带分商品列表，避免测试数据被调用方修改。
func cloneWeightedGoods(list []*contract.WeightedGoods) []*contract.WeightedGoods {
	result := make([]*contract.WeightedGoods, 0, len(list))
	for _, item := range list {
		if item == nil {
			continue
		}
		result = append(result, &contract.WeightedGoods{
			GoodsId: item.GoodsId,
			Score:   item.Score,
		})
	}
	return result
}

// cloneWeightedCategories 复制带分类分值列表。
func cloneWeightedCategories(list []*contract.WeightedCategory) []*contract.WeightedCategory {
	result := make([]*contract.WeightedCategory, 0, len(list))
	for _, item := range list {
		if item == nil {
			continue
		}
		result = append(result, &contract.WeightedCategory{
			CategoryId: item.CategoryId,
			Score:      item.Score,
		})
	}
	return result
}

// cloneWeightedUsers 复制带分用户列表。
func cloneWeightedUsers(list []*contract.WeightedUser) []*contract.WeightedUser {
	result := make([]*contract.WeightedUser, 0, len(list))
	for _, item := range list {
		if item == nil {
			continue
		}
		result = append(result, &contract.WeightedUser{
			UserId: item.UserId,
			Score:  item.Score,
		})
	}
	return result
}

// cloneRequestFacts 复制推荐请求事实列表。
func cloneRequestFacts(list []*contract.RequestFact) []*contract.RequestFact {
	result := make([]*contract.RequestFact, 0, len(list))
	for _, item := range list {
		if item == nil {
			continue
		}
		result = append(result, &contract.RequestFact{
			RequestId: item.RequestId,
			Scene:     item.Scene,
			ActorType: item.ActorType,
			ActorId:   item.ActorId,
			CreatedAt: item.CreatedAt,
			GoodsIds:  append([]int64(nil), item.GoodsIds...),
		})
	}
	return result
}

// cloneExposureFacts 复制推荐曝光事实列表。
func cloneExposureFacts(list []*contract.ExposureFact) []*contract.ExposureFact {
	result := make([]*contract.ExposureFact, 0, len(list))
	for _, item := range list {
		if item == nil {
			continue
		}
		result = append(result, &contract.ExposureFact{
			RequestId: item.RequestId,
			Scene:     item.Scene,
			ActorType: item.ActorType,
			ActorId:   item.ActorId,
			CreatedAt: item.CreatedAt,
			GoodsIds:  append([]int64(nil), item.GoodsIds...),
		})
	}
	return result
}

// cloneActionFacts 复制推荐行为事实列表。
func cloneActionFacts(list []*contract.ActionFact) []*contract.ActionFact {
	result := make([]*contract.ActionFact, 0, len(list))
	for _, item := range list {
		if item == nil {
			continue
		}
		result = append(result, &contract.ActionFact{
			RequestId: item.RequestId,
			Scene:     item.Scene,
			ActorType: item.ActorType,
			ActorId:   item.ActorId,
			GoodsId:   item.GoodsId,
			EventType: item.EventType,
			GoodsNum:  item.GoodsNum,
			CreatedAt: item.CreatedAt,
		})
	}
	return result
}

// buildExternalKey 生成测试外部池查询键。
func buildExternalKey(scene string, strategy string, actorType int32, actorId int64) string {
	return fmt.Sprintf("%s:%s:%d:%d", scene, strategy, actorType, actorId)
}

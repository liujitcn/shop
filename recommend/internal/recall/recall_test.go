package recall

import (
	"context"
	"recommend"
	"recommend/contract"
	"recommend/internal/model"
	"testing"
	"time"
)

func TestRecallLatest(t *testing.T) {
	dependencies := recommend.Dependencies{
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

func TestRecallSessionContext(t *testing.T) {
	dependencies := buildRecallDependencies()
	list, err := RecallSessionContext(context.Background(), Request{
		Actor: model.Actor{
			Type: recommend.ActorTypeUser,
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

func buildRecallDependencies() recommend.Dependencies {
	return recommend.Dependencies{
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

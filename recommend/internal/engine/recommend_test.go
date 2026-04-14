package engine

import (
	"context"
	"recommend"
	"recommend/contract"
	"testing"
	"time"
)

func TestRecommend(t *testing.T) {
	dependencies := recommend.Dependencies{
		Goods: &fakeGoodsSource{
			goodsById: map[int64]*contract.Goods{
				1: {Id: 1, CategoryId: 11, OnSale: true, InStock: true},
				2: {Id: 2, CategoryId: 12, OnSale: true, InStock: true},
				3: {Id: 3, CategoryId: 13, OnSale: true, InStock: true},
			},
			latestGoods: []*contract.Goods{
				{Id: 3, CategoryId: 13, OnSale: true, InStock: true},
			},
		},
		Recommend: &fakeRecommendSource{},
		Behavior:  &fakeBehaviorSource{},
		Order:     &fakeOrderSource{},
	}

	result, err := Recommend(context.Background(), dependencies, recommend.RecommendRequest{
		Scene: recommend.SceneHome,
		Pager: recommend.Pager{
			PageNum:  1,
			PageSize: 2,
		},
		Context: recommend.RecommendContext{
			RequestId: "trace-1",
		},
	})
	if err != nil {
		t.Fatalf("执行推荐主链路失败: %v", err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("推荐结果数量不符合预期: %d", len(result.Items))
	}
	if result.TraceId != "trace-1" {
		t.Fatalf("追踪编号不符合预期: %+v", result)
	}
}

type fakeGoodsSource struct {
	goodsById   map[int64]*contract.Goods
	latestGoods []*contract.Goods
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

func (s *fakeGoodsSource) ListGoodsByCategoryIds(_ context.Context, _ []int64, _ int32) ([]*contract.Goods, error) {
	return nil, nil
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
		{GoodsId: 1, Score: 6},
		{GoodsId: 2, Score: 5},
	}, nil
}

func (s *fakeRecommendSource) ListGlobalHotGoods(_ context.Context, _ time.Time, _ int32) ([]*contract.WeightedGoods, error) {
	return []*contract.WeightedGoods{
		{GoodsId: 2, Score: 4},
	}, nil
}

func (s *fakeRecommendSource) ListRelatedGoods(_ context.Context, _ int64, _ int32) ([]*contract.WeightedGoods, error) {
	return nil, nil
}

func (s *fakeRecommendSource) ListUserGoodsPreference(_ context.Context, _ int64, _ int32) ([]*contract.WeightedGoods, error) {
	return nil, nil
}

func (s *fakeRecommendSource) ListUserCategoryPreference(_ context.Context, _ int64, _ int32) ([]*contract.WeightedCategory, error) {
	return nil, nil
}

func (s *fakeRecommendSource) ListNeighborUsers(_ context.Context, _ int64, _ int32) ([]*contract.WeightedUser, error) {
	return nil, nil
}

func (s *fakeRecommendSource) ListUserToUserGoods(_ context.Context, _ int64, _ int32) ([]*contract.WeightedGoods, error) {
	return nil, nil
}

func (s *fakeRecommendSource) ListCollaborativeGoods(_ context.Context, _ int64, _ int32) ([]*contract.WeightedGoods, error) {
	return nil, nil
}

func (s *fakeRecommendSource) ListExternalGoods(_ context.Context, _, _ string, _ int32, _ int64, _ int32) ([]*contract.WeightedGoods, error) {
	return nil, nil
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
	return nil, nil
}

func (s *fakeBehaviorSource) ListBehaviorEvents(_ context.Context, _ int32, _ int64, _ time.Time, _ int32) ([]*contract.BehaviorEvent, error) {
	return nil, nil
}

type fakeOrderSource struct{}

func (s *fakeOrderSource) ListOrderGoods(_ context.Context, _ int64) ([]*contract.OrderGoods, error) {
	return nil, nil
}

func (s *fakeOrderSource) ListRecentPaidGoods(_ context.Context, _ int64, _ time.Time, _ int32) ([]*contract.OrderGoods, error) {
	return nil, nil
}

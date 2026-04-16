package feature

import (
	"testing"

	"shop/api/gen/go/common"
	recommendDomain "shop/pkg/recommend/domain"
)

// TestBuildAnonymousSignalLoadPlan 验证匿名态信号加载计划。
func TestBuildAnonymousSignalLoadPlan(t *testing.T) {
	request := &recommendDomain.GoodsRequest{
		Scene:   common.RecommendScene_GOODS_DETAIL,
		GoodsId: 99,
	}
	snapshot := SignalSnapshot{
		GoodsIds: []int64{1, 2, 3},
	}

	plan := BuildAnonymousSignalLoadPlan(request, snapshot)

	if plan.Scene != int32(common.RecommendScene_GOODS_DETAIL) {
		t.Fatalf("unexpected scene: %d", plan.Scene)
	}
	if len(plan.CandidateGoodsIds) != 3 {
		t.Fatalf("unexpected candidate goods ids: %+v", plan.CandidateGoodsIds)
	}
	if len(plan.CandidateCategoryIds) != 0 {
		t.Fatalf("unexpected candidate category ids: %+v", plan.CandidateCategoryIds)
	}
	if len(plan.RelationSourceGoodsIds) != 1 || plan.RelationSourceGoodsIds[0] != 99 {
		t.Fatalf("unexpected relation source goods ids: %+v", plan.RelationSourceGoodsIds)
	}
}

// TestBuildPersonalizedSignalLoadPlan 验证登录态信号加载计划。
func TestBuildPersonalizedSignalLoadPlan(t *testing.T) {
	request := &recommendDomain.GoodsRequest{
		Scene: common.RecommendScene_CART,
	}
	snapshot := SignalSnapshot{
		GoodsIds:    []int64{1, 2},
		CategoryIds: []int64{10, 20},
	}

	plan := BuildPersonalizedSignalLoadPlan(request, []int64{7, 8}, snapshot)

	if plan.Scene != int32(common.RecommendScene_CART) {
		t.Fatalf("unexpected scene: %d", plan.Scene)
	}
	if len(plan.CandidateGoodsIds) != 2 || plan.CandidateGoodsIds[0] != 1 || plan.CandidateGoodsIds[1] != 2 {
		t.Fatalf("unexpected candidate goods ids: %+v", plan.CandidateGoodsIds)
	}
	if len(plan.CandidateCategoryIds) != 2 || plan.CandidateCategoryIds[0] != 10 || plan.CandidateCategoryIds[1] != 20 {
		t.Fatalf("unexpected candidate category ids: %+v", plan.CandidateCategoryIds)
	}
	if len(plan.RelationSourceGoodsIds) != 2 || plan.RelationSourceGoodsIds[0] != 7 || plan.RelationSourceGoodsIds[1] != 8 {
		t.Fatalf("unexpected relation source goods ids: %+v", plan.RelationSourceGoodsIds)
	}
}

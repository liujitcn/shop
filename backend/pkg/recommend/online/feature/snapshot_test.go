package feature

import (
	"testing"

	app "shop/api/gen/go/app"
	"shop/api/gen/go/common"
	recommendDomain "shop/pkg/recommend/domain"
)

// TestBuildAnonymousSignalSnapshot 验证匿名态候选快照过滤规则。
func TestBuildAnonymousSignalSnapshot(t *testing.T) {
	request := &recommendDomain.GoodsRequest{
		Scene:   common.RecommendScene_GOODS_DETAIL,
		GoodsId: 2,
	}
	goodsList := []*app.GoodsInfo{
		nil,
		{Id: 0},
		{Id: 1, CategoryId: 11},
		{Id: 2, CategoryId: 12},
		{Id: 3, CategoryId: 13},
	}

	snapshot := BuildAnonymousSignalSnapshot(request, goodsList)

	if len(snapshot.GoodsList) != 2 {
		t.Fatalf("unexpected goods list length: %d", len(snapshot.GoodsList))
	}
	if len(snapshot.GoodsIds) != 2 || snapshot.GoodsIds[0] != 1 || snapshot.GoodsIds[1] != 3 {
		t.Fatalf("unexpected goods ids: %+v", snapshot.GoodsIds)
	}
	if len(snapshot.CategoryIds) != 0 {
		t.Fatalf("unexpected category ids: %+v", snapshot.CategoryIds)
	}
}

// TestBuildPersonalizedSignalSnapshot 验证登录态候选快照提取结果。
func TestBuildPersonalizedSignalSnapshot(t *testing.T) {
	goodsList := []*app.GoodsInfo{
		nil,
		{Id: 0},
		{Id: 1, CategoryId: 11},
		{Id: 2, CategoryId: 12},
	}

	snapshot := BuildPersonalizedSignalSnapshot(goodsList)

	if len(snapshot.GoodsList) != 2 {
		t.Fatalf("unexpected goods list length: %d", len(snapshot.GoodsList))
	}
	if len(snapshot.GoodsIds) != 2 || snapshot.GoodsIds[0] != 1 || snapshot.GoodsIds[1] != 2 {
		t.Fatalf("unexpected goods ids: %+v", snapshot.GoodsIds)
	}
	if len(snapshot.CategoryIds) != 2 || snapshot.CategoryIds[0] != 11 || snapshot.CategoryIds[1] != 12 {
		t.Fatalf("unexpected category ids: %+v", snapshot.CategoryIds)
	}
}

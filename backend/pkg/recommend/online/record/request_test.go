package record

import (
	"encoding/json"
	"testing"
	"time"

	"shop/api/gen/go/common"
	recommendcore "shop/pkg/recommend/core"
	recommendDomain "shop/pkg/recommend/domain"
)

// TestBuildRecommendRequestEntity 验证主表模型构建会收口来源上下文并回写请求字段。
func TestBuildRecommendRequestEntity(t *testing.T) {
	createdAt := time.Date(2026, 4, 16, 12, 30, 0, 0, time.UTC)
	entity, err := BuildRecommendRequestEntity(
		"req-1",
		&recommendDomain.Actor{
			ActorType: 1,
			ActorId:   9,
		},
		&recommendDomain.GoodsRequest{
			Scene:    common.RecommendScene_GOODS_DETAIL,
			PageNum:  2,
			PageSize: 10,
		},
		map[string]any{
			"goodsId":              int64(88),
			"actorType":            int32(1),
			"actorId":              int64(9),
			"cacheHitSources":      []string{"latest_cache"},
			"returnedScoreDetails": []recommendcore.ScoreDetail{{GoodsId: 88, FinalScore: 1.2}},
		},
		createdAt,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entity.RequestID != "req-1" || entity.ActorID != 9 || entity.Scene != int32(common.RecommendScene_GOODS_DETAIL) {
		t.Fatalf("unexpected request entity: %+v", entity)
	}
	if entity.PageNum != 2 || entity.PageSize != 10 || !entity.CreatedAt.Equal(createdAt) {
		t.Fatalf("unexpected paging fields in request entity: %+v", entity)
	}

	sourceContext := make(map[string]any)
	if err := json.Unmarshal([]byte(entity.SourceContext), &sourceContext); err != nil {
		t.Fatalf("unexpected source context json: %v", err)
	}
	if _, ok := sourceContext["actorType"]; ok {
		t.Fatalf("unexpected actorType in persisted source context: %+v", sourceContext)
	}
	if _, ok := sourceContext["returnedScoreDetails"]; ok {
		t.Fatalf("unexpected returnedScoreDetails in persisted source context: %+v", sourceContext)
	}
	if sourceContext["goodsId"] != float64(88) {
		t.Fatalf("unexpected goodsId in persisted source context: %+v", sourceContext)
	}
	onlineDebugContext, ok := sourceContext["onlineDebugContext"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected online debug context: %+v", sourceContext)
	}
	cacheHitSources, ok := onlineDebugContext["cacheHitSources"].([]any)
	if !ok || len(cacheHitSources) != 1 || cacheHitSources[0] != "latest_cache" {
		t.Fatalf("unexpected cache hit sources in online debug context: %+v", onlineDebugContext)
	}
}

// TestBuildRecommendRequestEntityNilInput 验证空主体和空请求也能构建最小主表模型。
func TestBuildRecommendRequestEntityNilInput(t *testing.T) {
	entity, err := BuildRecommendRequestEntity("req-2", nil, nil, nil, time.Time{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entity.RequestID != "req-2" || entity.ActorType != 0 || entity.PageNum != 0 {
		t.Fatalf("unexpected minimal request entity: %+v", entity)
	}
	if entity.SourceContext != "{}" {
		t.Fatalf("unexpected minimal source context: %s", entity.SourceContext)
	}
}

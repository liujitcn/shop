package biz

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	app "shop/api/gen/go/app"
	recommendLlmRerank "shop/pkg/llm/rerank"
	recommendCache "shop/pkg/recommend/cache"
	recommendDomain "shop/pkg/recommend/domain"
)

// TestLoadCachedLlmRerankScoresLiveFallback 验证缓存 miss 时会走在线重排并回写缓存。
func TestLoadCachedLlmRerankScoresLiveFallback(t *testing.T) {
	store, cleanup, err := recommendCache.NewStore(nil)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer cleanup()

	callCount := 0
	mockClient := httpDoerFunc(func(r *http.Request) (*http.Response, error) {
		callCount++
		responseBody, err := json.Marshal(map[string]any{
			"id":      "chatcmpl-test",
			"object":  "chat.completion",
			"created": 1710000000,
			"model":   "gpt-4o-mini",
			"choices": []map[string]any{
				{
					"index": 0,
					"message": map[string]any{
						"role": "assistant",
						"content": "```json\n" +
							`{"items":[{"goodsId":101,"score":0.9,"rank":1,"reason":"更匹配"},{"goodsId":102,"score":0.6,"rank":2,"reason":"可补充"}]}` +
							"\n```",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     12,
				"completion_tokens": 6,
				"total_tokens":      18,
			},
		})
		if err != nil {
			return nil, err
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			Body: io.NopCloser(strings.NewReader(string(responseBody))),
		}, nil
	})

	recommendCase := &RecommendRequestCase{
		recommendCacheStore: store,
		recommendLlmRerankService: recommendLlmRerank.NewService(recommendLlmRerank.Config{
			APIKey:       "test-key",
			BaseURL:      "http://mock.local/v1",
			DefaultModel: "gpt-4o-mini",
			HTTPTimeout:  time.Second,
			HTTPClient:   mockClient,
		}),
	}
	actor := &recommendDomain.Actor{
		ActorType: 1,
		ActorId:   18,
	}
	request := &recommendDomain.GoodsRequest{
		Scene:    3,
		GoodsId:  55,
		PageNum:  1,
		PageSize: 10,
	}
	strategy := &recommendDomain.LlmRerankStrategy{
		Enabled:         true,
		Model:           "gpt-4o-mini",
		TopN:            2,
		Weight:          1,
		CacheTTLSeconds: 60,
		PromptTemplate:  "候选={{ candidatesJson }}",
	}
	sceneStrategyContext := &recommendDomain.SceneStrategyContext{
		Scene:              3,
		EffectiveVersion:   "v2",
		VersionPublishedAt: time.Date(2026, 4, 17, 10, 0, 0, 0, time.Local),
		Config: &recommendDomain.StrategyVersionConfig{
			LlmRerank: strategy,
		},
	}
	candidateGoodsIds := []int64{101, 102}
	candidateGoodsList := []*app.GoodsInfo{
		{Id: 101, Name: "goods-a", SaleNum: 1, Price: 1000},
		{Id: 102, Name: "goods-b", SaleNum: 2, Price: 2000},
	}

	firstResult, err := recommendCase.loadCachedLlmRerankScores(
		context.Background(),
		actor,
		request,
		sceneStrategyContext,
		candidateGoodsIds,
		candidateGoodsList,
	)
	if err != nil {
		t.Fatalf("first loadCachedLlmRerankScores: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("unexpected live call count after first request: %d", callCount)
	}
	if len(firstResult.Scores) != 2 || !firstResult.ReadContext["liveRecovered"].(bool) {
		t.Fatalf("unexpected first result: %+v", firstResult)
	}
	if firstResult.ReadContext["writeBack"] != true {
		t.Fatalf("expected write back context, got %+v", firstResult.ReadContext)
	}

	secondResult, err := recommendCase.loadCachedLlmRerankScores(
		context.Background(),
		actor,
		request,
		sceneStrategyContext,
		candidateGoodsIds,
		candidateGoodsList,
	)
	if err != nil {
		t.Fatalf("second loadCachedLlmRerankScores: %v", err)
	}
	// 第二次命中缓存时，不应该再次访问在线模型。
	if callCount != 1 {
		t.Fatalf("unexpected live call count after cache hit: %d", callCount)
	}
	if secondResult.ReadContext["hit"] != true {
		t.Fatalf("expected cache hit on second request, got %+v", secondResult.ReadContext)
	}
	if len(secondResult.Scores) != 2 || secondResult.Scores[101] <= secondResult.Scores[102] {
		t.Fatalf("unexpected cached scores: %+v", secondResult.Scores)
	}
}

// httpDoerFunc 把函数适配成 OpenAI 客户端使用的 HTTPDoer。
type httpDoerFunc func(req *http.Request) (*http.Response, error)

// Do 执行一次内存 HTTP 请求。
func (f httpDoerFunc) Do(req *http.Request) (*http.Response, error) {
	return f(req)
}

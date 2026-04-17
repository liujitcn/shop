package rerank

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	app "shop/api/gen/go/app"
	recommendDomain "shop/pkg/recommend/domain"
)

// TestServiceRerank 验证在线重排会串起模板、表达式、脚本和 OpenAI 兼容调用。
func TestServiceRerank(t *testing.T) {
	var requestBody map[string]any
	mockClient := httpDoerFunc(func(r *http.Request) (*http.Response, error) {
		// 仅允许测试命中 ChatCompletion 路径，避免误调其它接口。
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
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
							`{"items":[{"goodsId":101,"score":0.5,"rank":1,"reason":"更匹配"},{"goodsId":102,"score":0.3,"rank":2,"reason":"可补充"}]}` +
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
			t.Fatalf("marshal response body: %v", err)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			Body: io.NopCloser(strings.NewReader(string(responseBody))),
		}, nil
	})

	service := NewService(Config{
		APIKey:       "test-key",
		BaseURL:      "http://mock.local/v1",
		DefaultModel: "gpt-4o-mini",
		HTTPTimeout:  time.Second,
		HTTPClient:   mockClient,
	})
	temperature := 0.0
	result, err := service.Rerank(context.Background(), Request{
		Strategy: &recommendDomain.LlmRerankStrategy{
			Enabled:             true,
			PromptTemplate:      "候选={{ candidatesJson }}",
			CandidateFilterExpr: "candidate.saleNum <= 2",
			ScoreExpr:           "score + candidate.saleNum",
			ScoreScript:         "return input.score + input.rank;",
			Temperature:         &temperature,
			MaxCompletionTokens: 128,
		},
		Actor: &recommendDomain.Actor{
			ActorType: 1,
			ActorId:   99,
		},
		GoodsRequest: &recommendDomain.GoodsRequest{
			Scene:    3,
			GoodsId:  55,
			PageNum:  1,
			PageSize: 10,
		},
		CandidateGoodsIds: []int64{101, 102, 103},
		CandidateGoodsList: []*app.GoodsInfo{
			{Id: 101, Name: "goods-a", SaleNum: 1, Price: 1000},
			{Id: 102, Name: "goods-b", SaleNum: 2, Price: 2000},
			{Id: 103, Name: "goods-c", SaleNum: 3, Price: 3000},
		},
	})
	if err != nil {
		t.Fatalf("rerank: %v", err)
	}

	if result.Scores[101] != 2.5 {
		t.Fatalf("unexpected goods 101 score: %+v", result.Scores)
	}
	if result.Scores[102] != 4.3 {
		t.Fatalf("unexpected goods 102 score: %+v", result.Scores)
	}
	if len(result.Documents) != 2 || result.Documents[0].Id != "102" {
		t.Fatalf("unexpected rerank documents: %+v", result.Documents)
	}
	if result.DebugContext["filteredCandidateCount"] != 2 {
		t.Fatalf("unexpected debug context: %+v", result.DebugContext)
	}
	promptTokens, ok := result.DebugContext["promptTokenEstimate"].(int)
	if !ok || promptTokens <= 0 {
		t.Fatalf("unexpected prompt token estimate: %+v", result.DebugContext["promptTokenEstimate"])
	}

	messageList, ok := requestBody["messages"].([]any)
	if !ok || len(messageList) != 2 {
		t.Fatalf("unexpected request messages: %+v", requestBody)
	}
	userMessage, ok := messageList[1].(map[string]any)
	if !ok {
		t.Fatalf("unexpected user message: %+v", messageList[1])
	}
	content, _ := userMessage["content"].(string)
	// 过滤后的提示词必须只保留前两个候选，不再携带被过滤商品。
	if !strings.Contains(content, "goods-a") || !strings.Contains(content, "goods-b") || strings.Contains(content, "goods-c") {
		t.Fatalf("unexpected prompt content: %s", content)
	}
}

// TestExtractJSONContent 验证代码块和前缀文本中的 JSON 会被正确抽取。
func TestExtractJSONContent(t *testing.T) {
	content, err := extractJSONContent("输出如下：```json\n{\"items\":[{\"goodsId\":1,\"score\":1}]}\n```")
	if err != nil {
		t.Fatalf("extract json content: %v", err)
	}
	if !strings.Contains(content, "\"goodsId\":1") {
		t.Fatalf("unexpected extracted content: %s", content)
	}
}

// httpDoerFunc 把函数适配为 OpenAI 客户端需要的 HTTPDoer。
type httpDoerFunc func(req *http.Request) (*http.Response, error)

// Do 执行一次内存 HTTP 请求。
func (f httpDoerFunc) Do(req *http.Request) (*http.Response, error) {
	return f(req)
}

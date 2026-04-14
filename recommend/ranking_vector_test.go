package recommend

import (
	"context"
	"fmt"
	"recommend/contract"
	"testing"
	"time"
)

// TestBuildVectorAndRecommendFromVectorPool 校验离线向量池会被在线推荐优先消费。
func TestBuildVectorAndRecommendFromVectorPool(t *testing.T) {
	ctx := context.Background()
	cacheSource := newRuntimeTestCacheSource(t)
	goodsSource := &runtimeTestGoodsSource{
		goodsById: map[int64]*contract.Goods{
			1: {Id: 1, CategoryId: 11, OnSale: true, InStock: true},
			2: {Id: 2, CategoryId: 12, OnSale: true, InStock: true},
			3: {Id: 3, CategoryId: 13, OnSale: true, InStock: true},
		},
		latestGoods: []*contract.Goods{
			{Id: 3, CategoryId: 13, OnSale: true, InStock: true},
		},
	}

	builder := newTestRecommend(
		t,
		WithDependencies(Dependencies{
			Goods: goodsSource,
			Vector: &vectorTestSource{actorGoods: map[string][]*contract.WeightedGoods{
				"home:1:101": {
					{GoodsId: 2, Score: 9},
				},
			}},
			Cache: cacheSource,
		}),
	)

	_, err := builder.BuildVector(ctx, BuildVectorRequest{
		Scenes:  []Scene{SceneHome},
		UserIds: []int64{101},
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("构建向量召回池失败: %v", err)
	}

	recommender := newTestRecommend(
		t,
		WithDependencies(Dependencies{
			Goods:     goodsSource,
			Recommend: &runtimeTestRecommendSource{},
			Cache:     cacheSource,
		}),
	)

	result, err := recommender.Recommend(ctx, RecommendRequest{
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
			RequestId: "vector-request",
		},
	})
	if err != nil {
		t.Fatalf("消费向量离线池推荐失败: %v", err)
	}
	if len(result.Items) == 0 || result.Items[0].GoodsId != 2 {
		t.Fatalf("向量召回未成为主结果: %+v", result.Items)
	}
}

// TestTrainRankingAndRecommendByFm 校验训练后的 FM 排序会改变在线结果顺序。
func TestTrainRankingAndRecommendByFm(t *testing.T) {
	ctx := context.Background()
	cacheSource := newRuntimeTestCacheSource(t)
	goodsSource := &runtimeTestGoodsSource{
		goodsById: map[int64]*contract.Goods{
			1: {Id: 1, CategoryId: 11, OnSale: true, InStock: true},
			2: {Id: 2, CategoryId: 12, OnSale: true, InStock: true},
		},
		latestGoods: []*contract.Goods{
			{Id: 1, CategoryId: 11, OnSale: true, InStock: true},
		},
	}
	recommendSource := &offlineTestRecommendSource{
		sceneHotGoods: map[string][]*contract.WeightedGoods{
			"home": {
				{GoodsId: 1, Score: 20},
				{GoodsId: 2, Score: 2},
			},
		},
		userGoodsPref: map[int64][]*contract.WeightedGoods{
			101: {
				{GoodsId: 2, Score: 4},
			},
		},
		requestFacts: map[string][]*contract.RequestFact{
			"home": {
				{RequestId: "trace-1", GoodsIds: []int64{1, 2}},
				{RequestId: "trace-2", GoodsIds: []int64{1, 2}},
			},
		},
		actionFacts: map[string][]*contract.ActionFact{
			"home": {
				{RequestId: "trace-1", GoodsId: 2, EventType: string(BehaviorOrderPay)},
				{RequestId: "trace-2", GoodsId: 2, EventType: string(BehaviorOrderPay)},
			},
		},
	}

	ruleConfig := DefaultConfig()
	ruleConfig.Materialize.DefaultScenes = []Scene{SceneHome}
	ruleConfig.Training.MinSampleCount = 1
	ruleConfig.Training.Epochs = 60
	ruleConfig.Training.LearningRate = 0.2

	ruleRecommender := newTestRecommend(
		t,
		WithConfig(ruleConfig),
		WithDependencies(Dependencies{
			Goods:     goodsSource,
			Recommend: recommendSource,
			Cache:     cacheSource,
		}),
	)

	firstResult, err := ruleRecommender.Recommend(ctx, RecommendRequest{
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
			RequestId: "trace-1",
		},
	})
	if err != nil {
		t.Fatalf("训练前首次推荐失败: %v", err)
	}
	if len(firstResult.Items) < 2 || firstResult.Items[0].GoodsId != 1 {
		t.Fatalf("训练前规则排序结果不符合预期: %+v", firstResult.Items)
	}

	_, err = ruleRecommender.Recommend(ctx, RecommendRequest{
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
			RequestId: "trace-2",
		},
	})
	if err != nil {
		t.Fatalf("训练前第二次推荐失败: %v", err)
	}

	trainResult, err := ruleRecommender.TrainRanking(ctx, TrainRankingRequest{
		Scenes:   []Scene{SceneHome},
		StatDate: time.Now(),
	})
	if err != nil {
		t.Fatalf("训练学习排序模型失败: %v", err)
	}
	if trainResult.KeyCount != 1 {
		t.Fatalf("训练模型未写入缓存: %+v", trainResult)
	}

	fmConfig := ruleConfig
	fmConfig.Ranking.Mode = RankingModeFm
	fmRecommender := newTestRecommend(
		t,
		WithConfig(fmConfig),
		WithDependencies(Dependencies{
			Goods:     goodsSource,
			Recommend: recommendSource,
			Cache:     cacheSource,
		}),
	)

	fmResult, err := fmRecommender.Recommend(ctx, RecommendRequest{
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
			RequestId: "fm-request",
		},
		Explain: true,
	})
	if err != nil {
		t.Fatalf("FM 排序推荐失败: %v", err)
	}
	if len(fmResult.Items) == 0 || fmResult.Items[0].GoodsId != 2 {
		t.Fatalf("FM 排序未提升正反馈商品: %+v", fmResult.Items)
	}

	explainResult, err := fmRecommender.Explain(ctx, ExplainRequest{
		RequestId: "fm-request",
	})
	if err != nil {
		t.Fatalf("FM explain 查询失败: %v", err)
	}
	if len(explainResult.ScoreDetails) == 0 || explainResult.ScoreDetails[0].FmScore == 0 {
		t.Fatalf("FM explain 未写入模型分值: %+v", explainResult.ScoreDetails)
	}
}

// TestRecommendByLlmReranker 校验 LLM 重排会在规则分基础上执行二阶段重排。
func TestRecommendByLlmReranker(t *testing.T) {
	ctx := context.Background()
	cacheSource := newRuntimeTestCacheSource(t)
	recommender := newTestRecommend(
		t,
		WithConfig(func() Config {
			config := DefaultConfig()
			config.Ranking.Mode = RankingModeLlm
			config.Ranking.LlmCandidateLimit = 2
			config.Ranking.LlmBlendWeight = 0.95
			return config
		}()),
		WithDependencies(Dependencies{
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
				sceneHotGoods: []*contract.WeightedGoods{
					{GoodsId: 1, Score: 8},
					{GoodsId: 2, Score: 2},
				},
			},
			Reranker: &llmTestReranker{
				scores: map[int64]float64{
					1: 1,
					2: 10,
				},
			},
			Cache: cacheSource,
		}),
	)

	result, err := recommender.Recommend(ctx, RecommendRequest{
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
			RequestId: "llm-request",
		},
		Explain: true,
	})
	if err != nil {
		t.Fatalf("LLM 重排推荐失败: %v", err)
	}
	if len(result.Items) == 0 || result.Items[0].GoodsId != 2 {
		t.Fatalf("LLM 重排未调整结果顺序: %+v", result.Items)
	}

	explainResult, err := recommender.Explain(ctx, ExplainRequest{
		RequestId: "llm-request",
	})
	if err != nil {
		t.Fatalf("LLM explain 查询失败: %v", err)
	}
	if len(explainResult.ScoreDetails) == 0 || explainResult.ScoreDetails[0].LlmScore == 0 {
		t.Fatalf("LLM explain 未写入重排分值: %+v", explainResult.ScoreDetails)
	}
}

type vectorTestSource struct {
	actorGoods map[string][]*contract.WeightedGoods
	goodsGoods map[string][]*contract.WeightedGoods
}

// ListVectorGoods 返回测试用向量召回结果。
func (s *vectorTestSource) ListVectorGoods(_ context.Context, request contract.VectorRecallRequest) ([]*contract.WeightedGoods, error) {
	if len(request.SourceGoodsIds) > 0 {
		key := fmt.Sprintf("%s:%d", request.Scene, request.SourceGoodsIds[0])
		return cloneWeightedGoods(s.goodsGoods[key]), nil
	}
	key := fmt.Sprintf("%s:%d:%d", request.Scene, request.ActorType, request.ActorId)
	return cloneWeightedGoods(s.actorGoods[key]), nil
}

type llmTestReranker struct {
	scores map[int64]float64
}

// Rerank 返回测试用 LLM 重排结果。
func (r *llmTestReranker) Rerank(_ context.Context, request contract.LlmRerankRequest) ([]*contract.LlmRerankResult, error) {
	result := make([]*contract.LlmRerankResult, 0, len(request.Candidates))
	for _, item := range request.Candidates {
		if item == nil {
			continue
		}
		result = append(result, &contract.LlmRerankResult{
			GoodsId: item.GoodsId,
			Score:   r.scores[item.GoodsId],
			Reason:  fmt.Sprintf("llm rerank goods=%d", item.GoodsId),
		})
	}
	return result, nil
}

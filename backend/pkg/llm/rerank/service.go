package rerank

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	app "shop/api/gen/go/app"
	recommendCache "shop/pkg/recommend/cache"
	recommendDomain "shop/pkg/recommend/domain"

	"github.com/expr-lang/expr"
	exprVm "github.com/expr-lang/expr/vm"
	gonja "github.com/nikolalohinski/gonja/v2"
	gonjaExec "github.com/nikolalohinski/gonja/v2/exec"
	openai "github.com/sashabaranov/go-openai"
	"github.com/tiktoken-go/tokenizer"
	"modernc.org/quickjs"
)

const (
	// EnvRecommendOpenaiAPIKey 表示推荐在线重排读取的 API Key 环境变量名。
	EnvRecommendOpenaiAPIKey = "RECOMMEND_OPENAI_API_KEY"
	// EnvRecommendOpenaiBaseURL 表示推荐在线重排读取的 BaseURL 环境变量名。
	EnvRecommendOpenaiBaseURL = "RECOMMEND_OPENAI_BASE_URL"
	// EnvRecommendOpenaiModel 表示推荐在线重排读取的默认模型环境变量名。
	EnvRecommendOpenaiModel = "RECOMMEND_OPENAI_MODEL"
)

const (
	defaultModel               = "gpt-4o-mini"
	defaultTimeout             = 8 * time.Second
	defaultTemperature         = 0
	defaultMaxCompletionTokens = int64(512)
	defaultSystemPrompt        = "你是商城推荐链路的二次重排器。你只能基于给定候选商品做排序，只返回 JSON，不要输出额外解释。"
	defaultPromptTemplate      = `
请根据当前请求上下文，对候选商品做二次重排。
只返回 JSON，不要输出 Markdown、代码块或额外说明。
输出格式：
{"items":[{"goodsId":123,"score":0.98,"reason":"一句话原因"}]}

当前请求：
- scene={{ request.scene }}
- goodsId={{ request.goodsId }}
- orderId={{ request.orderId }}
- actorType={{ actor.actorType }}
- actorId={{ actor.actorId }}

候选商品：
{% for candidate in candidates -%}
- goodsId={{ candidate.goodsId }}, categoryId={{ candidate.categoryId }}, name={{ candidate.name }}, desc={{ candidate.desc }}, price={{ candidate.price }}, saleNum={{ candidate.saleNum }}, updatedAt={{ candidate.updatedAt }}
{% endfor %}
`
)

// Config 表示在线 LLM 重排的基础运行配置。
type Config struct {
	APIKey       string          // 当前在线重排使用的 API Key。
	BaseURL      string          // 当前在线重排使用的 OpenAI 兼容 BaseURL。
	DefaultModel string          // 当前在线重排默认模型。
	HTTPTimeout  time.Duration   // 当前在线重排 HTTP 超时。
	HTTPClient   openai.HTTPDoer // 当前在线重排使用的 HTTP 客户端，测试场景可注入内存实现。
}

// Service 表示在线 LLM 重排执行器。
type Service struct {
	config Config // 当前执行器的基础运行配置。
}

// Request 表示一次在线 LLM 重排请求。
type Request struct {
	Strategy           *recommendDomain.LlmRerankStrategy // 当前场景版本的 LLM 重排配置。
	Actor              *recommendDomain.Actor             // 当前推荐主体。
	GoodsRequest       *recommendDomain.GoodsRequest      // 当前推荐请求快照。
	CandidateGoodsIds  []int64                            // 当前候选商品 ID 快照。
	CandidateGoodsList []*app.GoodsInfo                   // 当前候选商品详情快照。
}

// Result 表示一次在线 LLM 重排的输出结果。
type Result struct {
	Scores       map[int64]float64      // 当前重排生成的分数映射。
	Documents    []recommendCache.Score // 当前重排生成的缓存文档。
	DebugContext map[string]any         // 当前重排执行过程的调试上下文。
}

// NewService 创建在线 LLM 重排执行器。
func NewService(config Config) *Service {
	// 调用方未显式配置超时时，统一回退到默认超时。
	if config.HTTPTimeout <= 0 {
		config.HTTPTimeout = defaultTimeout
	}
	// 调用方未显式配置默认模型时，统一回退到内置默认模型。
	if strings.TrimSpace(config.DefaultModel) == "" {
		config.DefaultModel = defaultModel
	}
	return &Service{
		config: config,
	}
}

// NewServiceFromEnv 从环境变量创建在线 LLM 重排执行器。
func NewServiceFromEnv() *Service {
	return NewService(Config{
		APIKey:       resolveEnvValue(EnvRecommendOpenaiAPIKey, "OPENAI_API_KEY"),
		BaseURL:      resolveEnvValue(EnvRecommendOpenaiBaseURL, "OPENAI_BASE_URL"),
		DefaultModel: resolveEnvValue(EnvRecommendOpenaiModel, "OPENAI_MODEL"),
		HTTPTimeout:  defaultTimeout,
	})
}

// IsConfigured 判断当前执行器是否已具备在线调用所需配置。
func (s *Service) IsConfigured() bool {
	// 执行器为空或 API Key 为空时，不允许继续调用在线 LLM。
	return s != nil && strings.TrimSpace(s.config.APIKey) != ""
}

// Rerank 执行一次在线 LLM 二次重排。
func (s *Service) Rerank(ctx context.Context, req Request) (*Result, error) {
	result := &Result{
		Scores:       map[int64]float64{},
		Documents:    []recommendCache.Score{},
		DebugContext: map[string]any{},
	}
	// 执行器未配置 API Key 时，直接返回配置错误，避免发出无效请求。
	if !s.IsConfigured() {
		return nil, fmt.Errorf("llm rerank api key is empty")
	}
	// 当前策略为空时，不继续执行在线重排。
	if req.Strategy == nil {
		return nil, fmt.Errorf("llm rerank strategy is nil")
	}

	model := req.Strategy.ResolveModel(s.config.DefaultModel)
	// 当前模型名仍为空时，说明配置不足，直接拒绝调用。
	if strings.TrimSpace(model) == "" {
		return nil, fmt.Errorf("llm rerank model is empty")
	}
	timeout := req.Strategy.ResolveTimeout(s.config.HTTPTimeout)
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	topN := req.Strategy.ResolveTopN(int64(len(req.CandidateGoodsIds)))
	candidateList, missingCount := buildPromptCandidateList(req.CandidateGoodsIds, req.CandidateGoodsList, topN)
	result.DebugContext["candidateCount"] = len(candidateList)
	result.DebugContext["missingCandidateCount"] = missingCount
	// 当前没有可参与提示词的候选商品时，直接返回空结果。
	if len(candidateList) == 0 {
		result.DebugContext["skipped"] = true
		result.DebugContext["skipReason"] = "empty_candidates"
		return result, nil
	}

	filteredCandidateList, filterApplied, err := filterCandidates(req.Strategy.CandidateFilterExpr, candidateList, req.GoodsRequest, req.Actor)
	if err != nil {
		return nil, err
	}
	result.DebugContext["candidateFilterApplied"] = filterApplied
	result.DebugContext["filteredCandidateCount"] = len(filteredCandidateList)
	// 过滤后没有候选商品时，直接返回空结果，避免继续调用外部模型。
	if len(filteredCandidateList) == 0 {
		result.DebugContext["skipped"] = true
		result.DebugContext["skipReason"] = "all_candidates_filtered"
		return result, nil
	}

	systemPrompt := strings.TrimSpace(req.Strategy.SystemPrompt)
	// 当前未配置系统提示词时，统一回退到默认系统提示词。
	if systemPrompt == "" {
		systemPrompt = defaultSystemPrompt
	}
	prompt, err := renderPrompt(req.Strategy.PromptTemplate, filteredCandidateList, req.GoodsRequest, req.Actor)
	if err != nil {
		return nil, err
	}
	result.DebugContext["model"] = model
	result.DebugContext["baseURL"] = strings.TrimSpace(s.config.BaseURL)
	result.DebugContext["promptTokenEstimate"] = estimateTokenCount(model, systemPrompt, prompt)
	result.DebugContext["promptLength"] = len(prompt)

	chatResponse, responseContent, err := s.callChatCompletion(ctx, timeout, model, systemPrompt, prompt, req.Actor, req.Strategy)
	if err != nil {
		return nil, err
	}
	result.DebugContext["openaiUsage"] = map[string]any{
		"promptTokens":     chatResponse.Usage.PromptTokens,
		"completionTokens": chatResponse.Usage.CompletionTokens,
		"totalTokens":      chatResponse.Usage.TotalTokens,
	}

	parsedItemList, err := parseResponseItems(responseContent)
	if err != nil {
		return nil, err
	}
	result.DebugContext["parsedItemCount"] = len(parsedItemList)
	// 模型返回空结果时，直接返回空分数，避免继续写缓存。
	if len(parsedItemList) == 0 {
		result.DebugContext["skipped"] = true
		result.DebugContext["skipReason"] = "empty_model_output"
		return result, nil
	}

	candidateMap := make(map[int64]map[string]any, len(filteredCandidateList))
	for _, candidate := range filteredCandidateList {
		// 非法候选不参与最终分数回写。
		if candidate["goodsId"] == nil {
			continue
		}
		candidateMap[toInt64(candidate["goodsId"])] = candidate
	}

	scoreProgram, scoreExprApplied, err := compileScoreExpr(req.Strategy.ScoreExpr)
	if err != nil {
		return nil, err
	}
	result.DebugContext["scoreExprApplied"] = scoreExprApplied
	result.DebugContext["scoreScriptApplied"] = strings.TrimSpace(req.Strategy.ScoreScript) != ""

	now := time.Now()
	documentList := make([]recommendCache.Score, 0, len(parsedItemList))
	scoreMap := make(map[int64]float64, len(parsedItemList))
	for itemIndex, item := range parsedItemList {
		candidate, ok := candidateMap[item.GoodsId]
		// 模型返回了未知商品编号时，直接忽略，避免污染排序结果。
		if !ok || item.GoodsId <= 0 {
			continue
		}
		score, scoreErr := buildFinalScore(item, itemIndex, candidate, req.GoodsRequest, req.Actor, scoreProgram, req.Strategy.ScoreScript)
		if scoreErr != nil {
			return nil, scoreErr
		}
		// 最终分数非法时，直接跳过当前候选，避免把 NaN 写回排序链路。
		if math.IsNaN(score) || math.IsInf(score, 0) {
			continue
		}
		scoreMap[item.GoodsId] = score
		documentList = append(documentList, recommendCache.Score{
			Id:        strconv.FormatInt(item.GoodsId, 10),
			Score:     score,
			Timestamp: now,
		})
	}
	sort.SliceStable(documentList, func(i, j int) bool {
		return documentList[i].Score > documentList[j].Score
	})
	result.Scores = scoreMap
	result.Documents = documentList
	result.DebugContext["returnedCount"] = len(scoreMap)
	return result, nil
}

// callChatCompletion 发起一次 OpenAI 兼容的 ChatCompletion 调用。
func (s *Service) callChatCompletion(
	ctx context.Context,
	timeout time.Duration,
	model string,
	systemPrompt string,
	prompt string,
	actor *recommendDomain.Actor,
	strategy *recommendDomain.LlmRerankStrategy,
) (openai.ChatCompletionResponse, string, error) {
	clientConfig := openai.DefaultConfig(strings.TrimSpace(s.config.APIKey))
	// 调用方显式配置了 BaseURL 时，继续覆盖默认官方地址。
	if strings.TrimSpace(s.config.BaseURL) != "" {
		clientConfig.BaseURL = strings.TrimSpace(s.config.BaseURL)
	}
	// 调用方显式注入了 HTTP 客户端时，优先复用外部实现，便于测试或统一网关代理。
	if s.config.HTTPClient != nil {
		clientConfig.HTTPClient = s.config.HTTPClient
	} else {
		clientConfig.HTTPClient = &http.Client{Timeout: timeout}
	}
	client := openai.NewClientWithConfig(clientConfig)

	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	request := openai.ChatCompletionRequest{
		Model: model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: systemPrompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
		Temperature:         float32(strategy.ResolveTemperature(defaultTemperature)),
		MaxCompletionTokens: int(strategy.ResolveMaxCompletionTokens(defaultMaxCompletionTokens)),
	}
	// 当前存在主体信息时，再把主体编号写入 user 字段，便于上游网关做审计和限流。
	if actor != nil {
		request.User = fmt.Sprintf("%d:%d", actor.ActorType, actor.ActorId)
	}
	response, err := client.CreateChatCompletion(callCtx, request)
	if err != nil {
		return openai.ChatCompletionResponse{}, "", err
	}
	// 模型没有返回候选结果时，统一视为异常输出。
	if len(response.Choices) == 0 {
		return openai.ChatCompletionResponse{}, "", fmt.Errorf("llm rerank response choices is empty")
	}
	return response, strings.TrimSpace(response.Choices[0].Message.Content), nil
}

// buildPromptCandidateList 按候选商品 ID 顺序构建提示词候选快照。
func buildPromptCandidateList(candidateGoodsIds []int64, candidateGoodsList []*app.GoodsInfo, topN int64) ([]map[string]any, int) {
	if len(candidateGoodsIds) == 0 || len(candidateGoodsList) == 0 {
		return []map[string]any{}, 0
	}
	if topN <= 0 || topN > int64(len(candidateGoodsIds)) {
		topN = int64(len(candidateGoodsIds))
	}

	goodsMap := make(map[int64]*app.GoodsInfo, len(candidateGoodsList))
	for _, goods := range candidateGoodsList {
		// 非法商品快照不参与提示词构建。
		if goods == nil || goods.Id <= 0 {
			continue
		}
		goodsMap[goods.Id] = goods
	}

	result := make([]map[string]any, 0, topN)
	missingCount := 0
	for index, goodsId := range candidateGoodsIds[:topN] {
		goods, ok := goodsMap[goodsId]
		// 商品快照缺失时，记录缺口并跳过当前候选。
		if !ok {
			missingCount++
			continue
		}
		result = append(result, map[string]any{
			"goodsId":    goods.Id,
			"categoryId": goods.CategoryId,
			"name":       goods.Name,
			"desc":       goods.Desc,
			"price":      goods.Price,
			"priceYuan":  float64(goods.Price) / 100,
			"saleNum":    goods.SaleNum,
			"updatedAt":  goods.UpdatedAt,
			"position":   index + 1,
		})
	}
	return result, missingCount
}

// filterCandidates 按表达式过滤候选商品。
func filterCandidates(
	filterExpr string,
	candidateList []map[string]any,
	request *recommendDomain.GoodsRequest,
	actor *recommendDomain.Actor,
) ([]map[string]any, bool, error) {
	trimmedExpr := strings.TrimSpace(filterExpr)
	// 当前未配置过滤表达式时，直接返回原始候选集合。
	if trimmedExpr == "" {
		return append([]map[string]any{}, candidateList...), false, nil
	}
	program, err := expr.Compile(
		trimmedExpr,
		expr.Env(map[string]any{
			"candidate": map[string]any{},
			"request":   map[string]any{},
			"actor":     map[string]any{},
			"index":     0,
		}),
		expr.AllowUndefinedVariables(),
		expr.AsBool(),
	)
	if err != nil {
		return nil, false, fmt.Errorf("compile candidate filter expr: %w", err)
	}

	requestContext := buildRequestContext(request)
	actorContext := buildActorContext(actor)
	result := make([]map[string]any, 0, len(candidateList))
	for index, candidate := range candidateList {
		decision, runErr := expr.Run(program, map[string]any{
			"candidate": candidate,
			"request":   requestContext,
			"actor":     actorContext,
			"index":     index,
		})
		if runErr != nil {
			return nil, false, fmt.Errorf("run candidate filter expr: %w", runErr)
		}
		// 过滤表达式明确返回 false 时，当前候选不继续进入提示词。
		if allowed, ok := decision.(bool); ok && !allowed {
			continue
		}
		result = append(result, candidate)
	}
	return result, true, nil
}

// compileScoreExpr 编译分数表达式。
func compileScoreExpr(scoreExpr string) (*exprVm.Program, bool, error) {
	trimmedExpr := strings.TrimSpace(scoreExpr)
	// 当前未配置分数表达式时，直接返回空程序。
	if trimmedExpr == "" {
		return nil, false, nil
	}
	program, err := expr.Compile(
		trimmedExpr,
		expr.Env(map[string]any{
			"candidate": map[string]any{},
			"request":   map[string]any{},
			"actor":     map[string]any{},
			"score":     0.0,
			"rank":      0,
			"reason":    "",
			"llm":       map[string]any{},
		}),
		expr.AllowUndefinedVariables(),
		expr.AsFloat64(),
	)
	if err != nil {
		return nil, false, fmt.Errorf("compile score expr: %w", err)
	}
	return program, true, nil
}

// renderPrompt 渲染在线重排提示词。
func renderPrompt(
	promptTemplate string,
	candidateList []map[string]any,
	request *recommendDomain.GoodsRequest,
	actor *recommendDomain.Actor,
) (string, error) {
	trimmedTemplate := strings.TrimSpace(promptTemplate)
	// 当前未配置模板时，统一回退到内置模板。
	if trimmedTemplate == "" {
		trimmedTemplate = defaultPromptTemplate
	}

	template, err := gonja.FromString(trimmedTemplate)
	if err != nil {
		return "", fmt.Errorf("parse llm prompt template: %w", err)
	}
	contextMap := buildPromptContext(candidateList, request, actor)
	rendered, err := template.ExecuteToString(gonjaExec.NewContext(contextMap))
	if err != nil {
		return "", fmt.Errorf("render llm prompt template: %w", err)
	}
	return strings.TrimSpace(rendered), nil
}

// buildPromptContext 组装模板渲染上下文。
func buildPromptContext(candidateList []map[string]any, request *recommendDomain.GoodsRequest, actor *recommendDomain.Actor) map[string]any {
	requestContext := buildRequestContext(request)
	actorContext := buildActorContext(actor)
	candidatesJSON, _ := json.Marshal(candidateList)
	requestJSON, _ := json.Marshal(requestContext)
	actorJSON, _ := json.Marshal(actorContext)
	return map[string]any{
		"candidates":     candidateList,
		"request":        requestContext,
		"actor":          actorContext,
		"candidatesJson": string(candidatesJSON),
		"requestJson":    string(requestJSON),
		"actorJson":      string(actorJSON),
	}
}

// buildFinalScore 按“原始分 -> expr -> js”顺序计算最终写回分数。
func buildFinalScore(
	item parsedResponseItem,
	itemIndex int,
	candidate map[string]any,
	request *recommendDomain.GoodsRequest,
	actor *recommendDomain.Actor,
	scoreProgram *exprVm.Program,
	scoreScript string,
) (float64, error) {
	score := item.Score
	// 模型未显式返回分数时，回退到基于名次的递减分，确保仍然可以参与排序。
	if !item.HasScore {
		score = 1.0 / float64(item.ResolveRank(itemIndex))
	}

	requestContext := buildRequestContext(request)
	actorContext := buildActorContext(actor)
	if scoreProgram != nil {
		computed, err := expr.Run(scoreProgram, map[string]any{
			"candidate": candidate,
			"request":   requestContext,
			"actor":     actorContext,
			"score":     score,
			"rank":      item.ResolveRank(itemIndex),
			"reason":    item.Reason,
			"llm":       item.Raw,
		})
		if err != nil {
			return 0, fmt.Errorf("run score expr: %w", err)
		}
		score = toFloat64(computed)
	}
	// 当前未配置脚本时，直接返回表达式后的分数。
	if strings.TrimSpace(scoreScript) == "" {
		return score, nil
	}
	return executeScoreScript(scoreScript, map[string]any{
		"candidate": candidate,
		"request":   requestContext,
		"actor":     actorContext,
		"score":     score,
		"rank":      item.ResolveRank(itemIndex),
		"reason":    item.Reason,
		"llm":       item.Raw,
	})
}

// executeScoreScript 执行分数后处理脚本。
func executeScoreScript(scoreScript string, input map[string]any) (float64, error) {
	vm, err := quickjs.NewVM()
	if err != nil {
		return 0, fmt.Errorf("new quickjs vm: %w", err)
	}
	defer vm.Close()

	if setErr := vm.SetEvalTimeout(time.Second); setErr != nil {
		return 0, fmt.Errorf("set quickjs timeout: %w", setErr)
	}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return 0, fmt.Errorf("marshal score script input: %w", err)
	}
	script := fmt.Sprintf("(function(input){%s})(%s)", scoreScript, string(inputJSON))
	value, err := vm.Eval(script, quickjs.EvalGlobal)
	if err != nil {
		return 0, fmt.Errorf("run score script: %w", err)
	}
	return toFloat64(value), nil
}

// estimateTokenCount 估算本次提示词 token 数量。
func estimateTokenCount(model string, parts ...string) int {
	codec, err := tokenizer.ForModel(tokenizer.Model(strings.TrimSpace(strings.ToLower(model))))
	// 当前模型未被 tokenizer 直接识别时，统一回退到 O200kBase 编码估算。
	if err != nil {
		codec, err = tokenizer.Get(tokenizer.O200kBase)
		if err != nil {
			return 0
		}
	}
	total := 0
	for _, part := range parts {
		// 空片段不继续参与 token 估算。
		if strings.TrimSpace(part) == "" {
			continue
		}
		count, countErr := codec.Count(part)
		if countErr != nil {
			continue
		}
		total += count
	}
	return total
}

// parseResponseItems 解析模型输出的商品分数列表。
func parseResponseItems(content string) ([]parsedResponseItem, error) {
	jsonContent, err := extractJSONContent(content)
	if err != nil {
		return nil, err
	}

	itemList, parsed, parseErr := tryParseWrappedResponseItems(jsonContent)
	if parseErr != nil {
		return nil, parseErr
	}
	if parsed {
		return itemList, nil
	}
	return tryParseArrayResponseItems(jsonContent)
}

// tryParseWrappedResponseItems 解析带 items / scores / list 包裹的响应格式。
func tryParseWrappedResponseItems(content string) ([]parsedResponseItem, bool, error) {
	envelope := make(map[string]json.RawMessage)
	err := json.Unmarshal([]byte(content), &envelope)
	// 当前输出不是对象结构时，交给数组解析分支继续处理。
	if err != nil {
		return nil, false, nil
	}
	for _, key := range []string{"items", "scores", "list"} {
		payload, ok := envelope[key]
		// 当前包裹字段不存在时，继续尝试下一个兼容字段。
		if !ok {
			continue
		}
		itemList, convertErr := decodeResponseItemArray(payload)
		if convertErr != nil {
			return nil, true, convertErr
		}
		return itemList, true, nil
	}
	return nil, false, nil
}

// tryParseArrayResponseItems 解析数组形式的响应格式。
func tryParseArrayResponseItems(content string) ([]parsedResponseItem, error) {
	return decodeResponseItemArray([]byte(content))
}

// decodeResponseItemArray 解析单个响应数组。
func decodeResponseItemArray(payload []byte) ([]parsedResponseItem, error) {
	rawItemList := make([]map[string]any, 0)
	if err := json.Unmarshal(payload, &rawItemList); err != nil {
		return nil, fmt.Errorf("unmarshal llm rerank response items: %w", err)
	}
	result := make([]parsedResponseItem, 0, len(rawItemList))
	for _, rawItem := range rawItemList {
		item := parsedResponseItem{
			GoodsId:  pickInt64(rawItem, "goodsId", "goods_id", "id"),
			Score:    pickFloat64(rawItem, "score", "value"),
			HasScore: hasAnyKey(rawItem, "score", "value"),
			Rank:     int(pickInt64(rawItem, "rank", "position")),
			Reason:   pickString(rawItem, "reason", "explain"),
			Raw:      rawItem,
		}
		// 商品编号非法时，直接丢弃当前条目，避免后续排序阶段写入无效键。
		if item.GoodsId <= 0 {
			continue
		}
		result = append(result, item)
	}
	return result, nil
}

// extractJSONContent 从模型输出中抽取可反序列化的 JSON 内容。
func extractJSONContent(content string) (string, error) {
	trimmedContent := strings.TrimSpace(content)
	// 模型没有返回文本时，直接报错，避免后续解析空内容。
	if trimmedContent == "" {
		return "", fmt.Errorf("llm rerank response content is empty")
	}
	// 当前是 Markdown 代码块时，优先抽取代码块内部内容。
	if strings.Contains(trimmedContent, "```") {
		parts := strings.Split(trimmedContent, "```")
		for _, part := range parts {
			candidate := strings.TrimSpace(part)
			// 空代码块片段不继续尝试。
			if candidate == "" {
				continue
			}
			candidate = strings.TrimPrefix(candidate, "json")
			candidate = strings.TrimSpace(candidate)
			// 仅保留可能是 JSON 的片段继续尝试。
			if strings.HasPrefix(candidate, "{") || strings.HasPrefix(candidate, "[") {
				return candidate, nil
			}
		}
	}
	// 当前内容本身就是 JSON 时，直接返回。
	if strings.HasPrefix(trimmedContent, "{") || strings.HasPrefix(trimmedContent, "[") {
		return trimmedContent, nil
	}
	startIndex := strings.IndexAny(trimmedContent, "{[")
	// 完全找不到 JSON 起始符时，说明响应格式不符合约定。
	if startIndex < 0 {
		return "", fmt.Errorf("llm rerank response does not contain json")
	}
	return strings.TrimSpace(trimmedContent[startIndex:]), nil
}

// buildRequestContext 构建请求上下文映射。
func buildRequestContext(request *recommendDomain.GoodsRequest) map[string]any {
	if request == nil {
		return map[string]any{}
	}
	return map[string]any{
		"scene":    request.Scene,
		"orderId":  request.OrderId,
		"goodsId":  request.GoodsId,
		"pageNum":  request.PageNum,
		"pageSize": request.PageSize,
	}
}

// buildActorContext 构建主体上下文映射。
func buildActorContext(actor *recommendDomain.Actor) map[string]any {
	if actor == nil {
		return map[string]any{
			"actorType": 0,
			"actorId":   0,
		}
	}
	return map[string]any{
		"actorType": actor.ActorType,
		"actorId":   actor.ActorId,
	}
}

// resolveEnvValue 按优先顺序读取环境变量。
func resolveEnvValue(keys ...string) string {
	for _, key := range keys {
		// 当前键名为空时，继续尝试下一个环境变量。
		if strings.TrimSpace(key) == "" {
			continue
		}
		value := strings.TrimSpace(os.Getenv(key))
		// 命中了有效值时，直接返回当前环境变量。
		if value != "" {
			return value
		}
	}
	return ""
}

// hasAnyKey 判断映射中是否至少存在一个目标键。
func hasAnyKey(raw map[string]any, keys ...string) bool {
	for _, key := range keys {
		// 找到了目标键时，直接返回 true。
		if _, ok := raw[key]; ok {
			return true
		}
	}
	return false
}

// pickInt64 从动态映射中读取第一个有效整数值。
func pickInt64(raw map[string]any, keys ...string) int64 {
	for _, key := range keys {
		value, ok := raw[key]
		// 当前键不存在时，继续尝试下一个候选键。
		if !ok {
			continue
		}
		return toInt64(value)
	}
	return 0
}

// pickFloat64 从动态映射中读取第一个有效浮点值。
func pickFloat64(raw map[string]any, keys ...string) float64 {
	for _, key := range keys {
		value, ok := raw[key]
		// 当前键不存在时，继续尝试下一个候选键。
		if !ok {
			continue
		}
		return toFloat64(value)
	}
	return 0
}

// pickString 从动态映射中读取第一个有效字符串值。
func pickString(raw map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := raw[key]
		// 当前键不存在时，继续尝试下一个候选键。
		if !ok {
			continue
		}
		if text, ok := value.(string); ok {
			return strings.TrimSpace(text)
		}
	}
	return ""
}

// toFloat64 把动态值转换为 float64。
func toFloat64(value any) float64 {
	switch current := value.(type) {
	case float64:
		return current
	case float32:
		return float64(current)
	case int:
		return float64(current)
	case int32:
		return float64(current)
	case int64:
		return float64(current)
	case uint:
		return float64(current)
	case uint32:
		return float64(current)
	case uint64:
		return float64(current)
	case json.Number:
		parsed, err := current.Float64()
		if err == nil {
			return parsed
		}
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(current), 64)
		if err == nil {
			return parsed
		}
	}
	return 0
}

// toInt64 把动态值转换为 int64。
func toInt64(value any) int64 {
	switch current := value.(type) {
	case int:
		return int64(current)
	case int32:
		return int64(current)
	case int64:
		return current
	case uint:
		return int64(current)
	case uint32:
		return int64(current)
	case uint64:
		return int64(current)
	case float32:
		return int64(current)
	case float64:
		return int64(current)
	case json.Number:
		parsed, err := current.Int64()
		if err == nil {
			return parsed
		}
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(current), 10, 64)
		if err == nil {
			return parsed
		}
	}
	return 0
}

// parsedResponseItem 表示解析后的单个模型输出条目。
type parsedResponseItem struct {
	GoodsId  int64          // 当前条目对应的商品编号。
	Score    float64        // 当前条目的原始分数。
	HasScore bool           // 当前条目是否显式带了分数字段。
	Rank     int            // 当前条目的名次。
	Reason   string         // 当前条目的解释文本。
	Raw      map[string]any // 当前条目的原始 JSON 内容。
}

// ResolveRank 返回当前条目的有效名次。
func (i parsedResponseItem) ResolveRank(index int) int {
	// 当前显式带了有效名次时，优先使用模型返回值。
	if i.Rank > 0 {
		return i.Rank
	}
	return index + 1
}

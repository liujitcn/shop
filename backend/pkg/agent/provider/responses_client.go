package provider

import (
	"strings"

	agentopenai "shop/pkg/agent/openai"

	"github.com/go-kratos/blades"
	bootstrapConfigv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
)

// ResponsesClient 表示 AI 助手专用 Responses 模型客户端。
type ResponsesClient struct {
	blades.ModelProvider
}

// NewResponsesClient 创建 AI 助手专用 Responses 模型客户端。
func NewResponsesClient(bootstrapCfg *bootstrapConfigv1.Client_Llm) *ResponsesClient {
	client := &ResponsesClient{}
	if bootstrapCfg == nil {
		return client
	}
	baseURL := strings.TrimRight(bootstrapCfg.GetBaseUrl(), "/")
	apiKey := bootstrapCfg.GetApiKey()
	model := bootstrapCfg.GetModel()
	// 启动配置不完整时，保持客户端关闭状态。
	if baseURL == "" || apiKey == "" || model == "" {
		return client
	}
	client.ModelProvider = agentopenai.NewResponses(model, agentopenai.ResponsesConfig{
		BaseURL:         baseURL,
		APIKey:          apiKey,
		MaxOutputTokens: bootstrapCfg.GetMaxOutputTokens(),
		Temperature:     bootstrapCfg.GetTemperature(),
		TopP:            bootstrapCfg.GetTopP(),
		ExtraFields:     llmExtraFields(bootstrapCfg),
		ReasoningEffort: llmReasoningEffort(bootstrapCfg),
	})
	return client
}

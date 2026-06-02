package provider

import (
	"strings"

	agentopenai "shop/pkg/agent/openai"

	"github.com/cloudwego/eino/components/model"
	bootstrapConfigv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
)

// ResponsesClient 表示 AI 助手专用 Responses 模型客户端。
type ResponsesClient struct {
	model.BaseChatModel
	name string
}

// Name 返回当前 Responses 模型名称。
func (c *ResponsesClient) Name() string {
	if c == nil {
		return ""
	}
	return c.name
}

// NewResponsesClient 创建 AI 助手专用 Responses 模型客户端。
func NewResponsesClient(bootstrapCfg *bootstrapConfigv1.Client_Llm) *ResponsesClient {
	client := &ResponsesClient{}
	if bootstrapCfg == nil {
		return client
	}
	baseURL := strings.TrimRight(bootstrapCfg.GetBaseUrl(), "/")
	apiKey := bootstrapCfg.GetApiKey()
	modelName := bootstrapCfg.GetModel()
	// 启动配置不完整时，保持客户端关闭状态。
	if baseURL == "" || apiKey == "" || modelName == "" {
		return client
	}
	client.name = modelName
	client.BaseChatModel = agentopenai.NewResponses(modelName, agentopenai.ResponsesConfig{
		BaseURL:         baseURL,
		APIKey:          apiKey,
		MaxOutputTokens: bootstrapCfg.GetMaxOutputTokens(),
		Temperature:     bootstrapCfg.GetTemperature(),
		TopP:            bootstrapCfg.GetTopP(),
		ExtraFields:     llmExtraFields(bootstrapCfg),
		ReasoningEffort: llmReasoningEffort(bootstrapCfg),
		EnableWebSearch: true,
	})
	return client
}

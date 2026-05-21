package provider

import (
	"strings"

	"shop/pkg/agent/sub2api"

	"github.com/go-kratos/blades"
	bootstrapConfigv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
)

// ResponsesClient 表示 AI 助手专用 Responses 模型客户端。
type ResponsesClient struct {
	provider blades.ModelProvider
}

// NewResponsesClient 创建 AI 助手专用 Responses 模型客户端。
func NewResponsesClient(bootstrapCfg *bootstrapConfigv1.Client_Llm) *ResponsesClient {
	client := &ResponsesClient{}
	if bootstrapCfg == nil {
		return client
	}
	baseURL := strings.TrimRight(strings.TrimSpace(bootstrapCfg.GetBaseUrl()), "/")
	apiKey := strings.TrimSpace(bootstrapCfg.GetApiKey())
	model := strings.TrimSpace(bootstrapCfg.GetModel())
	// 启动配置不完整时，保持客户端关闭状态。
	if baseURL == "" || apiKey == "" || model == "" {
		return client
	}
	client.provider = sub2api.NewResponses(model, sub2api.ResponsesConfig{
		BaseURL:         baseURL,
		APIKey:          apiKey,
		MaxOutputTokens: bootstrapCfg.GetMaxOutputTokens(),
		Temperature:     bootstrapCfg.GetTemperature(),
		TopP:            bootstrapCfg.GetTopP(),
		ExtraFields:     llmExtraFields(bootstrapCfg),
		ReasoningEffort: strings.TrimSpace(bootstrapCfg.GetReasoningEffort()),
	})
	return client
}

// Enabled 判断 Responses 模型客户端是否可用。
func (c *ResponsesClient) Enabled() bool {
	return c != nil && c.provider != nil
}

// Provider 返回底层 Responses 模型提供者。
func (c *ResponsesClient) Provider() blades.ModelProvider {
	if c == nil {
		return nil
	}
	return c.provider
}

// Model 返回当前模型名称。
func (c *ResponsesClient) Model() string {
	if !c.Enabled() {
		return ""
	}
	return c.provider.Name()
}

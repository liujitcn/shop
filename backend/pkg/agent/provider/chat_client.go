package provider

import (
	"strings"

	"shop/pkg/agent/sub2api"

	"github.com/go-kratos/blades"
	bootstrapConfigv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
)

// ChatClient 表示智能体对话模型客户端。
type ChatClient struct {
	provider blades.ModelProvider
}

// NewChatClient 创建智能体对话模型客户端。
func NewChatClient(bootstrapCfg *bootstrapConfigv1.Client_Llm) *ChatClient {
	client := &ChatClient{}
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
	client.provider = sub2api.NewChat(model, sub2api.ChatConfig{
		BaseURL:          baseURL,
		APIKey:           apiKey,
		Seed:             bootstrapCfg.GetSeed(),
		MaxOutputTokens:  bootstrapCfg.GetMaxOutputTokens(),
		FrequencyPenalty: bootstrapCfg.GetFrequencyPenalty(),
		PresencePenalty:  bootstrapCfg.GetPresencePenalty(),
		Temperature:      bootstrapCfg.GetTemperature(),
		TopP:             bootstrapCfg.GetTopP(),
		StopSequences:    bootstrapCfg.GetStopSequences(),
		ExtraFields:      llmExtraFields(bootstrapCfg),
		ReasoningEffort:  strings.TrimSpace(bootstrapCfg.GetReasoningEffort()),
	})
	return client
}

// Enabled 判断对话模型客户端是否可用。
func (c *ChatClient) Enabled() bool {
	return c != nil && c.provider != nil
}

// Provider 返回底层模型提供者。
func (c *ChatClient) Provider() blades.ModelProvider {
	if c == nil {
		return nil
	}
	return c.provider
}

// Model 返回当前模型名称。
func (c *ChatClient) Model() string {
	if !c.Enabled() {
		return ""
	}
	return c.provider.Name()
}

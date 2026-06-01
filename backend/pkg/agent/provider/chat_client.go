package provider

import (
	"strings"

	agentopenai "shop/pkg/agent/openai"

	"github.com/cloudwego/eino/components/model"
	bootstrapConfigv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
)

// ChatClient 表示智能体对话模型客户端。
type ChatClient struct {
	model.BaseChatModel
	name string
}

// Name 返回当前聊天模型名称。
func (c *ChatClient) Name() string {
	if c == nil {
		return ""
	}
	return c.name
}

// NewChatClient 创建智能体对话模型客户端。
func NewChatClient(bootstrapCfg *bootstrapConfigv1.Client_Llm) *ChatClient {
	client := &ChatClient{}
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
	client.name = model
	client.BaseChatModel = agentopenai.NewResponses(model, agentopenai.ResponsesConfig{
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

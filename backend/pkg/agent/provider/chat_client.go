package provider

import (
	"strings"

	"github.com/go-kratos/blades"
	bladesopenai "github.com/go-kratos/blades/contrib/openai"
	bootstrapConfigv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
)

// ChatClient 表示智能体对话模型客户端。
type ChatClient struct {
	blades.ModelProvider
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
	client.ModelProvider = bladesopenai.NewModel(model, bladesopenai.Config{
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
		ReasoningEffort:  llmReasoningEffort(bootstrapCfg),
	})
	return client
}

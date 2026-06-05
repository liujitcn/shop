package provider

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/agenticopenai"
	"github.com/cloudwego/eino/components/model"
	aiEino "github.com/liujitcn/kratos-kit/ai/eino"
	bootstrapConfigv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
)

// ChatClient 表示智能体对话模型客户端。
type ChatClient struct {
	model.AgenticModel
	name string
}

// Name 返回当前聊天模型名称。
func (c *ChatClient) Name() string {
	if c == nil {
		return ""
	}
	return c.name
}

// NewChatClient 创建评论审核与摘要专用聊天模型客户端。
func NewChatClient(modelCfg *bootstrapConfigv1.AI_Model) *ChatClient {
	client := &ChatClient{}
	if !aiModelConfigured(modelCfg) {
		return client
	}
	agenticModel, err := aiEino.NewChatModel(
		context.Background(),
		modelCfg,
		aiEino.WithChatConfigMutator(func(modelConfig *agenticopenai.ChatConfig) {
			// 评论结构化输出不依赖采样温度，交给服务端使用模型默认值。
			modelConfig.Temperature = nil
		}),
	)
	if err != nil {
		panic(fmt.Errorf("创建评论智能体模型失败: %w", err))
	}
	client.name = modelCfg.GetModelName()
	client.AgenticModel = agenticModel
	return client
}

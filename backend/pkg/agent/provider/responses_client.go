package provider

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/model"
	aiEino "github.com/liujitcn/kratos-kit/ai/eino"
	bootstrapConfigv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
)

// ResponsesClient 表示 AI 助手专用 Responses 模型客户端。
type ResponsesClient struct {
	model.AgenticModel
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
func NewResponsesClient(modelCfg *bootstrapConfigv1.AI_Model) *ResponsesClient {
	client := &ResponsesClient{}
	if !aiModelConfigured(modelCfg) {
		return client
	}
	agenticModel, err := aiEino.NewResponsesModel(context.Background(), modelCfg)
	if err != nil {
		panic(fmt.Errorf("创建 AI 助手 Responses 模型失败: %w", err))
	}
	client.name = modelCfg.GetModelName()
	client.AgenticModel = agenticModel
	return client
}

package biz

import (
	"context"
	"encoding/json"

	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	queueData "github.com/liujitcn/kratos-kit/queue/data"
)

// BaseLogCase 日志业务实例
type BaseLogCase struct {
	*data.BaseLogRepo
}

// NewBaseLogCase 创建日志业务实例
func NewBaseLogCase(baseLogRepo *data.BaseLogRepo) *BaseLogCase {
	return &BaseLogCase{BaseLogRepo: baseLogRepo}
}

// SaveLog 保存日志队列消息
func (c *BaseLogCase) SaveLog(message queueData.Message) error {
	rawBody, err := json.Marshal(message.Values)
	if err != nil {
		return err
	}

	var payload map[string]*models.BaseLog
	err = json.Unmarshal(rawBody, &payload)
	if err != nil {
		return err
	}
	if baseLog, ok := payload["data"]; ok {
		return c.Create(context.TODO(), baseLog)
	}
	return nil
}

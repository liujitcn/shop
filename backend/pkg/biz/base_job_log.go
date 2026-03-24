package biz

import (
	"context"
	"encoding/json"

	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/go-kratos/kratos/v2/log"
	queueData "github.com/liujitcn/kratos-kit/queue/data"
)

// BaseJobLogCase 任务日志业务实例
type BaseJobLogCase struct {
	*data.BaseJobLogRepo
}

// NewBaseJobLogCase 创建任务日志业务实例
func NewBaseJobLogCase(baseJobLogRepo *data.BaseJobLogRepo) *BaseJobLogCase {
	return &BaseJobLogCase{
		BaseJobLogRepo: baseJobLogRepo,
	}
}

func (c *BaseJobLogCase) SaveJobLog(message queueData.Message) error {
	rb, err := json.Marshal(message.Values)
	if err != nil {
		log.Errorf("json Marshal error, %s", err.Error())
		return err
	}
	var m map[string]*models.BaseJobLog
	err = json.Unmarshal(rb, &m)
	if err != nil {
		log.Errorf("json Unmarshal error, %s", err.Error())
		return err
	}
	if v, ok := m["data"]; ok {
		err = c.Create(context.TODO(), v)
		if err != nil {
			return err
		}
	}
	return nil
}

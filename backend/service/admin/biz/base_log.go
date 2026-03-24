package biz

import (
	"context"
	"encoding/json"
	"time"

	"shop/api/gen/go/admin"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	_time "github.com/liujitcn/go-utils/time"
	"github.com/liujitcn/gorm-kit/repo"
	queueData "github.com/liujitcn/kratos-kit/queue/data"
)

// BaseLogCase 日志业务实例
type BaseLogCase struct {
	*biz.BaseCase
	*data.BaseLogRepo
	mapper *mapper.CopierMapper[admin.BaseLog, models.BaseLog]
}

// NewBaseLogCase 创建日志业务实例
func NewBaseLogCase(baseCase *biz.BaseCase, baseLogRepo *data.BaseLogRepo) *BaseLogCase {
	return &BaseLogCase{
		BaseCase:    baseCase,
		BaseLogRepo: baseLogRepo,
		mapper:      mapper.NewCopierMapper[admin.BaseLog, models.BaseLog](),
	}
}

// PageBaseLog 分页查询日志
func (c *BaseLogCase) PageBaseLog(ctx context.Context, req *admin.PageBaseLogRequest) (*admin.PageBaseLogResponse, error) {
	query := c.Query(ctx).BaseLog
	opts := make([]repo.QueryOption, 0, 3)
	if req.GetOperation() != "" {
		opts = append(opts, repo.Where(query.Operation.Like("%"+req.GetOperation()+"%")))
	}
	if req.StatusCode != nil {
		opts = append(opts, repo.Where(query.StatusCode.Eq(req.GetStatusCode())))
	}

	requestTime := req.GetRequestTime()
	if len(requestTime) == 2 {
		startTime := _time.StringTimeToTime(requestTime[0])
		endTime := _time.StringTimeToTime(requestTime[1])
		if startTime != nil {
			opts = append(opts, repo.Where(query.RequestTime.Gte(*startTime)))
		}
		if endTime != nil {
			endValue := endTime.AddDate(0, 0, 1)
			opts = append(opts, repo.Where(query.RequestTime.Lt(endValue)))
		}
	}

	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*admin.BaseLog, 0, len(list))
	for _, item := range list {
		resList = append(resList, c.toBaseLog(item))
	}
	return &admin.PageBaseLogResponse{List: resList, Total: int32(total)}, nil
}

// GetBaseLog 获取日志
func (c *BaseLogCase) GetBaseLog(ctx context.Context, id int64) (*admin.BaseLog, error) {
	baseLog, err := c.FindById(ctx, id)
	if err != nil {
		return nil, err
	}
	return c.toBaseLog(baseLog), nil
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

// toBaseLog 转换日志响应数据
func (c *BaseLogCase) toBaseLog(item *models.BaseLog) *admin.BaseLog {
	costTime := time.Duration(item.CostTime) * time.Millisecond
	baseLog := c.mapper.ToDTO(item)
	baseLog.RequestTime = _time.TimeToTimeString(item.RequestTime)
	baseLog.CostTime = costTime.String()
	return baseLog
}

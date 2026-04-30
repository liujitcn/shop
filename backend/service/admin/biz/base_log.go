package biz

import (
	"context"
	_const "shop/pkg/const"
	"time"

	adminv1 "shop/api/gen/go/admin/v1"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"shop/pkg/queue"

	"github.com/liujitcn/go-utils/mapper"
	_time "github.com/liujitcn/go-utils/time"
	"github.com/liujitcn/gorm-kit/repository"
	queueData "github.com/liujitcn/kratos-kit/queue/data"
)

// BaseLogCase 日志业务实例
type BaseLogCase struct {
	*biz.BaseCase
	*data.BaseLogRepository
	mapper *mapper.CopierMapper[adminv1.BaseLog, models.BaseLog]
}

// NewBaseLogCase 创建日志业务实例
func NewBaseLogCase(baseCase *biz.BaseCase, baseLogRepo *data.BaseLogRepository) *BaseLogCase {
	c := &BaseLogCase{
		BaseCase:          baseCase,
		BaseLogRepository: baseLogRepo,
		mapper:            mapper.NewCopierMapper[adminv1.BaseLog, models.BaseLog](),
	}

	// 注册日志队列
	c.RegisterQueueConsumer(_const.LOG, c.saveLog)
	return c
}

// PageBaseLogs 分页查询日志
func (c *BaseLogCase) PageBaseLogs(ctx context.Context, req *adminv1.PageBaseLogsRequest) (*adminv1.PageBaseLogsResponse, error) {
	query := c.Query(ctx).BaseLog
	opts := make([]repository.QueryOption, 0, 5)
	opts = append(opts, repository.Order(query.RequestTime.Desc()))
	// 传入操作名时，按操作名模糊匹配日志。
	if req.GetOperation() != "" {
		opts = append(opts, repository.Where(query.Operation.Like("%"+req.GetOperation()+"%")))
	}
	if req.StatusCode != nil {
		opts = append(opts, repository.Where(query.StatusCode.Eq(req.GetStatusCode())))
	}

	requestTime := req.GetRequestTime()
	// 仅在传入完整时间区间时，按请求时间范围过滤日志。
	if len(requestTime) == 2 {
		startTime := _time.StringTimeToTime(requestTime[0])
		endTime := _time.StringTimeToTime(requestTime[1])
		// 开始时间解析成功时，补充请求时间下界。
		if startTime != nil {
			opts = append(opts, repository.Where(query.RequestTime.Gte(*startTime)))
		}
		// 结束时间解析成功时，补充请求时间上界。
		if endTime != nil {
			endValue := endTime.AddDate(0, 0, 1)
			opts = append(opts, repository.Where(query.RequestTime.Lt(endValue)))
		}
	}

	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*adminv1.BaseLog, 0, len(list))
	for _, item := range list {
		resList = append(resList, c.toBaseLog(item))
	}
	return &adminv1.PageBaseLogsResponse{BaseLogs: resList, Total: int32(total)}, nil
}

// GetBaseLog 获取日志
func (c *BaseLogCase) GetBaseLog(ctx context.Context, id int64) (*adminv1.BaseLog, error) {
	baseLog, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return c.toBaseLog(baseLog), nil
}

// saveLog 保存日志队列消息
func (c *BaseLogCase) saveLog(message queueData.Message) error {
	baseLog, err := queue.DecodeQueueData[models.BaseLog](message)
	if err != nil {
		return err
	}
	// 队列消息里没有有效日志实体时，直接忽略当前消息。
	if baseLog == nil {
		return nil
	}
	return c.Create(context.TODO(), baseLog)
}

// toBaseLog 转换日志响应数据
func (c *BaseLogCase) toBaseLog(item *models.BaseLog) *adminv1.BaseLog {
	costTime := time.Duration(item.CostTime) * time.Millisecond
	baseLog := c.mapper.ToDTO(item)
	baseLog.RequestTime = _time.TimeToTimeString(item.RequestTime)
	baseLog.CostTime = costTime.String()
	return baseLog
}

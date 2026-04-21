package biz

import (
	"context"
	_const "shop/pkg/const"
	"strconv"

	"shop/api/gen/go/admin"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	pkgQueue "shop/pkg/queue"

	"github.com/liujitcn/go-utils/mapper"
	_time "github.com/liujitcn/go-utils/time"
	"github.com/liujitcn/gorm-kit/repo"
	queueData "github.com/liujitcn/kratos-kit/queue/data"
)

// BaseJobLogCase 任务日志业务实例
type BaseJobLogCase struct {
	*biz.BaseCase
	*data.BaseJobLogRepo
	mapper *mapper.CopierMapper[admin.BaseJobLog, models.BaseJobLog]
}

// NewBaseJobLogCase 创建任务日志业务实例
func NewBaseJobLogCase(baseCase *biz.BaseCase, baseJobLogRepo *data.BaseJobLogRepo) *BaseJobLogCase {
	c := &BaseJobLogCase{
		BaseCase:       baseCase,
		BaseJobLogRepo: baseJobLogRepo,
		mapper:         mapper.NewCopierMapper[admin.BaseJobLog, models.BaseJobLog](),
	}

	// 注册定时任务日志队列
	c.RegisterQueueConsumer(_const.JobLog, c.saveJobLog)
	return c
}

// PageBaseJobLog 分页查询任务日志
func (c *BaseJobLogCase) PageBaseJobLog(ctx context.Context, req *admin.PageBaseJobLogRequest) (*admin.PageBaseJobLogResponse, error) {
	query := c.Query(ctx).BaseJobLog
	opts := make([]repo.QueryOption, 0, 5)
	opts = append(opts, repo.Order(query.ExecuteTime.Desc()))
	// 传入任务编号时，仅查询对应任务的执行日志。
	if req.GetJobId() > 0 {
		opts = append(opts, repo.Where(query.JobID.Eq(req.GetJobId())))
	}
	if req.Status != nil {
		opts = append(opts, repo.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	// 仅在传入完整时间区间时，按执行时间范围过滤任务日志。
	if len(req.GetExecuteTime()) == 2 {
		startTime := _time.StringTimeToTime(req.GetExecuteTime()[0])
		endTime := _time.StringTimeToTime(req.GetExecuteTime()[1])
		// 开始时间解析成功时，补充执行时间下界。
		if startTime != nil {
			opts = append(opts, repo.Where(query.ExecuteTime.Gte(*startTime)))
		}
		// 结束时间解析成功时，补充执行时间上界。
		if endTime != nil {
			opts = append(opts, repo.Where(query.ExecuteTime.Lte(*endTime)))
		}
	}

	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*admin.BaseJobLog, 0, len(list))
	for _, item := range list {
		resList = append(resList, c.toBaseJobLog(item))
	}
	return &admin.PageBaseJobLogResponse{List: resList, Total: int32(total)}, nil
}

// GetBaseJobLog 获取任务日志
func (c *BaseJobLogCase) GetBaseJobLog(ctx context.Context, id int64) (*admin.BaseJobLog, error) {
	baseJobLog, err := c.FindById(ctx, id)
	if err != nil {
		return nil, err
	}
	return c.toBaseJobLog(baseJobLog), nil
}

// saveJobLog 保存任务日志队列消息。
func (c *BaseJobLogCase) saveJobLog(message queueData.Message) error {
	baseJobLog, err := pkgQueue.DecodeQueueData[models.BaseJobLog](message)
	if err != nil {
		return err
	}
	// 队列消息里没有有效任务日志实体时，直接忽略当前消息。
	if baseJobLog == nil {
		return nil
	}
	return c.Create(context.TODO(), baseJobLog)
}

// toBaseJobLog 转换任务日志响应
func (c *BaseJobLogCase) toBaseJobLog(item *models.BaseJobLog) *admin.BaseJobLog {
	baseJobLog := c.mapper.ToDTO(item)
	baseJobLog.ProcessTime = strconv.FormatInt(int64(item.ProcessTime), 10)
	return baseJobLog
}

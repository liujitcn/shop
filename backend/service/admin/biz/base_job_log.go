package biz

import (
	"context"
	"strconv"

	"shop/api/gen/go/admin"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	_time "github.com/liujitcn/go-utils/time"
	"github.com/liujitcn/gorm-kit/repo"
)

// BaseJobLogCase 任务日志业务实例
type BaseJobLogCase struct {
	*biz.BaseCase
	*data.BaseJobLogRepo
	mapper *mapper.CopierMapper[admin.BaseJobLog, models.BaseJobLog]
}

// NewBaseJobLogCase 创建任务日志业务实例
func NewBaseJobLogCase(baseCase *biz.BaseCase, baseJobLogRepo *data.BaseJobLogRepo) *BaseJobLogCase {
	return &BaseJobLogCase{
		BaseCase:       baseCase,
		BaseJobLogRepo: baseJobLogRepo,
		mapper:         mapper.NewCopierMapper[admin.BaseJobLog, models.BaseJobLog](),
	}
}

// PageBaseJobLog 分页查询任务日志
func (c *BaseJobLogCase) PageBaseJobLog(ctx context.Context, req *admin.PageBaseJobLogRequest) (*admin.PageBaseJobLogResponse, error) {
	query := c.Query(ctx).BaseJobLog
	opts := make([]repo.QueryOption, 0, 3)
	if req.GetJobId() > 0 {
		opts = append(opts, repo.Where(query.JobID.Eq(req.GetJobId())))
	}
	if req.Status != nil {
		opts = append(opts, repo.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	if len(req.GetExecuteTime()) == 2 {
		startTime := _time.StringTimeToTime(req.GetExecuteTime()[0])
		endTime := _time.StringTimeToTime(req.GetExecuteTime()[1])
		if startTime != nil {
			opts = append(opts, repo.Where(query.ExecuteTime.Gte(*startTime)))
		}
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

// toBaseJobLog 转换任务日志响应
func (c *BaseJobLogCase) toBaseJobLog(item *models.BaseJobLog) *admin.BaseJobLog {
	baseJobLog := c.mapper.ToDTO(item)
	baseJobLog.ProcessTime = strconv.FormatInt(int64(item.ProcessTime), 10)
	return baseJobLog
}

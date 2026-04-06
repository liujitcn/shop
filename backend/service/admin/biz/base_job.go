package biz

import (
	"context"
	"errors"
	"shop/service/admin/task"
	"time"

	"shop/api/gen/go/admin"
	"shop/api/gen/go/common"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/go-kratos/kratos/v2/log"
	_mapper "github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repo"
	"github.com/robfig/cron/v3"
)

// BaseJobCase 定时任务业务实例
type BaseJobCase struct {
	*biz.BaseCase
	*data.BaseJobRepo
	baseJobLogCase *BaseJobLogCase
	cron           *cron.Cron
	task           map[string]task.TaskExec
	formMapper     *_mapper.CopierMapper[admin.BaseJobForm, models.BaseJob]
	mapper         *_mapper.CopierMapper[admin.BaseJob, models.BaseJob]
}

// NewBaseJobCase 创建定时任务业务实例
func NewBaseJobCase(baseCase *biz.BaseCase, baseJobRepo *data.BaseJobRepo, baseJobLogCase *BaseJobLogCase, task map[string]task.TaskExec) *BaseJobCase {
	formMapper := _mapper.NewCopierMapper[admin.BaseJobForm, models.BaseJob]()
	formMapper.AppendConverters(_mapper.NewJSONTypeConverter[[]*admin.BaseJobArgs]().NewConverterPair())
	mapper := _mapper.NewCopierMapper[admin.BaseJob, models.BaseJob]()
	mapper.AppendConverters(_mapper.NewJSONTypeConverter[[]*admin.BaseJobArgs]().NewConverterPair())

	c := &BaseJobCase{
		BaseCase:       baseCase,
		BaseJobRepo:    baseJobRepo,
		baseJobLogCase: baseJobLogCase,
		cron:           cron.New(cron.WithSeconds()),
		task:           task,
		formMapper:     formMapper,
		mapper:         mapper,
	}

	ctx := c.Context.Context()
	// 查询全部定是任务，先停止，然后启动启用的
	list, err := c.List(ctx)
	if err != nil {
		log.Errorf("查询定时任务失败: %+v", err)
	}
	for _, item := range list {
		// 停止
		err = c.stopBaseJob(ctx, item)
		if err != nil {
			log.Errorf("停止定时任务失败: %+v", err)
			continue
		}
		// 启动
		err = c.startBaseJob(ctx, item)
		if err != nil {
			log.Errorf("启动定时任务失败: %+v", err)
			continue
		}
	}
	return c
}

// PageBaseJob 分页查询定时任务
func (c *BaseJobCase) PageBaseJob(ctx context.Context, req *admin.PageBaseJobRequest) (*admin.PageBaseJobResponse, error) {
	query := c.Query(ctx).BaseJob
	opts := make([]repo.QueryOption, 0, 4)
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))
	if req.GetName() != "" {
		opts = append(opts, repo.Where(query.Name.Like("%"+req.GetName()+"%")))
	}
	if req.GetInvokeTarget() != "" {
		opts = append(opts, repo.Where(query.InvokeTarget.Like("%"+req.GetInvokeTarget()+"%")))
	}
	if req.Status != nil {
		opts = append(opts, repo.Where(query.Status.Eq(int32(req.GetStatus()))))
	}

	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*admin.BaseJob, 0, len(list))
	for _, item := range list {
		baseJob := c.mapper.ToDTO(item)
		resList = append(resList, baseJob)
	}
	return &admin.PageBaseJobResponse{List: resList, Total: int32(total)}, nil
}

// GetBaseJob 获取定时任务
func (c *BaseJobCase) GetBaseJob(ctx context.Context, id int64) (*admin.BaseJobForm, error) {
	baseJob, err := c.FindById(ctx, id)
	if err != nil {
		return nil, err
	}
	res := c.formMapper.ToDTO(baseJob)
	return res, nil
}

// CreateBaseJob 创建定时任务
func (c *BaseJobCase) CreateBaseJob(ctx context.Context, req *admin.BaseJobForm) error {
	baseJob := c.formMapper.ToEntity(req)
	baseJob.Args = _string.ConvertAnyToJsonString(req.GetArgs())
	return c.Create(ctx, baseJob)
}

// UpdateBaseJob 更新定时任务
func (c *BaseJobCase) UpdateBaseJob(ctx context.Context, req *admin.BaseJobForm) error {
	baseJob := c.formMapper.ToEntity(req)
	baseJob.Args = _string.ConvertAnyToJsonString(req.GetArgs())
	return c.UpdateById(ctx, baseJob)
}

// DeleteBaseJob 删除定时任务
func (c *BaseJobCase) DeleteBaseJob(ctx context.Context, id string) error {
	return c.DeleteByIds(ctx, _string.ConvertStringToInt64Array(id))
}

// SetBaseJobStatus 设置定时任务状态
func (c *BaseJobCase) SetBaseJobStatus(ctx context.Context, req *common.SetStatusRequest) error {
	return c.UpdateById(ctx, &models.BaseJob{
		ID:     req.GetId(),
		Status: req.GetStatus(),
	})
}

// StartBaseJob 启动定时任务
func (c *BaseJobCase) StartBaseJob(ctx context.Context, req *admin.StartBaseJobRequest) error {
	baseJob, err := c.FindById(ctx, req.GetId())
	if err != nil {
		return err
	}
	return c.startBaseJob(ctx, baseJob)
}

// StopBaseJob 停止定时任务
func (c *BaseJobCase) StopBaseJob(ctx context.Context, req *admin.StopBaseJobRequest) error {
	baseJob, err := c.FindById(ctx, req.GetId())
	if err != nil {
		return err
	}
	return c.stopBaseJob(ctx, baseJob)
}

// ExecBaseJob 立即执行定时任务
func (c *BaseJobCase) ExecBaseJob(ctx context.Context, req *admin.ExecBaseJobRequest) error {
	baseJob, err := c.FindById(ctx, req.GetId())
	if err != nil {
		return err
	}
	return c.runJob(ctx, baseJob)
}

// runJob 执行任务并保存日志
func (c *BaseJobCase) runJob(ctx context.Context, baseJob *models.BaseJob) error {
	invokeTarget, ok := c.task[baseJob.InvokeTarget]
	if !ok {
		return errors.New("调用目标不存在")
	}

	baseJobForm := c.formMapper.ToDTO(baseJob)

	argsMap := make(map[string]string)
	for _, item := range baseJobForm.Args {
		argsMap[item.Key] = item.Value
	}

	baseJobLog := &models.BaseJobLog{
		JobID:       baseJob.ID,
		Input:       _string.ConvertAnyToJsonString(argsMap),
		ExecuteTime: time.Now(),
	}
	startAt := time.Now()
	ret, err := invokeTarget.Exec(argsMap)
	if err != nil {
		baseJobLog.Status = int32(common.BaseJobLogStatus_FAIL)
		baseJobLog.Error = err.Error()
	} else {
		baseJobLog.Status = int32(common.BaseJobLogStatus_SUCCESS)
	}
	baseJobLog.Output = _string.ConvertAnyToJsonString(ret)
	baseJobLog.ProcessTime = int32(time.Since(startAt).Milliseconds())

	saveErr := c.baseJobLogCase.Create(ctx, baseJobLog)
	if saveErr != nil {
		return saveErr
	}
	return err
}

func (c *BaseJobCase) startBaseJob(ctx context.Context, baseJob *models.BaseJob) error {
	if _, ok := c.task[baseJob.InvokeTarget]; !ok {
		return errors.New("调用目标不存在")
	}

	entryId, err := c.cron.AddFunc(baseJob.CronExpression, func() {
		err := c.runJob(context.Background(), baseJob)
		if err != nil {
			log.Errorf("runJob err:%+v", err)
		}
	})
	if err != nil {
		return err
	}

	c.cron.Start()
	return c.UpdateById(ctx, &models.BaseJob{
		ID:      baseJob.ID,
		EntryID: int32(entryId),
	})
}

func (c *BaseJobCase) stopBaseJob(ctx context.Context, baseJob *models.BaseJob) error {
	if baseJob.EntryID > 0 {
		c.cron.Remove(cron.EntryID(baseJob.EntryID))
	}
	baseJob.EntryID = 0
	return c.Query(ctx).BaseJob.WithContext(ctx).Save(baseJob)
}

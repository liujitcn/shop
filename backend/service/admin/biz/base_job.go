package biz

import (
	"context"
	"shop/api/gen/go/admin"
	"shop/api/gen/go/common"
	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"shop/pkg/job"

	_mapper "github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repo"
)

// BaseJobCase 定时任务业务实例
type BaseJobCase struct {
	*biz.BaseCase
	*data.BaseJobRepo
	baseJobLogCase *BaseJobLogCase
	cronServer     *job.CronServer
	formMapper     *_mapper.CopierMapper[admin.BaseJobForm, models.BaseJob]
	mapper         *_mapper.CopierMapper[admin.BaseJob, models.BaseJob]
}

// NewBaseJobCase 创建定时任务业务实例
func NewBaseJobCase(baseCase *biz.BaseCase, baseJobRepo *data.BaseJobRepo, baseJobLogCase *BaseJobLogCase, cronServer *job.CronServer) *BaseJobCase {
	formMapper := _mapper.NewCopierMapper[admin.BaseJobForm, models.BaseJob]()
	formMapper.AppendConverters(_mapper.NewJSONTypeConverter[[]*admin.BaseJobArgs]().NewConverterPair())
	mapper := _mapper.NewCopierMapper[admin.BaseJob, models.BaseJob]()
	mapper.AppendConverters(_mapper.NewJSONTypeConverter[[]*admin.BaseJobArgs]().NewConverterPair())

	return &BaseJobCase{
		BaseCase:       baseCase,
		BaseJobRepo:    baseJobRepo,
		baseJobLogCase: baseJobLogCase,
		cronServer:     cronServer,
		formMapper:     formMapper,
		mapper:         mapper,
	}
}

// PageBaseJob 分页查询定时任务
func (c *BaseJobCase) PageBaseJob(ctx context.Context, req *admin.PageBaseJobRequest) (*admin.PageBaseJobResponse, error) {
	query := c.Query(ctx).BaseJob
	opts := make([]repo.QueryOption, 0, 4)
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))
	// 传入任务名称时，按名称模糊匹配定时任务。
	if req.GetName() != "" {
		opts = append(opts, repo.Where(query.Name.Like("%"+req.GetName()+"%")))
	}
	// 传入调用目标时，按调用目标模糊匹配定时任务。
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
	err := c.Create(ctx, baseJob)
	if err != nil {
		// 命中调用目标唯一索引冲突时，返回稳定的业务冲突错误。
		if errorsx.IsMySQLDuplicateKey(err) {
			return errorsx.UniqueConflict("调用目标重复", "base_job", "invoke_target", "unique_base_job").WithCause(err)
		}
		return err
	}
	return nil
}

// UpdateBaseJob 更新定时任务
func (c *BaseJobCase) UpdateBaseJob(ctx context.Context, req *admin.BaseJobForm) error {
	baseJob := c.formMapper.ToEntity(req)
	baseJob.Args = _string.ConvertAnyToJsonString(req.GetArgs())
	err := c.UpdateById(ctx, baseJob)
	if err != nil {
		// 命中调用目标唯一索引冲突时，返回稳定的业务冲突错误。
		if errorsx.IsMySQLDuplicateKey(err) {
			return errorsx.UniqueConflict("调用目标重复", "base_job", "invoke_target", "unique_base_job").WithCause(err)
		}
		return err
	}
	return nil
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
	return c.cronServer.StartJob(ctx, baseJob)
}

// StopBaseJob 停止定时任务
func (c *BaseJobCase) StopBaseJob(ctx context.Context, req *admin.StopBaseJobRequest) error {
	baseJob, err := c.FindById(ctx, req.GetId())
	if err != nil {
		return err
	}
	return c.cronServer.StopJob(ctx, baseJob)
}

// ExecuteBaseJob 立即执行定时任务
func (c *BaseJobCase) ExecuteBaseJob(ctx context.Context, req *admin.ExecuteBaseJobRequest) error {
	baseJob, err := c.FindById(ctx, req.GetId())
	if err != nil {
		return err
	}
	return c.cronServer.RunJob(ctx, baseJob)
}

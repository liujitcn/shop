package biz

import (
	"context"
	_const "shop/pkg/const"
	"time"

	"shop/api/gen/go/admin"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	pkgUtils "shop/pkg/utils"

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
	c := &BaseLogCase{
		BaseCase:    baseCase,
		BaseLogRepo: baseLogRepo,
		mapper:      mapper.NewCopierMapper[admin.BaseLog, models.BaseLog](),
	}

	// 注册日志队列
	c.RegisterQueueConsumer(_const.Log, c.saveLog)
	return c
}

// PageBaseLog 分页查询日志
func (c *BaseLogCase) PageBaseLog(ctx context.Context, req *admin.PageBaseLogRequest) (*admin.PageBaseLogResponse, error) {
	query := c.Query(ctx).BaseLog
	opts := make([]repo.QueryOption, 0, 5)
	opts = append(opts, repo.Order(query.RequestTime.Desc()))
	// 传入操作名时，按操作名模糊匹配日志。
	if req.GetOperation() != "" {
		opts = append(opts, repo.Where(query.Operation.Like("%"+req.GetOperation()+"%")))
	}
	if req.StatusCode != nil {
		opts = append(opts, repo.Where(query.StatusCode.Eq(req.GetStatusCode())))
	}

	requestTime := req.GetRequestTime()
	// 仅在传入完整时间区间时，按请求时间范围过滤日志。
	if len(requestTime) == 2 {
		startTime := _time.StringTimeToTime(requestTime[0])
		endTime := _time.StringTimeToTime(requestTime[1])
		// 开始时间解析成功时，补充请求时间下界。
		if startTime != nil {
			opts = append(opts, repo.Where(query.RequestTime.Gte(*startTime)))
		}
		// 结束时间解析成功时，补充请求时间上界。
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

// saveLog 保存日志队列消息
func (c *BaseLogCase) saveLog(message queueData.Message) error {
	baseLog, err := pkgUtils.DecodeQueueData[models.BaseLog](message)
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
func (c *BaseLogCase) toBaseLog(item *models.BaseLog) *admin.BaseLog {
	costTime := time.Duration(item.CostTime) * time.Millisecond
	baseLog := c.mapper.ToDTO(item)
	baseLog.RequestTime = _time.TimeToTimeString(item.RequestTime)
	baseLog.CostTime = costTime.String()
	return baseLog
}

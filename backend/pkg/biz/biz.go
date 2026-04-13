package biz

import (
	"context"
	"shop/internal/cmd/server/assets"
	"sync"
	"time"

	_const "shop/pkg/const"

	"shop/api/gen/go/common"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/liujitcn/kratos-kit/auth"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/bootstrap"
	"github.com/liujitcn/kratos-kit/cache"
	"github.com/liujitcn/kratos-kit/database/gorm"
	"github.com/liujitcn/kratos-kit/pprof"
	"github.com/liujitcn/kratos-kit/queue"
	queueData "github.com/liujitcn/kratos-kit/queue/data"
	"github.com/liujitcn/kratos-kit/sdk"
)

type BaseCase struct {
	*bootstrap.Context
	queue          queue.Queue
	casbinRuleCase *CasbinRuleCase
	baseApiCase    *BaseApiCase
	quitChan       chan struct{} //退出Chan
	closeOnce      sync.Once
	taskTimer      *time.Timer
	rwLock         sync.RWMutex //异步数据锁
}

// NewBaseCase 创建基础业务实例。
func NewBaseCase(
	ctx *bootstrap.Context,
	cache cache.Cache,
	queue queue.Queue,
	gorm *gorm.Client,
	pprof pprof.Pprof,
	casbinRuleCase *CasbinRuleCase,
	baseApiCase *BaseApiCase,
) (*BaseCase, func(), error) {

	// 设置全局变量
	sdk.Runtime.SetGormClient(gorm)
	sdk.Runtime.SetCache(cache)
	sdk.Runtime.SetQueue(queue)

	// 启动服务监控
	// 配置了 pprof 时，启动运行时性能分析服务。
	if pprof != nil {
		pprof.Start()
	}

	s := BaseCase{
		Context:        ctx,
		queue:          queue,
		casbinRuleCase: casbinRuleCase,
		baseApiCase:    baseApiCase,
		quitChan:       make(chan struct{}),
		closeOnce:      sync.Once{},
		taskTimer:      nil,
		rwLock:         sync.RWMutex{},
	}
	// 启动后台服务
	go s.serve()

	cleanup := func() {
		s.close()
		// 启用了 pprof 时，清理阶段同步停止性能分析服务。
		if pprof != nil {
			pprof.Stop()
		}
	}

	// 检查API
	err := s.baseApiCase.apiCheck(assets.OpenApiData)
	if err != nil {
		return nil, cleanup, err
	}

	// 加载casbin
	err = s.RebuildPolicyRule(ctx.Context())
	if err != nil {
		return nil, cleanup, err
	}

	return &s, cleanup, nil
}

// RegisterQueueConsumer 注册异步队列消费者。
func (c *BaseCase) RegisterQueueConsumer(queueName _const.Queue, fn func(message queueData.Message) error) {
	c.queue.Register(string(queueName), fn)
}

// GetAuthInfo 获取当前登录用户认证信息
func (c *BaseCase) GetAuthInfo(ctx context.Context) (*authData.UserTokenPayload, error) {
	authInfo, err := auth.FromContext(ctx)
	if err != nil {
		log.Errorf("用户认证失败[%s]", err.Error())
		return nil, common.ErrorAccessForbidden("用户认证失败")
	}
	return authInfo, nil
}

// RebuildPolicyRule 重建内存权限策略。
func (c *BaseCase) RebuildPolicyRule(ctx context.Context) error {
	return c.casbinRuleCase.rebuildPolicyRule(ctx)
}

// close 关闭后台任务资源。
func (c *BaseCase) close() {
	c.closeOnce.Do(func() {
		// 定时器存在时，先停止后再关闭退出信号。
		// 已创建后台定时器时，先停止避免关闭后继续触发任务。
		if c.taskTimer != nil {
			c.taskTimer.Stop()
		}
		close(c.quitChan)
	})
}

// serve 启动后台队列消费线程。
func (c *BaseCase) serve() {
	// 启动队列
	c.queue.Run()
	// 循环处理同步事件
	for {
		select {
		case <-c.quitChan:
			return
		}
	}
}

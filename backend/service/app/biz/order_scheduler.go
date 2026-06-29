package biz

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"shop/pkg/biz"

	"github.com/go-kratos/kratos/v3/log"
)

// OrderSchedulerCase 维护订单自动取消调度任务。
type OrderSchedulerCase struct {
	*biz.BaseCase
	timers sync.Map // 存储订单ID和对应的定时器
	log    *slog.Logger
}

// NewOrderSchedulerCase 创建订单调度业务处理对象
func NewOrderSchedulerCase(baseCase *biz.BaseCase) *OrderSchedulerCase {
	return &OrderSchedulerCase{
		BaseCase: baseCase,
		log:      baseCase.NewLoggerHelper("service.app.biz"),
	}
}

// AddSchedule 添加订单自动取消调度任务
func (s *OrderSchedulerCase) AddSchedule(orderID int64, d time.Duration, cancelFunc func()) {
	s.log.Info(fmt.Sprintf("order schedule add %d", orderID))
	timer := time.AfterFunc(d, func() {
		cancelFunc()
		s.timers.Delete(orderID)
	})
	s.timers.Store(orderID, timer)
}

// DeleteScheduled 删除订单自动取消调度任务
func (s *OrderSchedulerCase) DeleteScheduled(orderID int64) {
	// 命中已注册的定时器时，先停止再清理调度记录。
	if timer, ok := s.timers.Load(orderID); ok {
		timer.(*time.Timer).Stop()
		log.Info(fmt.Sprintf("order schedule delete %d", orderID))
		s.timers.Delete(orderID)
	}
}

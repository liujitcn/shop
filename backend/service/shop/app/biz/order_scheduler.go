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
	timers sync.Map // 存储交易单ID和对应的定时器
	log    *slog.Logger
}

// NewOrderSchedulerCase 创建订单调度业务处理对象
func NewOrderSchedulerCase(baseCase *biz.BaseCase) *OrderSchedulerCase {
	return &OrderSchedulerCase{
		BaseCase: baseCase,
		log:      baseCase.NewLoggerHelper("service.app.biz"),
	}
}

// AddSchedule 添加交易单自动取消调度任务。
func (s *OrderSchedulerCase) AddSchedule(tradeID int64, d time.Duration, cancelFunc func()) {
	s.log.Info(fmt.Sprintf("order schedule add %d", tradeID))
	var timer *time.Timer
	timer = time.AfterFunc(d, func() {
		// 先移除当前任务，回调失败时才能为同一交易重新注册重试任务。
		s.timers.CompareAndDelete(tradeID, timer)
		cancelFunc()
	})
	previous, loaded := s.timers.Swap(tradeID, timer)
	if loaded {
		previous.(*time.Timer).Stop()
	}
}

// DeleteScheduled 删除交易单自动取消调度任务。
func (s *OrderSchedulerCase) DeleteScheduled(tradeID int64) {
	// 命中已注册的定时器时，先移除再停止，避免并发回调误删后续任务。
	if timer, ok := s.timers.LoadAndDelete(tradeID); ok {
		timer.(*time.Timer).Stop()
		log.Info(fmt.Sprintf("order schedule delete %d", tradeID))
	}
}

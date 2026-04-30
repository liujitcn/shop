package biz

import (
	"sync"
	"time"

	"shop/pkg/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type OrderSchedulerCase struct {
	*biz.BaseCase
	timers sync.Map // 存储订单ID和对应的定时器
	log    *log.Helper
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
	s.log.Infof("order schedule add %d", orderID)
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
		log.Infof("order schedule delete %d", orderID)
		s.timers.Delete(orderID)
	}
}

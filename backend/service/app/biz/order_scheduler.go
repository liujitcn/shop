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
func (s *OrderSchedulerCase) AddSchedule(orderId int64, d time.Duration, cancelFunc func()) {
	s.log.Infof("order schedule add %d", orderId)
	timer := time.AfterFunc(d, func() {
		cancelFunc()
		s.timers.Delete(orderId)
	})
	s.timers.Store(orderId, timer)
}

// DeleteScheduled 删除订单自动取消调度任务
func (s *OrderSchedulerCase) DeleteScheduled(orderId int64) {
	if timer, ok := s.timers.Load(orderId); ok {
		timer.(*time.Timer).Stop()
		log.Infof("order schedule delete %d", orderId)
		s.timers.Delete(orderId)
	}
}

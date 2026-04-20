package job

import (
	"shop/pkg/job/task"

	"github.com/google/wire"
)

// ProviderSet 注册定时任务模块依赖。
var ProviderSet = wire.NewSet(
	NewCronServer,
	task.NewTradeBill,
	task.NewOrderStatDay,
	task.NewGoodsStatDay,
	task.NewTaskList,
)

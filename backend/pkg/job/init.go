package job

import (
	"shop/pkg/job/task"

	"github.com/google/wire"
)

// ProviderSet is server providers.
var ProviderSet = wire.NewSet(
	NewCronServer,
	task.NewTradeBill,
	task.NewTaskList,
)

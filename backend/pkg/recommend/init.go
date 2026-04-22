package recommend

import (
	"github.com/google/wire"
)

// ProviderSet 汇总配置层依赖注入提供者。
var ProviderSet = wire.NewSet(
	NewRecommend,
	NewUserSyncReceiver,
	NewGoodsSyncReceiver,
	NewOnlineUserReceiver,
	NewOnlineSessionReceiver,
	NewOnlineNamedReceiver,
	NewOnlineChainReceiver,
	NewQueueReceiver,
)

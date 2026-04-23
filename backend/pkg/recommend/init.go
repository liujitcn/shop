package recommend

import (
	pkgLocal "shop/pkg/recommend/local"
	pkgRemote "shop/pkg/recommend/remote"

	"github.com/google/wire"
)

// ProviderSet 汇总配置层依赖注入提供者。
var ProviderSet = wire.NewSet(
	pkgRemote.NewRecommend,
	pkgRemote.NewUserSyncReceiver,
	pkgRemote.NewGoodsSyncReceiver,
	pkgRemote.NewUserReceiver,
	pkgRemote.NewSessionReceiver,
	pkgRemote.NewNamedReceiver,
	pkgRemote.NewChainReceiver,
	pkgRemote.NewQueueReceiver,
	pkgLocal.NewRecommend,
	pkgLocal.NewContextReceiver,
	pkgLocal.NewHotReceiver,
	pkgLocal.NewExploreReceiver,
	pkgLocal.NewChainReceiver,
	NewGoodsReceiver,
)

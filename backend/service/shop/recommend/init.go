package recommend

import (
	"shop/service/shop/recommend/gorse"
	"shop/service/shop/recommend/local"

	"github.com/google/wire"
)

// ProviderSet 汇总配置层依赖注入提供者。
var ProviderSet = wire.NewSet(
	gorse.NewRecommend,
	gorse.NewDashboard,
	gorse.NewUserSyncReceiver,
	gorse.NewGoodsSyncReceiver,
	gorse.NewUserReceiver,
	gorse.NewSessionReceiver,
	gorse.NewNamedReceiver,
	gorse.NewChainReceiver,
	gorse.NewQueueReceiver,
	local.NewRecommend,
	local.NewContextReceiver,
	local.NewHotReceiver,
	local.NewExploreReceiver,
	local.NewChainReceiver,
	NewGoodsReceiver,
	NewRecommendSync,
)

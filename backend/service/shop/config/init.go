package config

import (
	"github.com/google/wire"
)

// ProviderSet 汇总商城业务配置依赖注入提供者。
var ProviderSet = wire.NewSet(
	NewShopConfig,
	ParseWxPay,
	ParseRecommend,
)

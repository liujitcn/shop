package configs

import (
	"github.com/google/wire"
)

// ProviderSet 汇总配置层依赖注入提供者。
var ProviderSet = wire.NewSet(
	NewShopConfig,
	ParseWxMiniApp,
	ParseWxPay,
	ParseRecommend,
	ParseOss,
	ParseData,
	ParseDatabase,
	ParseRedis,
	ParseQueue,
	ParsePprof,
	ParseAuthnJwt,
)

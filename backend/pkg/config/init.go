package config

import (
	"github.com/google/wire"
)

// ProviderSet 汇总配置层依赖注入提供者。
var ProviderSet = wire.NewSet(
	NewShopConfig,
	ParseWxPay,
	ParseRecommend,
	ParseAIModel,
	ParseOSS,
	ParseData,
	ParseDatabase,
	ParseRedis,
	ParseQueue,
	ParsePprof,
	ParseAuthnJWT,
	ParseOAuth,
)

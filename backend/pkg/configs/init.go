package configs

import (
	"github.com/google/wire"
)

// ProviderSet is server providers.
var ProviderSet = wire.NewSet(
	NewShopConfig,
	ParseWxMiniApp,
	ParseWxPay,
	ParseOss,
	ParseData,
	ParseDatabase,
	ParseRedis,
	ParseQueue,
	ParsePprof,
	ParseAuthnJwt,
)

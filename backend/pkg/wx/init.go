package wx

import (
	"github.com/google/wire"
)

// ProviderSet 汇总微信能力依赖注入提供者。
var ProviderSet = wire.NewSet(
	NewWxPayCase,
)

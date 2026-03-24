package base

import (
	"shop/service/base/biz"

	"github.com/google/wire"
)

// ProviderSet is server providers.
var ProviderSet = wire.NewSet(
	biz.NewBaseDeptCase,
	biz.NewBaseRoleCase,
	biz.NewBaseUserCase,
	biz.NewConfigCase,
	biz.NewFileCase,
	biz.NewLoginCase,

	NewConfigService,
	NewFileService,
	NewLoginService,
)

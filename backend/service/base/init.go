package base

import (
	"shop/service/base/biz"

	"github.com/google/wire"
)

// ProviderSet 汇总基础服务依赖注入提供者。
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

package app

import (
	"shop/service/system/app/biz"

	"github.com/google/wire"
)

// ProviderSet 汇总系统商城端服务依赖注入提供者。
var ProviderSet = wire.NewSet(
	biz.NewAuthCase,
	biz.NewBaseAreaCase,
	biz.NewBaseDeptCase,
	biz.NewBaseDictCase,
	biz.NewBaseDictItemCase,
	biz.NewBaseRoleCase,
	biz.NewBaseUserCase,
	NewAuthService,
	NewBaseAreaService,
	NewBaseDictService,
)

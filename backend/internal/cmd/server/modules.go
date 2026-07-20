package main

import (
	host "shop/server"
	baseserver "shop/server/base"
	shopadminserver "shop/server/shop/admin"
	shopappserver "shop/server/shop/app"
	systemadminserver "shop/server/system/admin"
	systemappserver "shop/server/system/app"
)

// newModules 汇总当前进程启用的基础、系统和商城业务模块。
func newModules(
	baseModule baseserver.Services,
	systemAdminModule systemadminserver.Services,
	systemAppModule systemappserver.Services,
	shopAdminModule shopadminserver.Services,
	shopAppModule shopappserver.Services,
) host.Modules {
	return host.Modules{
		baseModule,
		systemAdminModule,
		systemAppModule,
		shopAdminModule,
		shopAppModule,
	}
}

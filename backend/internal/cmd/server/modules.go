package main

import (
	host "shop/server"
	baseserver "shop/server/base"
	systemadminserver "shop/server/system/admin"
	systemappserver "shop/server/system/app"
)

// newModules 汇总当前进程启用的基础与系统模块。
func newModules(
	baseModule baseserver.Services,
	systemAdminModule systemadminserver.Services,
	systemAppModule systemappserver.Services,
) (host.Modules, error) {
	return host.Modules{
		baseModule,
		systemAdminModule,
		systemAppModule,
	}, nil
}

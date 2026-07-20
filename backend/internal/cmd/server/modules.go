package main

import (
	"shop/pkg/event"
	"shop/pkg/job"
	host "shop/server"
	baseserver "shop/server/base"
	shopadminserver "shop/server/shop/admin"
	shopappserver "shop/server/shop/app"
	systemadminserver "shop/server/system/admin"
	systemappserver "shop/server/system/app"
	shopcodegen "shop/service/shop/admin/codegen"
	"shop/service/shop/agent/aiflow"
	"shop/service/shop/queue"
	"shop/service/shop/workspaceevent"
)

// newModules 汇总当前进程启用的基础、系统和商城业务模块。
func newModules(
	baseModule baseserver.Services,
	systemAdminModule systemadminserver.Services,
	systemAppModule systemappserver.Services,
	shopAdminModule shopadminserver.Services,
	shopAppModule shopappserver.Services,
	userEvents *event.UserEvents,
	recommendSubscriber *queue.RecommendUserEventSubscriber,
	taskRegistry *job.Registry,
	shopAdminTasks shopadminserver.TaskSet,
	shopAppTasks shopappserver.TaskSet,
	_ aiflow.Registration,
	_ shopcodegen.Registration,
	_ workspaceevent.SSEReady,
) (host.Modules, error) {
	err := host.RegisterTasks(taskRegistry, shopAdminTasks, shopAppTasks)
	if err != nil {
		return nil, err
	}
	userEvents.Subscribe(recommendSubscriber)
	return host.Modules{
		baseModule,
		systemAdminModule,
		systemAppModule,
		shopAdminModule,
		shopAppModule,
	}, nil
}

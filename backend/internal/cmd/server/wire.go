//go:build wireinject
// +build wireinject

package main

import (
	"github.com/go-kratos/kratos/v3"
	"github.com/google/wire"
	"github.com/liujitcn/kratos-kit/bootstrap"

	einoModel "shop/pkg/agent/eino/model"
	"shop/pkg/biz"
	systemConfig "shop/pkg/config"
	"shop/pkg/event"
	"shop/pkg/gen/data"
	"shop/pkg/job"
	"shop/pkg/middleware"
	transportSSE "shop/pkg/sse"
	"shop/server"
	baseserver "shop/server/base"
	systemadminserver "shop/server/system/admin"
	systemappserver "shop/server/system/app"
	"shop/service/base"
	"shop/service/base/agent/ai"
	systemadmin "shop/service/system/admin"
	systemapp "shop/service/system/app"
)

// initApp 初始化 Kratos 应用实例。
func initApp(*bootstrap.Context) (*kratos.App, func(), error) {
	panic(wire.Build(
		event.NewUserEvents,
		job.ProviderSet,
		transportSSE.NewRegistry,
		transportSSE.NewPublisher,
		biz.ProviderSet,
		einoModel.NewResponsesClient,
		ai.NewRuntime,
		systemConfig.ProviderSet,
		data.ProviderSet,
		middleware.ProviderSet,
		systemadmin.ProviderSet,
		systemapp.ProviderSet,
		base.ProviderSet,
		baseserver.ProviderSet,
		systemadminserver.ProviderSet,
		systemappserver.ProviderSet,
		baseserver.NewSSEHandler,
		newModules,
		shopProviderSet,
		wire.Bind(new(server.TerminalToolSetter), new(*ai.Runtime)),
		server.ProviderSet,
		newApp,
	))
}

//go:build wireinject
// +build wireinject

package main

import (
	pkgBiz "shop/pkg/biz"
	pkgConfigs "shop/pkg/configs"
	pkgGenData "shop/pkg/gen/data"
	pkgJob "shop/pkg/job"
	pkgMiddleware "shop/pkg/middleware"
	pkgWx "shop/pkg/wx"
	"shop/server"
	"shop/service/admin"
	"shop/service/app"
	"shop/service/base"

	"github.com/go-kratos/kratos/v2"
	"github.com/google/wire"

	"github.com/liujitcn/kratos-kit/bootstrap"
)

// initApp 初始化 Kratos 应用实例。
func initApp(*bootstrap.Context) (*kratos.App, func(), error) {
	panic(wire.Build(
		pkgJob.ProviderSet,
		pkgWx.ProviderSet,
		pkgBiz.ProviderSet,
		pkgConfigs.ProviderSet,
		pkgGenData.ProviderSet,
		pkgMiddleware.ProviderSet,
		admin.ProviderSet,
		app.ProviderSet,
		base.ProviderSet,
		server.ProviderSet,
		newApp,
	))
}

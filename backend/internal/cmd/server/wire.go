//go:build wireinject
// +build wireinject

package main

import (
	pkgBiz "shop/pkg/biz"
	pkgConfigs "shop/pkg/configs"
	pkgGenData "shop/pkg/gen/data"
	pkgMiddleware "shop/pkg/middleware"
	"shop/server"
	"shop/service/admin"
	"shop/service/app"
	"shop/service/base"

	"github.com/go-kratos/kratos/v2"
	"github.com/google/wire"

	"github.com/liujitcn/kratos-kit/bootstrap"
)

// initApp init kratos application.
func initApp(*bootstrap.Context) (*kratos.App, func(), error) {
	panic(wire.Build(
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

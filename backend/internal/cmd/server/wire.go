//go:build wireinject
// +build wireinject

package main

import (
	"shop/pkg/biz"
	"shop/pkg/config"
	"shop/pkg/gen/data"
	"shop/pkg/job"
	"shop/pkg/llm"
	"shop/pkg/middleware"
	"shop/pkg/recommend"
	"shop/pkg/wx"
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
		job.ProviderSet,
		wx.ProviderSet,
		biz.ProviderSet,
		config.ProviderSet,
		data.ProviderSet,
		recommend.ProviderSet,
		llm.ProviderSet,
		middleware.ProviderSet,
		admin.ProviderSet,
		app.ProviderSet,
		base.ProviderSet,
		server.ProviderSet,
		newApp,
	))
}

//go:build wireinject
// +build wireinject

package main

import (
	"github.com/go-kratos/kratos/v3"
	"github.com/google/wire"
	"github.com/liujitcn/kratos-kit/bootstrap"

	einoModel "shop/pkg/agent/eino/model"
	einoStructured "shop/pkg/agent/eino/structured"
	"shop/pkg/biz"
	systemConfig "shop/pkg/config"
	"shop/pkg/gen/data"
	"shop/pkg/job"
	"shop/pkg/middleware"
	"shop/server"
	baseserver "shop/server/base"
	shopadminserver "shop/server/shop/admin"
	shopappserver "shop/server/shop/app"
	systemadminserver "shop/server/system/admin"
	systemappserver "shop/server/system/app"
	"shop/service/base"
	"shop/service/base/agent/ai"
	"shop/service/shop/admin"
	shopadminbiz "shop/service/shop/admin/biz"
	"shop/service/shop/app"
	commentagent "shop/service/shop/app/agent/comment"
	shopappbiz "shop/service/shop/app/biz"
	shopConfig "shop/service/shop/config"
	"shop/service/shop/recommend"
	"shop/service/shop/wx"
	systemadmin "shop/service/system/admin"
	systemapp "shop/service/system/app"
)

// newTaskList 组装各业务模块提供的定时任务执行器。
func newTaskList(
	tradeBill *shopadminbiz.TradeBill,
	orderStatDay *shopadminbiz.OrderStatDay,
	goodsStatDay *shopadminbiz.GoodsStatDay,
	recommendSync *recommend.RecommendSync,
	commentAuditRetry *shopappbiz.CommentAuditRetry,
	orderRefundRetry *shopappbiz.OrderRefundRetry,
) map[string]job.TaskExec {
	return job.NewTaskList(tradeBill, orderStatDay, goodsStatDay, recommendSync, commentAuditRetry, orderRefundRetry)
}

// initApp 初始化 Kratos 应用实例。
func initApp(*bootstrap.Context) (*kratos.App, func(), error) {
	panic(wire.Build(
		job.ProviderSet,
		newTaskList,
		biz.ProviderSet,
		einoModel.NewChatClient,
		einoModel.NewResponsesClient,
		einoStructured.NewRunner,
		commentagent.NewRuntime,
		ai.NewRuntime,
		systemConfig.ProviderSet,
		shopConfig.ProviderSet,
		data.ProviderSet,
		recommend.ProviderSet,
		wx.ProviderSet,
		middleware.ProviderSet,
		admin.ProviderSet,
		app.ProviderSet,
		systemadmin.ProviderSet,
		systemapp.ProviderSet,
		base.ProviderSet,
		baseserver.ProviderSet,
		shopadminserver.ProviderSet,
		shopappserver.ProviderSet,
		systemadminserver.ProviderSet,
		systemappserver.ProviderSet,
		baseserver.NewSSEHandler,
		newModules,
		wire.Bind(new(server.TerminalToolSetter), new(*ai.Runtime)),
		server.ProviderSet,
		newApp,
	))
}

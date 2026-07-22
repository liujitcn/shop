//go:build wireinject
// +build wireinject

package main

import (
	einoModel "shop/pkg/agent/eino/model"
	einoStructured "shop/pkg/agent/eino/structured"
	shopadminserver "shop/server/shop/admin"
	shopappserver "shop/server/shop/app"
	"shop/service/shop/admin"
	"shop/service/shop/agent/aiflow"
	"shop/service/shop/app"
	commentagent "shop/service/shop/app/agent/comment"
	shopConfig "shop/service/shop/config"
	"shop/service/shop/queue"
	"shop/service/shop/recommend"
	"shop/service/shop/workspaceevent"
	"shop/service/shop/wx"

	"github.com/google/wire"
)

// shopProviderSet 汇总商城模块在组合根中的全部依赖。
var shopProviderSet = wire.NewSet(
	aiflow.NewProvider,
	aiflow.NewRegistration,
	einoModel.NewChatClient,
	einoStructured.NewRunner,
	commentagent.NewRuntime,
	shopConfig.ProviderSet,
	recommend.ProviderSet,
	wx.ProviderSet,
	admin.ProviderSet,
	app.ProviderSet,
	shopadminserver.ProviderSet,
	shopappserver.ProviderSet,
	queue.NewRecommendUserEventSubscriber,
	shopadminserver.NewTaskSet,
	shopappserver.NewTaskSet,
	workspaceevent.NewSSEReady,
)

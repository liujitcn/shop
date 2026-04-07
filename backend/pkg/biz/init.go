package biz

import (
	"github.com/google/wire"
	"github.com/liujitcn/kratos-kit/cache"
	"github.com/liujitcn/kratos-kit/database/gorm"
	"github.com/liujitcn/kratos-kit/oss"
	"github.com/liujitcn/kratos-kit/pprof"
	"github.com/liujitcn/kratos-kit/queue"
)

// ProviderSet is server providers.
var ProviderSet = wire.NewSet(
	NewBaseApiCase,
	NewCasbinRuleCase,
	NewBaseCase,
	oss.NewOSS,
	gorm.NewGormClient,
	queue.NewQueue,
	cache.NewCache,
	pprof.NewPprof,
)

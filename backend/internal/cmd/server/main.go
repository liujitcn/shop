package main

import (
	"context"
	"shop/api/gen/go/conf"
	"shop/pkg/configs"
	"shop/pkg/job"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	bootstrapConf "github.com/liujitcn/kratos-kit/api/gen/go/conf"
	"github.com/liujitcn/kratos-kit/bootstrap"

	//_ "github.com/liujitcn/kratos-kit/database/gorm/driver/bigquery"
	_ "github.com/liujitcn/kratos-kit/database/gorm/driver/mysql"
	//_ "github.com/liujitcn/kratos-kit/database/gorm/driver/oracle"
	//_ "github.com/liujitcn/kratos-kit/database/gorm/driver/postgres"
	//_ "github.com/liujitcn/kratos-kit/database/gorm/driver/sqlite"
	//_ "github.com/liujitcn/kratos-kit/database/gorm/driver/sqlserver"

	//_ "github.com/liujitcn/kratos-kit/config/apollo"
	//_ "github.com/liujitcn/kratos-kit/config/consul"
	//_ "github.com/liujitcn/kratos-kit/config/etcd"
	//_ "github.com/liujitcn/kratos-kit/config/kubernetes"
	//_ "github.com/liujitcn/kratos-kit/config/nacos"
	//_ "github.com/liujitcn/kratos-kit/config/polaris"

	//_ "github.com/liujitcn/kratos-kit/logger/aliyun"
	//_ "github.com/liujitcn/kratos-kit/logger/fluent"
	//_ "github.com/liujitcn/kratos-kit/logger/logrus"
	//_ "github.com/liujitcn/kratos-kit/logger/tencent"
	_ "github.com/liujitcn/kratos-kit/logger/zap"
	//_ "github.com/liujitcn/kratos-kit/logger/zerolog"
	//_ "github.com/liujitcn/kratos-kit/registry/consul"
	//_ "github.com/liujitcn/kratos-kit/registry/etcd"
	//_ "github.com/liujitcn/kratos-kit/registry/eureka"
	//_ "github.com/liujitcn/kratos-kit/registry/kubernetes"
	//_ "github.com/liujitcn/kratos-kit/registry/nacos"
	//_ "github.com/liujitcn/kratos-kit/registry/polaris"
	//_ "github.com/liujitcn/kratos-kit/registry/servicecomb"
	//_ "github.com/liujitcn/kratos-kit/registry/zookeeper"
)

var (
	Project = "shop"
	AppId   = "app"
	version = "1.0.0"
)

// newApp 组装应用实例并挂载定时任务、GRPC 与 HTTP 服务。
func newApp(
	ctx *bootstrap.Context,
	cron *job.CronServer,
	gs *grpc.Server,
	hs *http.Server,
) *kratos.App {
	return bootstrap.NewApp(ctx,
		cron,
		gs,
		hs,
	)
}

// runApp 初始化上下文并启动应用主流程。
func runApp() error {
	ctx := bootstrap.NewContext(
		context.Background(),
		&bootstrapConf.AppInfo{
			Project: Project,
			AppId:   AppId,
			Version: version,
		},
	)
	ctx.RegisterCustomConfig(configs.WrapperConfigKey, &conf.ShopConfigWrapper{})
	return bootstrap.RunApp(ctx, initApp)
}

// main 作为服务启动入口，负责执行应用启动并在失败时中止进程。
func main() {
	// 应用启动失败时直接中止进程，避免服务以异常状态继续运行。
	if err := runApp(); err != nil {
		panic(err)
	}
}

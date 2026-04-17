package server

import (
	"io/fs"
	stdhttp "net/http"
	"os"
	"path/filepath"
	adminApi "shop/api/gen/go/admin"
	appApi "shop/api/gen/go/app"
	baseApi "shop/api/gen/go/base"
	"shop/internal/cmd/server/assets"
	"shop/pkg/gen/data"
	"shop/pkg/middleware/logging"
	"shop/service/admin"
	"shop/service/app"
	"shop/service/base"
	"strings"

	kratosMiddleware "github.com/go-kratos/kratos/v2/middleware"
	kratosHttp "github.com/go-kratos/kratos/v2/transport/http"
	bootstrapConf "github.com/liujitcn/kratos-kit/api/gen/go/conf"
	"github.com/liujitcn/kratos-kit/auth"
	authnEngine "github.com/liujitcn/kratos-kit/auth/authn/engine"
	authzEngine "github.com/liujitcn/kratos-kit/auth/authz/engine"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/bootstrap"
	"github.com/liujitcn/kratos-kit/rpc"
	swaggerUI "github.com/liujitcn/kratos-kit/swagger-ui"
)

type HttpMiddlewares []kratosMiddleware.Middleware

// NewHttpMiddleware 创建 HTTP 服务统一中间件链。
func NewHttpMiddleware(
	ctx *bootstrap.Context,
	authenticator authnEngine.Authenticator,
	baseUserRepo *data.BaseUserRepo,
	authorizer authzEngine.Engine,
	userToken *authData.UserToken,
	jwtCfg *bootstrapConf.Authentication_Jwt,
) HttpMiddlewares {
	var ms HttpMiddlewares
	cfg := ctx.GetConfig()
	// 开启日志中间件时，统一挂载请求日志与操作者解析逻辑。
	if cfg != nil && cfg.Server != nil && cfg.Server.Http != nil && cfg.Server.Http.Middleware != nil && cfg.Server.Http.Middleware.EnableLogging {
		ms = append(ms, logging.Server(ctx.GetLogger(), baseUserRepo, authenticator))
	}
	ms = append(ms, auth.NewAuthMiddleware(authenticator, authorizer, userToken, jwtCfg))
	return ms
}

// NewHttpServer 创建 HTTP Server 并注册后端与前端静态路由。
func NewHttpServer(
	ctx *bootstrap.Context,
	middlewares HttpMiddlewares,

	adminAuth *admin.AuthService,
	adminBaseApi *admin.BaseApiService,
	adminBaseConfig *admin.BaseConfigService,
	adminBaseDept *admin.BaseDeptService,
	adminBaseDict *admin.BaseDictService,
	adminBaseJob *admin.BaseJobService,
	adminBaseLog *admin.BaseLogService,
	adminBaseMenu *admin.BaseMenuService,
	adminBaseRole *admin.BaseRoleService,
	adminBaseUser *admin.BaseUserService,
	adminGoodsAnalytics *admin.GoodsAnalyticsService,
	adminGoodsReport *admin.GoodsReportService,
	adminGoodsCategory *admin.GoodsCategoryService,
	adminGoodsProp *admin.GoodsPropService,
	adminGoods *admin.GoodsInfoService,
	adminGoodsSku *admin.GoodsSkuService,
	adminGoodsSpec *admin.GoodsSpecService,
	adminOrderAnalytics *admin.OrderAnalyticsService,
	adminOrderReport *admin.OrderReportService,
	adminOrder *admin.OrderInfoService,
	adminPayBill *admin.PayBillService,
	adminRecommendModelVersion *admin.RecommendModelVersionService,
	adminShopBanner *admin.ShopBannerService,
	adminShopHot *admin.ShopHotService,
	adminShopService *admin.ShopServiceService,
	adminUserAnalytics *admin.UserAnalyticsService,
	adminUserStore *admin.UserStoreService,
	adminWorkspace *admin.WorkspaceService,

	appAuth *app.AuthService,
	appBaseArea *app.BaseAreaService,
	appBaseDict *app.BaseDictService,
	appGoodsCategory *app.GoodsCategoryService,
	appGoods *app.GoodsInfoService,
	appOrder *app.OrderInfoService,
	appPay *app.PayService,
	appRecommend *app.RecommendService,
	appShopBanner *app.ShopBannerService,
	appShopHot *app.ShopHotService,
	appShopService *app.ShopServiceService,
	appUserAddress *app.UserAddressService,
	appUserCart *app.UserCartService,
	appUserCollect *app.UserCollectService,
	appUserStore *app.UserStoreService,

	config *base.ConfigService,
	fileSvc *base.FileService,
	login *base.LoginService,
) (*kratosHttp.Server, error) {
	cfg := ctx.GetConfig()
	// 未启用 HTTP 配置时，跳过 HTTP 服务创建。
	if cfg == nil || cfg.Server == nil || cfg.Server.Http == nil {
		return nil, nil
	}

	srv, err := rpc.CreateHttpServer(cfg, middlewares...)
	if err != nil {
		return nil, err
	}

	adminApi.RegisterAuthServiceHTTPServer(srv, adminAuth)
	adminApi.RegisterBaseApiServiceHTTPServer(srv, adminBaseApi)
	adminApi.RegisterBaseConfigServiceHTTPServer(srv, adminBaseConfig)
	adminApi.RegisterBaseDeptServiceHTTPServer(srv, adminBaseDept)
	adminApi.RegisterBaseDictServiceHTTPServer(srv, adminBaseDict)
	adminApi.RegisterBaseJobServiceHTTPServer(srv, adminBaseJob)
	adminApi.RegisterBaseLogServiceHTTPServer(srv, adminBaseLog)
	adminApi.RegisterBaseMenuServiceHTTPServer(srv, adminBaseMenu)
	adminApi.RegisterBaseRoleServiceHTTPServer(srv, adminBaseRole)
	adminApi.RegisterBaseUserServiceHTTPServer(srv, adminBaseUser)
	adminApi.RegisterGoodsAnalyticsServiceHTTPServer(srv, adminGoodsAnalytics)
	adminApi.RegisterGoodsReportServiceHTTPServer(srv, adminGoodsReport)
	adminApi.RegisterGoodsCategoryServiceHTTPServer(srv, adminGoodsCategory)
	adminApi.RegisterGoodsPropServiceHTTPServer(srv, adminGoodsProp)
	adminApi.RegisterGoodsInfoServiceHTTPServer(srv, adminGoods)
	adminApi.RegisterGoodsSkuServiceHTTPServer(srv, adminGoodsSku)
	adminApi.RegisterGoodsSpecServiceHTTPServer(srv, adminGoodsSpec)
	adminApi.RegisterOrderAnalyticsServiceHTTPServer(srv, adminOrderAnalytics)
	adminApi.RegisterOrderReportServiceHTTPServer(srv, adminOrderReport)
	adminApi.RegisterOrderInfoServiceHTTPServer(srv, adminOrder)
	adminApi.RegisterPayBillServiceHTTPServer(srv, adminPayBill)
	adminApi.RegisterRecommendModelVersionServiceHTTPServer(srv, adminRecommendModelVersion)
	adminApi.RegisterShopBannerServiceHTTPServer(srv, adminShopBanner)
	adminApi.RegisterShopHotServiceHTTPServer(srv, adminShopHot)
	adminApi.RegisterShopServiceServiceHTTPServer(srv, adminShopService)
	adminApi.RegisterUserAnalyticsServiceHTTPServer(srv, adminUserAnalytics)
	adminApi.RegisterUserStoreServiceHTTPServer(srv, adminUserStore)
	adminApi.RegisterWorkspaceServiceHTTPServer(srv, adminWorkspace)

	appApi.RegisterAuthServiceHTTPServer(srv, appAuth)
	appApi.RegisterBaseAreaServiceHTTPServer(srv, appBaseArea)
	appApi.RegisterBaseDictServiceHTTPServer(srv, appBaseDict)
	appApi.RegisterGoodsCategoryServiceHTTPServer(srv, appGoodsCategory)
	appApi.RegisterGoodsInfoServiceHTTPServer(srv, appGoods)
	appApi.RegisterOrderInfoServiceHTTPServer(srv, appOrder)
	appApi.RegisterPayServiceHTTPServer(srv, appPay)
	appApi.RegisterRecommendServiceHTTPServer(srv, appRecommend)
	appApi.RegisterShopBannerServiceHTTPServer(srv, appShopBanner)
	appApi.RegisterShopHotServiceHTTPServer(srv, appShopHot)
	appApi.RegisterShopServiceServiceHTTPServer(srv, appShopService)
	appApi.RegisterUserAddressServiceHTTPServer(srv, appUserAddress)
	appApi.RegisterUserCartServiceHTTPServer(srv, appUserCart)
	appApi.RegisterUserCollectServiceHTTPServer(srv, appUserCollect)
	appApi.RegisterUserStoreServiceHTTPServer(srv, appUserStore)

	baseApi.RegisterConfigServiceHTTPServer(srv, config)
	// 修改http接口实现
	base.RegisterFileServiceHTTPServer(srv, fileSvc)
	baseApi.RegisterLoginServiceHTTPServer(srv, login)

	ossRootDirectory := "./data"
	// 配置了本地 OSS 根目录时，优先使用配置值覆盖默认目录。
	if cfg.GetOss() != nil && cfg.GetOss().GetRootDirectory() != "" {
		ossRootDirectory = cfg.GetOss().GetRootDirectory()
	}
	var shopStaticDirectory = filepath.Join(ossRootDirectory, "shop")
	// 将本地 OSS 目录暴露为静态资源目录 访问 /shop/* 时直接映射到 ./data/shop/*
	staticHandler := stdhttp.StripPrefix("/shop/", stdhttp.FileServer(stdhttp.Dir(shopStaticDirectory)))
	srv.HandlePrefix("/shop/", staticHandler)

	// 自动发现本地 OSS 根目录下的前端入口，按子目录名称挂载为 SPA 路由。
	registerLocalSPARoutes(srv, ossRootDirectory)

	// 显式启用 Swagger 时，注册内存中的 OpenAPI 文档页面。
	if cfg.GetServer().GetHttp().GetEnableSwagger() {
		swaggerUI.RegisterSwaggerUIServerWithOption(
			srv,
			swaggerUI.WithTitle(ctx.GetAppInfo().GetName()),
			swaggerUI.WithMemoryData(assets.OpenApiData, "yaml"),
		)
	}

	return srv, nil
}

// registerLocalSPARoutes 扫描根目录下包含 index.html 的子目录，并按目录名注册单页应用路由。
func registerLocalSPARoutes(srv *kratosHttp.Server, rootDirectory string) {
	entries, err := os.ReadDir(rootDirectory)
	if err != nil {
		return
	}
	for _, entry := range entries {
		// 仅处理目录，忽略根目录下的普通文件。
		if !entry.IsDir() {
			continue
		}
		var directoryName = entry.Name()
		var indexPath = filepath.Join(rootDirectory, directoryName, "index.html")
		// 子目录未提供入口页面时，不注册为单页应用。
		if _, err = os.Stat(indexPath); err != nil {
			continue
		}
		var routePrefix = "/" + directoryName
		var spaHandler = newSPAHandler(os.DirFS(filepath.Join(rootDirectory, directoryName)), routePrefix)
		srv.Handle(routePrefix, spaHandler)
		srv.HandlePrefix(routePrefix+"/", spaHandler)
	}
}

// newSPAHandler 创建基于文件系统的单页应用处理器。
func newSPAHandler(webFS fs.FS, urlPrefix string) stdhttp.Handler {
	var fileHandler = stdhttp.StripPrefix(urlPrefix, stdhttp.FileServer(stdhttp.FS(webFS)))
	return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		var relativePath = strings.TrimPrefix(r.URL.Path, urlPrefix)
		relativePath = strings.TrimPrefix(relativePath, "/")
		// 访问应用根路径时，直接返回入口页面。
		if relativePath == "" {
			stdhttp.ServeFileFS(w, r, webFS, "index.html")
			return
		}
		// 命中真实静态文件时，交给文件服务直接返回。
		if _, err := fs.Stat(webFS, relativePath); err == nil {
			fileHandler.ServeHTTP(w, r)
			return
		}
		// 前端路由命中不到真实文件时，统一回退到入口页面。
		stdhttp.ServeFileFS(w, r, webFS, "index.html")
	})
}

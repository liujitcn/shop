package server

import (
	"io/fs"
	stdhttp "net/http"
	"os"
	"path/filepath"
	adminv1 "shop/api/gen/go/admin/v1"
	appv1 "shop/api/gen/go/app/v1"
	basev1 "shop/api/gen/go/base/v1"
	"shop/internal/cmd/server/assets"
	"shop/pkg/gen/data"
	"shop/pkg/middleware/logging"
	"shop/service/base"
	"strings"

	bootstrapConfigv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"

	kratosMiddleware "github.com/go-kratos/kratos/v2/middleware"
	kratosHTTP "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/liujitcn/kratos-kit/auth"
	authnEngine "github.com/liujitcn/kratos-kit/auth/authn/engine"
	authzEngine "github.com/liujitcn/kratos-kit/auth/authz/engine"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/bootstrap"
	"github.com/liujitcn/kratos-kit/rpc"
	"github.com/liujitcn/kratos-kit/rpc/middleware/requestid"
	swaggerUI "github.com/liujitcn/kratos-kit/swagger-ui"
)

type HTTPMiddlewares []kratosMiddleware.Middleware

// NewHTTPMiddleware 创建 HTTP 服务统一中间件链。
func NewHTTPMiddleware(
	ctx *bootstrap.Context,
	authenticator authnEngine.Authenticator,
	baseUserRepo *data.BaseUserRepository,
	authorizer authzEngine.Engine,
	userToken *authData.UserToken,
	jwtCfg *bootstrapConfigv1.Authentication_Jwt,
) HTTPMiddlewares {
	var ms HTTPMiddlewares
	cfg := ctx.GetConfig()
	// 先补齐请求标识，再进入访问日志中间件，确保日志能读取到统一 request_id。
	ms = append(ms, requestid.NewRequestIDMiddleware())
	// 开启日志中间件时，统一挂载请求日志与操作者解析逻辑。
	if cfg != nil && cfg.Server != nil && cfg.Server.Http != nil && cfg.Server.Http.Middleware != nil && cfg.Server.Http.Middleware.EnableLogging {
		ms = append(ms, logging.Server(ctx.GetLogger(), baseUserRepo, authenticator))
	}
	ms = append(ms, auth.NewAuthMiddleware(authenticator, authorizer, userToken, jwtCfg))
	return ms
}

// NewHTTPServer 创建 HTTP Server 并注册后端与前端静态路由。
func NewHTTPServer(
	ctx *bootstrap.Context,
	middlewares HTTPMiddlewares,
	services *ServerServices,
	mcpSvc *base.McpService,
	sseSvc *base.SseService,
) (*kratosHTTP.Server, error) {
	cfg := ctx.GetConfig()
	// 未启用 HTTP 配置时，跳过 HTTP 服务创建。
	if cfg == nil || cfg.Server == nil || cfg.Server.Http == nil {
		return nil, nil
	}

	srv, err := rpc.CreateHttpServer(cfg, middlewares...)
	if err != nil {
		return nil, err
	}

	adminv1.RegisterAuthServiceHTTPServer(srv, services.adminAuth)
	adminv1.RegisterBaseApiServiceHTTPServer(srv, services.adminBaseAPI)
	adminv1.RegisterBaseConfigServiceHTTPServer(srv, services.adminBaseConfig)
	adminv1.RegisterBaseDeptServiceHTTPServer(srv, services.adminBaseDept)
	adminv1.RegisterBaseDictServiceHTTPServer(srv, services.adminBaseDict)
	adminv1.RegisterBaseJobServiceHTTPServer(srv, services.adminBaseJob)
	adminv1.RegisterBaseLogServiceHTTPServer(srv, services.adminBaseLog)
	adminv1.RegisterBaseMenuServiceHTTPServer(srv, services.adminBaseMenu)
	adminv1.RegisterBaseRoleServiceHTTPServer(srv, services.adminBaseRole)
	adminv1.RegisterBaseUserServiceHTTPServer(srv, services.adminBaseUser)
	adminv1.RegisterCommentInfoServiceHTTPServer(srv, services.adminCommentInfo)
	adminv1.RegisterGoodsAnalyticsServiceHTTPServer(srv, services.adminGoodsAnalytics)
	adminv1.RegisterGoodsReportServiceHTTPServer(srv, services.adminGoodsReport)
	adminv1.RegisterGoodsCategoryServiceHTTPServer(srv, services.adminGoodsCategory)
	adminv1.RegisterGoodsPropServiceHTTPServer(srv, services.adminGoodsProp)
	adminv1.RegisterGoodsInfoServiceHTTPServer(srv, services.adminGoods)
	adminv1.RegisterGoodsSkuServiceHTTPServer(srv, services.adminGoodsSKU)
	adminv1.RegisterGoodsSpecServiceHTTPServer(srv, services.adminGoodsSpec)
	adminv1.RegisterOrderAnalyticsServiceHTTPServer(srv, services.adminOrderAnalytics)
	adminv1.RegisterOrderReportServiceHTTPServer(srv, services.adminOrderReport)
	adminv1.RegisterOrderInfoServiceHTTPServer(srv, services.adminOrder)
	adminv1.RegisterPayBillServiceHTTPServer(srv, services.adminPayBill)
	adminv1.RegisterRecommendRequestServiceHTTPServer(srv, services.adminRecommendRequest)
	adminv1.RegisterRecommendGorseServiceHTTPServer(srv, services.adminRecommendGorse)
	adminv1.RegisterShopBannerServiceHTTPServer(srv, services.adminShopBanner)
	adminv1.RegisterShopHotServiceHTTPServer(srv, services.adminShopHot)
	adminv1.RegisterShopServiceServiceHTTPServer(srv, services.adminShopService)
	adminv1.RegisterUserAnalyticsServiceHTTPServer(srv, services.adminUserAnalytics)
	adminv1.RegisterUserStoreServiceHTTPServer(srv, services.adminUserStore)
	adminv1.RegisterWorkspaceServiceHTTPServer(srv, services.adminWorkspace)

	appv1.RegisterAuthServiceHTTPServer(srv, services.appAuth)
	appv1.RegisterBaseAreaServiceHTTPServer(srv, services.appBaseArea)
	appv1.RegisterBaseDictServiceHTTPServer(srv, services.appBaseDict)
	appv1.RegisterCommentServiceHTTPServer(srv, services.appComment)
	appv1.RegisterGoodsCategoryServiceHTTPServer(srv, services.appGoodsCategory)
	appv1.RegisterGoodsInfoServiceHTTPServer(srv, services.appGoods)
	appv1.RegisterOrderInfoServiceHTTPServer(srv, services.appOrder)
	appv1.RegisterPayServiceHTTPServer(srv, services.appPay)
	appv1.RegisterRecommendServiceHTTPServer(srv, services.appRecommend)
	appv1.RegisterShopBannerServiceHTTPServer(srv, services.appShopBanner)
	appv1.RegisterShopHotServiceHTTPServer(srv, services.appShopHot)
	appv1.RegisterShopServiceServiceHTTPServer(srv, services.appShopService)
	appv1.RegisterUserAddressServiceHTTPServer(srv, services.appUserAddress)
	appv1.RegisterUserCartServiceHTTPServer(srv, services.appUserCart)
	appv1.RegisterUserCollectServiceHTTPServer(srv, services.appUserCollect)
	appv1.RegisterUserStoreServiceHTTPServer(srv, services.appUserStore)

	basev1.RegisterAiAssistantServiceHTTPServer(srv, services.aiAssistant)
	// AI 助手消息发送使用直连 SSE，避免占用工作台共用 /events 流。
	base.RegisterAiAssistantMessageServiceHTTPServer(srv, services.aiAssistantMessage)
	basev1.RegisterConfigServiceHTTPServer(srv, services.config)
	// 文件上传需要兼容 uni.uploadFile 的 multipart/form-data 请求，使用自定义 HTTP 适配器。
	base.RegisterFileServiceHTTPServer(srv, services.file)
	basev1.RegisterLoginServiceHTTPServer(srv, services.login)
	// MCP 需要保留 Streamable HTTP 的原始请求体和流式响应，使用自定义 HTTP 适配器。
	base.RegisterMcpServiceHTTPServer(srv, mcpSvc)

	// SSE 需要直接写入事件流响应，使用自定义 HTTP 适配器避免默认 JSON 响应。
	base.RegisterSseServiceHTTPServer(srv, sseSvc)

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
			swaggerUI.WithMemoryData(assets.OpenAPIData, "yaml"),
		)
	}

	return srv, nil
}

// registerLocalSPARoutes 扫描根目录下包含 index.html 的子目录，并按目录名注册单页应用路由。
func registerLocalSPARoutes(srv *kratosHTTP.Server, rootDirectory string) {
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

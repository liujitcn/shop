package server

import (
	"io/fs"
	stdhttp "net/http"
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
	kratosHTTP "github.com/go-kratos/kratos/v2/transport/http"
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
	if cfg != nil && cfg.Server != nil && cfg.Server.Http != nil && cfg.Server.Http.Middleware != nil && cfg.Server.Http.Middleware.EnableLogging {
		ms = append(ms, logging.Server(ctx.GetLogger(), baseUserRepo, authenticator))
	}
	ms = append(ms, auth.NewAuthMiddleware(authenticator, authorizer, userToken, jwtCfg))
	return ms
}

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
	adminDashboard *admin.DashboardService,
	adminGoodsCategory *admin.GoodsCategoryService,
	adminGoodsProp *admin.GoodsPropService,
	adminGoods *admin.GoodsService,
	adminGoodsSku *admin.GoodsSkuService,
	adminGoodsSpec *admin.GoodsSpecService,
	adminOrder *admin.OrderService,
	adminPayBill *admin.PayBillService,
	adminShopBanner *admin.ShopBannerService,
	adminShopHot *admin.ShopHotService,
	adminShopService *admin.ShopServiceService,
	adminUserStore *admin.UserStoreService,

	appAuth *app.AuthService,
	appBaseArea *app.BaseAreaService,
	appBaseDict *app.BaseDictService,
	appGoodsCategory *app.GoodsCategoryService,
	appGoods *app.GoodsService,
	appOrder *app.OrderService,
	appPay *app.PayService,
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
) (*kratosHTTP.Server, error) {
	cfg := ctx.GetConfig()
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
	adminApi.RegisterDashboardServiceHTTPServer(srv, adminDashboard)
	adminApi.RegisterGoodsCategoryServiceHTTPServer(srv, adminGoodsCategory)
	adminApi.RegisterGoodsPropServiceHTTPServer(srv, adminGoodsProp)
	adminApi.RegisterGoodsServiceHTTPServer(srv, adminGoods)
	adminApi.RegisterGoodsSkuServiceHTTPServer(srv, adminGoodsSku)
	adminApi.RegisterGoodsSpecServiceHTTPServer(srv, adminGoodsSpec)
	adminApi.RegisterOrderServiceHTTPServer(srv, adminOrder)
	adminApi.RegisterPayBillServiceHTTPServer(srv, adminPayBill)
	adminApi.RegisterShopBannerServiceHTTPServer(srv, adminShopBanner)
	adminApi.RegisterShopHotServiceHTTPServer(srv, adminShopHot)
	adminApi.RegisterShopServiceServiceHTTPServer(srv, adminShopService)
	adminApi.RegisterUserStoreServiceHTTPServer(srv, adminUserStore)

	appApi.RegisterAuthServiceHTTPServer(srv, appAuth)
	appApi.RegisterBaseAreaServiceHTTPServer(srv, appBaseArea)
	appApi.RegisterBaseDictServiceHTTPServer(srv, appBaseDict)
	appApi.RegisterGoodsCategoryServiceHTTPServer(srv, appGoodsCategory)
	appApi.RegisterGoodsServiceHTTPServer(srv, appGoods)
	appApi.RegisterOrderServiceHTTPServer(srv, appOrder)
	appApi.RegisterPayServiceHTTPServer(srv, appPay)
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
	if cfg.GetOss() != nil && cfg.GetOss().GetRootDirectory() != "" {
		ossRootDirectory = cfg.GetOss().GetRootDirectory()
	}
	ossRootDirectory = filepath.Join(ossRootDirectory, "shop")
	// 将本地 OSS 目录暴露为静态资源目录 访问 /shop/* 时直接映射到 ./data/shop/*
	staticHandler := stdhttp.StripPrefix("/shop/", stdhttp.FileServer(stdhttp.Dir(ossRootDirectory)))
	srv.HandlePrefix("/shop/", staticHandler)

	if webFS, subErr := fs.Sub(assets.WebAssets, "web"); subErr == nil {
		webHandler := stdhttp.FileServer(stdhttp.FS(webFS))
		srv.HandlePrefix("/web/", stdhttp.StripPrefix("/web/", webHandler))
		srv.HandlePrefix("/assets/", webHandler)
		srv.HandlePrefix("/static/", webHandler)
		srv.Handle("/favicon.ico", webHandler)
		srv.HandleFunc("/", func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
			if r.URL.Path == "/" {
				stdhttp.ServeFileFS(w, r, webFS, "index.html")
				return
			}
			stdhttp.NotFound(w, r)
		})
	}

	if cfg.GetServer().GetHttp().GetEnableSwagger() {
		swaggerUI.RegisterSwaggerUIServerWithOption(
			srv,
			swaggerUI.WithTitle(ctx.GetAppInfo().GetName()),
			swaggerUI.WithMemoryData(assets.OpenApiData, "yaml"),
		)
	}

	return srv, nil
}

// newEmbeddedSPAHandler 创建基于嵌入式文件系统的单页应用处理器。
func newEmbeddedSPAHandler(webFS fs.FS, urlPrefix string) stdhttp.Handler {
	var fileHandler = stdhttp.StripPrefix(urlPrefix, stdhttp.FileServer(stdhttp.FS(webFS)))
	return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		var relativePath = strings.TrimPrefix(r.URL.Path, urlPrefix)
		relativePath = strings.TrimPrefix(relativePath, "/")
		if relativePath == "" {
			stdhttp.ServeFileFS(w, r, webFS, "index.html")
			return
		}
		if _, err := fs.Stat(webFS, relativePath); err == nil {
			fileHandler.ServeHTTP(w, r)
			return
		}
		// 前端路由命中不到真实文件时，统一回退到入口页面。
		stdhttp.ServeFileFS(w, r, webFS, "index.html")
	})
}

package server

import (
	adminApi "shop/api/gen/go/admin"
	appApi "shop/api/gen/go/app"
	baseApi "shop/api/gen/go/base"
	"shop/pkg/gen/data"
	"shop/pkg/middleware/logging"
	"shop/service/admin"
	"shop/service/app"
	"shop/service/base"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	bootstrapConf "github.com/liujitcn/kratos-kit/api/gen/go/conf"
	"github.com/liujitcn/kratos-kit/auth"
	authnEngine "github.com/liujitcn/kratos-kit/auth/authn/engine"
	authzEngine "github.com/liujitcn/kratos-kit/auth/authz/engine"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/bootstrap"
	"github.com/liujitcn/kratos-kit/rpc"
)

type GrpcMiddlewares []middleware.Middleware

func NewGrpcMiddleware(
	ctx *bootstrap.Context,
	authenticator authnEngine.Authenticator,
	baseUserRepo *data.BaseUserRepo,
	authorizer authzEngine.Engine,
	userToken *authData.UserToken,
	jwtCfg *bootstrapConf.Authentication_Jwt,
) GrpcMiddlewares {
	var ms GrpcMiddlewares
	cfg := ctx.GetConfig()
	if cfg != nil && cfg.Server != nil && cfg.Server.Grpc != nil && cfg.Server.Grpc.Middleware != nil && cfg.Server.Grpc.Middleware.EnableLogging {
		ms = append(ms, logging.Server(ctx.GetLogger(), baseUserRepo, authenticator))
	}
	ms = append(ms, auth.NewAuthMiddleware(authenticator, authorizer, userToken, jwtCfg))
	return ms
}

func NewGRPCServer(
	ctx *bootstrap.Context,
	middlewares GrpcMiddlewares,

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
	adminGoodsCategory *admin.GoodsCategoryService,
	adminGoodsProp *admin.GoodsPropService,
	adminGoods *admin.GoodsService,
	adminGoodsSku *admin.GoodsSkuService,
	adminGoodsSpec *admin.GoodsSpecService,
	adminOrderAnalytics *admin.OrderAnalyticsService,
	adminOrder *admin.OrderService,
	adminPayBill *admin.PayBillService,
	adminShopBanner *admin.ShopBannerService,
	adminShopHot *admin.ShopHotService,
	adminShopService *admin.ShopServiceService,
	adminUserAnalytics *admin.UserAnalyticsService,
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
	file *base.FileService,
	login *base.LoginService,
) (*grpc.Server, error) {
	cfg := ctx.GetConfig()
	if cfg == nil || cfg.Server == nil || cfg.Server.Grpc == nil {
		return nil, nil
	}

	srv, err := rpc.CreateGrpcServer(cfg, middlewares...)
	if err != nil {
		return nil, err
	}
	adminApi.RegisterAuthServiceServer(srv, adminAuth)
	adminApi.RegisterBaseApiServiceServer(srv, adminBaseApi)
	adminApi.RegisterBaseConfigServiceServer(srv, adminBaseConfig)
	adminApi.RegisterBaseDeptServiceServer(srv, adminBaseDept)
	adminApi.RegisterBaseDictServiceServer(srv, adminBaseDict)
	adminApi.RegisterBaseJobServiceServer(srv, adminBaseJob)
	adminApi.RegisterBaseLogServiceServer(srv, adminBaseLog)
	adminApi.RegisterBaseMenuServiceServer(srv, adminBaseMenu)
	adminApi.RegisterBaseRoleServiceServer(srv, adminBaseRole)
	adminApi.RegisterBaseUserServiceServer(srv, adminBaseUser)
	adminApi.RegisterGoodsAnalyticsServiceServer(srv, adminGoodsAnalytics)
	adminApi.RegisterGoodsCategoryServiceServer(srv, adminGoodsCategory)
	adminApi.RegisterGoodsPropServiceServer(srv, adminGoodsProp)
	adminApi.RegisterGoodsServiceServer(srv, adminGoods)
	adminApi.RegisterGoodsSkuServiceServer(srv, adminGoodsSku)
	adminApi.RegisterGoodsSpecServiceServer(srv, adminGoodsSpec)
	adminApi.RegisterOrderAnalyticsServiceServer(srv, adminOrderAnalytics)
	adminApi.RegisterOrderServiceServer(srv, adminOrder)
	adminApi.RegisterPayBillServiceServer(srv, adminPayBill)
	adminApi.RegisterShopBannerServiceServer(srv, adminShopBanner)
	adminApi.RegisterShopHotServiceServer(srv, adminShopHot)
	adminApi.RegisterShopServiceServiceServer(srv, adminShopService)
	adminApi.RegisterUserAnalyticsServiceServer(srv, adminUserAnalytics)
	adminApi.RegisterUserStoreServiceServer(srv, adminUserStore)

	appApi.RegisterAuthServiceServer(srv, appAuth)
	appApi.RegisterBaseAreaServiceServer(srv, appBaseArea)
	appApi.RegisterBaseDictServiceServer(srv, appBaseDict)
	appApi.RegisterGoodsCategoryServiceServer(srv, appGoodsCategory)
	appApi.RegisterGoodsServiceServer(srv, appGoods)
	appApi.RegisterOrderServiceServer(srv, appOrder)
	appApi.RegisterPayServiceServer(srv, appPay)
	appApi.RegisterShopBannerServiceServer(srv, appShopBanner)
	appApi.RegisterShopHotServiceServer(srv, appShopHot)
	appApi.RegisterShopServiceServiceServer(srv, appShopService)
	appApi.RegisterUserAddressServiceServer(srv, appUserAddress)
	appApi.RegisterUserCartServiceServer(srv, appUserCart)
	appApi.RegisterUserCollectServiceServer(srv, appUserCollect)
	appApi.RegisterUserStoreServiceServer(srv, appUserStore)

	baseApi.RegisterConfigServiceServer(srv, config)
	baseApi.RegisterFileServiceServer(srv, file)
	baseApi.RegisterLoginServiceServer(srv, login)

	return srv, nil
}

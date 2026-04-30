package server

import (
	adminv1 "shop/api/gen/go/admin/v1"
	appv1 "shop/api/gen/go/app/v1"
	basev1 "shop/api/gen/go/base/v1"
	"shop/pkg/gen/data"
	"shop/pkg/middleware/logging"
	"shop/service/admin"
	"shop/service/app"
	"shop/service/base"

	bootstrapConfigv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/liujitcn/kratos-kit/auth"
	authnEngine "github.com/liujitcn/kratos-kit/auth/authn/engine"
	authzEngine "github.com/liujitcn/kratos-kit/auth/authz/engine"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/bootstrap"
	"github.com/liujitcn/kratos-kit/rpc"
	"github.com/liujitcn/kratos-kit/rpc/middleware/requestid"
)

type GRPCMiddlewares []middleware.Middleware

// NewGRPCMiddleware 创建 GRPC 服务统一中间件链。
func NewGRPCMiddleware(
	ctx *bootstrap.Context,
	authenticator authnEngine.Authenticator,
	baseUserRepo *data.BaseUserRepository,
	authorizer authzEngine.Engine,
	userToken *authData.UserToken,
	jwtCfg *bootstrapConfigv1.Authentication_Jwt,
) GRPCMiddlewares {
	var ms GRPCMiddlewares
	cfg := ctx.GetConfig()
	// 先补齐请求标识，再进入访问日志中间件，确保日志能读取到统一 request_id。
	ms = append(ms, requestid.NewRequestIDMiddleware())
	// 开启日志中间件时，统一挂载请求日志与操作者解析逻辑。
	if cfg != nil && cfg.Server != nil && cfg.Server.Grpc != nil && cfg.Server.Grpc.Middleware != nil && cfg.Server.Grpc.Middleware.EnableLogging {
		ms = append(ms, logging.Server(ctx.GetLogger(), baseUserRepo, authenticator))
	}
	ms = append(ms, auth.NewAuthMiddleware(authenticator, authorizer, userToken, jwtCfg))
	return ms
}

// NewGRPCServer 创建 GRPC Server 并注册全部业务服务。
func NewGRPCServer(
	ctx *bootstrap.Context,
	middlewares GRPCMiddlewares,

	adminAuth *admin.AuthService,
	adminBaseAPI *admin.BaseApiService,
	adminBaseConfig *admin.BaseConfigService,
	adminBaseDept *admin.BaseDeptService,
	adminBaseDict *admin.BaseDictService,
	adminBaseJob *admin.BaseJobService,
	adminBaseLog *admin.BaseLogService,
	adminBaseMenu *admin.BaseMenuService,
	adminBaseRole *admin.BaseRoleService,
	adminBaseUser *admin.BaseUserService,
	adminCommentInfo *admin.CommentInfoService,
	adminGoodsAnalytics *admin.GoodsAnalyticsService,
	adminGoodsReport *admin.GoodsReportService,
	adminGoodsCategory *admin.GoodsCategoryService,
	adminGoodsProp *admin.GoodsPropService,
	adminGoods *admin.GoodsInfoService,
	adminGoodsSKU *admin.GoodsSkuService,
	adminGoodsSpec *admin.GoodsSpecService,
	adminOrderAnalytics *admin.OrderAnalyticsService,
	adminOrderReport *admin.OrderReportService,
	adminOrder *admin.OrderInfoService,
	adminPayBill *admin.PayBillService,
	adminRecommendRequest *admin.RecommendRequestService,
	adminRecommendGorse *admin.RecommendGorseService,
	adminShopBanner *admin.ShopBannerService,
	adminShopHot *admin.ShopHotService,
	adminShopService *admin.ShopServiceService,
	adminUserAnalytics *admin.UserAnalyticsService,
	adminUserStore *admin.UserStoreService,
	adminWorkspace *admin.WorkspaceService,

	appAuth *app.AuthService,
	appBaseArea *app.BaseAreaService,
	appBaseDict *app.BaseDictService,
	appComment *app.CommentService,
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
	file *base.FileService,
	login *base.LoginService,
) (*grpc.Server, error) {
	cfg := ctx.GetConfig()
	// 未启用 GRPC 配置时，跳过 GRPC 服务创建。
	if cfg == nil || cfg.Server == nil || cfg.Server.Grpc == nil {
		return nil, nil
	}

	srv, err := rpc.CreateGrpcServer(cfg, middlewares...)
	if err != nil {
		return nil, err
	}
	adminv1.RegisterAuthServiceServer(srv, adminAuth)
	adminv1.RegisterBaseApiServiceServer(srv, adminBaseAPI)
	adminv1.RegisterBaseConfigServiceServer(srv, adminBaseConfig)
	adminv1.RegisterBaseDeptServiceServer(srv, adminBaseDept)
	adminv1.RegisterBaseDictServiceServer(srv, adminBaseDict)
	adminv1.RegisterBaseJobServiceServer(srv, adminBaseJob)
	adminv1.RegisterBaseLogServiceServer(srv, adminBaseLog)
	adminv1.RegisterBaseMenuServiceServer(srv, adminBaseMenu)
	adminv1.RegisterBaseRoleServiceServer(srv, adminBaseRole)
	adminv1.RegisterBaseUserServiceServer(srv, adminBaseUser)
	adminv1.RegisterCommentInfoServiceServer(srv, adminCommentInfo)
	adminv1.RegisterGoodsAnalyticsServiceServer(srv, adminGoodsAnalytics)
	adminv1.RegisterGoodsReportServiceServer(srv, adminGoodsReport)
	adminv1.RegisterGoodsCategoryServiceServer(srv, adminGoodsCategory)
	adminv1.RegisterGoodsPropServiceServer(srv, adminGoodsProp)
	adminv1.RegisterGoodsInfoServiceServer(srv, adminGoods)
	adminv1.RegisterGoodsSkuServiceServer(srv, adminGoodsSKU)
	adminv1.RegisterGoodsSpecServiceServer(srv, adminGoodsSpec)
	adminv1.RegisterOrderAnalyticsServiceServer(srv, adminOrderAnalytics)
	adminv1.RegisterOrderReportServiceServer(srv, adminOrderReport)
	adminv1.RegisterOrderInfoServiceServer(srv, adminOrder)
	adminv1.RegisterPayBillServiceServer(srv, adminPayBill)
	adminv1.RegisterRecommendRequestServiceServer(srv, adminRecommendRequest)
	adminv1.RegisterRecommendGorseServiceServer(srv, adminRecommendGorse)
	adminv1.RegisterShopBannerServiceServer(srv, adminShopBanner)
	adminv1.RegisterShopHotServiceServer(srv, adminShopHot)
	adminv1.RegisterShopServiceServiceServer(srv, adminShopService)
	adminv1.RegisterUserAnalyticsServiceServer(srv, adminUserAnalytics)
	adminv1.RegisterUserStoreServiceServer(srv, adminUserStore)
	adminv1.RegisterWorkspaceServiceServer(srv, adminWorkspace)

	appv1.RegisterAuthServiceServer(srv, appAuth)
	appv1.RegisterBaseAreaServiceServer(srv, appBaseArea)
	appv1.RegisterBaseDictServiceServer(srv, appBaseDict)
	appv1.RegisterCommentServiceServer(srv, appComment)
	appv1.RegisterGoodsCategoryServiceServer(srv, appGoodsCategory)
	appv1.RegisterGoodsInfoServiceServer(srv, appGoods)
	appv1.RegisterOrderInfoServiceServer(srv, appOrder)
	appv1.RegisterPayServiceServer(srv, appPay)
	appv1.RegisterRecommendServiceServer(srv, appRecommend)
	appv1.RegisterShopBannerServiceServer(srv, appShopBanner)
	appv1.RegisterShopHotServiceServer(srv, appShopHot)
	appv1.RegisterShopServiceServiceServer(srv, appShopService)
	appv1.RegisterUserAddressServiceServer(srv, appUserAddress)
	appv1.RegisterUserCartServiceServer(srv, appUserCart)
	appv1.RegisterUserCollectServiceServer(srv, appUserCollect)
	appv1.RegisterUserStoreServiceServer(srv, appUserStore)

	basev1.RegisterConfigServiceServer(srv, config)
	basev1.RegisterFileServiceServer(srv, file)
	basev1.RegisterLoginServiceServer(srv, login)

	return srv, nil
}

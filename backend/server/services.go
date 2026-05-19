package server

import (
	"shop/service/admin"
	"shop/service/app"
	"shop/service/base"
)

// ServerServices 汇总 HTTP 与 MCP 需要注册的服务实例。
type ServerServices struct {
	adminAuth             *admin.AuthService
	adminBaseAPI          *admin.BaseApiService
	adminBaseConfig       *admin.BaseConfigService
	adminBaseDept         *admin.BaseDeptService
	adminBaseDict         *admin.BaseDictService
	adminBaseJob          *admin.BaseJobService
	adminBaseLog          *admin.BaseLogService
	adminBaseMenu         *admin.BaseMenuService
	adminBaseRole         *admin.BaseRoleService
	adminBaseUser         *admin.BaseUserService
	adminCommentInfo      *admin.CommentInfoService
	adminGoodsAnalytics   *admin.GoodsAnalyticsService
	adminGoodsReport      *admin.GoodsReportService
	adminGoodsCategory    *admin.GoodsCategoryService
	adminGoodsProp        *admin.GoodsPropService
	adminGoods            *admin.GoodsInfoService
	adminGoodsSKU         *admin.GoodsSkuService
	adminGoodsSpec        *admin.GoodsSpecService
	adminOrderAnalytics   *admin.OrderAnalyticsService
	adminOrderReport      *admin.OrderReportService
	adminOrder            *admin.OrderInfoService
	adminPayBill          *admin.PayBillService
	adminRecommendRequest *admin.RecommendRequestService
	adminRecommendGorse   *admin.RecommendGorseService
	adminShopBanner       *admin.ShopBannerService
	adminShopHot          *admin.ShopHotService
	adminShopService      *admin.ShopServiceService
	adminUserAnalytics    *admin.UserAnalyticsService
	adminUserStore        *admin.UserStoreService
	adminWorkspace        *admin.WorkspaceService

	appAuth          *app.AuthService
	appBaseArea      *app.BaseAreaService
	appBaseDict      *app.BaseDictService
	appComment       *app.CommentService
	appGoodsCategory *app.GoodsCategoryService
	appGoods         *app.GoodsInfoService
	appOrder         *app.OrderInfoService
	appPay           *app.PayService
	appRecommend     *app.RecommendService
	appShopBanner    *app.ShopBannerService
	appShopHot       *app.ShopHotService
	appShopService   *app.ShopServiceService
	appUserAddress   *app.UserAddressService
	appUserCart      *app.UserCartService
	appUserCollect   *app.UserCollectService
	appUserStore     *app.UserStoreService

	aiAssistant        *base.AiAssistantService
	aiAssistantMessage *base.AiAssistantMessageService
	aiImage            *base.AiImageService
	config             *base.ConfigService
	file               *base.FileService
	login              *base.LoginService
}

// NewServerServices 创建 HTTP 与 MCP 服务注册表。
func NewServerServices(
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

	aiAssistant *base.AiAssistantService,
	aiAssistantMessage *base.AiAssistantMessageService,
	aiImage *base.AiImageService,
	config *base.ConfigService,
	file *base.FileService,
	login *base.LoginService,
) *ServerServices {
	return &ServerServices{
		adminAuth:             adminAuth,
		adminBaseAPI:          adminBaseAPI,
		adminBaseConfig:       adminBaseConfig,
		adminBaseDept:         adminBaseDept,
		adminBaseDict:         adminBaseDict,
		adminBaseJob:          adminBaseJob,
		adminBaseLog:          adminBaseLog,
		adminBaseMenu:         adminBaseMenu,
		adminBaseRole:         adminBaseRole,
		adminBaseUser:         adminBaseUser,
		adminCommentInfo:      adminCommentInfo,
		adminGoodsAnalytics:   adminGoodsAnalytics,
		adminGoodsReport:      adminGoodsReport,
		adminGoodsCategory:    adminGoodsCategory,
		adminGoodsProp:        adminGoodsProp,
		adminGoods:            adminGoods,
		adminGoodsSKU:         adminGoodsSKU,
		adminGoodsSpec:        adminGoodsSpec,
		adminOrderAnalytics:   adminOrderAnalytics,
		adminOrderReport:      adminOrderReport,
		adminOrder:            adminOrder,
		adminPayBill:          adminPayBill,
		adminRecommendRequest: adminRecommendRequest,
		adminRecommendGorse:   adminRecommendGorse,
		adminShopBanner:       adminShopBanner,
		adminShopHot:          adminShopHot,
		adminShopService:      adminShopService,
		adminUserAnalytics:    adminUserAnalytics,
		adminUserStore:        adminUserStore,
		adminWorkspace:        adminWorkspace,
		appAuth:               appAuth,
		appBaseArea:           appBaseArea,
		appBaseDict:           appBaseDict,
		appComment:            appComment,
		appGoodsCategory:      appGoodsCategory,
		appGoods:              appGoods,
		appOrder:              appOrder,
		appPay:                appPay,
		appRecommend:          appRecommend,
		appShopBanner:         appShopBanner,
		appShopHot:            appShopHot,
		appShopService:        appShopService,
		appUserAddress:        appUserAddress,
		appUserCart:           appUserCart,
		appUserCollect:        appUserCollect,
		appUserStore:          appUserStore,
		aiAssistant:           aiAssistant,
		aiAssistantMessage:    aiAssistantMessage,
		aiImage:               aiImage,
		config:                config,
		file:                  file,
		login:                 login,
	}
}

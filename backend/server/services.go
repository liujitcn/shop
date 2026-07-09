package server

import (
	"shop/pkg/agent/assistant"
	einoTool "shop/pkg/agent/eino/tool"
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
	adminBaseTenant       *admin.BaseTenantService
	adminBaseUser         *admin.BaseUserService
	adminCommentInfo      *admin.CommentInfoService
	adminTenantStore      *admin.TenantStoreService
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
	appTenantStore   *app.TenantStoreService
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
	config             *base.ConfigService
	file               *base.FileService
	login              *base.LoginService
	oauth              *base.OauthService
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
	adminBaseTenant *admin.BaseTenantService,
	adminBaseUser *admin.BaseUserService,
	adminCommentInfo *admin.CommentInfoService,
	adminTenantStore *admin.TenantStoreService,
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
	appTenantStore *app.TenantStoreService,
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

	assistantRuntime *assistant.Runtime,
	aiAssistant *base.AiAssistantService,
	aiAssistantMessage *base.AiAssistantMessageService,
	config *base.ConfigService,
	file *base.FileService,
	login *base.LoginService,
	oauth *base.OauthService,
) (*ServerServices, error) {
	services := &ServerServices{
		adminAuth:             adminAuth,
		adminBaseAPI:          adminBaseAPI,
		adminBaseConfig:       adminBaseConfig,
		adminBaseDept:         adminBaseDept,
		adminBaseDict:         adminBaseDict,
		adminBaseJob:          adminBaseJob,
		adminBaseLog:          adminBaseLog,
		adminBaseMenu:         adminBaseMenu,
		adminBaseRole:         adminBaseRole,
		adminBaseTenant:       adminBaseTenant,
		adminBaseUser:         adminBaseUser,
		adminCommentInfo:      adminCommentInfo,
		adminTenantStore:      adminTenantStore,
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
		appTenantStore:        appTenantStore,
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
		config:                config,
		file:                  file,
		login:                 login,
		oauth:                 oauth,
	}
	var err error
	var adminTools []einoTool.Invokable
	adminTools, err = newAdminFlowAgentTools(services)
	if err != nil {
		return nil, err
	}
	var appTools []einoTool.Invokable
	appTools, err = newAppFlowAgentTools(services)
	if err != nil {
		return nil, err
	}
	assistantRuntime.SetTerminalTools(adminTools, appTools)
	return services, nil
}

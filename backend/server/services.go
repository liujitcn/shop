package server

import (
	"strings"

	adminv1 "shop/api/gen/go/admin/v1"
	appv1 "shop/api/gen/go/app/v1"
	basev1 "shop/api/gen/go/base/v1"
	"shop/pkg/agent/assistant"
	"shop/service/admin"
	"shop/service/app"
	"shop/service/base"

	"github.com/go-kratos/blades/tools"
)

const (
	// maxAiAssistantAgentTools 限制单次模型请求携带的内部工具数量，避免代理因工具定义过大返回 503。
	maxAiAssistantAgentTools = 64
	// maxAiAssistantAgentToolNameLength 对齐 Responses function tool 的工具名长度限制。
	maxAiAssistantAgentToolNameLength = 64
)

var aiAssistantReadToolPrefixes = []string{
	"get_",
	"list_",
	"page_",
	"tree_",
	"option_",
	"summary_",
	"trend_",
	"rank_",
	"count_",
}

var aiAssistantBlockedToolNameParts = []string{
	"ai_assistant",
	"auth_service",
	"file_service",
	"login_service",
	"mcp_service",
	"pay_service",
}

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

	assistantRuntime *assistant.Runtime,
	aiAssistant *base.AiAssistantService,
	aiAssistantMessage *base.AiAssistantMessageService,
	config *base.ConfigService,
	file *base.FileService,
	login *base.LoginService,
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
		config:                config,
		file:                  file,
		login:                 login,
	}
	agentTools := make([]tools.Tool, 0, 128)
	collectAgentTools := func(values []tools.Tool, err error) error {
		if err != nil {
			return err
		}
		agentTools = append(agentTools, values...)
		return nil
	}

	if err := collectAgentTools(adminv1.NewAuthServiceAgentTools(services.adminAuth)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(adminv1.NewBaseApiServiceAgentTools(services.adminBaseAPI)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(adminv1.NewBaseConfigServiceAgentTools(services.adminBaseConfig)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(adminv1.NewBaseDeptServiceAgentTools(services.adminBaseDept)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(adminv1.NewBaseDictServiceAgentTools(services.adminBaseDict)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(adminv1.NewBaseJobServiceAgentTools(services.adminBaseJob)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(adminv1.NewBaseLogServiceAgentTools(services.adminBaseLog)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(adminv1.NewBaseMenuServiceAgentTools(services.adminBaseMenu)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(adminv1.NewBaseRoleServiceAgentTools(services.adminBaseRole)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(adminv1.NewBaseUserServiceAgentTools(services.adminBaseUser)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(adminv1.NewCommentInfoServiceAgentTools(services.adminCommentInfo)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(adminv1.NewGoodsAnalyticsServiceAgentTools(services.adminGoodsAnalytics)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(adminv1.NewGoodsReportServiceAgentTools(services.adminGoodsReport)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(adminv1.NewGoodsCategoryServiceAgentTools(services.adminGoodsCategory)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(adminv1.NewGoodsPropServiceAgentTools(services.adminGoodsProp)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(adminv1.NewGoodsInfoServiceAgentTools(services.adminGoods)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(adminv1.NewGoodsSkuServiceAgentTools(services.adminGoodsSKU)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(adminv1.NewGoodsSpecServiceAgentTools(services.adminGoodsSpec)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(adminv1.NewOrderAnalyticsServiceAgentTools(services.adminOrderAnalytics)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(adminv1.NewOrderReportServiceAgentTools(services.adminOrderReport)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(adminv1.NewOrderInfoServiceAgentTools(services.adminOrder)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(adminv1.NewPayBillServiceAgentTools(services.adminPayBill)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(adminv1.NewRecommendRequestServiceAgentTools(services.adminRecommendRequest)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(adminv1.NewRecommendGorseServiceAgentTools(services.adminRecommendGorse)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(adminv1.NewShopBannerServiceAgentTools(services.adminShopBanner)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(adminv1.NewShopHotServiceAgentTools(services.adminShopHot)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(adminv1.NewShopServiceServiceAgentTools(services.adminShopService)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(adminv1.NewUserAnalyticsServiceAgentTools(services.adminUserAnalytics)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(adminv1.NewUserStoreServiceAgentTools(services.adminUserStore)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(adminv1.NewWorkspaceServiceAgentTools(services.adminWorkspace)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(appv1.NewAuthServiceAgentTools(services.appAuth)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(appv1.NewBaseAreaServiceAgentTools(services.appBaseArea)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(appv1.NewBaseDictServiceAgentTools(services.appBaseDict)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(appv1.NewCommentServiceAgentTools(services.appComment)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(appv1.NewGoodsCategoryServiceAgentTools(services.appGoodsCategory)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(appv1.NewGoodsInfoServiceAgentTools(services.appGoods)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(appv1.NewOrderInfoServiceAgentTools(services.appOrder)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(appv1.NewPayServiceAgentTools(services.appPay)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(appv1.NewRecommendServiceAgentTools(services.appRecommend)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(appv1.NewShopBannerServiceAgentTools(services.appShopBanner)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(appv1.NewShopHotServiceAgentTools(services.appShopHot)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(appv1.NewShopServiceServiceAgentTools(services.appShopService)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(appv1.NewUserAddressServiceAgentTools(services.appUserAddress)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(appv1.NewUserCartServiceAgentTools(services.appUserCart)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(appv1.NewUserCollectServiceAgentTools(services.appUserCollect)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(appv1.NewUserStoreServiceAgentTools(services.appUserStore)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(basev1.NewConfigServiceAgentTools(services.config)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(basev1.NewFileServiceAgentTools(services.file)); err != nil {
		return nil, err
	}
	if err := collectAgentTools(basev1.NewLoginServiceAgentTools(services.login)); err != nil {
		return nil, err
	}
	assistantRuntime.SetTools(filterAiAssistantAgentTools(agentTools))
	return services, nil
}

// filterAiAssistantAgentTools 筛选 AI 助手可挂载的安全工具集合。
func filterAiAssistantAgentTools(values []tools.Tool) []tools.Tool {
	result := make([]tools.Tool, 0, maxAiAssistantAgentTools)
	seen := make(map[string]struct{}, len(values))
	for _, item := range values {
		if !isAiAssistantAgentToolAllowed(item) {
			continue
		}
		name := item.Name()
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		result = append(result, item)
		if len(result) >= maxAiAssistantAgentTools {
			return result
		}
	}
	return result
}

// isAiAssistantAgentToolAllowed 判断工具是否适合作为 AI 助手内部工具。
func isAiAssistantAgentToolAllowed(item tools.Tool) bool {
	if item == nil {
		return false
	}
	name := item.Name()
	if name == "" || len(name) > maxAiAssistantAgentToolNameLength {
		return false
	}
	if isAiAssistantBlockedTool(name) {
		return false
	}
	return hasAiAssistantReadToolPrefix(resolveAiAssistantToolMethod(name))
}

// isAiAssistantBlockedTool 判断工具是否属于不应暴露给 AI 助手的服务能力。
func isAiAssistantBlockedTool(name string) bool {
	for _, part := range aiAssistantBlockedToolNameParts {
		if strings.Contains(name, part) {
			return true
		}
	}
	return false
}

// resolveAiAssistantToolMethod 从生成工具名中提取 RPC 方法语义片段。
func resolveAiAssistantToolMethod(name string) string {
	separator := "_service_"
	index := strings.LastIndex(name, separator)
	if index < 0 {
		return name
	}
	return name[index+len(separator):]
}

// hasAiAssistantReadToolPrefix 判断方法名是否是只读类工具。
func hasAiAssistantReadToolPrefix(method string) bool {
	for _, prefix := range aiAssistantReadToolPrefixes {
		if strings.HasPrefix(method, prefix) {
			return true
		}
	}
	return false
}

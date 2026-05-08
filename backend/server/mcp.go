package server

import (
	adminv1 "shop/api/gen/go/admin/v1"
	appv1 "shop/api/gen/go/app/v1"
	basev1 "shop/api/gen/go/base/v1"

	"github.com/liujitcn/kratos-kit/bootstrap"
	"github.com/liujitcn/kratos-kit/rpc"
	mcpserver "github.com/liujitcn/kratos-kit/transport/mcp"
)

// NewMCPHandler 创建进程内 MCP 服务。
func NewMCPHandler(ctx *bootstrap.Context, services *ServerServices) (*mcpserver.Server, error) {
	mcpSrv, err := rpc.CreateMcpHandler(ctx.GetConfig())
	if err != nil {
		return nil, err
	}
	registerMCPTools(mcpSrv, services)
	return mcpSrv, nil
}

// registerMCPTools 注册本地服务 MCP 工具。
func registerMCPTools(mcpSrv *mcpserver.Server, services *ServerServices) {
	mcpServer := mcpSrv.MCPServer()
	adminv1.RegisterAuthServiceMCPTools(mcpServer, services.adminAuth)
	adminv1.RegisterBaseApiServiceMCPTools(mcpServer, services.adminBaseAPI)
	adminv1.RegisterBaseConfigServiceMCPTools(mcpServer, services.adminBaseConfig)
	adminv1.RegisterBaseDeptServiceMCPTools(mcpServer, services.adminBaseDept)
	adminv1.RegisterBaseDictServiceMCPTools(mcpServer, services.adminBaseDict)
	adminv1.RegisterBaseJobServiceMCPTools(mcpServer, services.adminBaseJob)
	adminv1.RegisterBaseLogServiceMCPTools(mcpServer, services.adminBaseLog)
	adminv1.RegisterBaseMenuServiceMCPTools(mcpServer, services.adminBaseMenu)
	adminv1.RegisterBaseRoleServiceMCPTools(mcpServer, services.adminBaseRole)
	adminv1.RegisterBaseUserServiceMCPTools(mcpServer, services.adminBaseUser)
	adminv1.RegisterCommentInfoServiceMCPTools(mcpServer, services.adminCommentInfo)
	adminv1.RegisterGoodsAnalyticsServiceMCPTools(mcpServer, services.adminGoodsAnalytics)
	adminv1.RegisterGoodsReportServiceMCPTools(mcpServer, services.adminGoodsReport)
	adminv1.RegisterGoodsCategoryServiceMCPTools(mcpServer, services.adminGoodsCategory)
	adminv1.RegisterGoodsPropServiceMCPTools(mcpServer, services.adminGoodsProp)
	adminv1.RegisterGoodsInfoServiceMCPTools(mcpServer, services.adminGoods)
	adminv1.RegisterGoodsSkuServiceMCPTools(mcpServer, services.adminGoodsSKU)
	adminv1.RegisterGoodsSpecServiceMCPTools(mcpServer, services.adminGoodsSpec)
	adminv1.RegisterOrderAnalyticsServiceMCPTools(mcpServer, services.adminOrderAnalytics)
	adminv1.RegisterOrderReportServiceMCPTools(mcpServer, services.adminOrderReport)
	adminv1.RegisterOrderInfoServiceMCPTools(mcpServer, services.adminOrder)
	adminv1.RegisterPayBillServiceMCPTools(mcpServer, services.adminPayBill)
	adminv1.RegisterRecommendRequestServiceMCPTools(mcpServer, services.adminRecommendRequest)
	adminv1.RegisterRecommendGorseServiceMCPTools(mcpServer, services.adminRecommendGorse)
	adminv1.RegisterShopBannerServiceMCPTools(mcpServer, services.adminShopBanner)
	adminv1.RegisterShopHotServiceMCPTools(mcpServer, services.adminShopHot)
	adminv1.RegisterShopServiceServiceMCPTools(mcpServer, services.adminShopService)
	adminv1.RegisterUserAnalyticsServiceMCPTools(mcpServer, services.adminUserAnalytics)
	adminv1.RegisterUserStoreServiceMCPTools(mcpServer, services.adminUserStore)
	adminv1.RegisterWorkspaceServiceMCPTools(mcpServer, services.adminWorkspace)

	appv1.RegisterAuthServiceMCPTools(mcpServer, services.appAuth)
	appv1.RegisterBaseAreaServiceMCPTools(mcpServer, services.appBaseArea)
	appv1.RegisterBaseDictServiceMCPTools(mcpServer, services.appBaseDict)
	appv1.RegisterCommentServiceMCPTools(mcpServer, services.appComment)
	appv1.RegisterGoodsCategoryServiceMCPTools(mcpServer, services.appGoodsCategory)
	appv1.RegisterGoodsInfoServiceMCPTools(mcpServer, services.appGoods)
	appv1.RegisterOrderInfoServiceMCPTools(mcpServer, services.appOrder)
	appv1.RegisterPayServiceMCPTools(mcpServer, services.appPay)
	appv1.RegisterRecommendServiceMCPTools(mcpServer, services.appRecommend)
	appv1.RegisterShopBannerServiceMCPTools(mcpServer, services.appShopBanner)
	appv1.RegisterShopHotServiceMCPTools(mcpServer, services.appShopHot)
	appv1.RegisterShopServiceServiceMCPTools(mcpServer, services.appShopService)
	appv1.RegisterUserAddressServiceMCPTools(mcpServer, services.appUserAddress)
	appv1.RegisterUserCartServiceMCPTools(mcpServer, services.appUserCart)
	appv1.RegisterUserCollectServiceMCPTools(mcpServer, services.appUserCollect)
	appv1.RegisterUserStoreServiceMCPTools(mcpServer, services.appUserStore)

	basev1.RegisterConfigServiceMCPTools(mcpServer, services.config)
	basev1.RegisterFileServiceMCPTools(mcpServer, services.file)
	basev1.RegisterLoginServiceMCPTools(mcpServer, services.login)
}

package server

import (
	adminv1 "shop/api/gen/go/admin/v1"
	appv1 "shop/api/gen/go/app/v1"
	basev1 "shop/api/gen/go/base/v1"

	"github.com/cloudwego/eino/components/tool"
)

// newAdminAgentTools 创建管理端 AI 助手可调用的 Agent 工具列表。
func newAdminAgentTools(services *ServerServices) ([]tool.InvokableTool, error) {
	var err error
	var builder agentToolBuilder
	if err = builder.append(basev1.NewConfigServiceAgentTools(services.config)); err != nil {
		return nil, err
	}
	if err = builder.append(basev1.NewFileServiceAgentTools(services.file)); err != nil {
		return nil, err
	}
	if err = builder.append(basev1.NewLoginServiceAgentTools(services.login)); err != nil {
		return nil, err
	}
	if err = builder.append(adminv1.NewAuthServiceAgentTools(services.adminAuth)); err != nil {
		return nil, err
	}
	if err = builder.append(adminv1.NewBaseApiServiceAgentTools(services.adminBaseAPI)); err != nil {
		return nil, err
	}
	if err = builder.append(adminv1.NewBaseConfigServiceAgentTools(services.adminBaseConfig)); err != nil {
		return nil, err
	}
	if err = builder.append(adminv1.NewBaseDeptServiceAgentTools(services.adminBaseDept)); err != nil {
		return nil, err
	}
	if err = builder.append(adminv1.NewBaseDictServiceAgentTools(services.adminBaseDict)); err != nil {
		return nil, err
	}
	if err = builder.append(adminv1.NewBaseJobServiceAgentTools(services.adminBaseJob)); err != nil {
		return nil, err
	}
	if err = builder.append(adminv1.NewBaseLogServiceAgentTools(services.adminBaseLog)); err != nil {
		return nil, err
	}
	if err = builder.append(adminv1.NewBaseMenuServiceAgentTools(services.adminBaseMenu)); err != nil {
		return nil, err
	}
	if err = builder.append(adminv1.NewBaseRoleServiceAgentTools(services.adminBaseRole)); err != nil {
		return nil, err
	}
	if err = builder.append(adminv1.NewBaseUserServiceAgentTools(services.adminBaseUser)); err != nil {
		return nil, err
	}
	if err = builder.append(adminv1.NewCommentInfoServiceAgentTools(services.adminCommentInfo)); err != nil {
		return nil, err
	}
	if err = builder.append(adminv1.NewGoodsAnalyticsServiceAgentTools(services.adminGoodsAnalytics)); err != nil {
		return nil, err
	}
	if err = builder.append(adminv1.NewGoodsReportServiceAgentTools(services.adminGoodsReport)); err != nil {
		return nil, err
	}
	if err = builder.append(adminv1.NewGoodsCategoryServiceAgentTools(services.adminGoodsCategory)); err != nil {
		return nil, err
	}
	if err = builder.append(adminv1.NewGoodsPropServiceAgentTools(services.adminGoodsProp)); err != nil {
		return nil, err
	}
	if err = builder.append(adminv1.NewGoodsInfoServiceAgentTools(services.adminGoods)); err != nil {
		return nil, err
	}
	if err = builder.append(adminv1.NewGoodsSkuServiceAgentTools(services.adminGoodsSKU)); err != nil {
		return nil, err
	}
	if err = builder.append(adminv1.NewGoodsSpecServiceAgentTools(services.adminGoodsSpec)); err != nil {
		return nil, err
	}
	if err = builder.append(adminv1.NewOrderAnalyticsServiceAgentTools(services.adminOrderAnalytics)); err != nil {
		return nil, err
	}
	if err = builder.append(adminv1.NewOrderReportServiceAgentTools(services.adminOrderReport)); err != nil {
		return nil, err
	}
	if err = builder.append(adminv1.NewOrderInfoServiceAgentTools(services.adminOrder)); err != nil {
		return nil, err
	}
	if err = builder.append(adminv1.NewPayBillServiceAgentTools(services.adminPayBill)); err != nil {
		return nil, err
	}
	if err = builder.append(adminv1.NewRecommendRequestServiceAgentTools(services.adminRecommendRequest)); err != nil {
		return nil, err
	}
	if err = builder.append(adminv1.NewRecommendGorseServiceAgentTools(services.adminRecommendGorse)); err != nil {
		return nil, err
	}
	if err = builder.append(adminv1.NewShopBannerServiceAgentTools(services.adminShopBanner)); err != nil {
		return nil, err
	}
	if err = builder.append(adminv1.NewShopHotServiceAgentTools(services.adminShopHot)); err != nil {
		return nil, err
	}
	if err = builder.append(adminv1.NewShopServiceServiceAgentTools(services.adminShopService)); err != nil {
		return nil, err
	}
	if err = builder.append(adminv1.NewUserAnalyticsServiceAgentTools(services.adminUserAnalytics)); err != nil {
		return nil, err
	}
	if err = builder.append(adminv1.NewUserStoreServiceAgentTools(services.adminUserStore)); err != nil {
		return nil, err
	}
	if err = builder.append(adminv1.NewWorkspaceServiceAgentTools(services.adminWorkspace)); err != nil {
		return nil, err
	}
	return builder.tools, nil
}

// newAppAgentTools 创建商城端 AI 助手可调用的 Agent 工具列表。
func newAppAgentTools(services *ServerServices) ([]tool.InvokableTool, error) {
	var err error
	var builder agentToolBuilder
	if err = builder.append(basev1.NewConfigServiceAgentTools(services.config)); err != nil {
		return nil, err
	}
	if err = builder.append(basev1.NewFileServiceAgentTools(services.file)); err != nil {
		return nil, err
	}
	if err = builder.append(basev1.NewLoginServiceAgentTools(services.login)); err != nil {
		return nil, err
	}
	if err = builder.append(appv1.NewAuthServiceAgentTools(services.appAuth)); err != nil {
		return nil, err
	}
	if err = builder.append(appv1.NewBaseAreaServiceAgentTools(services.appBaseArea)); err != nil {
		return nil, err
	}
	if err = builder.append(appv1.NewBaseDictServiceAgentTools(services.appBaseDict)); err != nil {
		return nil, err
	}
	if err = builder.append(appv1.NewCommentServiceAgentTools(services.appComment)); err != nil {
		return nil, err
	}
	if err = builder.append(appv1.NewGoodsCategoryServiceAgentTools(services.appGoodsCategory)); err != nil {
		return nil, err
	}
	if err = builder.append(appv1.NewGoodsInfoServiceAgentTools(services.appGoods)); err != nil {
		return nil, err
	}
	if err = builder.append(appv1.NewOrderInfoServiceAgentTools(services.appOrder)); err != nil {
		return nil, err
	}
	if err = builder.append(appv1.NewPayServiceAgentTools(services.appPay)); err != nil {
		return nil, err
	}
	if err = builder.append(appv1.NewRecommendServiceAgentTools(services.appRecommend)); err != nil {
		return nil, err
	}
	if err = builder.append(appv1.NewShopBannerServiceAgentTools(services.appShopBanner)); err != nil {
		return nil, err
	}
	if err = builder.append(appv1.NewShopHotServiceAgentTools(services.appShopHot)); err != nil {
		return nil, err
	}
	if err = builder.append(appv1.NewShopServiceServiceAgentTools(services.appShopService)); err != nil {
		return nil, err
	}
	if err = builder.append(appv1.NewUserAddressServiceAgentTools(services.appUserAddress)); err != nil {
		return nil, err
	}
	if err = builder.append(appv1.NewUserCartServiceAgentTools(services.appUserCart)); err != nil {
		return nil, err
	}
	if err = builder.append(appv1.NewUserCollectServiceAgentTools(services.appUserCollect)); err != nil {
		return nil, err
	}
	if err = builder.append(appv1.NewUserStoreServiceAgentTools(services.appUserStore)); err != nil {
		return nil, err
	}
	return builder.tools, nil
}

// agentToolBuilder 负责合并各服务生成的 Agent 工具。
type agentToolBuilder struct {
	tools []tool.InvokableTool
}

// append 合并单个服务生成的 Agent 工具。
func (b *agentToolBuilder) append(values []tool.InvokableTool, err error) error {
	if err != nil {
		return err
	}
	b.tools = append(b.tools, values...)
	return nil
}

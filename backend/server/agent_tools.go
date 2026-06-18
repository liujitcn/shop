package server

import (
	adminv1 "shop/api/gen/go/admin/v1"
	appv1 "shop/api/gen/go/app/v1"
	einoTool "shop/pkg/agent/eino/tool"
)

// newAdminFlowAgentTools 创建管理端 AI 助手流程工具列表。
func newAdminFlowAgentTools(services *ServerServices) ([]einoTool.Invokable, error) {
	var err error
	var builder agentToolBuilder

	// 工作台服务（5 个工具）
	if err = builder.appendTools(adminv1.NewWorkspaceServiceAgentTools(services.adminWorkspace)); err != nil {
		return nil, err
	}

	// 订单信息服务（排除 refund_order_info，注册 5 个工具）
	if err = builder.appendTool(adminv1.NewOrderInfoServicePageOrderInfosAgentTool(services.adminOrder)); err != nil {
		return nil, err
	}
	if err = builder.appendTool(adminv1.NewOrderInfoServiceGetOrderInfoAgentTool(services.adminOrder)); err != nil {
		return nil, err
	}
	if err = builder.appendTool(adminv1.NewOrderInfoServiceGetOrderInfoRefundAgentTool(services.adminOrder)); err != nil {
		return nil, err
	}
	if err = builder.appendTool(adminv1.NewOrderInfoServiceGetOrderInfoShipmentAgentTool(services.adminOrder)); err != nil {
		return nil, err
	}
	if err = builder.appendTool(adminv1.NewOrderInfoServiceShipOrderInfoAgentTool(services.adminOrder)); err != nil {
		return nil, err
	}

	// 评价信息服务（7 个工具）
	if err = builder.appendTools(adminv1.NewCommentInfoServiceAgentTools(services.adminCommentInfo)); err != nil {
		return nil, err
	}

	// 商品信息服务（7 个工具）
	if err = builder.appendTools(adminv1.NewGoodsInfoServiceAgentTools(services.adminGoods)); err != nil {
		return nil, err
	}

	// 商品分析服务（4 个工具）
	if err = builder.appendTools(adminv1.NewGoodsAnalyticsServiceAgentTools(services.adminGoodsAnalytics)); err != nil {
		return nil, err
	}

	// 订单分析服务（3 个工具）
	if err = builder.appendTools(adminv1.NewOrderAnalyticsServiceAgentTools(services.adminOrderAnalytics)); err != nil {
		return nil, err
	}

	// 门店服务（3 个工具）
	if err = builder.appendTools(adminv1.NewUserStoreServiceAgentTools(services.adminUserStore)); err != nil {
		return nil, err
	}

	// 推荐 Gorse 服务（22 个工具）
	if err = builder.appendTools(adminv1.NewRecommendGorseServiceAgentTools(services.adminRecommendGorse)); err != nil {
		return nil, err
	}

	// 对账单服务（1 个工具）
	if err = builder.appendTools(adminv1.NewPayBillServiceAgentTools(services.adminPayBill)); err != nil {
		return nil, err
	}

	// 订单报表服务（4 个工具）
	if err = builder.appendTools(adminv1.NewOrderReportServiceAgentTools(services.adminOrderReport)); err != nil {
		return nil, err
	}

	// 商品报表服务（4 个工具）
	if err = builder.appendTools(adminv1.NewGoodsReportServiceAgentTools(services.adminGoodsReport)); err != nil {
		return nil, err
	}

	// 用户分析服务（3 个工具）
	if err = builder.appendTools(adminv1.NewUserAnalyticsServiceAgentTools(services.adminUserAnalytics)); err != nil {
		return nil, err
	}

	// 推荐请求服务（3 个工具）
	if err = builder.appendTools(adminv1.NewRecommendRequestServiceAgentTools(services.adminRecommendRequest)); err != nil {
		return nil, err
	}

	return builder.tools, nil
}

// newAppFlowAgentTools 创建商城端 AI 助手流程工具列表。
func newAppFlowAgentTools(services *ServerServices) ([]einoTool.Invokable, error) {
	var err error
	var builder agentToolBuilder
	if err = builder.appendTool(appv1.NewAuthServiceGetUserProfileAgentTool(services.appAuth)); err != nil {
		return nil, err
	}
	if err = builder.appendTool(appv1.NewAuthServiceUpdateUserProfileAgentTool(services.appAuth)); err != nil {
		return nil, err
	}
	if err = builder.appendTools(appv1.NewBaseAreaServiceAgentTools(services.appBaseArea)); err != nil {
		return nil, err
	}
	if err = builder.appendTools(appv1.NewBaseDictServiceAgentTools(services.appBaseDict)); err != nil {
		return nil, err
	}
	if err = builder.appendTools(appv1.NewCommentServiceAgentTools(services.appComment)); err != nil {
		return nil, err
	}
	if err = builder.appendTools(appv1.NewGoodsCategoryServiceAgentTools(services.appGoodsCategory)); err != nil {
		return nil, err
	}
	if err = builder.appendTools(appv1.NewGoodsInfoServiceAgentTools(services.appGoods)); err != nil {
		return nil, err
	}
	if err = builder.appendTools(appv1.NewOrderInfoServiceAgentTools(services.appOrder)); err != nil {
		return nil, err
	}
	if err = builder.appendTool(appv1.NewPayServiceJsapiPayAgentTool(services.appPay)); err != nil {
		return nil, err
	}
	if err = builder.appendTool(appv1.NewPayServiceH5PayAgentTool(services.appPay)); err != nil {
		return nil, err
	}
	if err = builder.appendTool(appv1.NewRecommendServiceRecommendAnonymousActorAgentTool(services.appRecommend)); err != nil {
		return nil, err
	}
	if err = builder.appendTool(appv1.NewRecommendServiceRecommendGoodsAgentTool(services.appRecommend)); err != nil {
		return nil, err
	}
	if err = builder.appendTool(appv1.NewRecommendServiceRecommendEventReportAgentTool(services.appRecommend)); err != nil {
		return nil, err
	}
	if err = builder.appendTools(appv1.NewShopBannerServiceAgentTools(services.appShopBanner)); err != nil {
		return nil, err
	}
	if err = builder.appendTools(appv1.NewShopHotServiceAgentTools(services.appShopHot)); err != nil {
		return nil, err
	}
	if err = builder.appendTools(appv1.NewShopServiceServiceAgentTools(services.appShopService)); err != nil {
		return nil, err
	}
	if err = builder.appendTools(appv1.NewUserAddressServiceAgentTools(services.appUserAddress)); err != nil {
		return nil, err
	}
	if err = builder.appendTools(appv1.NewUserCartServiceAgentTools(services.appUserCart)); err != nil {
		return nil, err
	}
	if err = builder.appendTools(appv1.NewUserCollectServiceAgentTools(services.appUserCollect)); err != nil {
		return nil, err
	}
	if err = builder.appendTools(appv1.NewUserStoreServiceAgentTools(services.appUserStore)); err != nil {
		return nil, err
	}
	return builder.tools, nil
}

// agentToolBuilder 负责合并各服务生成的 Agent 工具。
type agentToolBuilder struct {
	tools []einoTool.Invokable
}

// appendTool 合并单个生成的 Agent 工具。
func (b *agentToolBuilder) appendTool(value einoTool.Invokable, err error) error {
	if err != nil {
		return err
	}
	b.tools = append(b.tools, value)
	return nil
}

// appendTools 合并一组生成的 Agent 工具。
func (b *agentToolBuilder) appendTools(values []einoTool.Invokable, err error) error {
	if err != nil {
		return err
	}
	b.tools = append(b.tools, values...)
	return nil
}

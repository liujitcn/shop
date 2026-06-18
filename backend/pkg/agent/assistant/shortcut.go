package assistant

import basev1 "shop/api/gen/go/base/v1"

const (
	shortcutFlowShopping       = "shopping"
	shortcutFlowPendingPayment = "pending_payment"
	shortcutFlowPendingReview  = "pending_review"
	shortcutFlowOrderLogistics = "order_logistics"
	shortcutFlowUserCart       = "user_cart"
	shortcutFlowUserCollect    = "user_collect"
	shortcutFlowUserAddress    = "user_address"
	shortcutFlowUserProfile    = "user_profile"
	shortcutFlowUserStore      = "user_store"
	shortcutFlowGoodsCategory  = "goods_category"
	shortcutFlowShopHot        = "shop_hot"
	shortcutFlowShopService    = "shop_service"

	shortcutActionOpenShopping       = "open_shopping"
	shortcutActionOpenPendingPayment = "open_pending_payment"
	shortcutActionOpenPendingReview  = "open_pending_review"
	shortcutActionOpenOrderLogistics = "open_order_logistics"
	shortcutActionOpenUserCart       = "open_user_cart"
	shortcutActionOpenUserCollect    = "open_user_collect"
	shortcutActionOpenUserAddress    = "open_user_address"
	shortcutActionOpenUserProfile    = "open_user_profile"
	shortcutActionOpenUserStore      = "open_user_store"
	shortcutActionOpenGoodsCategory  = "open_goods_category"
	shortcutActionOpenShopHot        = "open_shop_hot"
	shortcutActionOpenShopService    = "open_shop_service"

	toolGetUserProfile     = "app_v1_auth_service_get_user_profile"
	toolListBaseAreas      = "app_v1_base_area_service_tree_base_areas"
	toolPageGoodsInfo      = "app_v1_goods_info_service_page_goods_info"
	toolGetGoodsInfo       = "app_v1_goods_info_service_get_goods_info"
	toolListGoodsCategory  = "app_v1_goods_category_service_list_goods_categories"
	toolBuyNowOrderInfo    = "app_v1_order_info_service_buy_now_order_info"
	toolCreateOrderInfo    = "app_v1_order_info_service_create_order_info"
	toolCountOrderInfo     = "app_v1_order_info_service_count_order_info"
	toolPageOrderInfo      = "app_v1_order_info_service_page_order_info"
	toolGetOrderInfoByID   = "app_v1_order_info_service_get_order_info_by_id"
	toolListShopHots       = "app_v1_shop_hot_service_list_shop_hots"
	toolListShopHotItems   = "app_v1_shop_hot_service_list_shop_hot_items"
	toolPageShopHotGoods   = "app_v1_shop_hot_service_page_shop_hot_goods"
	toolListShopServices   = "app_v1_shop_service_service_list_shop_services"
	toolListUserAddresses  = "app_v1_user_address_service_list_user_addresses"
	toolCreateUserAddress  = "app_v1_user_address_service_create_user_address"
	toolListUserCarts      = "app_v1_user_cart_service_list_user_carts"
	toolPageUserCollects   = "app_v1_user_collect_service_page_user_collects"
	toolGetUserStore       = "app_v1_user_store_service_get_user_store"
	toolPagePendingComment = "app_v1_comment_service_page_pending_comment_goods"
	toolCreateComment      = "app_v1_comment_service_create_comment"
	toolJSAPIPay           = "app_v1_pay_service_jsapi_pay"
	toolH5Pay              = "app_v1_pay_service_h5_pay"

	// 管理端流程名称
	shortcutAdminFlowWorkspaceOverview   = "workspace_overview"
	shortcutAdminFlowPendingShipment     = "pending_shipment"
	shortcutAdminFlowCommentReview       = "comment_review"
	shortcutAdminFlowGoodsInventoryAlert = "goods_inventory_alert"
	shortcutAdminFlowOrderRefund         = "order_refund"
	shortcutAdminFlowGoodsAnalytics      = "goods_analytics"
	shortcutAdminFlowOrderAnalytics      = "order_analytics"
	shortcutAdminFlowStoreAudit          = "store_audit"
	shortcutAdminFlowRecommendDashboard  = "recommend_dashboard"
	shortcutAdminFlowReputationInsight   = "reputation_insight"
	shortcutAdminFlowPayBillCheck        = "pay_bill_check"
	shortcutAdminFlowReportOverview      = "report_overview"

	// 管理端入口动作
	shortcutAdminActionOpenWorkspaceOverview   = "open_workspace_overview"
	shortcutAdminActionOpenPendingShipment     = "open_pending_shipment"
	shortcutAdminActionOpenCommentReview       = "open_comment_review"
	shortcutAdminActionOpenGoodsInventoryAlert = "open_goods_inventory_alert"
	shortcutAdminActionOpenOrderRefund         = "open_order_refund"
	shortcutAdminActionOpenGoodsAnalytics      = "open_goods_analytics"
	shortcutAdminActionOpenOrderAnalytics      = "open_order_analytics"
	shortcutAdminActionOpenStoreAudit          = "open_store_audit"
	shortcutAdminActionOpenRecommendDashboard  = "open_recommend_dashboard"
	shortcutAdminActionOpenReputationInsight   = "open_reputation_insight"
	shortcutAdminActionOpenPayBillCheck        = "open_pay_bill_check"
	shortcutAdminActionOpenReportOverview      = "open_report_overview"

	// 管理端工具
	toolAdminSummaryWorkspaceMetrics    = "admin_v1_workspace_service_summary_workspace_metrics"
	toolAdminPageOrderInfos             = "admin_v1_order_info_service_page_order_infos"
	toolAdminPageCommentInfos           = "admin_v1_comment_info_service_page_comment_infos"
	toolAdminPageGoodsInfos             = "admin_v1_goods_info_service_page_goods_infos"
	toolAdminSummaryGoodsAnalytics      = "admin_v1_goods_analytics_service_summary_goods_analytics"
	toolAdminSummaryOrderAnalytics      = "admin_v1_order_analytics_service_summary_order_analytics"
	toolAdminPageUserStores             = "admin_v1_user_store_service_page_user_stores"
	toolAdminListDashboardItems         = "admin_v1_recommend_gorse_service_list_dashboard_items"
	toolAdminSummaryWorkspaceReputation = "admin_v1_workspace_service_summary_workspace_reputation"
	toolAdminPagePayBills               = "admin_v1_pay_bill_service_page_pay_bills"
	toolAdminSummaryOrderMonthReport    = "admin_v1_order_report_service_summary_order_month_report"
)

var shortcutCatalog = []shortcutItem{
	{
		key:      "recommend_goods",
		title:    "帮我推荐商品",
		prompt:   "帮我推荐商品",
		terminal: TerminalApp,
		sort:     10,
		action: shortcutAction{
			flow: shortcutFlowShopping,
			step: "goods",
			typ:  shortcutActionOpenShopping,
		},
		requiredTools: []string{
			toolPageGoodsInfo,
			toolGetGoodsInfo,
			toolBuyNowOrderInfo,
			toolCreateOrderInfo,
			toolListUserAddresses,
			toolCreateUserAddress,
			toolJSAPIPay,
			toolH5Pay,
		},
	},
	{
		key:      "hot_goods",
		title:    "帮我找热销商品",
		prompt:   "帮我找热销商品",
		terminal: TerminalApp,
		sort:     20,
		action: shortcutAction{
			flow: shortcutFlowShopping,
			step: "goods",
			typ:  shortcutActionOpenShopping,
		},
		requiredTools: []string{
			toolPageGoodsInfo,
			toolGetGoodsInfo,
			toolBuyNowOrderInfo,
			toolCreateOrderInfo,
			toolListUserAddresses,
			toolCreateUserAddress,
			toolJSAPIPay,
			toolH5Pay,
		},
	},
	{
		key:      "pending_payment",
		title:    "查看待付款订单",
		prompt:   "查看待付款订单",
		terminal: TerminalApp,
		sort:     30,
		action: shortcutAction{
			flow: shortcutFlowPendingPayment,
			step: "list",
			typ:  shortcutActionOpenPendingPayment,
		},
		requiredTools: []string{
			toolPageOrderInfo,
			toolJSAPIPay,
			toolH5Pay,
		},
	},
	{
		key:      "recent_order",
		title:    "查询最近订单",
		prompt:   "查询最近订单",
		terminal: TerminalApp,
		sort:     40,
		action: shortcutAction{
			flow: shortcutFlowOrderLogistics,
			step: "list",
			typ:  shortcutActionOpenOrderLogistics,
		},
		requiredTools: []string{
			toolPageOrderInfo,
			toolGetOrderInfoByID,
		},
	},
	{
		key:      "order_logistics",
		title:    "我的物流到哪了",
		prompt:   "我的物流到哪了",
		terminal: TerminalApp,
		sort:     50,
		action: shortcutAction{
			flow: shortcutFlowOrderLogistics,
			step: "list",
			typ:  shortcutActionOpenOrderLogistics,
		},
		requiredTools: []string{
			toolPageOrderInfo,
			toolGetOrderInfoByID,
		},
	},
	{
		key:      "pending_review",
		title:    "收到商品后怎么评价",
		prompt:   "收到商品后怎么评价",
		terminal: TerminalApp,
		sort:     60,
		action: shortcutAction{
			flow: shortcutFlowPendingReview,
			step: "list",
			typ:  shortcutActionOpenPendingReview,
		},
		requiredTools: []string{
			toolPagePendingComment,
			toolCreateComment,
		},
	},
	{
		key:      "user_cart",
		title:    "看看购物车里有什么",
		prompt:   "看看购物车里有什么",
		terminal: TerminalApp,
		sort:     70,
		action: shortcutAction{
			flow: shortcutFlowUserCart,
			step: "list",
			typ:  shortcutActionOpenUserCart,
		},
		requiredTools: []string{
			toolListUserCarts,
		},
	},
	{
		key:      "user_collect",
		title:    "查看我的收藏商品",
		prompt:   "查看我的收藏商品",
		terminal: TerminalApp,
		sort:     80,
		action: shortcutAction{
			flow: shortcutFlowUserCollect,
			step: "list",
			typ:  shortcutActionOpenUserCollect,
		},
		requiredTools: []string{
			toolPageUserCollects,
		},
	},
	{
		key:      "user_address",
		title:    "管理我的收货地址",
		prompt:   "管理我的收货地址",
		terminal: TerminalApp,
		sort:     90,
		action: shortcutAction{
			flow: shortcutFlowUserAddress,
			step: "list",
			typ:  shortcutActionOpenUserAddress,
		},
		requiredTools: []string{
			toolListUserAddresses,
			toolCreateUserAddress,
			toolListBaseAreas,
		},
	},
	{
		key:      "user_profile",
		title:    "查看我的个人资料",
		prompt:   "查看我的个人资料",
		terminal: TerminalApp,
		sort:     100,
		action: shortcutAction{
			flow: shortcutFlowUserProfile,
			step: "detail",
			typ:  shortcutActionOpenUserProfile,
		},
		requiredTools: []string{
			toolGetUserProfile,
		},
	},
	{
		key:      "user_store",
		title:    "查看我的门店入驻",
		prompt:   "查看我的门店入驻",
		terminal: TerminalApp,
		sort:     110,
		action: shortcutAction{
			flow: shortcutFlowUserStore,
			step: "detail",
			typ:  shortcutActionOpenUserStore,
		},
		requiredTools: []string{
			toolGetUserStore,
		},
	},
	{
		key:      "goods_category",
		title:    "按分类逛商品",
		prompt:   "按分类逛商品",
		terminal: TerminalApp,
		sort:     120,
		action: shortcutAction{
			flow: shortcutFlowGoodsCategory,
			step: "list",
			typ:  shortcutActionOpenGoodsCategory,
		},
		requiredTools: []string{
			toolListGoodsCategory,
		},
	},
	{
		key:      "shop_hot",
		title:    "看看热门专区",
		prompt:   "看看热门专区",
		terminal: TerminalApp,
		sort:     130,
		action: shortcutAction{
			flow: shortcutFlowShopHot,
			step: "list",
			typ:  shortcutActionOpenShopHot,
		},
		requiredTools: []string{
			toolListShopHots,
			toolListShopHotItems,
			toolPageShopHotGoods,
		},
	},
	{
		key:      "shop_service",
		title:    "商城服务说明有哪些",
		prompt:   "商城服务说明有哪些",
		terminal: TerminalApp,
		sort:     140,
		action: shortcutAction{
			flow: shortcutFlowShopService,
			step: "list",
			typ:  shortcutActionOpenShopService,
		},
		requiredTools: []string{
			toolListShopServices,
		},
	},
	// ===== 管理端快捷入口 =====
	{
		key:      "admin_workspace_overview",
		title:    "经营总览",
		prompt:   "查看经营总览",
		terminal: TerminalAdmin,
		sort:     10,
		group:    "订单运营",
		action: shortcutAction{
			flow: shortcutAdminFlowWorkspaceOverview,
			step: "overview",
			typ:  shortcutAdminActionOpenWorkspaceOverview,
		},
		requiredTools: []string{
			toolAdminSummaryWorkspaceMetrics,
		},
	},
	{
		key:      "admin_pending_shipment",
		title:    "待发货订单",
		prompt:   "查看待发货订单",
		terminal: TerminalAdmin,
		sort:     20,
		group:    "订单运营",
		action: shortcutAction{
			flow: shortcutAdminFlowPendingShipment,
			step: "list",
			typ:  shortcutAdminActionOpenPendingShipment,
		},
		requiredTools: []string{
			toolAdminPageOrderInfos,
		},
	},
	{
		key:      "admin_comment_review",
		title:    "评价审核",
		prompt:   "查看待审核评价",
		terminal: TerminalAdmin,
		sort:     30,
		group:    "商品运营",
		action: shortcutAction{
			flow: shortcutAdminFlowCommentReview,
			step: "list",
			typ:  shortcutAdminActionOpenCommentReview,
		},
		requiredTools: []string{
			toolAdminPageCommentInfos,
		},
	},
	{
		key:      "admin_goods_inventory_alert",
		title:    "库存预警",
		prompt:   "查看库存预警商品",
		terminal: TerminalAdmin,
		sort:     40,
		group:    "商品运营",
		action: shortcutAction{
			flow: shortcutAdminFlowGoodsInventoryAlert,
			step: "list",
			typ:  shortcutAdminActionOpenGoodsInventoryAlert,
		},
		requiredTools: []string{
			toolAdminPageGoodsInfos,
		},
	},
	{
		key:      "admin_order_refund",
		title:    "退款记录",
		prompt:   "查看退款记录",
		terminal: TerminalAdmin,
		sort:     50,
		group:    "订单运营",
		action: shortcutAction{
			flow: shortcutAdminFlowOrderRefund,
			step: "list",
			typ:  shortcutAdminActionOpenOrderRefund,
		},
		requiredTools: []string{
			toolAdminPageOrderInfos,
		},
	},
	{
		key:      "admin_goods_analytics",
		title:    "商品分析",
		prompt:   "查看商品分析",
		terminal: TerminalAdmin,
		sort:     60,
		group:    "商品运营",
		action: shortcutAction{
			flow: shortcutAdminFlowGoodsAnalytics,
			step: "overview",
			typ:  shortcutAdminActionOpenGoodsAnalytics,
		},
		requiredTools: []string{
			toolAdminSummaryGoodsAnalytics,
		},
	},
	{
		key:      "admin_order_analytics",
		title:    "订单分析",
		prompt:   "查看订单分析",
		terminal: TerminalAdmin,
		sort:     70,
		group:    "订单运营",
		action: shortcutAction{
			flow: shortcutAdminFlowOrderAnalytics,
			step: "overview",
			typ:  shortcutAdminActionOpenOrderAnalytics,
		},
		requiredTools: []string{
			toolAdminSummaryOrderAnalytics,
		},
	},
	{
		key:      "admin_store_audit",
		title:    "门店审核",
		prompt:   "查看待审核门店",
		terminal: TerminalAdmin,
		sort:     80,
		group:    "商品运营",
		action: shortcutAction{
			flow: shortcutAdminFlowStoreAudit,
			step: "list",
			typ:  shortcutAdminActionOpenStoreAudit,
		},
		requiredTools: []string{
			toolAdminPageUserStores,
		},
	},
	{
		key:      "admin_recommend_dashboard",
		title:    "推荐看板",
		prompt:   "查看推荐效果看板",
		terminal: TerminalAdmin,
		sort:     90,
		group:    "数据分析",
		action: shortcutAction{
			flow: shortcutAdminFlowRecommendDashboard,
			step: "overview",
			typ:  shortcutAdminActionOpenRecommendDashboard,
		},
		requiredTools: []string{
			toolAdminListDashboardItems,
		},
	},
	{
		key:      "admin_reputation_insight",
		title:    "口碑洞察",
		prompt:   "查看口碑洞察",
		terminal: TerminalAdmin,
		sort:     100,
		group:    "数据分析",
		action: shortcutAction{
			flow: shortcutAdminFlowReputationInsight,
			step: "overview",
			typ:  shortcutAdminActionOpenReputationInsight,
		},
		requiredTools: []string{
			toolAdminSummaryWorkspaceReputation,
		},
	},
	{
		key:      "admin_pay_bill_check",
		title:    "对账检查",
		prompt:   "查看对账异常",
		terminal: TerminalAdmin,
		sort:     110,
		group:    "系统管理",
		action: shortcutAction{
			flow: shortcutAdminFlowPayBillCheck,
			step: "list",
			typ:  shortcutAdminActionOpenPayBillCheck,
		},
		requiredTools: []string{
			toolAdminPagePayBills,
		},
	},
	{
		key:      "admin_report_overview",
		title:    "经营报表",
		prompt:   "查看经营报表",
		terminal: TerminalAdmin,
		sort:     120,
		group:    "数据分析",
		action: shortcutAction{
			flow: shortcutAdminFlowReportOverview,
			step: "overview",
			typ:  shortcutAdminActionOpenReportOverview,
		},
		requiredTools: []string{
			toolAdminSummaryOrderMonthReport,
		},
	},
}

// shortcutItem 表示一个快捷助手入口的静态配置项。
type shortcutItem struct {
	// key 快捷入口唯一标识，用于前端稳定识别和埋点扩展。
	key string
	// title 快捷入口展示标题。
	title string
	// prompt 点击快捷入口后发送给 AI 助手的提示词。
	prompt string
	// terminal 快捷入口适用终端，对应数据库终端值。
	terminal int32
	// sort 快捷入口排序值，数值越小越靠前。
	sort int32
	// group 快捷入口分组名称，用于前端按分组展示。
	group string
	// action 快捷入口触发的前端流程动作。
	action shortcutAction
	// requiredTools 快捷入口依赖的后台 Agent 工具名，全部启用时才返回。
	requiredTools []string
}

// shortcutAction 表示快捷助手入口关联的前端流程动作。
type shortcutAction struct {
	// flow 流程标识。
	flow string
	// step 流程步骤。
	step string
	// typ 动作类型。
	typ string
}

// BuildShortcuts 根据终端和已启用工具生成 AI 助手快捷入口列表。
func BuildShortcuts(terminal int32, enabledTools map[string]bool) []*basev1.AiAssistantShortcut {
	shortcuts := make([]*basev1.AiAssistantShortcut, 0, len(shortcutCatalog))
	for _, item := range shortcutCatalog {
		if item.terminal != terminal {
			continue
		}
		if !hasRequiredTools(enabledTools, item.requiredTools) {
			continue
		}
		shortcuts = append(shortcuts, item.toDTO())
	}
	return shortcuts
}

// toDTO 转换快捷入口为接口响应对象。
func (i shortcutItem) toDTO() *basev1.AiAssistantShortcut {
	return &basev1.AiAssistantShortcut{
		Key:           i.key,
		Title:         i.title,
		Prompt:        i.prompt,
		Action:        i.action.toDTO(),
		RequiredTools: append([]string(nil), i.requiredTools...),
		Sort:          i.sort,
		Group:         i.group,
	}
}

// toDTO 转换快捷入口动作为接口响应对象。
func (a shortcutAction) toDTO() *basev1.AiAssistantShortcutAction {
	return &basev1.AiAssistantShortcutAction{
		Flow: a.flow,
		Step: a.step,
		Type: a.typ,
	}
}

// hasRequiredTools 判断快捷入口依赖工具是否全部可用。
func hasRequiredTools(enabledTools map[string]bool, requiredTools []string) bool {
	for _, name := range requiredTools {
		if !enabledTools[name] {
			return false
		}
	}
	return true
}

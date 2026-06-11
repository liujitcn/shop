package assistant

import basev1 "shop/api/gen/go/base/v1"

const (
	shortcutFlowShopping       = "shopping"
	shortcutFlowPendingPayment = "pending_payment"
	shortcutFlowPendingReview  = "pending_review"
	shortcutFlowOrderLogistics = "order_logistics"

	shortcutActionOpenShopping       = "open_shopping"
	shortcutActionOpenPendingPayment = "open_pending_payment"
	shortcutActionOpenPendingReview  = "open_pending_review"
	shortcutActionOpenOrderLogistics = "open_order_logistics"

	toolPageGoodsInfo      = "app_v1_goods_info_service_page_goods_info"
	toolGetGoodsInfo       = "app_v1_goods_info_service_get_goods_info"
	toolBuyNowOrderInfo    = "app_v1_order_info_service_buy_now_order_info"
	toolCreateOrderInfo    = "app_v1_order_info_service_create_order_info"
	toolPageOrderInfo      = "app_v1_order_info_service_page_order_info"
	toolGetOrderInfoByID   = "app_v1_order_info_service_get_order_info_by_id"
	toolListUserAddresses  = "app_v1_user_address_service_list_user_addresses"
	toolCreateUserAddress  = "app_v1_user_address_service_create_user_address"
	toolPagePendingComment = "app_v1_comment_service_page_pending_comment_goods"
	toolCreateComment      = "app_v1_comment_service_create_comment"
	toolJSAPIPay           = "app_v1_pay_service_jsapi_pay"
	toolH5Pay              = "app_v1_pay_service_h5_pay"
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

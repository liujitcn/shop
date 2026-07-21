package aiflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	shopcommonv1 "shop/api/gen/go/shop/common/v1"
	systemcommonv1 "shop/api/gen/go/system/common/v1"

	basev1 "shop/api/gen/go/base/v1"
	einoWorkflow "shop/pkg/agent/eino/workflow"
	"shop/pkg/errorsx"
	"shop/service/base/agent/ai"
)

const (
	aiFlowShopping       = "shopping"
	aiFlowPendingPayment = "pending_payment"
	aiFlowPendingReview  = "pending_review"
	aiFlowOrderLogistics = "order_logistics"
	aiFlowUserCart       = "user_cart"
	aiFlowUserCollect    = "user_collect"
	aiFlowUserAddress    = "user_address"
	aiFlowUserProfile    = "user_profile"
	aiFlowUserStore      = "user_store"
	aiFlowGoodsCategory  = "goods_category"
	aiFlowShopHot        = "shop_hot"
	aiFlowShopService    = "shop_service"

	aiToolGetUserProfile     = "app_v1_auth_service_get_user_profile"
	aiToolRecommendGoods     = "app_v1_recommend_service_recommend_goods"
	aiToolPageGoodsInfo      = "app_v1_goods_info_service_page_goods_info"
	aiToolGetGoodsInfo       = "app_v1_goods_info_service_get_goods_info"
	aiToolListGoodsCategory  = "app_v1_goods_category_service_list_goods_categories"
	aiToolBuyNowOrderInfo    = "app_v1_order_info_service_buy_now_order_info"
	aiToolCreateOrderInfo    = "app_v1_order_info_service_create_order_info"
	aiToolPageOrderInfo      = "app_v1_order_info_service_page_order_info"
	aiToolGetOrderInfoByID   = "app_v1_order_info_service_get_order_info_by_id"
	aiToolGetOrderTradeByID  = "app_v1_order_info_service_get_order_trade_by_id"
	aiToolReceiveOrderInfo   = "app_v1_order_info_service_receive_order_info"
	aiToolListShopHot        = "app_v1_shop_hot_service_list_shop_hots"
	aiToolListShopHotItem    = "app_v1_shop_hot_service_list_shop_hot_items"
	aiToolPageShopHotGoods   = "app_v1_shop_hot_service_page_shop_hot_goods"
	aiToolListShopService    = "app_v1_shop_service_service_list_shop_services"
	aiToolListUserAddress    = "app_v1_user_address_service_list_user_addresses"
	aiToolCreateUserAddress  = "app_v1_user_address_service_create_user_address"
	aiToolListUserCart       = "app_v1_user_cart_service_list_user_carts"
	aiToolPageUserCollect    = "app_v1_user_collect_service_page_user_collects"
	aiToolGetUserStore       = "app_v1_user_store_service_get_user_store"
	aiToolPagePendingComment = "app_v1_comment_service_page_pending_comment_goods"
	aiToolCreateComment      = "app_v1_comment_service_create_comment"
	aiToolJSAPIPay           = "app_v1_pay_service_jsapi_pay"
	aiToolH5Pay              = "app_v1_pay_service_h5_pay"
)

var aiFlowRegistry = einoWorkflow.MustNewRegistry[*ai.Response](appFlowDefinitions, "商城", "shop_app_fixed_flow")

// Runner 编排移动端助手闭环流程。
type Runner struct {
	runtime  *ai.Runtime
	terminal int32
}

// aiProfileField 表示资料面板中的字段展示规则。
type aiProfileField struct {
	label  string
	key    string
	format func(any) string
}

// GenerateReply 按终端分发 AI 助手闭环流程回复。
func GenerateReply(
	ctx context.Context,
	runtime *ai.Runtime,
	terminal int32,
	content string,
	action *basev1.AiAction,
) (*ai.Response, bool, error) {
	if terminal == ai.TerminalAdmin {
		return GenerateAdminReply(ctx, runtime, terminal, content, action)
	}
	return GenerateAppReply(ctx, runtime, terminal, content, action)
}

// GenerateAppReply 生成移动端闭环流程回复。
func GenerateAppReply(
	ctx context.Context,
	runtime *ai.Runtime,
	terminal int32,
	content string,
	action *basev1.AiAction,
) (*ai.Response, bool, error) {
	runner := &Runner{runtime: runtime, terminal: terminal}
	if action != nil && action.GetType() != "" {
		reply, err := runner.handleAiFlowAction(ctx, action)
		return reply, true, err
	}

	flow := matchAiFlowIntent(content)
	if flow == "" {
		return nil, false, nil
	}
	reply, err := runner.handleAiFlowAction(ctx, &basev1.AiAction{
		Flow: flow,
		Type: openAiFlowActionType(flow),
	})
	return reply, true, err
}

// IsEntryAction 判断动作是否为固定流程入口。
func IsEntryAction(terminal int32, flow string, actionType string) bool {
	if actionType == "" {
		return false
	}
	if terminal == ai.TerminalAdmin {
		return adminFlowRegistry.EntryAction(einoWorkflow.FlowName(flow)) == actionType
	}
	return aiFlowRegistry.EntryAction(einoWorkflow.FlowName(flow)) == actionType
}

// ExecuteWorkflowAction 执行 Eino Graph 路由后的移动端流程动作。
func (r *Runner) ExecuteWorkflowAction(ctx context.Context, action einoWorkflow.Action, payload map[string]any) (*ai.Response, error) {
	// 按前端按钮提交的动作类型进入对应流程步骤；动作合法性由 eino/workflow 的 Graph 分支负责。
	switch action.Type {
	case "open_shopping":
		return r.openAiShoppingFlow(ctx)
	case "select_goods":
		return r.openAiSkuSelector(ctx, payload)
	case "select_sku":
		return r.openAiCheckout(ctx, payload)
	case "create_address":
		return r.createAiAddress(ctx, payload)
	case "select_address":
		return r.selectAiAddress(payload), nil
	case "confirm_order":
		return r.confirmAiOrder(ctx, payload)
	case "start_payment":
		return r.startAiPayment(ctx, payload)
	case "open_pending_payment":
		return r.openAiPendingPaymentFlow(ctx)
	case "open_pending_review":
		return r.openAiPendingReviewFlow(ctx)
	case "open_review_form":
		return r.openAiReviewForm(payload), nil
	case "submit_review":
		return r.submitAiReview(ctx, payload)
	case "open_order_logistics":
		return r.openAiOrderLogisticsFlow(ctx)
	case "view_order":
		return r.viewAiOrder(ctx, payload)
	case "receive_order":
		return r.receiveAiOrder(ctx, payload)
	case "open_user_cart":
		return r.openAiUserCartFlow(ctx)
	case "open_user_collect":
		return r.openAiUserCollectFlow(ctx)
	case "open_user_address":
		return r.openAiUserAddressFlow(ctx)
	case "open_user_profile":
		return r.openAiUserProfileFlow(ctx)
	case "open_user_store":
		return r.openAiUserStoreFlow(ctx)
	case "open_goods_category":
		return r.openAiGoodsCategoryFlow(ctx)
	case "view_goods_category":
		return r.viewAiGoodsCategory(ctx, payload)
	case "open_shop_hot":
		return r.openAiShopHotFlow(ctx)
	case "view_shop_hot_item":
		return r.viewAiShopHotItem(ctx, payload)
	case "open_shop_service":
		return r.openAiShopServiceFlow(ctx)
	default:
		return nil, errorsx.InvalidArgument("助手动作不支持")
	}
}

// handleAiFlowAction 推进移动端闭环流程。
func (r *Runner) handleAiFlowAction(ctx context.Context, action *basev1.AiAction) (*ai.Response, error) {
	payload, err := parseAiActionPayload(action.GetPayloadJson())
	if err != nil {
		return nil, err
	}
	var result einoWorkflow.ActionResult[*ai.Response]
	result, err = aiFlowRegistry.Run(ctx, einoWorkflow.ActionRequest{
		Flow:       einoWorkflow.FlowName(action.GetFlow()),
		ActionType: action.GetType(),
		Payload:    payload,
	}, r.ExecuteWorkflowAction)
	if err != nil {
		return nil, err
	}
	// 固定流程动作先经过 Eino Graph 路由，避免前端传入未注册动作直接进入业务分支。
	if action.GetType() != "" && !result.Found {
		return nil, errorsx.InvalidArgument("助手动作不支持")
	}
	if result.Output == nil {
		return nil, errorsx.Internal("助手动作结果无效")
	}
	return result.Output, nil
}

// openAiShoppingFlow 打开推荐下单流程。
func (r *Runner) openAiShoppingFlow(ctx context.Context) (*ai.Response, error) {
	tools := make([]ai.ToolUsage, 0, 2)
	output, usage, err := r.invokeAiFlowTool(ctx, aiToolRecommendGoods, map[string]any{
		"scene":     int(shopcommonv1.RecommendScene_PROFILE),
		"page_num":  1,
		"page_size": 6,
	})
	tools = appendAiFlowTool(tools, usage)
	if err != nil {
		var fallbackUsage ai.ToolUsage
		output, fallbackUsage, err = r.invokeAiFlowTool(ctx, aiToolPageGoodsInfo, map[string]any{
			"page_num":  1,
			"page_size": 6,
		})
		tools = appendAiFlowTool(tools, fallbackUsage)
	}
	if err != nil {
		return r.aiFlowErrorResponse(aiFlowShopping, "goods", tools), nil
	}
	return r.aiFlowResponse(aiFlowShopping, "goods", "先给你推荐这些商品，选一个继续看规格。", []map[string]any{
		buildAiGoodsListBlock(output),
	}, tools), nil
}

// openAiSkuSelector 打开商品规格选择。
func (r *Runner) openAiSkuSelector(ctx context.Context, payload map[string]any) (*ai.Response, error) {
	goodsID := int64Value(payload["goods_id"])
	if goodsID <= 0 {
		return nil, errorsx.InvalidArgument("商品参数不合法")
	}
	output, usage, err := r.invokeAiFlowTool(ctx, aiToolGetGoodsInfo, map[string]any{"id": goodsID})
	tools := appendAiFlowTool(nil, usage)
	if err != nil {
		return r.aiFlowErrorResponse(aiFlowShopping, "sku", tools), nil
	}
	block := buildAiSkuSelectorBlock(output, mapValue(payload["recommend_context"]))
	return r.aiFlowResponse(aiFlowShopping, "sku", "这个商品可以选下面的规格和数量。", []map[string]any{block}, tools), nil
}

// openAiCheckout 打开订单确认流程。
func (r *Runner) openAiCheckout(ctx context.Context, payload map[string]any) (*ai.Response, error) {
	goodsID := int64Value(payload["goods_id"])
	skuCode := stringValue(payload["sku_code"])
	num := int64Value(payload["num"])
	if goodsID <= 0 || skuCode == "" {
		return nil, errorsx.InvalidArgument("商品规格参数不合法")
	}
	if num <= 0 {
		num = 1
	}
	selectedGoods := buildAiSelectedGoods(goodsID, skuCode, num, mapValue(payload["recommend_context"]))
	buyOutput, buyUsage, err := r.invokeAiFlowTool(ctx, aiToolBuyNowOrderInfo, map[string]any{
		"goods_id":          goodsID,
		"sku_code":          skuCode,
		"num":               num,
		"recommend_context": selectedGoods["recommend_context"],
	})
	tools := appendAiFlowTool(nil, buyUsage)
	if err != nil {
		return r.aiFlowErrorResponse(aiFlowShopping, "checkout", tools), nil
	}
	var addressOutput map[string]any
	var addressUsage ai.ToolUsage
	addressOutput, addressUsage, err = r.invokeAiFlowTool(ctx, aiToolListUserAddress, map[string]any{})
	tools = appendAiFlowTool(tools, addressUsage)
	if err != nil {
		return r.aiFlowErrorResponse(aiFlowShopping, "checkout", tools), nil
	}
	orderPayload := map[string]any{
		"goods":               []map[string]any{selectedGoods},
		"clear_cart":          boolValue(buyOutput["clear_cart"]),
		"pay_type":            int(shopcommonv1.OrderPayType_ONLINE_PAY),
		"pay_channel":         int(shopcommonv1.OrderPayChannel_WX_PAY),
		"order_store_options": buildAiOrderStoreOptions(buyOutput),
	}
	blocks := buildAiCheckoutBlocks(buyOutput, addressOutput, orderPayload)
	return r.aiFlowResponse(aiFlowShopping, "checkout", "规格已选好，再确认收货地址。", blocks, tools), nil
}

// createAiAddress 创建收货地址后回到订单确认流程。
func (r *Runner) createAiAddress(ctx context.Context, payload map[string]any) (*ai.Response, error) {
	userAddress := mapValue(payload["user_address"])
	if len(userAddress) == 0 {
		return nil, errorsx.InvalidArgument("收货地址不能为空")
	}
	orderPayload := mapValue(payload["order_payload"])
	flowName := aiFlowUserAddress
	if len(orderPayload) > 0 {
		flowName = aiFlowShopping
	}
	_, usage, err := r.invokeAiFlowTool(ctx, aiToolCreateUserAddress, map[string]any{"user_address": userAddress})
	tools := appendAiFlowTool(nil, usage)
	if err != nil {
		return r.aiFlowErrorResponse(flowName, "address", tools), nil
	}
	var addressOutput map[string]any
	var addressUsage ai.ToolUsage
	addressOutput, addressUsage, err = r.invokeAiFlowTool(ctx, aiToolListUserAddress, map[string]any{})
	tools = appendAiFlowTool(tools, addressUsage)
	if err != nil {
		return r.aiFlowErrorResponse(flowName, "address", tools), nil
	}
	if len(orderPayload) == 0 {
		blocks := []map[string]any{
			{"type": "success", "title": "地址已保存", "desc": "新的收货地址已经加入地址列表。"},
			buildAiAddressSelectorBlock(addressOutput, nil),
			buildAiAddressFormBlock(nil, aiFlowUserAddress),
		}
		return r.aiFlowResponse(aiFlowUserAddress, "address", "地址已经保存好了。", blocks, tools), nil
	}
	blocks := []map[string]any{
		{"type": "success", "title": "地址已保存", "desc": "可以继续选择这个地址下单。"},
		buildAiAddressSelectorBlock(addressOutput, orderPayload),
	}
	return r.aiFlowResponse(aiFlowShopping, "address", "地址已经加好了，选择一个地址继续确认订单。", blocks, tools), nil
}

// selectAiAddress 选择地址后展示最终确认。
func (r *Runner) selectAiAddress(payload map[string]any) *ai.Response {
	orderPayload := mapValue(payload["order_payload"])
	addressID := int64Value(payload["address_id"])
	orderPayload["address_id"] = addressID
	blocks := []map[string]any{
		{"type": "selected_address", "address": payload["address"]},
		{
			"type":    "confirm_order",
			"title":   "确认下单",
			"desc":    "订单将按当前商品、地址和在线支付方式创建。",
			"summary": payload["summary"],
			"action":  aiAction(aiFlowShopping, "confirm", "confirm_order", orderPayload),
		},
	}
	return r.aiFlowResponse(aiFlowShopping, "confirm", "地址已选好，确认无误后就可以提交订单。", blocks, nil)
}

// confirmAiOrder 创建订单并进入支付流程。
func (r *Runner) confirmAiOrder(ctx context.Context, payload map[string]any) (*ai.Response, error) {
	output, usage, err := r.invokeAiFlowTool(ctx, aiToolCreateOrderInfo, payload)
	tools := appendAiFlowTool(nil, usage)
	if err != nil {
		return r.aiFlowErrorResponse(aiFlowShopping, "payment", tools), nil
	}
	tradeID := int64Value(output["trade_id"])
	block := buildAiPaymentPanelBlock(tradeID)
	return r.aiFlowResponse(aiFlowShopping, "payment", "订单已创建，可以继续在聊天里发起支付。", []map[string]any{block}, tools), nil
}

// startAiPayment 调用支付工具并返回支付参数。
func (r *Runner) startAiPayment(ctx context.Context, payload map[string]any) (*ai.Response, error) {
	tradeID := int64Value(payload["trade_id"])
	if tradeID <= 0 {
		return nil, errorsx.InvalidArgument("交易单参数不合法")
	}
	platform := stringValue(payload["platform"])
	toolName := aiToolJSAPIPay
	if platform == "h5" || platform == "app" {
		toolName = aiToolH5Pay
	}
	output, usage, err := r.invokeAiFlowTool(ctx, toolName, map[string]any{"trade_id": tradeID})
	tools := appendAiFlowTool(nil, usage)
	if err != nil {
		return r.aiFlowErrorResponse(aiFlowPendingPayment, "payment", tools), nil
	}
	block := map[string]any{
		"type":     "payment_result",
		"trade_id": tradeID,
		"platform": platform,
		"pay_data": output,
	}
	return r.aiFlowResponse(aiFlowPendingPayment, "payment", "支付参数已准备好。", []map[string]any{block}, tools), nil
}

// openAiPendingPaymentFlow 打开待支付流程。
func (r *Runner) openAiPendingPaymentFlow(ctx context.Context) (*ai.Response, error) {
	output, usage, err := r.invokeAiFlowTool(ctx, aiToolPageOrderInfo, map[string]any{
		"trade_status": int(shopcommonv1.OrderTradeStatus_PENDING_PAYMENT_OTS),
		"page_num":     1,
		"page_size":    5,
	})
	tools := appendAiFlowTool(nil, usage)
	if err != nil {
		return r.aiFlowErrorResponse(aiFlowPendingPayment, "list", tools), nil
	}
	block := buildAiOrderListBlock(aiFlowPendingPayment, "待支付订单", output, "start_payment")
	return r.aiFlowResponse(aiFlowPendingPayment, "list", "这些订单还没有支付，可以直接在这里继续。", []map[string]any{block}, tools), nil
}

// openAiPendingReviewFlow 打开待评价流程。
func (r *Runner) openAiPendingReviewFlow(ctx context.Context) (*ai.Response, error) {
	output, usage, err := r.invokeAiFlowTool(ctx, aiToolPagePendingComment, map[string]any{
		"page_num":  1,
		"page_size": 5,
	})
	tools := appendAiFlowTool(nil, usage)
	if err != nil {
		return r.aiFlowErrorResponse(aiFlowPendingReview, "list", tools), nil
	}
	block := buildAiPendingReviewBlock(output)
	return r.aiFlowResponse(aiFlowPendingReview, "list", "找到这些待评价商品，选一个就能写评价。", []map[string]any{block}, tools), nil
}

// openAiReviewForm 打开评价表单。
func (r *Runner) openAiReviewForm(payload map[string]any) *ai.Response {
	block := map[string]any{
		"type":   "review_form",
		"title":  "写评价",
		"goods":  payload,
		"action": aiAction(aiFlowPendingReview, "submit", "submit_review", payload),
	}
	return r.aiFlowResponse(aiFlowPendingReview, "form", "可以直接在这里写评价。", []map[string]any{block}, nil)
}

// submitAiReview 提交商品评价。
func (r *Runner) submitAiReview(ctx context.Context, payload map[string]any) (*ai.Response, error) {
	if stringValue(payload["content"]) == "" {
		return nil, errorsx.InvalidArgument("评价内容不能为空")
	}
	if int64Value(payload["goods_score"]) <= 0 {
		payload["goods_score"] = 5
	}
	if int64Value(payload["package_score"]) <= 0 {
		payload["package_score"] = 5
	}
	if int64Value(payload["delivery_score"]) <= 0 {
		payload["delivery_score"] = 5
	}
	_, usage, err := r.invokeAiFlowTool(ctx, aiToolCreateComment, payload)
	tools := appendAiFlowTool(nil, usage)
	if err != nil {
		return r.aiFlowErrorResponse(aiFlowPendingReview, "submit", tools), nil
	}
	block := map[string]any{"type": "success", "title": "评价已提交", "desc": "评价提交成功，审核通过后会展示在商品页。"}
	return r.aiFlowResponse(aiFlowPendingReview, "done", "评价已经提交。", []map[string]any{block}, tools), nil
}

// openAiOrderLogisticsFlow 打开订单物流查询流程。
func (r *Runner) openAiOrderLogisticsFlow(ctx context.Context) (*ai.Response, error) {
	output, usage, err := r.invokeAiFlowTool(ctx, aiToolPageOrderInfo, map[string]any{
		"page_num":  1,
		"page_size": 5,
	})
	tools := appendAiFlowTool(nil, usage)
	if err != nil {
		return r.aiFlowErrorResponse(aiFlowOrderLogistics, "list", tools), nil
	}
	block := buildAiOrderListBlock(aiFlowOrderLogistics, "最近订单", output, "view_order")
	return r.aiFlowResponse(aiFlowOrderLogistics, "list", "这些是最近订单，选择一个查看物流和订单状态。", []map[string]any{block}, tools), nil
}

// viewAiOrder 查询订单详情和物流。
func (r *Runner) viewAiOrder(ctx context.Context, payload map[string]any) (*ai.Response, error) {
	orderID := int64Value(payload["order_id"])
	tradeID := int64Value(payload["trade_id"])
	if orderID <= 0 && tradeID <= 0 {
		return nil, errorsx.InvalidArgument("订单参数不合法")
	}
	toolName := aiToolGetOrderInfoByID
	toolPayload := map[string]any{"id": orderID}
	// 未支付或已关闭记录是交易聚合，必须按交易单查询全部门店商品。
	if tradeID > 0 {
		toolName = aiToolGetOrderTradeByID
		toolPayload = map[string]any{"trade_id": tradeID}
	}
	output, usage, err := r.invokeAiFlowTool(ctx, toolName, toolPayload)
	tools := appendAiFlowTool(nil, usage)
	if err != nil {
		return r.aiFlowErrorResponse(aiFlowOrderLogistics, "detail", tools), nil
	}
	block := buildAiOrderDetailBlock(output)
	return r.aiFlowResponse(aiFlowOrderLogistics, "detail", "订单详情和物流信息在这里。", []map[string]any{block}, tools), nil
}

// receiveAiOrder 确认收货后重新展示订单详情。
func (r *Runner) receiveAiOrder(ctx context.Context, payload map[string]any) (*ai.Response, error) {
	orderID := int64Value(payload["order_id"])
	if orderID <= 0 {
		return nil, errorsx.InvalidArgument("订单参数不合法")
	}
	_, usage, err := r.invokeAiFlowTool(ctx, aiToolReceiveOrderInfo, map[string]any{"order_id": orderID})
	tools := appendAiFlowTool(nil, usage)
	if err != nil {
		return r.aiFlowErrorResponse(aiFlowOrderLogistics, "receipt", tools), nil
	}
	var output map[string]any
	var detailUsage ai.ToolUsage
	output, detailUsage, err = r.invokeAiFlowTool(ctx, aiToolGetOrderInfoByID, map[string]any{"id": orderID})
	tools = appendAiFlowTool(tools, detailUsage)
	if err != nil {
		return r.aiFlowErrorResponse(aiFlowOrderLogistics, "detail", tools), nil
	}
	blocks := []map[string]any{
		{"type": "success", "title": "已确认收货", "desc": "订单已进入待评价流程。"},
		buildAiOrderDetailBlock(output),
	}
	return r.aiFlowResponse(aiFlowOrderLogistics, "detail", "已经确认收货。", blocks, tools), nil
}

// openAiUserCartFlow 打开购物车查询流程。
func (r *Runner) openAiUserCartFlow(ctx context.Context) (*ai.Response, error) {
	output, usage, err := r.invokeAiFlowTool(ctx, aiToolListUserCart, map[string]any{})
	tools := appendAiFlowTool(nil, usage)
	if err != nil {
		return r.aiFlowErrorResponse(aiFlowUserCart, "list", tools), nil
	}
	block := buildAiCartListBlock(output)
	return r.aiFlowResponse(aiFlowUserCart, "list", "购物车里的商品在这里。", []map[string]any{block}, tools), nil
}

// openAiUserCollectFlow 打开收藏商品查询流程。
func (r *Runner) openAiUserCollectFlow(ctx context.Context) (*ai.Response, error) {
	output, usage, err := r.invokeAiFlowTool(ctx, aiToolPageUserCollect, map[string]any{
		"page_num":  1,
		"page_size": 6,
	})
	tools := appendAiFlowTool(nil, usage)
	if err != nil {
		return r.aiFlowErrorResponse(aiFlowUserCollect, "list", tools), nil
	}
	block := buildAiGoodsListBlockFromItems("收藏商品", sliceMapValue(output["user_collects"]), 0)
	block["total"] = output["total"]
	return r.aiFlowResponse(aiFlowUserCollect, "list", "这些是你收藏过的商品，可以继续查看规格。", []map[string]any{block}, tools), nil
}

// openAiUserAddressFlow 打开收货地址管理流程。
func (r *Runner) openAiUserAddressFlow(ctx context.Context) (*ai.Response, error) {
	output, usage, err := r.invokeAiFlowTool(ctx, aiToolListUserAddress, map[string]any{})
	tools := appendAiFlowTool(nil, usage)
	if err != nil {
		return r.aiFlowErrorResponse(aiFlowUserAddress, "list", tools), nil
	}
	blocks := []map[string]any{
		buildAiAddressSelectorBlock(output, nil),
		buildAiAddressFormBlock(nil, aiFlowUserAddress),
	}
	return r.aiFlowResponse(aiFlowUserAddress, "list", "你的收货地址在这里，也可以继续新增一个地址。", blocks, tools), nil
}

// openAiUserProfileFlow 打开用户资料查询流程。
func (r *Runner) openAiUserProfileFlow(ctx context.Context) (*ai.Response, error) {
	output, usage, err := r.invokeAiFlowTool(ctx, aiToolGetUserProfile, map[string]any{})
	tools := appendAiFlowTool(nil, usage)
	if err != nil {
		return r.aiFlowErrorResponse(aiFlowUserProfile, "detail", tools), nil
	}
	block := buildAiProfilePanelBlock("个人资料", output, []aiProfileField{
		{label: "账号", key: "user_name"},
		{label: "昵称", key: "nick_name"},
		{label: "性别", key: "gender", format: aiGenderLabel},
		{label: "手机号", key: "phone"},
	})
	block["avatar"] = output["avatar"]
	return r.aiFlowResponse(aiFlowUserProfile, "detail", "你的个人资料在这里。", []map[string]any{block}, tools), nil
}

// openAiUserStoreFlow 打开用户门店查询流程。
func (r *Runner) openAiUserStoreFlow(ctx context.Context) (*ai.Response, error) {
	output, usage, err := r.invokeAiFlowTool(ctx, aiToolGetUserStore, map[string]any{})
	tools := appendAiFlowTool(nil, usage)
	if err != nil {
		return r.aiFlowErrorResponse(aiFlowUserStore, "detail", tools), nil
	}
	if int64Value(output["id"]) <= 0 {
		block := map[string]any{
			"type":  "notice",
			"title": "暂无门店入驻信息",
			"desc":  "当前账号还没有提交门店入驻资料。",
		}
		return r.aiFlowResponse(aiFlowUserStore, "detail", "当前还没有门店入驻信息。", []map[string]any{block}, tools), nil
	}
	block := buildAiProfilePanelBlock("门店入驻", output, []aiProfileField{
		{label: "门店名称", key: "name"},
		{label: "所在地区", key: "address_name", format: joinAnyStringList},
		{label: "详细地址", key: "detail"},
		{label: "审核状态", key: "status", format: aiUserStoreStatusLabel},
		{label: "备注", key: "remark"},
	})
	block["pictures"] = output["picture"]
	return r.aiFlowResponse(aiFlowUserStore, "detail", "你的门店入驻信息在这里。", []map[string]any{block}, tools), nil
}

// openAiGoodsCategoryFlow 打开商品分类查询流程。
func (r *Runner) openAiGoodsCategoryFlow(ctx context.Context) (*ai.Response, error) {
	output, usage, err := r.invokeAiFlowTool(ctx, aiToolListGoodsCategory, map[string]any{})
	tools := appendAiFlowTool(nil, usage)
	if err != nil {
		return r.aiFlowErrorResponse(aiFlowGoodsCategory, "list", tools), nil
	}
	block := buildAiCategoryListBlock(output)
	return r.aiFlowResponse(aiFlowGoodsCategory, "list", "这些分类可以继续点进去逛商品。", []map[string]any{block}, tools), nil
}

// viewAiGoodsCategory 查询指定分类下的商品。
func (r *Runner) viewAiGoodsCategory(ctx context.Context, payload map[string]any) (*ai.Response, error) {
	categoryID := int64Value(payload["category_id"])
	if categoryID <= 0 {
		return nil, errorsx.InvalidArgument("商品分类参数不合法")
	}
	output, usage, err := r.invokeAiFlowTool(ctx, aiToolPageGoodsInfo, map[string]any{
		"category_id": categoryID,
		"page_num":    1,
		"page_size":   6,
	})
	tools := appendAiFlowTool(nil, usage)
	if err != nil {
		return r.aiFlowErrorResponse(aiFlowGoodsCategory, "goods", tools), nil
	}
	title := stringValue(payload["category_name"])
	if title == "" {
		title = "分类商品"
	}
	block := buildAiGoodsListBlockFromItems(title, sliceMapValue(output["goods_infos"]), 0)
	block["total"] = output["total"]
	return r.aiFlowResponse(aiFlowGoodsCategory, "goods", "这个分类下有这些商品。", []map[string]any{block}, tools), nil
}

// openAiShopHotFlow 打开热门专区查询流程。
func (r *Runner) openAiShopHotFlow(ctx context.Context) (*ai.Response, error) {
	output, usage, err := r.invokeAiFlowTool(ctx, aiToolListShopHot, map[string]any{})
	tools := appendAiFlowTool(nil, usage)
	if err != nil {
		return r.aiFlowErrorResponse(aiFlowShopHot, "list", tools), nil
	}
	block := buildAiShopHotListBlock(output)
	return r.aiFlowResponse(aiFlowShopHot, "list", "这些热门专区可以继续查看。", []map[string]any{block}, tools), nil
}

// viewAiShopHotItem 查询热门专区选项或选项下商品。
func (r *Runner) viewAiShopHotItem(ctx context.Context, payload map[string]any) (*ai.Response, error) {
	hotItemID := int64Value(payload["hot_item_id"])
	if hotItemID > 0 {
		return r.viewAiShopHotGoods(ctx, hotItemID)
	}
	hotID := int64Value(payload["hot_id"])
	if hotID <= 0 {
		return nil, errorsx.InvalidArgument("热门专区参数不合法")
	}
	output, usage, err := r.invokeAiFlowTool(ctx, aiToolListShopHotItem, map[string]any{"id": hotID})
	tools := appendAiFlowTool(nil, usage)
	if err != nil {
		return r.aiFlowErrorResponse(aiFlowShopHot, "item", tools), nil
	}
	block := buildAiShopHotItemListBlock(output)
	return r.aiFlowResponse(aiFlowShopHot, "item", "这个专区下面还有这些热门选项。", []map[string]any{block}, tools), nil
}

// viewAiShopHotGoods 查询热门专区选项下商品。
func (r *Runner) viewAiShopHotGoods(ctx context.Context, hotItemID int64) (*ai.Response, error) {
	output, usage, err := r.invokeAiFlowTool(ctx, aiToolPageShopHotGoods, map[string]any{
		"hot_item_id": hotItemID,
		"page_num":    1,
		"page_size":   6,
	})
	tools := appendAiFlowTool(nil, usage)
	if err != nil {
		return r.aiFlowErrorResponse(aiFlowShopHot, "goods", tools), nil
	}
	block := buildAiGoodsListBlockFromItems("热门商品", sliceMapValue(output["goods_infos"]), 0)
	block["total"] = output["total"]
	return r.aiFlowResponse(aiFlowShopHot, "goods", "这个热门选项下有这些商品。", []map[string]any{block}, tools), nil
}

// openAiShopServiceFlow 打开商城服务说明流程。
func (r *Runner) openAiShopServiceFlow(ctx context.Context) (*ai.Response, error) {
	output, usage, err := r.invokeAiFlowTool(ctx, aiToolListShopService, map[string]any{})
	tools := appendAiFlowTool(nil, usage)
	if err != nil {
		return r.aiFlowErrorResponse(aiFlowShopService, "list", tools), nil
	}
	block := buildAiShopServiceListBlock(output)
	return r.aiFlowResponse(aiFlowShopService, "list", "商城服务说明在这里。", []map[string]any{block}, tools), nil
}

// invokeAiFlowTool 通过生成的 Agent Tool 调用业务能力。
func (r *Runner) invokeAiFlowTool(ctx context.Context, name string, input map[string]any) (map[string]any, ai.ToolUsage, error) {
	if r == nil || r.runtime == nil {
		return nil, ai.ToolUsage{}, errorsx.Internal("AI助手运行时未初始化")
	}
	raw, err := json.Marshal(input)
	if err != nil {
		return nil, ai.ToolUsage{}, errorsx.Internal("助手动作参数序列化失败").WithCause(err)
	}
	var result *ai.ToolInvokeResult
	result, err = r.runtime.InvokeTool(ctx, ai.NormalizeTerminalString(r.terminal), name, string(raw))
	if result == nil {
		return nil, ai.ToolUsage{}, err
	}
	output := make(map[string]any)
	if result.Output != "" {
		if decodeErr := json.Unmarshal([]byte(result.Output), &output); decodeErr != nil {
			return nil, result.Usage, errorsx.Internal("助手工具结果解析失败").WithCause(decodeErr)
		}
	}
	return output, result.Usage, err
}

// aiFlowErrorResponse 构造移动端流程失败提示。
func (r *Runner) aiFlowErrorResponse(flow string, step string, tools []ai.ToolUsage) *ai.Response {
	return r.aiFlowResponse(flow, step, "这个操作暂时没有完成，可以稍后再试或换一种方式继续。", []map[string]any{
		{"type": "notice", "title": "操作未完成", "desc": "当前步骤没有成功返回结果。"},
	}, tools)
}

// aiFlowResponse 构造移动端流程回复。
func (r *Runner) aiFlowResponse(flow string, step string, content string, blocks []map[string]any, tools []ai.ToolUsage) *ai.Response {
	model := ""
	if r != nil && r.runtime != nil {
		model = r.runtime.Model()
	}
	raw, err := json.Marshal(blocks)
	if err != nil {
		raw = []byte("[]")
	}
	return &ai.Response{
		Content:    content,
		Tools:      tools,
		Source:     "flow",
		Model:      model,
		Flow:       flow,
		Step:       step,
		BlocksJSON: string(raw),
	}
}

// buildAiGoodsListBlock 构造商品推荐卡片。
func buildAiGoodsListBlock(output map[string]any) map[string]any {
	goods := sliceMapValue(output["goods_infos"])
	requestID := int64Value(output["request_id"])
	return buildAiGoodsListBlockFromItems("推荐商品", goods, requestID)
}

// buildAiGoodsListBlockFromItems 按通用商品数组构造商品卡片。
func buildAiGoodsListBlockFromItems(title string, goods []map[string]any, requestID int64) map[string]any {
	items := make([]map[string]any, 0, len(goods))
	for index, item := range goods {
		goodsID := int64Value(item["goods_id"])
		if goodsID <= 0 {
			goodsID = int64Value(item["id"])
		}
		payload := map[string]any{
			"goods_id": goodsID,
			"recommend_context": map[string]any{
				"scene":      int(shopcommonv1.RecommendScene_PROFILE),
				"request_id": requestID,
				"position":   index + 1,
			},
		}
		items = append(items, map[string]any{
			"id":       goodsID,
			"name":     item["name"],
			"desc":     item["desc"],
			"picture":  item["picture"],
			"price":    item["price"],
			"sale_num": item["sale_num"],
			"action":   aiAction(aiFlowShopping, "sku", "select_goods", payload),
		})
	}
	return map[string]any{
		"type":  "goods_list",
		"title": title,
		"goods": items,
	}
}

// buildAiSkuSelectorBlock 构造规格选择卡片。
func buildAiSkuSelectorBlock(output map[string]any, recommendContext map[string]any) map[string]any {
	skus := sliceMapValue(output["sku_list"])
	items := make([]map[string]any, 0, len(skus))
	for _, item := range skus {
		items = append(items, map[string]any{
			"sku_code":  item["sku_code"],
			"spec_text": strings.Join(stringListValue(item["spec_item"]), " / "),
			"picture":   item["picture"],
			"price":     item["price"],
			"inventory": item["inventory"],
			"num":       1,
		})
	}
	goods := map[string]any{
		"id":      output["id"],
		"name":    output["name"],
		"desc":    output["desc"],
		"picture": output["picture"],
		"price":   output["price"],
	}
	return map[string]any{
		"type":   "sku_selector",
		"title":  "选择规格",
		"goods":  goods,
		"skus":   items,
		"action": aiAction(aiFlowShopping, "checkout", "select_sku", map[string]any{"goods_id": output["id"], "recommend_context": recommendContext}),
	}
}

// buildAiCheckoutBlocks 构造确认订单和地址选择卡片。
func buildAiCheckoutBlocks(orderOutput map[string]any, addressOutput map[string]any, orderPayload map[string]any) []map[string]any {
	blocks := []map[string]any{
		{
			"type":               "order_preview",
			"title":              "订单预览",
			"order_goods_stores": orderOutput["order_goods_stores"],
			"summary":            orderOutput["summary"],
			"order_payload":      orderPayload,
		},
	}
	addressBlock := buildAiAddressSelectorBlock(addressOutput, orderPayload)
	blocks = append(blocks, addressBlock)
	if len(sliceMapValue(addressOutput["user_addresses"])) == 0 {
		blocks = append(blocks, buildAiAddressFormBlock(orderPayload, aiFlowShopping))
	}
	return blocks
}

// buildAiAddressSelectorBlock 构造地址选择卡片。
func buildAiAddressSelectorBlock(output map[string]any, orderPayload map[string]any) map[string]any {
	addresses := sliceMapValue(output["user_addresses"])
	flowName := aiFlowUserAddress
	if len(orderPayload) > 0 {
		flowName = aiFlowShopping
	}
	items := make([]map[string]any, 0, len(addresses))
	for _, item := range addresses {
		addressID := int64Value(item["id"])
		payload := map[string]any{
			"address_id":    addressID,
			"address":       item,
			"order_payload": orderPayload,
		}
		items = append(items, map[string]any{
			"id":         addressID,
			"receiver":   item["receiver"],
			"contact":    item["contact"],
			"address":    item["address"],
			"detail":     item["detail"],
			"is_default": item["is_default"],
		})
		if len(orderPayload) > 0 {
			items[len(items)-1]["action"] = aiAction(flowName, "confirm", "select_address", payload)
		}
	}
	return map[string]any{
		"type":      "address_selector",
		"title":     "选择收货地址",
		"addresses": items,
	}
}

// buildAiAddressFormBlock 构造新增地址表单卡片。
func buildAiAddressFormBlock(orderPayload map[string]any, flowName string) map[string]any {
	if flowName == "" {
		flowName = aiFlowUserAddress
	}
	return map[string]any{
		"type":          "address_form",
		"title":         "新增收货地址",
		"order_payload": orderPayload,
		"action":        aiAction(flowName, "address", "create_address", map[string]any{"order_payload": orderPayload}),
	}
}

// buildAiSelectedGoods 构造创建订单所需商品项。
func buildAiSelectedGoods(goodsID int64, skuCode string, num int64, recommendContext map[string]any) map[string]any {
	return map[string]any{
		"goods_id":          goodsID,
		"sku_code":          skuCode,
		"num":               num,
		"recommend_context": recommendContext,
	}
}

// buildAiOrderStoreOptions 按后端确认单的门店分组构建默认配送选项。
func buildAiOrderStoreOptions(output map[string]any) []map[string]any {
	orderGoodsStores := sliceMapValue(output["order_goods_stores"])
	options := make([]map[string]any, 0, len(orderGoodsStores))
	for _, orderGoodsStore := range orderGoodsStores {
		store := mapValue(orderGoodsStore["store"])
		options = append(options, map[string]any{
			"tenant_store_id": int64Value(store["id"]),
			"delivery_time":   int(shopcommonv1.OrderDeliveryTime_ALL_TIME),
			"remark":          "",
		})
	}
	return options
}

// buildAiPaymentPanelBlock 构造交易单支付面板卡片。
func buildAiPaymentPanelBlock(tradeID int64) map[string]any {
	return map[string]any{
		"type":     "payment_panel",
		"title":    "订单支付",
		"trade_id": tradeID,
		"action":   aiAction(aiFlowPendingPayment, "payment", "start_payment", map[string]any{"trade_id": tradeID}),
	}
}

// buildAiOrderListBlock 构造订单列表卡片。
func buildAiOrderListBlock(flow string, title string, output map[string]any, actionType string) map[string]any {
	orders := sliceMapValue(output["order_infos"])
	items := make([]map[string]any, 0, len(orders))
	for _, item := range orders {
		orderID := int64Value(item["id"])
		payload := map[string]any{"order_id": orderID}
		step := "detail"
		if actionType == "start_payment" {
			step = "payment"
			payload = map[string]any{"trade_id": int64Value(item["trade_id"])}
		} else if boolValue(item["is_trade"]) {
			payload = map[string]any{"trade_id": int64Value(item["trade_id"])}
		}
		items = append(items, map[string]any{
			"id":                 orderID,
			"is_trade":           item["is_trade"],
			"trade_id":           item["trade_id"],
			"order_no":           item["order_no"],
			"pay_money":          item["pay_money"],
			"total_money":        item["total_money"],
			"status":             item["status"],
			"goods_num":          item["goods_num"],
			"order_goods_stores": item["order_goods_stores"],
			"action":             aiAction(flow, step, actionType, payload),
		})
	}
	return map[string]any{
		"type":   "order_list",
		"title":  title,
		"orders": items,
		"total":  output["total"],
	}
}

// buildAiPendingReviewBlock 构造待评价商品列表卡片。
func buildAiPendingReviewBlock(output map[string]any) map[string]any {
	values := sliceMapValue(output["pending_comment_goods"])
	if len(values) == 0 {
		values = sliceMapValue(output["goods"])
	}
	if len(values) == 0 {
		values = sliceMapValue(output["items"])
	}
	items := make([]map[string]any, 0, len(values))
	for _, item := range values {
		payload := map[string]any{
			"order_id":      item["order_id"],
			"goods_id":      item["goods_id"],
			"goods_name":    item["goods_name"],
			"goods_picture": item["goods_picture"],
			"sku_code":      item["sku_code"],
			"sku_desc":      item["sku_desc"],
		}
		items = append(items, map[string]any{
			"order_id":      item["order_id"],
			"goods_id":      item["goods_id"],
			"goods_name":    item["goods_name"],
			"goods_picture": item["goods_picture"],
			"sku_code":      item["sku_code"],
			"sku_desc":      item["sku_desc"],
			"desc":          item["desc"],
			"action":        aiAction(aiFlowPendingReview, "form", "open_review_form", payload),
		})
	}
	return map[string]any{
		"type":  "pending_review_list",
		"title": "待评价商品",
		"goods": items,
		"total": output["total"],
	}
}

// buildAiOrderDetailBlock 构造订单详情和物流卡片。
func buildAiOrderDetailBlock(output map[string]any) map[string]any {
	order := mapValue(output["order"])
	orderID := int64Value(order["id"])
	block := map[string]any{
		"type":      "order_logistics",
		"title":     "订单详情",
		"order":     order,
		"address":   output["address"],
		"logistics": output["logistics"],
		"countdown": output["countdown"],
	}
	if int64Value(order["status"]) == int64(shopcommonv1.OrderInfoStatus_SHIPPED_OIS) {
		block["action"] = aiAction(aiFlowOrderLogistics, "receipt", "receive_order", map[string]any{"order_id": orderID})
	}
	return block
}

// buildAiCartListBlock 构造购物车列表卡片。
func buildAiCartListBlock(output map[string]any) map[string]any {
	cartStores := sliceMapValue(output["user_cart_stores"])
	var items []map[string]any
	for _, cartStore := range cartStores {
		store := mapValue(cartStore["store"])
		for _, item := range sliceMapValue(cartStore["goods"]) {
			items = append(items, map[string]any{
				"id":        item["id"],
				"goods_id":  item["goods_id"],
				"name":      item["name"],
				"picture":   item["picture"],
				"sku_code":  item["sku_code"],
				"spec_text": strings.Join(stringListValue(item["spec_item"]), " / "),
				"num":       item["num"],
				"price":     item["price"],
				"checked":   item["is_checked"],
				"store":     store,
			})
		}
	}
	return map[string]any{
		"type":  "cart_list",
		"title": "购物车",
		"carts": items,
	}
}

// buildAiCategoryListBlock 构造商品分类列表卡片。
func buildAiCategoryListBlock(output map[string]any) map[string]any {
	categories := sliceMapValue(output["goods_categories"])
	items := make([]map[string]any, 0, len(categories))
	for _, item := range categories {
		categoryID := int64Value(item["id"])
		items = append(items, map[string]any{
			"id":      categoryID,
			"title":   item["name"],
			"picture": item["picture"],
			"desc":    buildAiCategoryDesc(item),
			"action": aiAction(aiFlowGoodsCategory, "goods", "view_goods_category", map[string]any{
				"category_id":   categoryID,
				"category_name": item["name"],
			}),
		})
	}
	return map[string]any{
		"type":  "simple_list",
		"title": "商品分类",
		"items": items,
	}
}

// buildAiCategoryDesc 构造分类辅助描述。
func buildAiCategoryDesc(item map[string]any) string {
	goods := sliceMapValue(item["goods"])
	if len(goods) == 0 {
		return "点击查看分类商品"
	}
	return fmt.Sprintf("%d 个精选商品", len(goods))
}

// buildAiShopHotListBlock 构造热门专区列表卡片。
func buildAiShopHotListBlock(output map[string]any) map[string]any {
	values := sliceMapValue(output["shop_hots"])
	items := make([]map[string]any, 0, len(values))
	for _, item := range values {
		hotID := int64Value(item["id"])
		items = append(items, map[string]any{
			"id":      hotID,
			"title":   item["title"],
			"desc":    item["desc"],
			"picture": firstStringValue(item["picture"]),
			"action":  aiAction(aiFlowShopHot, "item", "view_shop_hot_item", map[string]any{"hot_id": hotID}),
		})
	}
	return map[string]any{
		"type":  "simple_list",
		"title": "热门专区",
		"items": items,
	}
}

// buildAiShopHotItemListBlock 构造热门专区选项列表卡片。
func buildAiShopHotItemListBlock(output map[string]any) map[string]any {
	values := sliceMapValue(output["shop_hot_items"])
	items := make([]map[string]any, 0, len(values))
	for _, item := range values {
		hotItemID := int64Value(item["id"])
		items = append(items, map[string]any{
			"id":     hotItemID,
			"title":  item["title"],
			"desc":   "点击查看商品",
			"action": aiAction(aiFlowShopHot, "goods", "view_shop_hot_item", map[string]any{"hot_item_id": hotItemID}),
		})
	}
	title := stringValue(output["title"])
	if title == "" {
		title = "热门选项"
	}
	return map[string]any{
		"type":   "simple_list",
		"title":  title,
		"banner": output["banner"],
		"items":  items,
	}
}

// buildAiShopServiceListBlock 构造商城服务说明列表卡片。
func buildAiShopServiceListBlock(output map[string]any) map[string]any {
	values := sliceMapValue(output["shop_services"])
	items := make([]map[string]any, 0, len(values))
	for _, item := range values {
		items = append(items, map[string]any{
			"title": item["label"],
			"desc":  item["value"],
		})
	}
	return map[string]any{
		"type":  "simple_list",
		"title": "商城服务",
		"items": items,
	}
}

// buildAiProfilePanelBlock 构造资料详情卡片。
func buildAiProfilePanelBlock(title string, output map[string]any, fields []aiProfileField) map[string]any {
	items := make([]map[string]any, 0, len(fields))
	for _, field := range fields {
		value := stringValue(output[field.key])
		if field.format != nil {
			value = field.format(output[field.key])
		}
		if value == "" {
			value = "未填写"
		}
		items = append(items, map[string]any{
			"label": field.label,
			"value": value,
		})
	}
	return map[string]any{
		"type":   "profile_panel",
		"title":  title,
		"fields": items,
	}
}

// matchAiFlowIntent 根据用户文本识别移动端闭环流程。
func matchAiFlowIntent(content string) string {
	// 优先识别状态类流程，避免“待支付订单”被商品购买意图截走。
	if strings.Contains(content, "待支付") || strings.Contains(content, "付款") || strings.Contains(content, "支付") {
		return aiFlowPendingPayment
	}
	if strings.Contains(content, "待评价") || strings.Contains(content, "评价") {
		return aiFlowPendingReview
	}
	if strings.Contains(content, "物流") || strings.Contains(content, "查订单") || strings.Contains(content, "订单") || strings.Contains(content, "收货") || strings.Contains(content, "到哪") {
		return aiFlowOrderLogistics
	}
	if strings.Contains(content, "购物车") || strings.Contains(content, "加购") {
		return aiFlowUserCart
	}
	if strings.Contains(content, "收藏") {
		return aiFlowUserCollect
	}
	if strings.Contains(content, "地址") {
		return aiFlowUserAddress
	}
	if strings.Contains(content, "个人资料") || strings.Contains(content, "个人信息") || strings.Contains(content, "昵称") || strings.Contains(content, "头像") || strings.Contains(content, "手机号") {
		return aiFlowUserProfile
	}
	if strings.Contains(content, "门店") || strings.Contains(content, "入驻") || strings.Contains(content, "开店") {
		return aiFlowUserStore
	}
	if strings.Contains(content, "分类") || strings.Contains(content, "类目") {
		return aiFlowGoodsCategory
	}
	if strings.Contains(content, "热门") || strings.Contains(content, "热销") || strings.Contains(content, "专区") || strings.Contains(content, "榜") {
		return aiFlowShopHot
	}
	if strings.Contains(content, "服务") || strings.Contains(content, "保障") || strings.Contains(content, "说明") {
		return aiFlowShopService
	}
	if strings.Contains(content, "推荐") || strings.Contains(content, "下单") || strings.Contains(content, "购买") || strings.Contains(content, "买") || strings.Contains(content, "商品") {
		return aiFlowShopping
	}
	return ""
}

// openAiFlowActionType 返回流程入口动作。
func openAiFlowActionType(flow string) string {
	if actionType := aiFlowRegistry.EntryAction(einoWorkflow.FlowName(flow)); actionType != "" {
		return actionType
	}
	return aiFlowRegistry.EntryAction(einoWorkflow.FlowName(aiFlowShopping))
}

// parseAiActionPayload 解析前端动作负载。
func parseAiActionPayload(raw string) (map[string]any, error) {
	if raw == "" {
		return map[string]any{}, nil
	}
	result := make(map[string]any)
	err := json.Unmarshal([]byte(raw), &result)
	if err != nil {
		return nil, errorsx.InvalidArgument("助手动作参数不合法")
	}
	return result, nil
}

// aiAction 构造移动端按钮动作。
func aiAction(flow string, step string, actionType string, payload map[string]any) map[string]any {
	raw, err := json.Marshal(payload)
	if err != nil {
		raw = []byte("{}")
	}
	return map[string]any{
		"flow":         flow,
		"step":         step,
		"type":         actionType,
		"payload_json": string(raw),
	}
}

// appendAiFlowTool 追加有效工具调用记录。
func appendAiFlowTool(tools []ai.ToolUsage, usage ai.ToolUsage) []ai.ToolUsage {
	if usage.Name == "" {
		return tools
	}
	return append(tools, usage)
}

// mapValue 将任意 JSON 值收敛为对象。
func mapValue(value any) map[string]any {
	if result, ok := value.(map[string]any); ok {
		return result
	}
	return map[string]any{}
}

// sliceMapValue 将任意 JSON 值收敛为对象数组。
func sliceMapValue(value any) []map[string]any {
	values, ok := value.([]any)
	if !ok {
		return []map[string]any{}
	}
	result := make([]map[string]any, 0, len(values))
	for _, item := range values {
		if next, ok := item.(map[string]any); ok {
			result = append(result, next)
		}
	}
	return result
}

// stringListValue 将任意 JSON 值收敛为字符串数组。
func stringListValue(value any) []string {
	values, ok := value.([]any)
	if !ok {
		return []string{}
	}
	result := make([]string, 0, len(values))
	for _, item := range values {
		result = append(result, stringValue(item))
	}
	return result
}

// joinAnyStringList 将任意字符串数组拼接成展示文本。
func joinAnyStringList(value any) string {
	return strings.Join(stringListValue(value), " ")
}

// firstStringValue 返回字符串或字符串数组中的第一个非空值。
func firstStringValue(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case []any:
		for _, item := range v {
			text := stringValue(item)
			if text != "" {
				return text
			}
		}
	case []string:
		for _, item := range v {
			if item != "" {
				return item
			}
		}
	}
	return ""
}

// aiGenderLabel 转换用户性别枚举展示。
func aiGenderLabel(value any) string {
	switch int64Value(value) {
	case int64(systemcommonv1.BaseUserGender_SECRET):
		return "保密"
	case int64(systemcommonv1.BaseUserGender_BOY):
		return "男"
	case int64(systemcommonv1.BaseUserGender_GIRL):
		return "女"
	default:
		return "未填写"
	}
}

// aiUserStoreStatusLabel 转换门店审核状态展示。
func aiUserStoreStatusLabel(value any) string {
	switch int64Value(value) {
	case int64(shopcommonv1.UserStoreStatus_PENDING_REVIEW):
		return "待审核"
	case int64(shopcommonv1.UserStoreStatus_FAILED_REVIEW):
		return "审核失败"
	case int64(shopcommonv1.UserStoreStatus_APPROVED):
		return "审核通过"
	default:
		return "未提交"
	}
}

// stringValue 将任意 JSON 标量收敛为字符串。
func stringValue(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case float64:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case int:
		return strconv.Itoa(v)
	default:
		return ""
	}
}

// int64Value 将任意 JSON 标量收敛为 int64。
func int64Value(value any) int64 {
	switch v := value.(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	case string:
		result, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0
		}
		return result
	default:
		return 0
	}
}

// boolValue 将任意 JSON 标量收敛为 bool。
func boolValue(value any) bool {
	if result, ok := value.(bool); ok {
		return result
	}
	return false
}

package flow

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	basev1 "shop/api/gen/go/base/v1"
	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/agent/assistant"
	einoWorkflow "shop/pkg/agent/eino/workflow"
	"shop/pkg/errorsx"
)

const (
	aiAssistantFlowShopping       = string(einoWorkflow.FlowShopping)
	aiAssistantFlowPendingPayment = string(einoWorkflow.FlowPendingPayment)
	aiAssistantFlowPendingReview  = string(einoWorkflow.FlowPendingReview)
	aiAssistantFlowOrderLogistics = string(einoWorkflow.FlowOrderLogistics)
	aiAssistantFlowUserCart       = string(einoWorkflow.FlowUserCart)
	aiAssistantFlowUserCollect    = string(einoWorkflow.FlowUserCollect)
	aiAssistantFlowUserAddress    = string(einoWorkflow.FlowUserAddress)
	aiAssistantFlowUserProfile    = string(einoWorkflow.FlowUserProfile)
	aiAssistantFlowUserStore      = string(einoWorkflow.FlowUserStore)
	aiAssistantFlowGoodsCategory  = string(einoWorkflow.FlowGoodsCategory)
	aiAssistantFlowShopHot        = string(einoWorkflow.FlowShopHot)
	aiAssistantFlowShopService    = string(einoWorkflow.FlowShopService)

	aiAssistantToolGetUserProfile     = "app_v1_auth_service_get_user_profile"
	aiAssistantToolRecommendGoods     = "app_v1_recommend_service_recommend_goods"
	aiAssistantToolPageGoodsInfo      = "app_v1_goods_info_service_page_goods_info"
	aiAssistantToolGetGoodsInfo       = "app_v1_goods_info_service_get_goods_info"
	aiAssistantToolListGoodsCategory  = "app_v1_goods_category_service_list_goods_categories"
	aiAssistantToolBuyNowOrderInfo    = "app_v1_order_info_service_buy_now_order_info"
	aiAssistantToolCreateOrderInfo    = "app_v1_order_info_service_create_order_info"
	aiAssistantToolPageOrderInfo      = "app_v1_order_info_service_page_order_info"
	aiAssistantToolGetOrderInfoByID   = "app_v1_order_info_service_get_order_info_by_id"
	aiAssistantToolReceiveOrderInfo   = "app_v1_order_info_service_receive_order_info"
	aiAssistantToolListShopHots       = "app_v1_shop_hot_service_list_shop_hots"
	aiAssistantToolListShopHotItems   = "app_v1_shop_hot_service_list_shop_hot_items"
	aiAssistantToolPageShopHotGoods   = "app_v1_shop_hot_service_page_shop_hot_goods"
	aiAssistantToolListShopServices   = "app_v1_shop_service_service_list_shop_services"
	aiAssistantToolListUserAddresses  = "app_v1_user_address_service_list_user_addresses"
	aiAssistantToolCreateUserAddress  = "app_v1_user_address_service_create_user_address"
	aiAssistantToolListUserCarts      = "app_v1_user_cart_service_list_user_carts"
	aiAssistantToolPageUserCollects   = "app_v1_user_collect_service_page_user_collects"
	aiAssistantToolGetUserStore       = "app_v1_user_store_service_get_user_store"
	aiAssistantToolPagePendingComment = "app_v1_comment_service_page_pending_comment_goods"
	aiAssistantToolCreateComment      = "app_v1_comment_service_create_comment"
	aiAssistantToolJSAPIPay           = "app_v1_pay_service_jsapi_pay"
	aiAssistantToolH5Pay              = "app_v1_pay_service_h5_pay"
)

var aiAssistantFlowRegistry = einoWorkflow.MustNewAppRegistry[*assistant.Response]()

// Runner 编排移动端助手闭环流程。
type Runner struct {
	runtime  *assistant.Runtime
	terminal int32
}

// aiAssistantProfileField 表示资料面板中的字段展示规则。
type aiAssistantProfileField struct {
	label  string
	key    string
	format func(any) string
}

// GenerateReply 生成移动端闭环流程回复。
func GenerateReply(
	ctx context.Context,
	runtime *assistant.Runtime,
	terminal int32,
	content string,
	action *basev1.AiAssistantAction,
) (*assistant.Response, bool, error) {
	runner := &Runner{runtime: runtime, terminal: terminal}
	if action != nil && action.GetType() != "" {
		reply, err := runner.handleAiAssistantFlowAction(ctx, action)
		return reply, true, err
	}

	flow := matchAiAssistantFlowIntent(content)
	if flow == "" {
		return nil, false, nil
	}
	reply, err := runner.handleAiAssistantFlowAction(ctx, &basev1.AiAssistantAction{
		Flow: flow,
		Type: openAiAssistantFlowActionType(flow),
	})
	return reply, true, err
}

// handleAiAssistantFlowAction 推进移动端闭环流程。
func (r *Runner) handleAiAssistantFlowAction(ctx context.Context, action *basev1.AiAssistantAction) (*assistant.Response, error) {
	payload, err := parseAiAssistantActionPayload(action.GetPayloadJson())
	if err != nil {
		return nil, err
	}
	var result einoWorkflow.ActionResult[*assistant.Response]
	result, err = aiAssistantFlowRegistry.Run(ctx, einoWorkflow.ActionRequest{
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

// ExecuteWorkflowAction 执行 Eino Graph 路由后的移动端流程动作。
func (r *Runner) ExecuteWorkflowAction(ctx context.Context, action einoWorkflow.Action, payload map[string]any) (*assistant.Response, error) {
	// 按前端按钮提交的动作类型进入对应流程步骤；动作合法性由 eino/workflow 的 Graph 分支负责。
	switch action.Type {
	case "open_shopping":
		return r.openAiAssistantShoppingFlow(ctx)
	case "select_goods":
		return r.openAiAssistantSkuSelector(ctx, payload)
	case "select_sku":
		return r.openAiAssistantCheckout(ctx, payload)
	case "create_address":
		return r.createAiAssistantAddress(ctx, payload)
	case "select_address":
		return r.selectAiAssistantAddress(payload), nil
	case "confirm_order":
		return r.confirmAiAssistantOrder(ctx, payload)
	case "start_payment":
		return r.startAiAssistantPayment(ctx, payload)
	case "open_pending_payment":
		return r.openAiAssistantPendingPaymentFlow(ctx)
	case "open_pending_review":
		return r.openAiAssistantPendingReviewFlow(ctx)
	case "open_review_form":
		return r.openAiAssistantReviewForm(payload), nil
	case "submit_review":
		return r.submitAiAssistantReview(ctx, payload)
	case "open_order_logistics":
		return r.openAiAssistantOrderLogisticsFlow(ctx)
	case "view_order":
		return r.viewAiAssistantOrder(ctx, payload)
	case "receive_order":
		return r.receiveAiAssistantOrder(ctx, payload)
	case "open_user_cart":
		return r.openAiAssistantUserCartFlow(ctx)
	case "open_user_collect":
		return r.openAiAssistantUserCollectFlow(ctx)
	case "open_user_address":
		return r.openAiAssistantUserAddressFlow(ctx)
	case "open_user_profile":
		return r.openAiAssistantUserProfileFlow(ctx)
	case "open_user_store":
		return r.openAiAssistantUserStoreFlow(ctx)
	case "open_goods_category":
		return r.openAiAssistantGoodsCategoryFlow(ctx)
	case "view_goods_category":
		return r.viewAiAssistantGoodsCategory(ctx, payload)
	case "open_shop_hot":
		return r.openAiAssistantShopHotFlow(ctx)
	case "view_shop_hot_item":
		return r.viewAiAssistantShopHotItem(ctx, payload)
	case "open_shop_service":
		return r.openAiAssistantShopServiceFlow(ctx)
	default:
		return nil, errorsx.InvalidArgument("助手动作不支持")
	}
}

// openAiAssistantShoppingFlow 打开推荐下单流程。
func (r *Runner) openAiAssistantShoppingFlow(ctx context.Context) (*assistant.Response, error) {
	tools := make([]assistant.ToolUsage, 0, 2)
	output, usage, err := r.invokeAiAssistantFlowTool(ctx, aiAssistantToolRecommendGoods, map[string]any{
		"scene":     int(commonv1.RecommendScene_PROFILE),
		"page_num":  1,
		"page_size": 6,
	})
	tools = appendAiAssistantFlowTool(tools, usage)
	if err != nil {
		var fallbackUsage assistant.ToolUsage
		output, fallbackUsage, err = r.invokeAiAssistantFlowTool(ctx, aiAssistantToolPageGoodsInfo, map[string]any{
			"page_num":  1,
			"page_size": 6,
		})
		tools = appendAiAssistantFlowTool(tools, fallbackUsage)
	}
	if err != nil {
		return r.aiAssistantFlowErrorResponse(aiAssistantFlowShopping, "goods", tools), nil
	}
	return r.aiAssistantFlowResponse(aiAssistantFlowShopping, "goods", "先给你推荐这些商品，选一个继续看规格。", []map[string]any{
		buildAiAssistantGoodsListBlock(output),
	}, tools), nil
}

// openAiAssistantSkuSelector 打开商品规格选择。
func (r *Runner) openAiAssistantSkuSelector(ctx context.Context, payload map[string]any) (*assistant.Response, error) {
	goodsID := int64Value(payload["goods_id"])
	if goodsID <= 0 {
		return nil, errorsx.InvalidArgument("商品参数不合法")
	}
	output, usage, err := r.invokeAiAssistantFlowTool(ctx, aiAssistantToolGetGoodsInfo, map[string]any{"id": goodsID})
	tools := appendAiAssistantFlowTool(nil, usage)
	if err != nil {
		return r.aiAssistantFlowErrorResponse(aiAssistantFlowShopping, "sku", tools), nil
	}
	block := buildAiAssistantSkuSelectorBlock(output, mapValue(payload["recommend_context"]))
	return r.aiAssistantFlowResponse(aiAssistantFlowShopping, "sku", "这个商品可以选下面的规格和数量。", []map[string]any{block}, tools), nil
}

// openAiAssistantCheckout 打开订单确认流程。
func (r *Runner) openAiAssistantCheckout(ctx context.Context, payload map[string]any) (*assistant.Response, error) {
	goodsID := int64Value(payload["goods_id"])
	skuCode := stringValue(payload["sku_code"])
	num := int64Value(payload["num"])
	if goodsID <= 0 || skuCode == "" {
		return nil, errorsx.InvalidArgument("商品规格参数不合法")
	}
	if num <= 0 {
		num = 1
	}
	selectedGoods := buildAiAssistantSelectedGoods(goodsID, skuCode, num, mapValue(payload["recommend_context"]))
	buyOutput, buyUsage, err := r.invokeAiAssistantFlowTool(ctx, aiAssistantToolBuyNowOrderInfo, map[string]any{
		"goods_id":          goodsID,
		"sku_code":          skuCode,
		"num":               num,
		"recommend_context": selectedGoods["recommend_context"],
	})
	tools := appendAiAssistantFlowTool(nil, buyUsage)
	if err != nil {
		return r.aiAssistantFlowErrorResponse(aiAssistantFlowShopping, "checkout", tools), nil
	}
	addressOutput, addressUsage, err := r.invokeAiAssistantFlowTool(ctx, aiAssistantToolListUserAddresses, map[string]any{})
	tools = appendAiAssistantFlowTool(tools, addressUsage)
	if err != nil {
		return r.aiAssistantFlowErrorResponse(aiAssistantFlowShopping, "checkout", tools), nil
	}
	orderPayload := map[string]any{
		"goods":         []map[string]any{selectedGoods},
		"clear_cart":    boolValue(buyOutput["clear_cart"]),
		"pay_type":      int(commonv1.OrderPayType_ONLINE_PAY),
		"pay_channel":   int(commonv1.OrderPayChannel_WX_PAY),
		"delivery_time": int(commonv1.OrderDeliveryTime_ALL_TIME),
	}
	blocks := buildAiAssistantCheckoutBlocks(buyOutput, addressOutput, orderPayload)
	return r.aiAssistantFlowResponse(aiAssistantFlowShopping, "checkout", "规格已选好，再确认收货地址。", blocks, tools), nil
}

// createAiAssistantAddress 创建收货地址后回到订单确认流程。
func (r *Runner) createAiAssistantAddress(ctx context.Context, payload map[string]any) (*assistant.Response, error) {
	userAddress := mapValue(payload["user_address"])
	if len(userAddress) == 0 {
		return nil, errorsx.InvalidArgument("收货地址不能为空")
	}
	orderPayload := mapValue(payload["order_payload"])
	flowName := aiAssistantFlowUserAddress
	if len(orderPayload) > 0 {
		flowName = aiAssistantFlowShopping
	}
	_, usage, err := r.invokeAiAssistantFlowTool(ctx, aiAssistantToolCreateUserAddress, map[string]any{"user_address": userAddress})
	tools := appendAiAssistantFlowTool(nil, usage)
	if err != nil {
		return r.aiAssistantFlowErrorResponse(flowName, "address", tools), nil
	}
	addressOutput, addressUsage, err := r.invokeAiAssistantFlowTool(ctx, aiAssistantToolListUserAddresses, map[string]any{})
	tools = appendAiAssistantFlowTool(tools, addressUsage)
	if err != nil {
		return r.aiAssistantFlowErrorResponse(flowName, "address", tools), nil
	}
	if len(orderPayload) == 0 {
		blocks := []map[string]any{
			{"type": "success", "title": "地址已保存", "desc": "新的收货地址已经加入地址列表。"},
			buildAiAssistantAddressSelectorBlock(addressOutput, nil),
			buildAiAssistantAddressFormBlock(nil, aiAssistantFlowUserAddress),
		}
		return r.aiAssistantFlowResponse(aiAssistantFlowUserAddress, "address", "地址已经保存好了。", blocks, tools), nil
	}
	blocks := []map[string]any{
		{"type": "success", "title": "地址已保存", "desc": "可以继续选择这个地址下单。"},
		buildAiAssistantAddressSelectorBlock(addressOutput, orderPayload),
	}
	return r.aiAssistantFlowResponse(aiAssistantFlowShopping, "address", "地址已经加好了，选择一个地址继续确认订单。", blocks, tools), nil
}

// selectAiAssistantAddress 选择地址后展示最终确认。
func (r *Runner) selectAiAssistantAddress(payload map[string]any) *assistant.Response {
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
			"action":  aiAssistantAction(aiAssistantFlowShopping, "confirm", "confirm_order", orderPayload),
		},
	}
	return r.aiAssistantFlowResponse(aiAssistantFlowShopping, "confirm", "地址已选好，确认无误后就可以提交订单。", blocks, nil)
}

// confirmAiAssistantOrder 创建订单并进入支付流程。
func (r *Runner) confirmAiAssistantOrder(ctx context.Context, payload map[string]any) (*assistant.Response, error) {
	output, usage, err := r.invokeAiAssistantFlowTool(ctx, aiAssistantToolCreateOrderInfo, payload)
	tools := appendAiAssistantFlowTool(nil, usage)
	if err != nil {
		return r.aiAssistantFlowErrorResponse(aiAssistantFlowShopping, "payment", tools), nil
	}
	orderID := int64Value(output["order_id"])
	block := buildAiAssistantPaymentPanelBlock(orderID)
	return r.aiAssistantFlowResponse(aiAssistantFlowShopping, "payment", "订单已创建，可以继续在聊天里发起支付。", []map[string]any{block}, tools), nil
}

// startAiAssistantPayment 调用支付工具并返回支付参数。
func (r *Runner) startAiAssistantPayment(ctx context.Context, payload map[string]any) (*assistant.Response, error) {
	orderID := int64Value(payload["order_id"])
	if orderID <= 0 {
		return nil, errorsx.InvalidArgument("订单参数不合法")
	}
	platform := stringValue(payload["platform"])
	toolName := aiAssistantToolJSAPIPay
	if platform == "h5" || platform == "app" {
		toolName = aiAssistantToolH5Pay
	}
	output, usage, err := r.invokeAiAssistantFlowTool(ctx, toolName, map[string]any{"order_id": orderID})
	tools := appendAiAssistantFlowTool(nil, usage)
	if err != nil {
		return r.aiAssistantFlowErrorResponse(aiAssistantFlowPendingPayment, "payment", tools), nil
	}
	block := map[string]any{
		"type":     "payment_result",
		"order_id": orderID,
		"platform": platform,
		"pay_data": output,
	}
	return r.aiAssistantFlowResponse(aiAssistantFlowPendingPayment, "payment", "支付参数已准备好。", []map[string]any{block}, tools), nil
}

// openAiAssistantPendingPaymentFlow 打开待支付流程。
func (r *Runner) openAiAssistantPendingPaymentFlow(ctx context.Context) (*assistant.Response, error) {
	output, usage, err := r.invokeAiAssistantFlowTool(ctx, aiAssistantToolPageOrderInfo, map[string]any{
		"status":    int(commonv1.OrderStatus_CREATED),
		"page_num":  1,
		"page_size": 5,
	})
	tools := appendAiAssistantFlowTool(nil, usage)
	if err != nil {
		return r.aiAssistantFlowErrorResponse(aiAssistantFlowPendingPayment, "list", tools), nil
	}
	block := buildAiAssistantOrderListBlock(aiAssistantFlowPendingPayment, "待支付订单", output, "start_payment")
	return r.aiAssistantFlowResponse(aiAssistantFlowPendingPayment, "list", "这些订单还没有支付，可以直接在这里继续。", []map[string]any{block}, tools), nil
}

// openAiAssistantPendingReviewFlow 打开待评价流程。
func (r *Runner) openAiAssistantPendingReviewFlow(ctx context.Context) (*assistant.Response, error) {
	output, usage, err := r.invokeAiAssistantFlowTool(ctx, aiAssistantToolPagePendingComment, map[string]any{
		"page_num":  1,
		"page_size": 5,
	})
	tools := appendAiAssistantFlowTool(nil, usage)
	if err != nil {
		return r.aiAssistantFlowErrorResponse(aiAssistantFlowPendingReview, "list", tools), nil
	}
	block := buildAiAssistantPendingReviewBlock(output)
	return r.aiAssistantFlowResponse(aiAssistantFlowPendingReview, "list", "找到这些待评价商品，选一个就能写评价。", []map[string]any{block}, tools), nil
}

// openAiAssistantReviewForm 打开评价表单。
func (r *Runner) openAiAssistantReviewForm(payload map[string]any) *assistant.Response {
	block := map[string]any{
		"type":   "review_form",
		"title":  "写评价",
		"goods":  payload,
		"action": aiAssistantAction(aiAssistantFlowPendingReview, "submit", "submit_review", payload),
	}
	return r.aiAssistantFlowResponse(aiAssistantFlowPendingReview, "form", "可以直接在这里写评价。", []map[string]any{block}, nil)
}

// submitAiAssistantReview 提交商品评价。
func (r *Runner) submitAiAssistantReview(ctx context.Context, payload map[string]any) (*assistant.Response, error) {
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
	_, usage, err := r.invokeAiAssistantFlowTool(ctx, aiAssistantToolCreateComment, payload)
	tools := appendAiAssistantFlowTool(nil, usage)
	if err != nil {
		return r.aiAssistantFlowErrorResponse(aiAssistantFlowPendingReview, "submit", tools), nil
	}
	block := map[string]any{"type": "success", "title": "评价已提交", "desc": "评价提交成功，审核通过后会展示在商品页。"}
	return r.aiAssistantFlowResponse(aiAssistantFlowPendingReview, "done", "评价已经提交。", []map[string]any{block}, tools), nil
}

// openAiAssistantOrderLogisticsFlow 打开订单物流查询流程。
func (r *Runner) openAiAssistantOrderLogisticsFlow(ctx context.Context) (*assistant.Response, error) {
	output, usage, err := r.invokeAiAssistantFlowTool(ctx, aiAssistantToolPageOrderInfo, map[string]any{
		"page_num":  1,
		"page_size": 5,
	})
	tools := appendAiAssistantFlowTool(nil, usage)
	if err != nil {
		return r.aiAssistantFlowErrorResponse(aiAssistantFlowOrderLogistics, "list", tools), nil
	}
	block := buildAiAssistantOrderListBlock(aiAssistantFlowOrderLogistics, "最近订单", output, "view_order")
	return r.aiAssistantFlowResponse(aiAssistantFlowOrderLogistics, "list", "这些是最近订单，选择一个查看物流和订单状态。", []map[string]any{block}, tools), nil
}

// viewAiAssistantOrder 查询订单详情和物流。
func (r *Runner) viewAiAssistantOrder(ctx context.Context, payload map[string]any) (*assistant.Response, error) {
	orderID := int64Value(payload["order_id"])
	if orderID <= 0 {
		return nil, errorsx.InvalidArgument("订单参数不合法")
	}
	output, usage, err := r.invokeAiAssistantFlowTool(ctx, aiAssistantToolGetOrderInfoByID, map[string]any{"id": orderID})
	tools := appendAiAssistantFlowTool(nil, usage)
	if err != nil {
		return r.aiAssistantFlowErrorResponse(aiAssistantFlowOrderLogistics, "detail", tools), nil
	}
	block := buildAiAssistantOrderDetailBlock(output)
	return r.aiAssistantFlowResponse(aiAssistantFlowOrderLogistics, "detail", "订单详情和物流信息在这里。", []map[string]any{block}, tools), nil
}

// receiveAiAssistantOrder 确认收货后重新展示订单详情。
func (r *Runner) receiveAiAssistantOrder(ctx context.Context, payload map[string]any) (*assistant.Response, error) {
	orderID := int64Value(payload["order_id"])
	if orderID <= 0 {
		return nil, errorsx.InvalidArgument("订单参数不合法")
	}
	_, usage, err := r.invokeAiAssistantFlowTool(ctx, aiAssistantToolReceiveOrderInfo, map[string]any{"order_id": orderID})
	tools := appendAiAssistantFlowTool(nil, usage)
	if err != nil {
		return r.aiAssistantFlowErrorResponse(aiAssistantFlowOrderLogistics, "receipt", tools), nil
	}
	output, detailUsage, err := r.invokeAiAssistantFlowTool(ctx, aiAssistantToolGetOrderInfoByID, map[string]any{"id": orderID})
	tools = appendAiAssistantFlowTool(tools, detailUsage)
	if err != nil {
		return r.aiAssistantFlowErrorResponse(aiAssistantFlowOrderLogistics, "detail", tools), nil
	}
	blocks := []map[string]any{
		{"type": "success", "title": "已确认收货", "desc": "订单已进入待评价流程。"},
		buildAiAssistantOrderDetailBlock(output),
	}
	return r.aiAssistantFlowResponse(aiAssistantFlowOrderLogistics, "detail", "已经确认收货。", blocks, tools), nil
}

// openAiAssistantUserCartFlow 打开购物车查询流程。
func (r *Runner) openAiAssistantUserCartFlow(ctx context.Context) (*assistant.Response, error) {
	output, usage, err := r.invokeAiAssistantFlowTool(ctx, aiAssistantToolListUserCarts, map[string]any{})
	tools := appendAiAssistantFlowTool(nil, usage)
	if err != nil {
		return r.aiAssistantFlowErrorResponse(aiAssistantFlowUserCart, "list", tools), nil
	}
	block := buildAiAssistantCartListBlock(output)
	return r.aiAssistantFlowResponse(aiAssistantFlowUserCart, "list", "购物车里的商品在这里。", []map[string]any{block}, tools), nil
}

// openAiAssistantUserCollectFlow 打开收藏商品查询流程。
func (r *Runner) openAiAssistantUserCollectFlow(ctx context.Context) (*assistant.Response, error) {
	output, usage, err := r.invokeAiAssistantFlowTool(ctx, aiAssistantToolPageUserCollects, map[string]any{
		"page_num":  1,
		"page_size": 6,
	})
	tools := appendAiAssistantFlowTool(nil, usage)
	if err != nil {
		return r.aiAssistantFlowErrorResponse(aiAssistantFlowUserCollect, "list", tools), nil
	}
	block := buildAiAssistantGoodsListBlockFromItems("收藏商品", sliceMapValue(output["user_collects"]), 0)
	block["total"] = output["total"]
	return r.aiAssistantFlowResponse(aiAssistantFlowUserCollect, "list", "这些是你收藏过的商品，可以继续查看规格。", []map[string]any{block}, tools), nil
}

// openAiAssistantUserAddressFlow 打开收货地址管理流程。
func (r *Runner) openAiAssistantUserAddressFlow(ctx context.Context) (*assistant.Response, error) {
	output, usage, err := r.invokeAiAssistantFlowTool(ctx, aiAssistantToolListUserAddresses, map[string]any{})
	tools := appendAiAssistantFlowTool(nil, usage)
	if err != nil {
		return r.aiAssistantFlowErrorResponse(aiAssistantFlowUserAddress, "list", tools), nil
	}
	blocks := []map[string]any{
		buildAiAssistantAddressSelectorBlock(output, nil),
		buildAiAssistantAddressFormBlock(nil, aiAssistantFlowUserAddress),
	}
	return r.aiAssistantFlowResponse(aiAssistantFlowUserAddress, "list", "你的收货地址在这里，也可以继续新增一个地址。", blocks, tools), nil
}

// openAiAssistantUserProfileFlow 打开用户资料查询流程。
func (r *Runner) openAiAssistantUserProfileFlow(ctx context.Context) (*assistant.Response, error) {
	output, usage, err := r.invokeAiAssistantFlowTool(ctx, aiAssistantToolGetUserProfile, map[string]any{})
	tools := appendAiAssistantFlowTool(nil, usage)
	if err != nil {
		return r.aiAssistantFlowErrorResponse(aiAssistantFlowUserProfile, "detail", tools), nil
	}
	block := buildAiAssistantProfilePanelBlock("个人资料", output, []aiAssistantProfileField{
		{label: "账号", key: "user_name"},
		{label: "昵称", key: "nick_name"},
		{label: "性别", key: "gender", format: aiAssistantGenderLabel},
		{label: "手机号", key: "phone"},
	})
	block["avatar"] = output["avatar"]
	return r.aiAssistantFlowResponse(aiAssistantFlowUserProfile, "detail", "你的个人资料在这里。", []map[string]any{block}, tools), nil
}

// openAiAssistantUserStoreFlow 打开用户门店查询流程。
func (r *Runner) openAiAssistantUserStoreFlow(ctx context.Context) (*assistant.Response, error) {
	output, usage, err := r.invokeAiAssistantFlowTool(ctx, aiAssistantToolGetUserStore, map[string]any{})
	tools := appendAiAssistantFlowTool(nil, usage)
	if err != nil {
		return r.aiAssistantFlowErrorResponse(aiAssistantFlowUserStore, "detail", tools), nil
	}
	if int64Value(output["id"]) <= 0 {
		block := map[string]any{
			"type":  "notice",
			"title": "暂无门店入驻信息",
			"desc":  "当前账号还没有提交门店入驻资料。",
		}
		return r.aiAssistantFlowResponse(aiAssistantFlowUserStore, "detail", "当前还没有门店入驻信息。", []map[string]any{block}, tools), nil
	}
	block := buildAiAssistantProfilePanelBlock("门店入驻", output, []aiAssistantProfileField{
		{label: "门店名称", key: "name"},
		{label: "所在地区", key: "address_name", format: joinAnyStringList},
		{label: "详细地址", key: "detail"},
		{label: "审核状态", key: "status", format: aiAssistantUserStoreStatusLabel},
		{label: "备注", key: "remark"},
	})
	block["pictures"] = output["picture"]
	return r.aiAssistantFlowResponse(aiAssistantFlowUserStore, "detail", "你的门店入驻信息在这里。", []map[string]any{block}, tools), nil
}

// openAiAssistantGoodsCategoryFlow 打开商品分类查询流程。
func (r *Runner) openAiAssistantGoodsCategoryFlow(ctx context.Context) (*assistant.Response, error) {
	output, usage, err := r.invokeAiAssistantFlowTool(ctx, aiAssistantToolListGoodsCategory, map[string]any{})
	tools := appendAiAssistantFlowTool(nil, usage)
	if err != nil {
		return r.aiAssistantFlowErrorResponse(aiAssistantFlowGoodsCategory, "list", tools), nil
	}
	block := buildAiAssistantCategoryListBlock(output)
	return r.aiAssistantFlowResponse(aiAssistantFlowGoodsCategory, "list", "这些分类可以继续点进去逛商品。", []map[string]any{block}, tools), nil
}

// viewAiAssistantGoodsCategory 查询指定分类下的商品。
func (r *Runner) viewAiAssistantGoodsCategory(ctx context.Context, payload map[string]any) (*assistant.Response, error) {
	categoryID := int64Value(payload["category_id"])
	if categoryID <= 0 {
		return nil, errorsx.InvalidArgument("商品分类参数不合法")
	}
	output, usage, err := r.invokeAiAssistantFlowTool(ctx, aiAssistantToolPageGoodsInfo, map[string]any{
		"category_id": categoryID,
		"page_num":    1,
		"page_size":   6,
	})
	tools := appendAiAssistantFlowTool(nil, usage)
	if err != nil {
		return r.aiAssistantFlowErrorResponse(aiAssistantFlowGoodsCategory, "goods", tools), nil
	}
	title := stringValue(payload["category_name"])
	if title == "" {
		title = "分类商品"
	}
	block := buildAiAssistantGoodsListBlockFromItems(title, sliceMapValue(output["goods_infos"]), 0)
	block["total"] = output["total"]
	return r.aiAssistantFlowResponse(aiAssistantFlowGoodsCategory, "goods", "这个分类下有这些商品。", []map[string]any{block}, tools), nil
}

// openAiAssistantShopHotFlow 打开热门专区查询流程。
func (r *Runner) openAiAssistantShopHotFlow(ctx context.Context) (*assistant.Response, error) {
	output, usage, err := r.invokeAiAssistantFlowTool(ctx, aiAssistantToolListShopHots, map[string]any{})
	tools := appendAiAssistantFlowTool(nil, usage)
	if err != nil {
		return r.aiAssistantFlowErrorResponse(aiAssistantFlowShopHot, "list", tools), nil
	}
	block := buildAiAssistantShopHotListBlock(output)
	return r.aiAssistantFlowResponse(aiAssistantFlowShopHot, "list", "这些热门专区可以继续查看。", []map[string]any{block}, tools), nil
}

// viewAiAssistantShopHotItem 查询热门专区选项或选项下商品。
func (r *Runner) viewAiAssistantShopHotItem(ctx context.Context, payload map[string]any) (*assistant.Response, error) {
	hotItemID := int64Value(payload["hot_item_id"])
	if hotItemID > 0 {
		return r.viewAiAssistantShopHotGoods(ctx, hotItemID)
	}
	hotID := int64Value(payload["hot_id"])
	if hotID <= 0 {
		return nil, errorsx.InvalidArgument("热门专区参数不合法")
	}
	output, usage, err := r.invokeAiAssistantFlowTool(ctx, aiAssistantToolListShopHotItems, map[string]any{"id": hotID})
	tools := appendAiAssistantFlowTool(nil, usage)
	if err != nil {
		return r.aiAssistantFlowErrorResponse(aiAssistantFlowShopHot, "item", tools), nil
	}
	block := buildAiAssistantShopHotItemListBlock(output)
	return r.aiAssistantFlowResponse(aiAssistantFlowShopHot, "item", "这个专区下面还有这些热门选项。", []map[string]any{block}, tools), nil
}

// viewAiAssistantShopHotGoods 查询热门专区选项下商品。
func (r *Runner) viewAiAssistantShopHotGoods(ctx context.Context, hotItemID int64) (*assistant.Response, error) {
	output, usage, err := r.invokeAiAssistantFlowTool(ctx, aiAssistantToolPageShopHotGoods, map[string]any{
		"hot_item_id": hotItemID,
		"page_num":    1,
		"page_size":   6,
	})
	tools := appendAiAssistantFlowTool(nil, usage)
	if err != nil {
		return r.aiAssistantFlowErrorResponse(aiAssistantFlowShopHot, "goods", tools), nil
	}
	block := buildAiAssistantGoodsListBlockFromItems("热门商品", sliceMapValue(output["goods_infos"]), 0)
	block["total"] = output["total"]
	return r.aiAssistantFlowResponse(aiAssistantFlowShopHot, "goods", "这个热门选项下有这些商品。", []map[string]any{block}, tools), nil
}

// openAiAssistantShopServiceFlow 打开商城服务说明流程。
func (r *Runner) openAiAssistantShopServiceFlow(ctx context.Context) (*assistant.Response, error) {
	output, usage, err := r.invokeAiAssistantFlowTool(ctx, aiAssistantToolListShopServices, map[string]any{})
	tools := appendAiAssistantFlowTool(nil, usage)
	if err != nil {
		return r.aiAssistantFlowErrorResponse(aiAssistantFlowShopService, "list", tools), nil
	}
	block := buildAiAssistantShopServiceListBlock(output)
	return r.aiAssistantFlowResponse(aiAssistantFlowShopService, "list", "商城服务说明在这里。", []map[string]any{block}, tools), nil
}

// invokeAiAssistantFlowTool 通过生成的 Agent Tool 调用业务能力。
func (r *Runner) invokeAiAssistantFlowTool(ctx context.Context, name string, input map[string]any) (map[string]any, assistant.ToolUsage, error) {
	if r == nil || r.runtime == nil {
		return nil, assistant.ToolUsage{}, errorsx.Internal("AI助手运行时未初始化")
	}
	raw, err := json.Marshal(input)
	if err != nil {
		return nil, assistant.ToolUsage{}, errorsx.Internal("助手动作参数序列化失败").WithCause(err)
	}
	var result *assistant.ToolInvokeResult
	result, err = r.runtime.InvokeTool(ctx, assistant.NormalizeTerminalString(r.terminal), name, string(raw))
	if result == nil {
		return nil, assistant.ToolUsage{}, err
	}
	output := make(map[string]any)
	if result.Output != "" {
		if decodeErr := json.Unmarshal([]byte(result.Output), &output); decodeErr != nil {
			return nil, result.Usage, errorsx.Internal("助手工具结果解析失败").WithCause(decodeErr)
		}
	}
	return output, result.Usage, err
}

// aiAssistantFlowResponse 构造移动端流程回复。
func (r *Runner) aiAssistantFlowResponse(flow string, step string, content string, blocks []map[string]any, tools []assistant.ToolUsage) *assistant.Response {
	model := ""
	if r != nil && r.runtime != nil {
		model = r.runtime.Model()
	}
	raw, err := json.Marshal(blocks)
	if err != nil {
		raw = []byte("[]")
	}
	return &assistant.Response{
		Content:    content,
		Tools:      tools,
		Source:     "flow",
		Model:      model,
		Flow:       flow,
		Step:       step,
		BlocksJSON: string(raw),
	}
}

// aiAssistantFlowErrorResponse 构造移动端流程失败提示。
func (r *Runner) aiAssistantFlowErrorResponse(flow string, step string, tools []assistant.ToolUsage) *assistant.Response {
	return r.aiAssistantFlowResponse(flow, step, "这个操作暂时没有完成，可以稍后再试或换一种方式继续。", []map[string]any{
		{"type": "notice", "title": "操作未完成", "desc": "当前步骤没有成功返回结果。"},
	}, tools)
}

// buildAiAssistantGoodsListBlock 构造商品推荐卡片。
func buildAiAssistantGoodsListBlock(output map[string]any) map[string]any {
	goods := sliceMapValue(output["goods_infos"])
	requestID := int64Value(output["request_id"])
	return buildAiAssistantGoodsListBlockFromItems("推荐商品", goods, requestID)
}

// buildAiAssistantGoodsListBlockFromItems 按通用商品数组构造商品卡片。
func buildAiAssistantGoodsListBlockFromItems(title string, goods []map[string]any, requestID int64) map[string]any {
	items := make([]map[string]any, 0, len(goods))
	for index, item := range goods {
		goodsID := int64Value(item["goods_id"])
		if goodsID <= 0 {
			goodsID = int64Value(item["id"])
		}
		payload := map[string]any{
			"goods_id": goodsID,
			"recommend_context": map[string]any{
				"scene":      int(commonv1.RecommendScene_PROFILE),
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
			"action":   aiAssistantAction(aiAssistantFlowShopping, "sku", "select_goods", payload),
		})
	}
	return map[string]any{
		"type":  "goods_list",
		"title": title,
		"goods": items,
	}
}

// buildAiAssistantSkuSelectorBlock 构造规格选择卡片。
func buildAiAssistantSkuSelectorBlock(output map[string]any, recommendContext map[string]any) map[string]any {
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
		"action": aiAssistantAction(aiAssistantFlowShopping, "checkout", "select_sku", map[string]any{"goods_id": output["id"], "recommend_context": recommendContext}),
	}
}

// buildAiAssistantCheckoutBlocks 构造确认订单和地址选择卡片。
func buildAiAssistantCheckoutBlocks(orderOutput map[string]any, addressOutput map[string]any, orderPayload map[string]any) []map[string]any {
	blocks := []map[string]any{
		{
			"type":          "order_preview",
			"title":         "订单预览",
			"goods":         orderOutput["goods"],
			"summary":       orderOutput["summary"],
			"order_payload": orderPayload,
		},
	}
	addressBlock := buildAiAssistantAddressSelectorBlock(addressOutput, orderPayload)
	blocks = append(blocks, addressBlock)
	if len(sliceMapValue(addressOutput["user_addresses"])) == 0 {
		blocks = append(blocks, buildAiAssistantAddressFormBlock(orderPayload, aiAssistantFlowShopping))
	}
	return blocks
}

// buildAiAssistantAddressSelectorBlock 构造地址选择卡片。
func buildAiAssistantAddressSelectorBlock(output map[string]any, orderPayload map[string]any) map[string]any {
	addresses := sliceMapValue(output["user_addresses"])
	flowName := aiAssistantFlowUserAddress
	if len(orderPayload) > 0 {
		flowName = aiAssistantFlowShopping
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
			items[len(items)-1]["action"] = aiAssistantAction(flowName, "confirm", "select_address", payload)
		}
	}
	return map[string]any{
		"type":      "address_selector",
		"title":     "选择收货地址",
		"addresses": items,
	}
}

// buildAiAssistantAddressFormBlock 构造新增地址表单卡片。
func buildAiAssistantAddressFormBlock(orderPayload map[string]any, flowName string) map[string]any {
	if flowName == "" {
		flowName = aiAssistantFlowUserAddress
	}
	return map[string]any{
		"type":          "address_form",
		"title":         "新增收货地址",
		"order_payload": orderPayload,
		"action":        aiAssistantAction(flowName, "address", "create_address", map[string]any{"order_payload": orderPayload}),
	}
}

// buildAiAssistantSelectedGoods 构造创建订单所需商品项。
func buildAiAssistantSelectedGoods(goodsID int64, skuCode string, num int64, recommendContext map[string]any) map[string]any {
	return map[string]any{
		"goods_id":          goodsID,
		"sku_code":          skuCode,
		"num":               num,
		"recommend_context": recommendContext,
	}
}

// buildAiAssistantPaymentPanelBlock 构造支付面板卡片。
func buildAiAssistantPaymentPanelBlock(orderID int64) map[string]any {
	return map[string]any{
		"type":     "payment_panel",
		"title":    "订单支付",
		"order_id": orderID,
		"action":   aiAssistantAction(aiAssistantFlowPendingPayment, "payment", "start_payment", map[string]any{"order_id": orderID}),
	}
}

// buildAiAssistantOrderListBlock 构造订单列表卡片。
func buildAiAssistantOrderListBlock(flow string, title string, output map[string]any, actionType string) map[string]any {
	orders := sliceMapValue(output["order_infos"])
	items := make([]map[string]any, 0, len(orders))
	for _, item := range orders {
		orderID := int64Value(item["id"])
		payload := map[string]any{"order_id": orderID}
		step := "detail"
		if actionType == "start_payment" {
			step = "payment"
		}
		items = append(items, map[string]any{
			"id":          orderID,
			"order_no":    item["order_no"],
			"pay_money":   item["pay_money"],
			"total_money": item["total_money"],
			"status":      item["status"],
			"goods_num":   item["goods_num"],
			"goods":       item["goods"],
			"action":      aiAssistantAction(flow, step, actionType, payload),
		})
	}
	return map[string]any{
		"type":   "order_list",
		"title":  title,
		"orders": items,
		"total":  output["total"],
	}
}

// buildAiAssistantPendingReviewBlock 构造待评价商品列表卡片。
func buildAiAssistantPendingReviewBlock(output map[string]any) map[string]any {
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
			"action":        aiAssistantAction(aiAssistantFlowPendingReview, "form", "open_review_form", payload),
		})
	}
	return map[string]any{
		"type":  "pending_review_list",
		"title": "待评价商品",
		"goods": items,
		"total": output["total"],
	}
}

// buildAiAssistantOrderDetailBlock 构造订单详情和物流卡片。
func buildAiAssistantOrderDetailBlock(output map[string]any) map[string]any {
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
	if int64Value(order["status"]) == int64(commonv1.OrderStatus_SHIPPED) {
		block["action"] = aiAssistantAction(aiAssistantFlowOrderLogistics, "receipt", "receive_order", map[string]any{"order_id": orderID})
	}
	return block
}

// buildAiAssistantCartListBlock 构造购物车列表卡片。
func buildAiAssistantCartListBlock(output map[string]any) map[string]any {
	carts := sliceMapValue(output["user_carts"])
	items := make([]map[string]any, 0, len(carts))
	for _, item := range carts {
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
		})
	}
	return map[string]any{
		"type":  "cart_list",
		"title": "购物车",
		"carts": items,
	}
}

// buildAiAssistantCategoryListBlock 构造商品分类列表卡片。
func buildAiAssistantCategoryListBlock(output map[string]any) map[string]any {
	categories := sliceMapValue(output["goods_categories"])
	items := make([]map[string]any, 0, len(categories))
	for _, item := range categories {
		categoryID := int64Value(item["id"])
		items = append(items, map[string]any{
			"id":      categoryID,
			"title":   item["name"],
			"picture": item["picture"],
			"desc":    buildAiAssistantCategoryDesc(item),
			"action": aiAssistantAction(aiAssistantFlowGoodsCategory, "goods", "view_goods_category", map[string]any{
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

// buildAiAssistantCategoryDesc 构造分类辅助描述。
func buildAiAssistantCategoryDesc(item map[string]any) string {
	goods := sliceMapValue(item["goods"])
	if len(goods) == 0 {
		return "点击查看分类商品"
	}
	return fmt.Sprintf("%d 个精选商品", len(goods))
}

// buildAiAssistantShopHotListBlock 构造热门专区列表卡片。
func buildAiAssistantShopHotListBlock(output map[string]any) map[string]any {
	values := sliceMapValue(output["shop_hots"])
	items := make([]map[string]any, 0, len(values))
	for _, item := range values {
		hotID := int64Value(item["id"])
		items = append(items, map[string]any{
			"id":      hotID,
			"title":   item["title"],
			"desc":    item["desc"],
			"picture": firstStringValue(item["picture"]),
			"action":  aiAssistantAction(aiAssistantFlowShopHot, "item", "view_shop_hot_item", map[string]any{"hot_id": hotID}),
		})
	}
	return map[string]any{
		"type":  "simple_list",
		"title": "热门专区",
		"items": items,
	}
}

// buildAiAssistantShopHotItemListBlock 构造热门专区选项列表卡片。
func buildAiAssistantShopHotItemListBlock(output map[string]any) map[string]any {
	values := sliceMapValue(output["shop_hot_items"])
	items := make([]map[string]any, 0, len(values))
	for _, item := range values {
		hotItemID := int64Value(item["id"])
		items = append(items, map[string]any{
			"id":     hotItemID,
			"title":  item["title"],
			"desc":   "点击查看商品",
			"action": aiAssistantAction(aiAssistantFlowShopHot, "goods", "view_shop_hot_item", map[string]any{"hot_item_id": hotItemID}),
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

// buildAiAssistantShopServiceListBlock 构造商城服务说明列表卡片。
func buildAiAssistantShopServiceListBlock(output map[string]any) map[string]any {
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

// buildAiAssistantProfilePanelBlock 构造资料详情卡片。
func buildAiAssistantProfilePanelBlock(title string, output map[string]any, fields []aiAssistantProfileField) map[string]any {
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

// matchAiAssistantFlowIntent 根据用户文本识别移动端闭环流程。
func matchAiAssistantFlowIntent(content string) string {
	// 优先识别状态类流程，避免“待支付订单”被商品购买意图截走。
	if strings.Contains(content, "待支付") || strings.Contains(content, "付款") || strings.Contains(content, "支付") {
		return aiAssistantFlowPendingPayment
	}
	if strings.Contains(content, "待评价") || strings.Contains(content, "评价") {
		return aiAssistantFlowPendingReview
	}
	if strings.Contains(content, "物流") || strings.Contains(content, "查订单") || strings.Contains(content, "订单") || strings.Contains(content, "收货") || strings.Contains(content, "到哪") {
		return aiAssistantFlowOrderLogistics
	}
	if strings.Contains(content, "购物车") || strings.Contains(content, "加购") {
		return aiAssistantFlowUserCart
	}
	if strings.Contains(content, "收藏") {
		return aiAssistantFlowUserCollect
	}
	if strings.Contains(content, "地址") {
		return aiAssistantFlowUserAddress
	}
	if strings.Contains(content, "个人资料") || strings.Contains(content, "个人信息") || strings.Contains(content, "昵称") || strings.Contains(content, "头像") || strings.Contains(content, "手机号") {
		return aiAssistantFlowUserProfile
	}
	if strings.Contains(content, "门店") || strings.Contains(content, "入驻") || strings.Contains(content, "开店") {
		return aiAssistantFlowUserStore
	}
	if strings.Contains(content, "分类") || strings.Contains(content, "类目") {
		return aiAssistantFlowGoodsCategory
	}
	if strings.Contains(content, "热门") || strings.Contains(content, "热销") || strings.Contains(content, "专区") || strings.Contains(content, "榜") {
		return aiAssistantFlowShopHot
	}
	if strings.Contains(content, "服务") || strings.Contains(content, "保障") || strings.Contains(content, "说明") {
		return aiAssistantFlowShopService
	}
	if strings.Contains(content, "推荐") || strings.Contains(content, "下单") || strings.Contains(content, "购买") || strings.Contains(content, "买") || strings.Contains(content, "商品") {
		return aiAssistantFlowShopping
	}
	return ""
}

// openAiAssistantFlowActionType 返回流程入口动作。
func openAiAssistantFlowActionType(flow string) string {
	if actionType := aiAssistantFlowRegistry.EntryAction(einoWorkflow.FlowName(flow)); actionType != "" {
		return actionType
	}
	return aiAssistantFlowRegistry.EntryAction(einoWorkflow.FlowShopping)
}

// parseAiAssistantActionPayload 解析前端动作负载。
func parseAiAssistantActionPayload(raw string) (map[string]any, error) {
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

// aiAssistantAction 构造移动端按钮动作。
func aiAssistantAction(flow string, step string, actionType string, payload map[string]any) map[string]any {
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

// appendAiAssistantFlowTool 追加有效工具调用记录。
func appendAiAssistantFlowTool(tools []assistant.ToolUsage, usage assistant.ToolUsage) []assistant.ToolUsage {
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

// aiAssistantGenderLabel 转换用户性别枚举展示。
func aiAssistantGenderLabel(value any) string {
	switch int64Value(value) {
	case int64(commonv1.BaseUserGender_SECRET):
		return "保密"
	case int64(commonv1.BaseUserGender_BOY):
		return "男"
	case int64(commonv1.BaseUserGender_GIRL):
		return "女"
	default:
		return "未填写"
	}
}

// aiAssistantUserStoreStatusLabel 转换门店审核状态展示。
func aiAssistantUserStoreStatusLabel(value any) string {
	switch int64Value(value) {
	case int64(commonv1.UserStoreStatus_PENDING_REVIEW):
		return "待审核"
	case int64(commonv1.UserStoreStatus_FAILED_REVIEW):
		return "审核失败"
	case int64(commonv1.UserStoreStatus_APPROVED):
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

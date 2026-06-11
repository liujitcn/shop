package flow

import (
	"context"
	"encoding/json"
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

	aiAssistantToolRecommendGoods     = "app_v1_recommend_service_recommend_goods"
	aiAssistantToolPageGoodsInfo      = "app_v1_goods_info_service_page_goods_info"
	aiAssistantToolGetGoodsInfo       = "app_v1_goods_info_service_get_goods_info"
	aiAssistantToolBuyNowOrderInfo    = "app_v1_order_info_service_buy_now_order_info"
	aiAssistantToolCreateOrderInfo    = "app_v1_order_info_service_create_order_info"
	aiAssistantToolPageOrderInfo      = "app_v1_order_info_service_page_order_info"
	aiAssistantToolGetOrderInfoByID   = "app_v1_order_info_service_get_order_info_by_id"
	aiAssistantToolReceiveOrderInfo   = "app_v1_order_info_service_receive_order_info"
	aiAssistantToolListUserAddresses  = "app_v1_user_address_service_list_user_addresses"
	aiAssistantToolCreateUserAddress  = "app_v1_user_address_service_create_user_address"
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
	_, usage, err := r.invokeAiAssistantFlowTool(ctx, aiAssistantToolCreateUserAddress, map[string]any{"user_address": userAddress})
	tools := appendAiAssistantFlowTool(nil, usage)
	if err != nil {
		return r.aiAssistantFlowErrorResponse(aiAssistantFlowShopping, "address", tools), nil
	}
	addressOutput, addressUsage, err := r.invokeAiAssistantFlowTool(ctx, aiAssistantToolListUserAddresses, map[string]any{})
	tools = appendAiAssistantFlowTool(tools, addressUsage)
	if err != nil {
		return r.aiAssistantFlowErrorResponse(aiAssistantFlowShopping, "address", tools), nil
	}
	blocks := []map[string]any{
		{"type": "success", "title": "地址已保存", "desc": "可以继续选择这个地址下单。"},
		buildAiAssistantAddressSelectorBlock(addressOutput, mapValue(payload["order_payload"])),
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
	items := make([]map[string]any, 0, len(goods))
	for index, item := range goods {
		goodsID := int64Value(item["id"])
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
		"title": "推荐商品",
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
		blocks = append(blocks, buildAiAssistantAddressFormBlock(orderPayload))
	}
	return blocks
}

// buildAiAssistantAddressSelectorBlock 构造地址选择卡片。
func buildAiAssistantAddressSelectorBlock(output map[string]any, orderPayload map[string]any) map[string]any {
	addresses := sliceMapValue(output["user_addresses"])
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
			"action":     aiAssistantAction(aiAssistantFlowShopping, "confirm", "select_address", payload),
		})
	}
	return map[string]any{
		"type":      "address_selector",
		"title":     "选择收货地址",
		"addresses": items,
	}
}

// buildAiAssistantAddressFormBlock 构造新增地址表单卡片。
func buildAiAssistantAddressFormBlock(orderPayload map[string]any) map[string]any {
	return map[string]any{
		"type":          "address_form",
		"title":         "新增收货地址",
		"order_payload": orderPayload,
		"action":        aiAssistantAction(aiAssistantFlowShopping, "address", "create_address", map[string]any{"order_payload": orderPayload}),
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

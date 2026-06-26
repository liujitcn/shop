package flow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	basev1 "shop/api/gen/go/base/v1"
	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/agent/assistant"
	einoWorkflow "shop/pkg/agent/eino/workflow"
	"shop/pkg/errorsx"
)

const (
	adminFlowWorkspaceOverview   = string(einoWorkflow.AdminFlowWorkspaceOverview)
	adminFlowPendingShipment     = string(einoWorkflow.AdminFlowPendingShipment)
	adminFlowCommentReview       = string(einoWorkflow.AdminFlowCommentReview)
	adminFlowGoodsInventoryAlert = string(einoWorkflow.AdminFlowGoodsInventoryAlert)
	adminFlowOrderRefund         = string(einoWorkflow.AdminFlowOrderRefund)
	adminFlowGoodsAnalytics      = string(einoWorkflow.AdminFlowGoodsAnalytics)
	adminFlowOrderAnalytics      = string(einoWorkflow.AdminFlowOrderAnalytics)
	adminFlowStoreAudit          = string(einoWorkflow.AdminFlowStoreAudit)
	adminFlowRecommendDashboard  = string(einoWorkflow.AdminFlowRecommendDashboard)
	adminFlowReputationInsight   = string(einoWorkflow.AdminFlowReputationInsight)
	adminFlowPayBillCheck        = string(einoWorkflow.AdminFlowPayBillCheck)
	adminFlowReportOverview      = string(einoWorkflow.AdminFlowReportOverview)
)

const (
	adminToolSummaryWorkspaceMetrics      = "admin_v1_workspace_service_summary_workspace_metrics"
	adminToolSummaryWorkspaceTodo         = "admin_v1_workspace_service_summary_workspace_todo"
	adminToolSummaryWorkspaceRisk         = "admin_v1_workspace_service_summary_workspace_risk"
	adminToolSummaryWorkspaceReputation   = "admin_v1_workspace_service_summary_workspace_reputation"
	adminToolListWorkspacePendingComments = "admin_v1_workspace_service_list_workspace_pending_comments"
	adminToolPageOrderInfos               = "admin_v1_order_info_service_page_order_infos"
	adminToolGetOrderInfo                 = "admin_v1_order_info_service_get_order_info"
	adminToolGetOrderInfoRefund           = "admin_v1_order_info_service_get_order_info_refund"
	adminToolGetOrderInfoShipment         = "admin_v1_order_info_service_get_order_info_shipment"
	adminToolShipOrderInfo                = "admin_v1_order_info_service_ship_order_info"
	adminToolPageCommentInfos             = "admin_v1_comment_info_service_page_comment_infos"
	adminToolGetCommentInfo               = "admin_v1_comment_info_service_get_comment_info"
	adminToolSetCommentInfoStatus         = "admin_v1_comment_info_service_set_comment_info_status"
	adminToolPageGoodsInfos               = "admin_v1_goods_info_service_page_goods_infos"
	adminToolGetGoodsInfo                 = "admin_v1_goods_info_service_get_goods_info"
	adminToolSetGoodsInfoStatus           = "admin_v1_goods_info_service_set_goods_info_status"

	// P1: 商品分析工具
	adminToolSummaryGoodsAnalytics = "admin_v1_goods_analytics_service_summary_goods_analytics"
	adminToolRankGoodsAnalytics    = "admin_v1_goods_analytics_service_rank_goods_analytics"
	adminToolTrendGoodsAnalytics   = "admin_v1_goods_analytics_service_trend_goods_analytics"
	adminToolPieGoodsAnalytics     = "admin_v1_goods_analytics_service_pie_goods_analytics"

	// P1: 订单分析工具
	adminToolSummaryOrderAnalytics = "admin_v1_order_analytics_service_summary_order_analytics"
	adminToolTrendOrderAnalytics   = "admin_v1_order_analytics_service_trend_order_analytics"
	adminToolPieOrderAnalytics     = "admin_v1_order_analytics_service_pie_order_analytics"

	// P1: 门店审核工具
	adminToolPageUserStores = "admin_v1_user_store_service_page_user_stores"
	adminToolGetUserStore   = "admin_v1_user_store_service_get_user_store"
	adminToolAuditUserStore = "admin_v1_user_store_service_audit_user_store"

	// P1: 推荐看板工具
	adminToolListDashboardItems = "admin_v1_recommend_gorse_service_list_dashboard_items"
	adminToolListTasks          = "admin_v1_recommend_gorse_service_list_tasks"
	adminToolGetConfig          = "admin_v1_recommend_gorse_service_get_config"

	// P2: 对账检查工具
	adminToolPagePayBills = "admin_v1_pay_bill_service_page_pay_bills"

	// P2: 经营报表工具
	adminToolSummaryOrderMonthReport = "admin_v1_order_report_service_summary_order_month_report"
	adminToolSummaryGoodsMonthReport = "admin_v1_goods_report_service_summary_goods_month_report"
	adminToolSummaryUserAnalytics    = "admin_v1_user_analytics_service_summary_user_analytics"
)

var adminFlowRegistry = einoWorkflow.MustNewAdminRegistry[*assistant.Response]()

// AdminRunner 编排管理端助手闭环流程。
type AdminRunner struct {
	runtime  *assistant.Runtime
	terminal int32
}

// GenerateAdminReply 生成管理端闭环流程回复。
func GenerateAdminReply(
	ctx context.Context,
	runtime *assistant.Runtime,
	terminal int32,
	content string,
	action *basev1.AiAssistantAction,
) (*assistant.Response, bool, error) {
	runner := &AdminRunner{runtime: runtime, terminal: terminal}
	if action != nil && action.GetType() != "" {
		reply, err := runner.handleAdminFlowAction(ctx, action)
		return reply, true, err
	}

	flow := matchAdminFlowIntent(content)
	if flow == "" {
		return nil, false, nil
	}
	reply, err := runner.handleAdminFlowAction(ctx, &basev1.AiAssistantAction{
		Flow: flow,
		Type: openAdminFlowActionType(flow),
	})
	return reply, true, err
}

// IsAdminEntryAction 判断动作是否为管理端固定流程入口。
func IsAdminEntryAction(flow string, actionType string) bool {
	return actionType != "" && adminFlowRegistry.EntryAction(einoWorkflow.FlowName(flow)) == actionType
}

// handleAdminFlowAction 推进管理端闭环流程。
func (r *AdminRunner) handleAdminFlowAction(ctx context.Context, action *basev1.AiAssistantAction) (*assistant.Response, error) {
	payload, err := parseAiAssistantActionPayload(action.GetPayloadJson())
	if err != nil {
		return nil, err
	}
	var result einoWorkflow.ActionResult[*assistant.Response]
	result, err = adminFlowRegistry.Run(ctx, einoWorkflow.ActionRequest{
		Flow:       einoWorkflow.FlowName(action.GetFlow()),
		ActionType: action.GetType(),
		Payload:    payload,
	}, r.ExecuteAdminWorkflowAction)
	if err != nil {
		return nil, err
	}
	// 固定流程动作先经过 Eino Graph 路由，避免前端传入未注册动作直接进入业务分支。
	if action.GetType() != "" && !result.Found {
		return nil, errorsx.InvalidArgument("管理端助手动作不支持")
	}
	if result.Output == nil {
		return nil, errorsx.Internal("管理端助手动作结果无效")
	}
	return result.Output, nil
}

// ExecuteAdminWorkflowAction 执行 Eino Graph 路由后的管理端流程动作。
func (r *AdminRunner) ExecuteAdminWorkflowAction(ctx context.Context, action einoWorkflow.Action, payload map[string]any) (*assistant.Response, error) {
	switch action.Type {
	// P0: 经营总览
	case "open_workspace_overview":
		return r.openAdminWorkspaceOverviewFlow(ctx)
	// P0: 待发货
	case "open_pending_shipment":
		return r.openAdminPendingShipmentFlow(ctx)
	case "view_shipment_detail":
		return r.viewAdminShipmentDetail(ctx, payload)
	case "confirm_shipment":
		return r.confirmAdminShipment(ctx, payload)
	// P0: 评价审核
	case "open_comment_review":
		return r.openAdminCommentReviewFlow(ctx)
	case "view_comment_detail":
		return r.viewAdminCommentDetail(ctx, payload)
	case "confirm_comment_review":
		return r.confirmAdminCommentReview(ctx, payload)
	// P0: 库存预警
	case "open_goods_inventory_alert":
		return r.openAdminGoodsInventoryAlertFlow(ctx)
	case "view_goods_detail":
		return r.viewAdminGoodsDetail(ctx, payload)
	case "confirm_goods_status":
		return r.confirmAdminGoodsStatus(ctx, payload)
	// P1: 退款记录查看
	case "open_order_refund":
		return r.openAdminOrderRefundFlow(ctx)
	case "view_refund_detail":
		return r.viewAdminRefundDetail(ctx, payload)
	// P1: 商品分析
	case "open_goods_analytics":
		return r.openAdminGoodsAnalyticsFlow(ctx)
	// P1: 订单分析
	case "open_order_analytics":
		return r.openAdminOrderAnalyticsFlow(ctx)
	// P1: 门店审核
	case "open_store_audit":
		return r.openAdminStoreAuditFlow(ctx)
	case "view_store_detail":
		return r.viewAdminStoreDetail(ctx, payload)
	case "confirm_store_audit":
		return r.confirmAdminStoreAudit(ctx, payload)
	// P1: 推荐看板
	case "open_recommend_dashboard":
		return r.openAdminRecommendDashboardFlow(ctx)
	// P2: 口碑洞察
	case "open_reputation_insight":
		return r.openAdminReputationInsightFlow(ctx)
	// P2: 对账检查
	case "open_pay_bill_check":
		return r.openAdminPayBillCheckFlow(ctx)
	// P2: 报表总览
	case "open_report_overview":
		return r.openAdminReportOverviewFlow(ctx)
	default:
		return nil, errorsx.InvalidArgument("管理端助手动作不支持")
	}
}

// adminNotImplementedResponse 构造未实现流程的占位回复。
func (r *AdminRunner) adminNotImplementedResponse(flow string, step string) *assistant.Response {
	return r.adminFlowResponse(flow, step, "该功能正在开发中，敬请期待。", []map[string]any{
		adminNoticeBlock("功能开发中", "该流程尚未实现，后续版本将支持。"),
	}, nil)
}

// =========================================================================
// P0 Flow: 经营总览 (workspace_overview)
// =========================================================================

// openAdminWorkspaceOverviewFlow 打开经营总览流程。
//
// 扇出式查询：同时调用指标、待办、风险三个工具，任一失败不中断其他调用。
func (r *AdminRunner) openAdminWorkspaceOverviewFlow(ctx context.Context) (*assistant.Response, error) {
	tools := make([]assistant.ToolUsage, 0, 3)

	// 扇出调用：指标、待办、风险
	metricsOutput, metricsUsage, metricsErr := r.invokeAdminFlowTool(ctx, adminToolSummaryWorkspaceMetrics, map[string]any{})
	tools = appendAiAssistantFlowTool(tools, metricsUsage)

	todoOutput, todoUsage, todoErr := r.invokeAdminFlowTool(ctx, adminToolSummaryWorkspaceTodo, map[string]any{})
	tools = appendAiAssistantFlowTool(tools, todoUsage)

	riskOutput, riskUsage, riskErr := r.invokeAdminFlowTool(ctx, adminToolSummaryWorkspaceRisk, map[string]any{})
	tools = appendAiAssistantFlowTool(tools, riskUsage)

	// 全部失败才返回错误
	if metricsErr != nil && todoErr != nil && riskErr != nil {
		return r.adminFlowErrorResponse(adminFlowWorkspaceOverview, "overview", tools), nil
	}

	blocks := make([]map[string]any, 0, 3)
	if metricsErr == nil {
		blocks = append(blocks, buildAdminMetricsBlock(metricsOutput))
	}
	if todoErr == nil {
		blocks = append(blocks, buildAdminTodoBlock(todoOutput))
	}
	if riskErr == nil {
		blocks = append(blocks, buildAdminRiskBlock(riskOutput))
	}

	return r.adminFlowResponse(adminFlowWorkspaceOverview, "overview", "经营总览数据已加载。", blocks, tools), nil
}

// buildAdminMetricsBlock 构造经营指标卡片。
func buildAdminMetricsBlock(output map[string]any) map[string]any {
	items := adminMetricItems(output, []adminMetricField{
		{label: "今日订单", key: "today_order_count", unit: "单"},
		{label: "今日成交额", key: "today_sale_amount", unit: "元", format: formatAmount},
		{label: "客单价", key: "average_order_amount", unit: "元", format: formatAmount},
		{label: "支付转化率", key: "pay_conversion_rate", unit: "‰"},
		{label: "今日下单用户", key: "today_order_user_count", unit: "人"},
		{label: "今日新增用户", key: "today_new_user_count", unit: "人"},
		{label: "今日销量", key: "today_sale_count", unit: "件"},
		{label: "动销商品", key: "active_goods_count", unit: "个"},
		{label: "今日评价", key: "today_comment_count", unit: "条"},
		{label: "近7日平均评分", key: "average_comment_score", unit: "分"},
	})
	return map[string]any{
		"type":  "metric_panel",
		"title": "经营指标",
		"items": items,
	}
}

// buildAdminTodoBlock 构造待办事项卡片。
func buildAdminTodoBlock(output map[string]any) map[string]any {
	items := adminMetricItems(output, []adminMetricField{
		{label: "待支付订单", key: "pending_pay_order_count", unit: "单"},
		{label: "待发货订单", key: "pending_shipped_order_count", unit: "单"},
		{label: "低库存SKU", key: "low_inventory_sku_count", unit: "个"},
		{label: "待上架商品", key: "pending_put_on_goods_count", unit: "个"},
		{label: "待审核评价", key: "pending_comment_count", unit: "条"},
		{label: "待审核讨论", key: "pending_comment_discussion_count", unit: "条"},
	})
	return map[string]any{
		"type":  "todo_list",
		"title": "待办事项",
		"items": items,
	}
}

// buildAdminRiskBlock 构造风险预警卡片。
func buildAdminRiskBlock(output map[string]any) map[string]any {
	items := adminMetricItems(output, []adminMetricField{
		{label: "对账异常", key: "abnormal_pay_bill_count", unit: "笔"},
		{label: "零库存仍上架", key: "zero_inventory_put_on_sku_count", unit: "个"},
		{label: "价格异常", key: "abnormal_price_sku_count", unit: "个"},
		{label: "近7日低分评价", key: "low_score_comment_count", unit: "条"},
	})
	return map[string]any{
		"type":  "risk_alerts",
		"title": "风险预警",
		"items": items,
	}
}

// =========================================================================
// P0 Flow: 待发货 (pending_shipment)
// =========================================================================

// openAdminPendingShipmentFlow 打开待发货流程。
func (r *AdminRunner) openAdminPendingShipmentFlow(ctx context.Context) (*assistant.Response, error) {
	output, usage, err := r.invokeAdminFlowTool(ctx, adminToolPageOrderInfos, map[string]any{
		"status":    int(commonv1.OrderStatus_PAID),
		"page_num":  1,
		"page_size": 5,
	})
	tools := appendAiAssistantFlowTool(nil, usage)
	if err != nil {
		return r.adminFlowErrorResponse(adminFlowPendingShipment, "list", tools), nil
	}
	block := buildAdminOrderListBlock(adminFlowPendingShipment, "待发货订单", output, "view_shipment_detail")
	return r.adminFlowResponse(adminFlowPendingShipment, "list", "这些订单等待发货，选择一个查看详情并填写物流信息。", []map[string]any{block}, tools), nil
}

// viewAdminShipmentDetail 查看订单发货详情。
func (r *AdminRunner) viewAdminShipmentDetail(ctx context.Context, payload map[string]any) (*assistant.Response, error) {
	orderID := int64Value(payload["order_id"])
	if orderID <= 0 {
		return nil, errorsx.InvalidArgument("订单参数不合法")
	}
	output, usage, err := r.invokeAdminFlowTool(ctx, adminToolGetOrderInfoShipment, map[string]any{"id": orderID})
	tools := appendAiAssistantFlowTool(nil, usage)
	if err != nil {
		return r.adminFlowErrorResponse(adminFlowPendingShipment, "detail", tools), nil
	}
	block := buildAdminShipmentFormBlock(output, orderID)
	return r.adminFlowResponse(adminFlowPendingShipment, "detail", "订单详情已加载，填写物流信息后确认发货。", []map[string]any{block}, tools), nil
}

// confirmAdminShipment 确认发货。
func (r *AdminRunner) confirmAdminShipment(ctx context.Context, payload map[string]any) (*assistant.Response, error) {
	orderID := int64Value(payload["order_id"])
	if orderID <= 0 {
		return nil, errorsx.InvalidArgument("订单参数不合法")
	}
	name := stringValue(payload["name"])
	no := stringValue(payload["no"])
	contact := stringValue(payload["contact"])
	if name == "" || no == "" {
		return nil, errorsx.InvalidArgument("物流公司名称和物流单号不能为空")
	}
	_, usage, err := r.invokeAdminFlowTool(ctx, adminToolShipOrderInfo, map[string]any{
		"order_id": orderID,
		"name":     name,
		"no":       no,
		"contact":  contact,
	})
	tools := appendAiAssistantFlowTool(nil, usage)
	if err != nil {
		return r.adminFlowErrorResponse(adminFlowPendingShipment, "confirm", tools), nil
	}
	block := adminSuccessBlock("发货成功", "订单已发货，物流信息已记录。")
	return r.adminFlowResponse(adminFlowPendingShipment, "confirm", "订单已成功发货。", []map[string]any{block}, tools), nil
}

// =========================================================================
// P0 Flow: 评价审核 (comment_review)
// =========================================================================

// openAdminCommentReviewFlow 打开评价审核流程。
func (r *AdminRunner) openAdminCommentReviewFlow(ctx context.Context) (*assistant.Response, error) {
	output, usage, err := r.invokeAdminFlowTool(ctx, adminToolPageCommentInfos, map[string]any{
		"status":    int(commonv1.CommentStatus_PENDING_REVIEW_CS),
		"page_num":  1,
		"page_size": 5,
	})
	tools := appendAiAssistantFlowTool(nil, usage)
	if err != nil {
		return r.adminFlowErrorResponse(adminFlowCommentReview, "list", tools), nil
	}
	block := buildAdminCommentListBlock(adminFlowCommentReview, output)
	return r.adminFlowResponse(adminFlowCommentReview, "list", "这些评价等待审核，选择一个查看详情。", []map[string]any{block}, tools), nil
}

// viewAdminCommentDetail 查看评价详情。
func (r *AdminRunner) viewAdminCommentDetail(ctx context.Context, payload map[string]any) (*assistant.Response, error) {
	commentID := int64Value(payload["comment_id"])
	if commentID <= 0 {
		commentID = int64Value(payload["id"])
	}
	if commentID <= 0 {
		return nil, errorsx.InvalidArgument("评价参数不合法")
	}
	output, usage, err := r.invokeAdminFlowTool(ctx, adminToolGetCommentInfo, map[string]any{"id": commentID})
	tools := appendAiAssistantFlowTool(nil, usage)
	if err != nil {
		return r.adminFlowErrorResponse(adminFlowCommentReview, "detail", tools), nil
	}
	block := buildAdminCommentDetailBlock(adminFlowCommentReview, output)
	return r.adminFlowResponse(adminFlowCommentReview, "detail", "评价详情已加载，可以选择通过或不通过。", []map[string]any{block}, tools), nil
}

// confirmAdminCommentReview 确认评价审核。
func (r *AdminRunner) confirmAdminCommentReview(ctx context.Context, payload map[string]any) (*assistant.Response, error) {
	commentID := int64Value(payload["comment_id"])
	if commentID <= 0 {
		commentID = int64Value(payload["id"])
	}
	if commentID <= 0 {
		return nil, errorsx.InvalidArgument("评价参数不合法")
	}
	status := int64Value(payload["status"])
	if status <= 0 {
		return nil, errorsx.InvalidArgument("审核状态不合法")
	}
	reason := stringValue(payload["reason"])
	_, usage, err := r.invokeAdminFlowTool(ctx, adminToolSetCommentInfoStatus, map[string]any{
		"id":     commentID,
		"status": status,
		"reason": reason,
	})
	tools := appendAiAssistantFlowTool(nil, usage)
	if err != nil {
		return r.adminFlowErrorResponse(adminFlowCommentReview, "confirm", tools), nil
	}
	label := commentStatusLabel(status)
	block := adminSuccessBlock("审核完成", fmt.Sprintf("评价已%s。", label))
	return r.adminFlowResponse(adminFlowCommentReview, "confirm", fmt.Sprintf("评价已%s。", label), []map[string]any{block}, tools), nil
}

// =========================================================================
// P0 Flow: 库存预警 (goods_inventory_alert)
// =========================================================================

// openAdminGoodsInventoryAlertFlow 打开库存预警流程。
func (r *AdminRunner) openAdminGoodsInventoryAlertFlow(ctx context.Context) (*assistant.Response, error) {
	output, usage, err := r.invokeAdminFlowTool(ctx, adminToolPageGoodsInfos, map[string]any{
		"inventory_alert": int(commonv1.GoodsInventoryAlert_LOW_STOCK),
		"page_num":        1,
		"page_size":       5,
	})
	tools := appendAiAssistantFlowTool(nil, usage)
	if err != nil {
		return r.adminFlowErrorResponse(adminFlowGoodsInventoryAlert, "list", tools), nil
	}
	block := buildAdminGoodsAlertListBlock(adminFlowGoodsInventoryAlert, output)
	return r.adminFlowResponse(adminFlowGoodsInventoryAlert, "list", "这些商品库存不足，选择一个查看详情。", []map[string]any{block}, tools), nil
}

// viewAdminGoodsDetail 查看库存预警商品详情。
func (r *AdminRunner) viewAdminGoodsDetail(ctx context.Context, payload map[string]any) (*assistant.Response, error) {
	goodsID := int64Value(payload["goods_id"])
	if goodsID <= 0 {
		goodsID = int64Value(payload["id"])
	}
	if goodsID <= 0 {
		return nil, errorsx.InvalidArgument("商品参数不合法")
	}
	output, usage, err := r.invokeAdminFlowTool(ctx, adminToolGetGoodsInfo, map[string]any{"id": goodsID})
	tools := appendAiAssistantFlowTool(nil, usage)
	if err != nil {
		return r.adminFlowErrorResponse(adminFlowGoodsInventoryAlert, "detail", tools), nil
	}
	block := buildAdminGoodsAlertDetailBlock(adminFlowGoodsInventoryAlert, output)
	return r.adminFlowResponse(adminFlowGoodsInventoryAlert, "detail", "商品详情已加载，可以选择下架或补充库存。", []map[string]any{block}, tools), nil
}

// confirmAdminGoodsStatus 确认商品状态变更。
func (r *AdminRunner) confirmAdminGoodsStatus(ctx context.Context, payload map[string]any) (*assistant.Response, error) {
	goodsID := int64Value(payload["goods_id"])
	if goodsID <= 0 {
		goodsID = int64Value(payload["id"])
	}
	if goodsID <= 0 {
		return nil, errorsx.InvalidArgument("商品参数不合法")
	}
	status := int64Value(payload["status"])
	if status <= 0 {
		status = int64(commonv1.GoodsStatus_PULL_OFF)
	}
	_, usage, err := r.invokeAdminFlowTool(ctx, adminToolSetGoodsInfoStatus, map[string]any{
		"id":     goodsID,
		"status": status,
	})
	tools := appendAiAssistantFlowTool(nil, usage)
	if err != nil {
		return r.adminFlowErrorResponse(adminFlowGoodsInventoryAlert, "confirm", tools), nil
	}
	label := goodsStatusLabel(status)
	block := adminSuccessBlock("操作完成", fmt.Sprintf("商品已%s。", label))
	return r.adminFlowResponse(adminFlowGoodsInventoryAlert, "confirm", fmt.Sprintf("商品已%s。", label), []map[string]any{block}, tools), nil
}

// =========================================================================
// P1 Flow: 退款记录查看 (order_refund)
// =========================================================================

// openAdminOrderRefundFlow 打开退款记录流程。
func (r *AdminRunner) openAdminOrderRefundFlow(ctx context.Context) (*assistant.Response, error) {
	output, usage, err := r.invokeAdminFlowTool(ctx, adminToolPageOrderInfos, map[string]any{
		"status":    int(commonv1.OrderStatus_REFUNDING),
		"page_num":  1,
		"page_size": 5,
	})
	tools := appendAiAssistantFlowTool(nil, usage)
	if err != nil {
		return r.adminFlowErrorResponse(adminFlowOrderRefund, "list", tools), nil
	}
	block := buildAdminOrderListBlock(adminFlowOrderRefund, "退款订单", output, "view_refund_detail")
	return r.adminFlowResponse(adminFlowOrderRefund, "list", "这些订单已退款，选择一个查看退款详情。", []map[string]any{block}, tools), nil
}

// viewAdminRefundDetail 查看退款详情（纯展示，无操作按钮）。
func (r *AdminRunner) viewAdminRefundDetail(ctx context.Context, payload map[string]any) (*assistant.Response, error) {
	orderID := int64Value(payload["order_id"])
	if orderID <= 0 {
		return nil, errorsx.InvalidArgument("订单参数不合法")
	}
	orderOutput, orderUsage, orderErr := r.invokeAdminFlowTool(ctx, adminToolGetOrderInfo, map[string]any{"id": orderID})
	tools := appendAiAssistantFlowTool(nil, orderUsage)
	refundOutput, refundUsage, refundErr := r.invokeAdminFlowTool(ctx, adminToolGetOrderInfoRefund, map[string]any{"id": orderID})
	tools = appendAiAssistantFlowTool(tools, refundUsage)
	if orderErr != nil && refundErr != nil {
		return r.adminFlowErrorResponse(adminFlowOrderRefund, "detail", tools), nil
	}
	block := buildAdminRefundDetailBlock(orderOutput, refundOutput)
	return r.adminFlowResponse(adminFlowOrderRefund, "detail", "退款详情已加载。", []map[string]any{block}, tools), nil
}

// =========================================================================
// P1 Flow: 商品分析 (goods_analytics)
// =========================================================================

// openAdminGoodsAnalyticsFlow 打开商品分析流程。
//
// 扇出式查询：同时调用摘要、排行、趋势、饼图四个工具，任一失败不中断其他调用。
func (r *AdminRunner) openAdminGoodsAnalyticsFlow(ctx context.Context) (*assistant.Response, error) {
	tools := make([]assistant.ToolUsage, 0, 4)
	timeType := int(commonv1.AnalyticsTimeType_ANALYTICS_TIME_TYPE_WEEK)

	summaryOutput, summaryUsage, summaryErr := r.invokeAdminFlowTool(ctx, adminToolSummaryGoodsAnalytics, map[string]any{
		"time_type": timeType,
	})
	tools = appendAiAssistantFlowTool(tools, summaryUsage)

	rankOutput, rankUsage, rankErr := r.invokeAdminFlowTool(ctx, adminToolRankGoodsAnalytics, map[string]any{
		"time_type": timeType,
	})
	tools = appendAiAssistantFlowTool(tools, rankUsage)

	trendOutput, trendUsage, trendErr := r.invokeAdminFlowTool(ctx, adminToolTrendGoodsAnalytics, map[string]any{
		"time_type": timeType,
	})
	tools = appendAiAssistantFlowTool(tools, trendUsage)

	pieOutput, pieUsage, pieErr := r.invokeAdminFlowTool(ctx, adminToolPieGoodsAnalytics, map[string]any{
		"time_type": timeType,
	})
	tools = appendAiAssistantFlowTool(tools, pieUsage)

	if summaryErr != nil && rankErr != nil && trendErr != nil && pieErr != nil {
		return r.adminFlowErrorResponse(adminFlowGoodsAnalytics, "overview", tools), nil
	}

	blocks := make([]map[string]any, 0, 4)
	if summaryErr == nil {
		blocks = append(blocks, buildAdminAnalyticsSummaryBlock("商品分析摘要", summaryOutput, []adminMetricField{
			{label: "新增商品", key: "new_goods_count", unit: "个"},
			{label: "上架占比", key: "put_on_goods_rate", unit: "%"},
			{label: "活跃商品", key: "active_goods_count", unit: "个"},
			{label: "活跃占比", key: "active_goods_rate", unit: "%"},
			{label: "销售件数", key: "sale_count", unit: "件"},
			{label: "销售增长率", key: "sale_growth_rate", unit: "%"},
			{label: "浏览次数", key: "view_count", unit: "次"},
			{label: "收藏次数", key: "collect_count", unit: "次"},
			{label: "加购件数", key: "cart_count", unit: "件"},
			{label: "下单次数", key: "order_count", unit: "次"},
			{label: "支付次数", key: "pay_count", unit: "次"},
			{label: "支付金额", key: "pay_amount", unit: "元", format: formatAmount},
			{label: "浏览加购转化率", key: "cart_conversion_rate", unit: "%"},
			{label: "加购下单转化率", key: "order_conversion_rate", unit: "%"},
		}))
	}
	if rankErr == nil {
		blocks = append(blocks, buildAdminAnalyticsRankBlock("商品支付排行", rankOutput))
	}
	if trendErr == nil {
		blocks = append(blocks, buildAdminAnalyticsTrendBlock("商品趋势", trendOutput, analyticsTimeTypeLabel(timeType)))
	}
	if pieErr == nil {
		blocks = append(blocks, buildAdminAnalyticsPieBlock("商品分类分布", pieOutput))
	}

	return r.adminFlowResponse(adminFlowGoodsAnalytics, "overview", "商品分析数据已加载。", blocks, tools), nil
}

// =========================================================================
// P1 Flow: 订单分析 (order_analytics)
// =========================================================================

// openAdminOrderAnalyticsFlow 打开订单分析流程。
//
// 扇出式查询：同时调用摘要、趋势、饼图三个工具，任一失败不中断其他调用。
func (r *AdminRunner) openAdminOrderAnalyticsFlow(ctx context.Context) (*assistant.Response, error) {
	tools := make([]assistant.ToolUsage, 0, 3)
	timeType := int(commonv1.AnalyticsTimeType_ANALYTICS_TIME_TYPE_WEEK)

	summaryOutput, summaryUsage, summaryErr := r.invokeAdminFlowTool(ctx, adminToolSummaryOrderAnalytics, map[string]any{
		"time_type": timeType,
	})
	tools = appendAiAssistantFlowTool(tools, summaryUsage)

	trendOutput, trendUsage, trendErr := r.invokeAdminFlowTool(ctx, adminToolTrendOrderAnalytics, map[string]any{
		"time_type": timeType,
	})
	tools = appendAiAssistantFlowTool(tools, trendUsage)

	pieOutput, pieUsage, pieErr := r.invokeAdminFlowTool(ctx, adminToolPieOrderAnalytics, map[string]any{
		"time_type": timeType,
	})
	tools = appendAiAssistantFlowTool(tools, pieUsage)

	if summaryErr != nil && trendErr != nil && pieErr != nil {
		return r.adminFlowErrorResponse(adminFlowOrderAnalytics, "overview", tools), nil
	}

	blocks := make([]map[string]any, 0, 3)
	if summaryErr == nil {
		blocks = append(blocks, buildAdminAnalyticsSummaryBlock("订单分析摘要", summaryOutput, []adminMetricField{
			{label: "新增订单", key: "new_order_count", unit: "单"},
			{label: "订单增长率", key: "new_order_growth_rate", unit: "%"},
			{label: "销售额", key: "sale_amount", unit: "元", format: formatAmount},
			{label: "客单价", key: "average_order_amount", unit: "元", format: formatAmount},
			{label: "下单用户", key: "order_user_count", unit: "人"},
			{label: "复购率", key: "repurchase_rate", unit: "%"},
		}))
	}
	if trendErr == nil {
		blocks = append(blocks, buildAdminAnalyticsTrendBlock("订单趋势", trendOutput, analyticsTimeTypeLabel(timeType)))
	}
	if pieErr == nil {
		blocks = append(blocks, buildAdminAnalyticsPieBlock("订单状态分布", pieOutput))
	}

	return r.adminFlowResponse(adminFlowOrderAnalytics, "overview", "订单分析数据已加载。", blocks, tools), nil
}

// =========================================================================
// P1 Flow: 门店入驻审核 (store_audit)
// =========================================================================

// openAdminStoreAuditFlow 打开门店审核流程。
func (r *AdminRunner) openAdminStoreAuditFlow(ctx context.Context) (*assistant.Response, error) {
	output, usage, err := r.invokeAdminFlowTool(ctx, adminToolPageUserStores, map[string]any{
		"status":    int(commonv1.UserStoreStatus_PENDING_REVIEW),
		"page_num":  1,
		"page_size": 5,
	})
	tools := appendAiAssistantFlowTool(nil, usage)
	if err != nil {
		return r.adminFlowErrorResponse(adminFlowStoreAudit, "list", tools), nil
	}
	block := buildAdminStoreAuditListBlock(adminFlowStoreAudit, output)
	return r.adminFlowResponse(adminFlowStoreAudit, "list", "这些门店等待审核，选择一个查看详情。", []map[string]any{block}, tools), nil
}

// viewAdminStoreDetail 查看门店审核详情。
func (r *AdminRunner) viewAdminStoreDetail(ctx context.Context, payload map[string]any) (*assistant.Response, error) {
	storeID := int64Value(payload["store_id"])
	if storeID <= 0 {
		storeID = int64Value(payload["id"])
	}
	if storeID <= 0 {
		return nil, errorsx.InvalidArgument("门店参数不合法")
	}
	output, usage, err := r.invokeAdminFlowTool(ctx, adminToolGetUserStore, map[string]any{"id": storeID})
	tools := appendAiAssistantFlowTool(nil, usage)
	if err != nil {
		return r.adminFlowErrorResponse(adminFlowStoreAudit, "detail", tools), nil
	}
	block := buildAdminStoreDetailBlock(adminFlowStoreAudit, output)
	return r.adminFlowResponse(adminFlowStoreAudit, "detail", "门店详情已加载，可以选择通过或拒绝。", []map[string]any{block}, tools), nil
}

// confirmAdminStoreAudit 确认门店审核。
func (r *AdminRunner) confirmAdminStoreAudit(ctx context.Context, payload map[string]any) (*assistant.Response, error) {
	storeID := int64Value(payload["store_id"])
	if storeID <= 0 {
		storeID = int64Value(payload["id"])
	}
	if storeID <= 0 {
		return nil, errorsx.InvalidArgument("门店参数不合法")
	}
	status := int64Value(payload["status"])
	if status <= 0 {
		return nil, errorsx.InvalidArgument("审核状态不合法")
	}
	remark := stringValue(payload["remark"])
	_, usage, err := r.invokeAdminFlowTool(ctx, adminToolAuditUserStore, map[string]any{
		"id":     storeID,
		"status": status,
		"remark": remark,
	})
	tools := appendAiAssistantFlowTool(nil, usage)
	if err != nil {
		return r.adminFlowErrorResponse(adminFlowStoreAudit, "confirm", tools), nil
	}
	label := storeAuditStatusLabel(status)
	block := adminSuccessBlock("审核完成", fmt.Sprintf("门店已%s。", label))
	return r.adminFlowResponse(adminFlowStoreAudit, "confirm", fmt.Sprintf("门店已%s。", label), []map[string]any{block}, tools), nil
}

// =========================================================================
// P1 Flow: 推荐效果总览 (recommend_dashboard)
// =========================================================================

// openAdminRecommendDashboardFlow 打开推荐效果总览流程。
//
// 扇出式查询：同时调用仪表盘、任务状态、配置三个工具，任一失败不中断其他调用。
func (r *AdminRunner) openAdminRecommendDashboardFlow(ctx context.Context) (*assistant.Response, error) {
	tools := make([]assistant.ToolUsage, 0, 3)

	dashboardOutput, dashboardUsage, dashboardErr := r.invokeAdminFlowTool(ctx, adminToolListDashboardItems, map[string]any{})
	tools = appendAiAssistantFlowTool(tools, dashboardUsage)

	tasksOutput, tasksUsage, tasksErr := r.invokeAdminFlowTool(ctx, adminToolListTasks, map[string]any{})
	tools = appendAiAssistantFlowTool(tools, tasksUsage)

	configOutput, configUsage, configErr := r.invokeAdminFlowTool(ctx, adminToolGetConfig, map[string]any{})
	tools = appendAiAssistantFlowTool(tools, configUsage)

	if dashboardErr != nil && tasksErr != nil && configErr != nil {
		return r.adminFlowErrorResponse(adminFlowRecommendDashboard, "overview", tools), nil
	}

	blocks := make([]map[string]any, 0, 3)
	if dashboardErr == nil {
		blocks = append(blocks, buildAdminRecommendDashboardItemsBlock(dashboardOutput))
	}
	if tasksErr == nil {
		blocks = append(blocks, buildAdminRecommendTaskListBlock(tasksOutput))
	}
	if configErr == nil {
		blocks = append(blocks, buildAdminGorseConfigBlock(configOutput))
	}

	return r.adminFlowResponse(adminFlowRecommendDashboard, "overview", "推荐效果数据已加载。", blocks, tools), nil
}

// =========================================================================
// P2 Flow: 口碑洞察 (reputation_insight)
// =========================================================================

// openAdminReputationInsightFlow 打开口碑洞察流程。
func (r *AdminRunner) openAdminReputationInsightFlow(ctx context.Context) (*assistant.Response, error) {
	output, usage, err := r.invokeAdminFlowTool(ctx, adminToolSummaryWorkspaceReputation, map[string]any{})
	tools := appendAiAssistantFlowTool(nil, usage)
	if err != nil {
		return r.adminFlowErrorResponse(adminFlowReputationInsight, "overview", tools), nil
	}
	block := buildAdminReputationBlock(output)
	return r.adminFlowResponse(adminFlowReputationInsight, "overview", "口碑洞察数据已加载。", []map[string]any{block}, tools), nil
}

// =========================================================================
// P2 Flow: 对账检查 (pay_bill_check)
// =========================================================================

// openAdminPayBillCheckFlow 打开对账检查流程。
func (r *AdminRunner) openAdminPayBillCheckFlow(ctx context.Context) (*assistant.Response, error) {
	output, usage, err := r.invokeAdminFlowTool(ctx, adminToolPagePayBills, map[string]any{
		"status":    int(commonv1.PayBillStatus_HAS_ERROR),
		"page_num":  1,
		"page_size": 5,
	})
	tools := appendAiAssistantFlowTool(nil, usage)
	if err != nil {
		return r.adminFlowErrorResponse(adminFlowPayBillCheck, "list", tools), nil
	}
	block := buildAdminPayBillListBlock(adminFlowPayBillCheck, output)
	return r.adminFlowResponse(adminFlowPayBillCheck, "list", "这些账单存在对账异常，请核查。", []map[string]any{block}, tools), nil
}

// =========================================================================
// P2 Flow: 经营报表总览 (report_overview)
// =========================================================================

// openAdminReportOverviewFlow 打开经营报表总览流程。
//
// 扇出式查询：同时调用订单月报、商品月报、用户分析三个工具，任一失败不中断其他调用。
func (r *AdminRunner) openAdminReportOverviewFlow(ctx context.Context) (*assistant.Response, error) {
	tools := make([]assistant.ToolUsage, 0, 3)

	orderOutput, orderUsage, orderErr := r.invokeAdminFlowTool(ctx, adminToolSummaryOrderMonthReport, map[string]any{})
	tools = appendAiAssistantFlowTool(tools, orderUsage)

	goodsOutput, goodsUsage, goodsErr := r.invokeAdminFlowTool(ctx, adminToolSummaryGoodsMonthReport, map[string]any{})
	tools = appendAiAssistantFlowTool(tools, goodsUsage)

	userOutput, userUsage, userErr := r.invokeAdminFlowTool(ctx, adminToolSummaryUserAnalytics, map[string]any{
		"time_type": int(commonv1.AnalyticsTimeType_ANALYTICS_TIME_TYPE_MONTH),
	})
	tools = appendAiAssistantFlowTool(tools, userUsage)

	if orderErr != nil && goodsErr != nil && userErr != nil {
		return r.adminFlowErrorResponse(adminFlowReportOverview, "overview", tools), nil
	}

	blocks := make([]map[string]any, 0, 3)
	if orderErr == nil {
		blocks = append(blocks, buildAdminOrderReportBlock(orderOutput))
	}
	if goodsErr == nil {
		blocks = append(blocks, buildAdminGoodsReportBlock(goodsOutput))
	}
	if userErr == nil {
		blocks = append(blocks, buildAdminUserAnalyticsBlock(userOutput))
	}

	return r.adminFlowResponse(adminFlowReportOverview, "overview", "经营报表数据已加载。", blocks, tools), nil
}

// =========================================================================
// 管理端 Flow 辅助方法
// =========================================================================

// invokeAdminFlowTool 通过生成的 Agent Tool 调用管理端业务能力。
func (r *AdminRunner) invokeAdminFlowTool(ctx context.Context, name string, input map[string]any) (map[string]any, assistant.ToolUsage, error) {
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

// adminFlowResponse 构造管理端流程回复。
func (r *AdminRunner) adminFlowResponse(flow string, step string, content string, blocks []map[string]any, tools []assistant.ToolUsage) *assistant.Response {
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

// adminFlowErrorResponse 构造管理端流程失败提示。
func (r *AdminRunner) adminFlowErrorResponse(flow string, step string, tools []assistant.ToolUsage) *assistant.Response {
	return r.adminFlowResponse(flow, step, "这个操作暂时没有完成，可以稍后再试或换一种方式继续。", []map[string]any{
		adminNoticeBlock("操作未完成", "当前步骤没有成功返回结果。"),
	}, tools)
}

// matchAdminFlowIntent 根据用户文本识别管理端闭环流程。
func matchAdminFlowIntent(content string) string {
	// 优先识别写操作类流程，避免被分析类意图截走。
	if strings.Contains(content, "待发货") || strings.Contains(content, "发货") {
		return adminFlowPendingShipment
	}
	if strings.Contains(content, "评价审核") || strings.Contains(content, "评论审核") || strings.Contains(content, "评价管理") {
		return adminFlowCommentReview
	}
	if strings.Contains(content, "库存预警") || strings.Contains(content, "库存不足") || strings.Contains(content, "缺货") || strings.Contains(content, "低库存") {
		return adminFlowGoodsInventoryAlert
	}
	if strings.Contains(content, "退款") || strings.Contains(content, "退货") {
		return adminFlowOrderRefund
	}
	if strings.Contains(content, "门店审核") || strings.Contains(content, "入驻审核") || strings.Contains(content, "开店审核") {
		return adminFlowStoreAudit
	}
	if strings.Contains(content, "对账") {
		return adminFlowPayBillCheck
	}
	// 分析类流程
	if strings.Contains(content, "经营总览") || strings.Contains(content, "工作台") || strings.Contains(content, "首页") || strings.Contains(content, "概览") {
		return adminFlowWorkspaceOverview
	}
	if strings.Contains(content, "商品分析") || strings.Contains(content, "商品数据") || strings.Contains(content, "商品销量") || strings.Contains(content, "商品统计") {
		return adminFlowGoodsAnalytics
	}
	if strings.Contains(content, "订单分析") || strings.Contains(content, "订单数据") || strings.Contains(content, "订单趋势") || strings.Contains(content, "订单统计") {
		return adminFlowOrderAnalytics
	}
	if strings.Contains(content, "推荐效果") || strings.Contains(content, "推荐数据") || strings.Contains(content, "推荐总览") || strings.Contains(content, "推荐看板") || strings.Contains(content, "推荐系统") || strings.Contains(content, "推荐配置") || strings.Contains(content, "gorse") {
		return adminFlowRecommendDashboard
	}
	if strings.Contains(content, "口碑") || strings.Contains(content, "评价洞察") {
		return adminFlowReputationInsight
	}
	if strings.Contains(content, "报表") || strings.Contains(content, "报告") {
		return adminFlowReportOverview
	}
	return ""
}

// openAdminFlowActionType 返回管理端流程入口动作。
func openAdminFlowActionType(flow string) string {
	if actionType := adminFlowRegistry.EntryAction(einoWorkflow.FlowName(flow)); actionType != "" {
		return actionType
	}
	return adminFlowRegistry.EntryAction(einoWorkflow.FlowName(adminFlowWorkspaceOverview))
}

// =========================================================================
// 管理端 Flow 卡片构造函数
// =========================================================================

// buildAdminOrderListBlock 构造管理端订单列表卡片。
func buildAdminOrderListBlock(flow string, title string, output map[string]any, actionType string) map[string]any {
	orders := sliceMapValue(output["order_infos"])
	items := make([]map[string]any, 0, len(orders))
	for _, item := range orders {
		orderID := int64Value(item["id"])
		payload := map[string]any{"order_id": orderID}
		items = append(items, map[string]any{
			"id":           orderID,
			"order_no":     item["order_no"],
			"pay_money":    item["pay_money"],
			"total_money":  item["total_money"],
			"status":       item["status"],
			"status_label": orderStatusLabel(item["status"]),
			"goods_num":    item["goods_num"],
			"goods":        item["goods"],
			"action":       aiAssistantAction(flow, "detail", actionType, payload),
		})
	}
	return map[string]any{
		"type":   "admin_order_list",
		"title":  title,
		"orders": items,
		"total":  output["total"],
	}
}

// buildAdminShipmentFormBlock 构造发货表单卡片。
func buildAdminShipmentFormBlock(output map[string]any, orderID int64) map[string]any {
	order := mapValue(output["order"])
	if len(order) == 0 {
		order = output
	}
	return map[string]any{
		"type":  "shipment_form",
		"title": "确认发货",
		"order": order,
		"action": aiAssistantAction(adminFlowPendingShipment, "confirm", "confirm_shipment", map[string]any{
			"order_id": orderID,
		}),
	}
}

// buildAdminCommentListBlock 构造评价审核列表卡片。
func buildAdminCommentListBlock(flow string, output map[string]any) map[string]any {
	comments := sliceMapValue(output["comment_infos"])
	items := make([]map[string]any, 0, len(comments))
	for _, item := range comments {
		commentID := int64Value(item["id"])
		payload := map[string]any{"comment_id": commentID}
		items = append(items, map[string]any{
			"id":           commentID,
			"goods_name":   item["goods_name"],
			"user_name":    item["user_name"],
			"content":      item["content"],
			"goods_score":  item["goods_score"],
			"status":       item["status"],
			"status_label": commentStatusLabel(item["status"]),
			"created_at":   item["created_at"],
			"action":       aiAssistantAction(flow, "detail", "view_comment_detail", payload),
		})
	}
	return map[string]any{
		"type":     "comment_list",
		"title":    "待审核评价",
		"comments": items,
		"total":    output["total"],
	}
}

// buildAdminCommentDetailBlock 构造评价详情卡片（含审核按钮）。
func buildAdminCommentDetailBlock(flow string, output map[string]any) map[string]any {
	comment := mapValue(output["comment_info"])
	if len(comment) == 0 {
		comment = output
	}
	commentID := int64Value(comment["id"])
	return map[string]any{
		"type":    "comment_detail",
		"title":   "评价详情",
		"comment": comment,
		"actions": []map[string]any{
			aiAssistantAction(flow, "confirm", "confirm_comment_review", map[string]any{
				"comment_id": commentID,
				"status":     int(commonv1.CommentStatus_APPROVED_CS),
				"reason":     "",
			}),
			aiAssistantAction(flow, "confirm", "confirm_comment_review", map[string]any{
				"comment_id": commentID,
				"status":     int(commonv1.CommentStatus_REJECTED_CS),
				"reason":     "",
			}),
		},
	}
}

// buildAdminGoodsAlertListBlock 构造库存预警商品列表卡片。
func buildAdminGoodsAlertListBlock(flow string, output map[string]any) map[string]any {
	goods := sliceMapValue(output["goods_infos"])
	items := make([]map[string]any, 0, len(goods))
	for _, item := range goods {
		goodsID := int64Value(item["id"])
		payload := map[string]any{"goods_id": goodsID}
		items = append(items, map[string]any{
			"id":              goodsID,
			"name":            item["name"],
			"picture":         item["picture"],
			"price":           item["price"],
			"inventory":       item["inventory"],
			"inventory_alert": item["inventory_alert"],
			"alert_label":     inventoryAlertLabel(item["inventory_alert"]),
			"status":          item["status"],
			"status_label":    goodsStatusLabel(item["status"]),
			"action":          aiAssistantAction(flow, "detail", "view_goods_detail", payload),
		})
	}
	return map[string]any{
		"type":  "goods_alert_list",
		"title": "库存预警商品",
		"goods": items,
		"total": output["total"],
	}
}

// buildAdminGoodsAlertDetailBlock 构造库存预警商品详情卡片（含下架按钮）。
func buildAdminGoodsAlertDetailBlock(flow string, output map[string]any) map[string]any {
	goods := mapValue(output["goods_info"])
	if len(goods) == 0 {
		goods = output
	}
	goodsID := int64Value(goods["id"])
	return map[string]any{
		"type":  "goods_alert_detail",
		"title": "商品详情",
		"goods": goods,
		"actions": []map[string]any{
			aiAssistantAction(flow, "confirm", "confirm_goods_status", map[string]any{
				"goods_id": goodsID,
				"status":   int(commonv1.GoodsStatus_PULL_OFF),
			}),
		},
	}
}

// =========================================================================
// P1 Flow 卡片构造函数
// =========================================================================

// buildAdminRefundDetailBlock 构造退款详情卡片（纯展示，无操作按钮）。
func buildAdminRefundDetailBlock(orderOutput map[string]any, refundOutput map[string]any) map[string]any {
	order := mapValue(orderOutput["order_info"])
	if len(order) == 0 {
		order = orderOutput
	}
	refund := mapValue(refundOutput["order_info_refund"])
	if len(refund) == 0 {
		refund = refundOutput
	}
	return map[string]any{
		"type":   "refund_detail",
		"title":  "退款详情",
		"order":  order,
		"refund": refund,
	}
}

// buildAdminAnalyticsSummaryBlock 构造分析摘要指标卡片。
func buildAdminAnalyticsSummaryBlock(title string, output map[string]any, fields []adminMetricField) map[string]any {
	items := adminMetricItems(output, fields)
	return map[string]any{
		"type":  "analytics_summary",
		"title": title,
		"items": items,
	}
}

// buildAdminAnalyticsRankBlock 构造分析排行卡片。
func buildAdminAnalyticsRankBlock(title string, output map[string]any) map[string]any {
	rawItems := sliceMapValue(output["items"])
	items := make([]map[string]any, 0, len(rawItems))
	for _, item := range rawItems {
		items = append(items, map[string]any{
			"name":  item["name"],
			"value": item["value"],
		})
	}
	return map[string]any{
		"type":  "analytics_rank_list",
		"title": title,
		"items": items,
	}
}

// buildAdminAnalyticsTrendBlock 构造分析趋势卡片。
func buildAdminAnalyticsTrendBlock(title string, output map[string]any, timeLabel string) map[string]any {
	axis := make([]string, 0)
	for _, item := range sliceAnyValue(output["axis"]) {
		axis = append(axis, stringValue(item))
	}
	rawSeries := sliceMapValue(output["series"])
	series := make([]map[string]any, 0, len(rawSeries))
	for _, item := range rawSeries {
		data := make([]int64, 0)
		for _, d := range sliceAnyValue(item["data"]) {
			data = append(data, int64Value(d))
		}
		series = append(series, map[string]any{
			"name":         item["name"],
			"type":         item["type"],
			"data":         data,
			"y_axis_index": item["y_axis_index"],
		})
	}
	yAxisNames := make([]string, 0)
	for _, item := range sliceAnyValue(output["y_axis_names"]) {
		yAxisNames = append(yAxisNames, stringValue(item))
	}
	return map[string]any{
		"type":         "analytics_trend",
		"title":        title,
		"time_label":   timeLabel,
		"axis":         axis,
		"series":       series,
		"y_axis_names": yAxisNames,
	}
}

// buildAdminAnalyticsPieBlock 构造分析饼图卡片。
func buildAdminAnalyticsPieBlock(title string, output map[string]any) map[string]any {
	rawItems := sliceMapValue(output["items"])
	items := make([]map[string]any, 0, len(rawItems))
	for _, item := range rawItems {
		items = append(items, map[string]any{
			"name":  item["name"],
			"value": item["value"],
		})
	}
	return map[string]any{
		"type":  "analytics_pie",
		"title": title,
		"items": items,
	}
}

// buildAdminStoreAuditListBlock 构造门店审核列表卡片。
func buildAdminStoreAuditListBlock(flow string, output map[string]any) map[string]any {
	stores := sliceMapValue(output["user_stores"])
	items := make([]map[string]any, 0, len(stores))
	for _, item := range stores {
		storeID := int64Value(item["id"])
		payload := map[string]any{"store_id": storeID}
		items = append(items, map[string]any{
			"id":           storeID,
			"name":         item["name"],
			"nick_name":    item["nick_name"],
			"detail":       item["detail"],
			"status":       item["status"],
			"status_label": storeAuditStatusLabel(item["status"]),
			"remark":       item["remark"],
			"action":       aiAssistantAction(flow, "detail", "view_store_detail", payload),
		})
	}
	return map[string]any{
		"type":   "store_audit_list",
		"title":  "待审核门店",
		"stores": items,
		"total":  output["total"],
	}
}

// buildAdminStoreDetailBlock 构造门店详情卡片（含审核按钮）。
func buildAdminStoreDetailBlock(flow string, output map[string]any) map[string]any {
	store := mapValue(output["user_store"])
	if len(store) == 0 {
		store = output
	}
	storeID := int64Value(store["id"])
	return map[string]any{
		"type":         "store_detail",
		"title":        "门店详情",
		"store":        store,
		"status_label": storeAuditStatusLabel(store["status"]),
		"actions": []map[string]any{
			aiAssistantAction(flow, "confirm", "confirm_store_audit", map[string]any{
				"store_id": storeID,
				"status":   int(commonv1.UserStoreStatus_APPROVED),
				"remark":   "",
			}),
			aiAssistantAction(flow, "confirm", "confirm_store_audit", map[string]any{
				"store_id": storeID,
				"status":   int(commonv1.UserStoreStatus_FAILED_REVIEW),
				"remark":   "",
			}),
		},
	}
}

// buildAdminRecommendDashboardItemsBlock 构造推荐仪表盘商品列表卡片。
func buildAdminRecommendDashboardItemsBlock(output map[string]any) map[string]any {
	items := sliceMapValue(output["items"])
	return map[string]any{
		"type":          "recommend_dashboard_items",
		"title":         "推荐商品",
		"items":         items,
		"last_modified": output["last_modified"],
	}
}

// buildAdminRecommendTaskListBlock 构造推荐任务状态卡片。
func buildAdminRecommendTaskListBlock(output map[string]any) map[string]any {
	tasks := sliceMapValue(output["tasks"])
	return map[string]any{
		"type":  "recommend_task_list",
		"title": "推荐任务状态",
		"tasks": tasks,
	}
}

// buildAdminGorseConfigBlock 构造 Gorse 推荐配置卡片（脱敏处理）。
func buildAdminGorseConfigBlock(output map[string]any) map[string]any {
	safeConfig := redactGorseConfig(output)
	return map[string]any{
		"type":   "gorse_config",
		"title":  "推荐系统配置",
		"config": safeConfig,
	}
}

// =========================================================================
// P2 Flow 卡片构造函数
// =========================================================================

// buildAdminReputationBlock 构造口碑洞察卡片。
func buildAdminReputationBlock(output map[string]any) map[string]any {
	score := int64Value(output["average_comment_score"])
	scoreText := fmt.Sprintf("%d.%d", score/10, score%10)
	rawTags := sliceMapValue(output["hot_tags"])
	tags := make([]map[string]any, 0, len(rawTags))
	for _, tag := range rawTags {
		tags = append(tags, map[string]any{
			"name":          tag["name"],
			"mention_count": tag["mention_count"],
		})
	}
	return map[string]any{
		"type":            "reputation_insight",
		"title":           "口碑洞察",
		"average_score":   scoreText,
		"hot_tags":        tags,
		"comment_summary": output["comment_summary"],
	}
}

// buildAdminPayBillListBlock 构造对账异常列表卡片。
func buildAdminPayBillListBlock(flow string, output map[string]any) map[string]any {
	bills := sliceMapValue(output["pay_bills"])
	items := make([]map[string]any, 0, len(bills))
	for _, bill := range bills {
		items = append(items, map[string]any{
			"id":                 bill["id"],
			"bill_date":          bill["bill_date"],
			"bill_type":          bill["bill_type"],
			"total_count":        bill["total_count"],
			"total_amount":       bill["total_amount"],
			"third_total_count":  bill["third_total_count"],
			"third_total_amount": bill["third_total_amount"],
			"status":             bill["status"],
			"status_label":       payBillStatusLabel(bill["status"]),
		})
	}
	return map[string]any{
		"type":  "pay_bill_list",
		"title": "对账异常账单",
		"bills": items,
		"total": output["total"],
	}
}

// buildAdminOrderReportBlock 构造订单月报汇总卡片。
func buildAdminOrderReportBlock(output map[string]any) map[string]any {
	items := adminMetricItems(output, []adminMetricField{
		{label: "支付订单数", key: "paid_order_count", unit: "单"},
		{label: "支付金额", key: "paid_order_amount", unit: "元", format: formatAmount},
		{label: "退款订单数", key: "refund_order_count", unit: "单"},
		{label: "退款金额", key: "refund_order_amount", unit: "元", format: formatAmount},
		{label: "净销售额", key: "net_order_amount", unit: "元", format: formatAmount},
		{label: "支付用户数", key: "paid_user_count", unit: "人"},
		{label: "商品件数", key: "goods_count", unit: "件"},
		{label: "客单价", key: "customer_unit_price", unit: "元", format: formatAmount},
	})
	return map[string]any{
		"type":  "analytics_summary",
		"title": "订单月报汇总",
		"items": items,
	}
}

// buildAdminGoodsReportBlock 构造商品月报汇总卡片。
func buildAdminGoodsReportBlock(output map[string]any) map[string]any {
	items := adminMetricItems(output, []adminMetricField{
		{label: "浏览次数", key: "view_count", unit: "次"},
		{label: "收藏次数", key: "collect_count", unit: "次"},
		{label: "加购件数", key: "cart_count", unit: "件"},
		{label: "下单次数", key: "order_count", unit: "次"},
		{label: "支付次数", key: "pay_count", unit: "次"},
		{label: "支付件数", key: "pay_goods_num", unit: "件"},
		{label: "支付金额", key: "pay_amount", unit: "元", format: formatAmount},
		{label: "浏览加购转化率", key: "cart_conversion_rate", unit: "%"},
		{label: "加购下单转化率", key: "order_conversion_rate", unit: "%"},
		{label: "浏览支付转化率", key: "pay_conversion_rate", unit: "%"},
		{label: "件均成交价", key: "pay_unit_price", unit: "元", format: formatAmount},
	})
	return map[string]any{
		"type":  "analytics_summary",
		"title": "商品月报汇总",
		"items": items,
	}
}

// buildAdminUserAnalyticsBlock 构造用户分析摘要卡片。
func buildAdminUserAnalyticsBlock(output map[string]any) map[string]any {
	items := adminMetricItems(output, []adminMetricField{
		{label: "新增用户", key: "new_user_count", unit: "人"},
		{label: "新增增长率", key: "new_user_growth_rate", unit: "%"},
		{label: "下单用户", key: "order_user_count", unit: "人"},
		{label: "下单转化率", key: "order_user_conversion_rate", unit: "%"},
		{label: "活跃用户", key: "active_user_count", unit: "人"},
		{label: "活跃覆盖率", key: "active_user_coverage_rate", unit: "%"},
	})
	return map[string]any{
		"type":  "analytics_summary",
		"title": "用户分析摘要",
		"items": items,
	}
}

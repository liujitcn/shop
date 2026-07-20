package aiflow

import einoWorkflow "shop/pkg/agent/eino/workflow"

var appFlowDefinitions = []einoWorkflow.Definition{
	{
		Name:        einoWorkflow.FlowName(aiFlowShopping),
		EntryAction: "open_shopping",
		Actions: []einoWorkflow.Action{
			{Flow: einoWorkflow.FlowName(aiFlowShopping), Step: "goods", Type: "open_shopping"},
			{Flow: einoWorkflow.FlowName(aiFlowShopping), Step: "sku", Type: "select_goods"},
			{Flow: einoWorkflow.FlowName(aiFlowShopping), Step: "checkout", Type: "select_sku"},
			{Flow: einoWorkflow.FlowName(aiFlowShopping), Step: "address", Type: "create_address"},
			{Flow: einoWorkflow.FlowName(aiFlowShopping), Step: "confirm", Type: "select_address"},
			{Flow: einoWorkflow.FlowName(aiFlowShopping), Step: "confirm", Type: "confirm_order"},
			{Flow: einoWorkflow.FlowName(aiFlowShopping), Step: "payment", Type: "start_payment"},
		},
	},
	{Name: einoWorkflow.FlowName(aiFlowPendingPayment), EntryAction: "open_pending_payment", Actions: []einoWorkflow.Action{{Flow: einoWorkflow.FlowName(aiFlowPendingPayment), Step: "list", Type: "open_pending_payment"}, {Flow: einoWorkflow.FlowName(aiFlowPendingPayment), Step: "payment", Type: "start_payment"}}},
	{Name: einoWorkflow.FlowName(aiFlowPendingReview), EntryAction: "open_pending_review", Actions: []einoWorkflow.Action{{Flow: einoWorkflow.FlowName(aiFlowPendingReview), Step: "list", Type: "open_pending_review"}, {Flow: einoWorkflow.FlowName(aiFlowPendingReview), Step: "form", Type: "open_review_form"}, {Flow: einoWorkflow.FlowName(aiFlowPendingReview), Step: "submit", Type: "submit_review"}}},
	{Name: einoWorkflow.FlowName(aiFlowOrderLogistics), EntryAction: "open_order_logistics", Actions: []einoWorkflow.Action{{Flow: einoWorkflow.FlowName(aiFlowOrderLogistics), Step: "list", Type: "open_order_logistics"}, {Flow: einoWorkflow.FlowName(aiFlowOrderLogistics), Step: "detail", Type: "view_order"}, {Flow: einoWorkflow.FlowName(aiFlowOrderLogistics), Step: "receipt", Type: "receive_order"}}},
	{Name: einoWorkflow.FlowName(aiFlowUserCart), EntryAction: "open_user_cart", Actions: []einoWorkflow.Action{{Flow: einoWorkflow.FlowName(aiFlowUserCart), Step: "list", Type: "open_user_cart"}}},
	{Name: einoWorkflow.FlowName(aiFlowUserCollect), EntryAction: "open_user_collect", Actions: []einoWorkflow.Action{{Flow: einoWorkflow.FlowName(aiFlowUserCollect), Step: "list", Type: "open_user_collect"}}},
	{Name: einoWorkflow.FlowName(aiFlowUserAddress), EntryAction: "open_user_address", Actions: []einoWorkflow.Action{{Flow: einoWorkflow.FlowName(aiFlowUserAddress), Step: "list", Type: "open_user_address"}, {Flow: einoWorkflow.FlowName(aiFlowUserAddress), Step: "address", Type: "create_address"}}},
	{Name: einoWorkflow.FlowName(aiFlowUserProfile), EntryAction: "open_user_profile", Actions: []einoWorkflow.Action{{Flow: einoWorkflow.FlowName(aiFlowUserProfile), Step: "detail", Type: "open_user_profile"}}},
	{Name: einoWorkflow.FlowName(aiFlowUserStore), EntryAction: "open_user_store", Actions: []einoWorkflow.Action{{Flow: einoWorkflow.FlowName(aiFlowUserStore), Step: "detail", Type: "open_user_store"}}},
	{Name: einoWorkflow.FlowName(aiFlowGoodsCategory), EntryAction: "open_goods_category", Actions: []einoWorkflow.Action{{Flow: einoWorkflow.FlowName(aiFlowGoodsCategory), Step: "list", Type: "open_goods_category"}, {Flow: einoWorkflow.FlowName(aiFlowGoodsCategory), Step: "goods", Type: "view_goods_category"}}},
	{Name: einoWorkflow.FlowName(aiFlowShopHot), EntryAction: "open_shop_hot", Actions: []einoWorkflow.Action{{Flow: einoWorkflow.FlowName(aiFlowShopHot), Step: "list", Type: "open_shop_hot"}, {Flow: einoWorkflow.FlowName(aiFlowShopHot), Step: "goods", Type: "view_shop_hot_item"}, {Flow: einoWorkflow.FlowName(aiFlowShopHot), Step: "sku", Type: "select_goods"}}},
	{Name: einoWorkflow.FlowName(aiFlowShopService), EntryAction: "open_shop_service", Actions: []einoWorkflow.Action{{Flow: einoWorkflow.FlowName(aiFlowShopService), Step: "list", Type: "open_shop_service"}}},
}

var adminFlowDefinitions = []einoWorkflow.Definition{
	{Name: einoWorkflow.FlowName(adminFlowWorkspaceOverview), EntryAction: "open_workspace_overview", Actions: []einoWorkflow.Action{{Flow: einoWorkflow.FlowName(adminFlowWorkspaceOverview), Step: "overview", Type: "open_workspace_overview"}}},
	{Name: einoWorkflow.FlowName(adminFlowPendingShipment), EntryAction: "open_pending_shipment", Actions: []einoWorkflow.Action{{Flow: einoWorkflow.FlowName(adminFlowPendingShipment), Step: "list", Type: "open_pending_shipment"}, {Flow: einoWorkflow.FlowName(adminFlowPendingShipment), Step: "detail", Type: "view_shipment_detail"}, {Flow: einoWorkflow.FlowName(adminFlowPendingShipment), Step: "confirm", Type: "confirm_shipment"}}},
	{Name: einoWorkflow.FlowName(adminFlowCommentReview), EntryAction: "open_comment_review", Actions: []einoWorkflow.Action{{Flow: einoWorkflow.FlowName(adminFlowCommentReview), Step: "list", Type: "open_comment_review"}, {Flow: einoWorkflow.FlowName(adminFlowCommentReview), Step: "detail", Type: "view_comment_detail"}, {Flow: einoWorkflow.FlowName(adminFlowCommentReview), Step: "confirm", Type: "confirm_comment_review"}}},
	{Name: einoWorkflow.FlowName(adminFlowGoodsInventoryAlert), EntryAction: "open_goods_inventory_alert", Actions: []einoWorkflow.Action{{Flow: einoWorkflow.FlowName(adminFlowGoodsInventoryAlert), Step: "list", Type: "open_goods_inventory_alert"}, {Flow: einoWorkflow.FlowName(adminFlowGoodsInventoryAlert), Step: "detail", Type: "view_goods_detail"}, {Flow: einoWorkflow.FlowName(adminFlowGoodsInventoryAlert), Step: "confirm", Type: "confirm_goods_status"}}},
	{Name: einoWorkflow.FlowName(adminFlowOrderRefund), EntryAction: "open_order_refund", Actions: []einoWorkflow.Action{{Flow: einoWorkflow.FlowName(adminFlowOrderRefund), Step: "list", Type: "open_order_refund"}, {Flow: einoWorkflow.FlowName(adminFlowOrderRefund), Step: "detail", Type: "view_refund_detail"}}},
	{Name: einoWorkflow.FlowName(adminFlowGoodsAnalytics), EntryAction: "open_goods_analytics", Actions: []einoWorkflow.Action{{Flow: einoWorkflow.FlowName(adminFlowGoodsAnalytics), Step: "overview", Type: "open_goods_analytics"}}},
	{Name: einoWorkflow.FlowName(adminFlowOrderAnalytics), EntryAction: "open_order_analytics", Actions: []einoWorkflow.Action{{Flow: einoWorkflow.FlowName(adminFlowOrderAnalytics), Step: "overview", Type: "open_order_analytics"}}},
	{Name: einoWorkflow.FlowName(adminFlowStoreAudit), EntryAction: "open_store_audit", Actions: []einoWorkflow.Action{{Flow: einoWorkflow.FlowName(adminFlowStoreAudit), Step: "list", Type: "open_store_audit"}, {Flow: einoWorkflow.FlowName(adminFlowStoreAudit), Step: "detail", Type: "view_store_detail"}, {Flow: einoWorkflow.FlowName(adminFlowStoreAudit), Step: "confirm", Type: "confirm_store_audit"}}},
	{Name: einoWorkflow.FlowName(adminFlowRecommendDashboard), EntryAction: "open_recommend_dashboard", Actions: []einoWorkflow.Action{{Flow: einoWorkflow.FlowName(adminFlowRecommendDashboard), Step: "overview", Type: "open_recommend_dashboard"}}},
	{Name: einoWorkflow.FlowName(adminFlowReputationInsight), EntryAction: "open_reputation_insight", Actions: []einoWorkflow.Action{{Flow: einoWorkflow.FlowName(adminFlowReputationInsight), Step: "overview", Type: "open_reputation_insight"}}},
	{Name: einoWorkflow.FlowName(adminFlowPayBillCheck), EntryAction: "open_pay_bill_check", Actions: []einoWorkflow.Action{{Flow: einoWorkflow.FlowName(adminFlowPayBillCheck), Step: "list", Type: "open_pay_bill_check"}}},
	{Name: einoWorkflow.FlowName(adminFlowReportOverview), EntryAction: "open_report_overview", Actions: []einoWorkflow.Action{{Flow: einoWorkflow.FlowName(adminFlowReportOverview), Step: "overview", Type: "open_report_overview"}}},
}

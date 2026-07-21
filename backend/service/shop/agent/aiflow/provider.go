package aiflow

import (
	"context"

	basev1 "shop/api/gen/go/base/v1"
	"shop/service/base/agent/ai"
)

// Provider 提供商城固定流程、入口校验和快捷入口。
type Provider struct{}

var _ ai.FixedFlowProvider = (*Provider)(nil)

// NewProvider 创建商城固定流程提供者。
func NewProvider() *Provider {
	return &Provider{}
}

// NewRegistration 将商城固定流程显式注册到基础 AI 运行时。
func NewRegistration(runtime *ai.Runtime, provider *Provider) (Registration, error) {
	err := runtime.RegisterFixedFlow(provider)
	if err != nil {
		return Registration{}, err
	}
	return Registration{}, nil
}

// FlowNames 返回商城固定流程标识。
func (p *Provider) FlowNames() []string {
	return []string{
		aiFlowShopping, aiFlowPendingPayment, aiFlowPendingReview, aiFlowOrderLogistics,
		aiFlowUserCart, aiFlowUserCollect, aiFlowUserAddress, aiFlowUserProfile,
		aiFlowUserStore, aiFlowGoodsCategory, aiFlowShopHot, aiFlowShopService,
		adminFlowWorkspaceOverview, adminFlowPendingShipment, adminFlowCommentReview,
		adminFlowGoodsInventoryAlert, adminFlowOrderRefund, adminFlowGoodsAnalytics,
		adminFlowOrderAnalytics, adminFlowStoreAudit, adminFlowRecommendDashboard,
		adminFlowReputationInsight, adminFlowPayBillCheck, adminFlowReportOverview,
	}
}

// GenerateFixedFlowReply 生成商城固定流程回复。
func (p *Provider) GenerateFixedFlowReply(ctx context.Context, runtime *ai.Runtime, terminal int32, content string, action *basev1.AiAction) (*ai.Response, bool, error) {
	return GenerateReply(ctx, runtime, terminal, content, action)
}

// IsFixedFlowEntryAction 判断商城固定流程入口。
func (p *Provider) IsFixedFlowEntryAction(terminal int32, flow string, actionType string) bool {
	return IsEntryAction(terminal, flow, actionType)
}

// Registration 表示商城固定流程已在组合根注册。
type Registration struct{}

// FixedFlowShortcuts 返回商城快捷入口。
func (p *Provider) FixedFlowShortcuts(terminal int32, enabledTools map[string]bool) []*basev1.AiShortcut {
	return BuildShortcuts(terminal, enabledTools)
}
